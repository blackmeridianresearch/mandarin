// Copyright 2024 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

// Package hotcache provides an in-memory cache for frequently-accessed DeFi contract state.
// It maintains decoded state for a watchlist of contracts (e.g., Uniswap pools, Aave markets)
// with sub-microsecond read latency for co-located trading operations.
package hotcache

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

var (
	ErrNotFound        = errors.New("contract not in cache")
	ErrNotWatched      = errors.New("contract not in watchlist")
	ErrInconsistentState = errors.New("cache state inconsistent with canonical state")
)

// Config contains configuration for the hot state cache.
type Config struct {
	// Enabled controls whether the cache is active
	Enabled bool
	
	// Watchlist is the list of contract addresses to cache
	Watchlist []common.Address
	
	// ShadowMode enables validation against canonical state
	// Should be true initially to verify correctness
	ShadowMode bool
	
	// MaxSnapshots is the maximum number of historical snapshots to keep
	// for reorg protection (default: 64)
	MaxSnapshots int
}

// DefaultConfig returns the default configuration.
func DefaultConfig() Config {
	return Config{
		Enabled:      false, // Disabled by default - high risk feature
		Watchlist:    []common.Address{},
		ShadowMode:   true, // Always start in shadow mode
		MaxSnapshots: 64,
	}
}

// Cache maintains an in-memory cache of DeFi contract state.
// It uses copy-on-write snapshots for lock-free reads and atomic updates.
type Cache struct {
	config Config
	
	// Current canonical state (atomic pointer for lock-free reads)
	current atomic.Pointer[Snapshot]
	
	// Historical snapshots for reorg protection, keyed by block hash
	snapshots map[common.Hash]*Snapshot
	snapshotMu sync.RWMutex
	
	// Watchlist map for O(1) lookup
	watchlist map[common.Address]bool
	
	// Decoders for known contract types
	decoders map[common.Address]ContractDecoder
	decoderMu sync.RWMutex
	
	// Statistics
	stats Statistics
}

// Statistics tracks cache performance metrics.
type Statistics struct {
	Hits              atomic.Uint64
	Misses            atomic.Uint64
	Updates           atomic.Uint64
	ValidationErrors  atomic.Uint64
	ReorgCount        atomic.Uint64
}

// Snapshot represents a point-in-time view of cached contract states.
// Snapshots are immutable once published for lock-free reads.
type Snapshot struct {
	BlockNumber uint64
	BlockHash   common.Hash
	BlockTime   uint64
	
	// Contract states keyed by address
	Contracts map[common.Address]*ContractState
}

// ContractState holds the cached state for a single contract.
type ContractState struct {
	Address     common.Address
	Type        ContractType
	
	// Raw storage slots (always populated)
	RawSlots    map[common.Hash]common.Hash
	
	// Decoded state (populated if decoder available)
	Decoded     interface{}
	
	// Metadata
	LastUpdated uint64 // Block number
}

// ContractType identifies the contract type for specialized decoding.
type ContractType uint8

const (
	ContractTypeUnknown ContractType = iota
	ContractTypeUniswapV2
	ContractTypeUniswapV3
	ContractTypeAave
	ContractTypeCurve
)

func (t ContractType) String() string {
	switch t {
	case ContractTypeUniswapV2:
		return "UniswapV2"
	case ContractTypeUniswapV3:
		return "UniswapV3"
	case ContractTypeAave:
		return "Aave"
	case ContractTypeCurve:
		return "Curve"
	default:
		return "Unknown"
	}
}

// New creates a new hot state cache with the given configuration.
func New(config Config) *Cache {
	if config.MaxSnapshots == 0 {
		config.MaxSnapshots = 64
	}
	
	// Build watchlist map
	watchlist := make(map[common.Address]bool, len(config.Watchlist))
	for _, addr := range config.Watchlist {
		watchlist[addr] = true
	}
	
	cache := &Cache{
		config:    config,
		snapshots: make(map[common.Hash]*Snapshot),
		watchlist: watchlist,
		decoders:  make(map[common.Address]ContractDecoder),
	}
	
	// Initialize with empty snapshot
	initial := &Snapshot{
		Contracts: make(map[common.Address]*ContractState),
	}
	cache.current.Store(initial)
	
	if config.Enabled {
		log.Info("Hot state cache initialized",
			"watchlist", len(config.Watchlist),
			"shadowMode", config.ShadowMode,
			"maxSnapshots", config.MaxSnapshots)
	}
	
	return cache
}

// IsEnabled returns whether the cache is enabled.
func (c *Cache) IsEnabled() bool {
	return c.config.Enabled
}

// IsWatched returns whether an address is in the watchlist.
func (c *Cache) IsWatched(addr common.Address) bool {
	return c.watchlist[addr]
}

// RegisterDecoder registers a decoder for a specific contract address.
func (c *Cache) RegisterDecoder(addr common.Address, decoder ContractDecoder) {
	c.decoderMu.Lock()
	defer c.decoderMu.Unlock()
	c.decoders[addr] = decoder
	log.Debug("Registered contract decoder", "address", addr, "type", decoder.Type())
}

// GetSnapshot returns the current cache snapshot.
// This is a lock-free operation using atomic pointer load.
func (c *Cache) GetSnapshot() *Snapshot {
	return c.current.Load()
}

// GetContractState returns the cached state for a specific contract.
// Returns ErrNotFound if the contract is not in the cache.
func (c *Cache) GetContractState(addr common.Address) (*ContractState, error) {
	snapshot := c.GetSnapshot()
	state, ok := snapshot.Contracts[addr]
	if !ok {
		c.stats.Misses.Add(1)
		return nil, ErrNotFound
	}
	c.stats.Hits.Add(1)
	return state, nil
}

// GetRawSlot returns a raw storage slot value for a contract.
func (c *Cache) GetRawSlot(addr common.Address, slot common.Hash) (common.Hash, error) {
	state, err := c.GetContractState(addr)
	if err != nil {
		return common.Hash{}, err
	}
	value, ok := state.RawSlots[slot]
	if !ok {
		return common.Hash{}, fmt.Errorf("slot %s not found", slot.Hex())
	}
	return value, nil
}

// GetStatistics returns the current cache statistics.
func (c *Cache) GetStatistics() Statistics {
	return c.stats
}

// ContractDecoder defines the interface for decoding contract-specific state.
type ContractDecoder interface {
	// Type returns the contract type
	Type() ContractType
	
	// Decode decodes raw storage slots into a structured format
	Decode(slots map[common.Hash]common.Hash) (interface{}, error)
	
	// RequiredSlots returns the storage slots needed for decoding
	RequiredSlots() []common.Hash
}

