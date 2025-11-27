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

package hotcache

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

// StateReader provides read access to the canonical state.
// This interface allows the cache to query state without circular dependencies.
type StateReader interface {
	GetState(addr common.Address, slot common.Hash) common.Hash
}

// Update updates the cache with state from a newly imported block.
// This should be called after a block is written to the canonical chain.
func (c *Cache) Update(block *types.Header, stateDB StateReader) error {
	if !c.config.Enabled {
		return nil
	}
	
	c.stats.Updates.Add(1)
	
	// Create new snapshot
	newSnapshot := &Snapshot{
		BlockNumber: block.Number.Uint64(),
		BlockHash:   block.Hash(),
		BlockTime:   block.Time,
		Contracts:   make(map[common.Address]*ContractState),
	}
	
	// Update state for each watched contract
	for addr := range c.watchlist {
		contractState, err := c.updateContract(addr, stateDB)
		if err != nil {
			log.Warn("Failed to update contract state",
				"address", addr,
				"block", block.Number.Uint64(),
				"err", err)
			continue
		}
		newSnapshot.Contracts[addr] = contractState
	}
	
	// Store snapshot for reorg protection
	c.snapshotMu.Lock()
	c.snapshots[block.Hash()] = newSnapshot
	c.cleanupOldSnapshots(block.Number.Uint64())
	c.snapshotMu.Unlock()
	
	// Atomic update of current snapshot (lock-free for readers)
	c.current.Store(newSnapshot)
	
	log.Debug("Hot cache updated",
		"block", block.Number.Uint64(),
		"hash", block.Hash().Hex()[:10],
		"contracts", len(newSnapshot.Contracts))
	
	return nil
}

// updateContract reads and decodes state for a single contract.
func (c *Cache) updateContract(addr common.Address, stateDB StateReader) (*ContractState, error) {
	contractState := &ContractState{
		Address:  addr,
		Type:     ContractTypeUnknown,
		RawSlots: make(map[common.Hash]common.Hash),
	}
	
	// Get decoder if available
	c.decoderMu.RLock()
	decoder, hasDecoder := c.decoders[addr]
	c.decoderMu.RUnlock()
	
	if hasDecoder {
		contractState.Type = decoder.Type()
		
		// Read required slots
		slots := decoder.RequiredSlots()
		for _, slot := range slots {
			value := stateDB.GetState(addr, slot)
			contractState.RawSlots[slot] = value
		}
		
		// Decode to structured format
		decoded, err := decoder.Decode(contractState.RawSlots)
		if err != nil {
			return nil, fmt.Errorf("failed to decode %s: %w", decoder.Type(), err)
		}
		contractState.Decoded = decoded
		
		log.Trace("Contract state decoded",
			"address", addr,
			"type", decoder.Type(),
			"slots", len(contractState.RawSlots))
	}
	
	return contractState, nil
}

// Validate checks if the cached state matches the canonical state.
// This should be called periodically in shadow mode to verify correctness.
func (c *Cache) Validate(stateDB StateReader) error {
	if !c.config.ShadowMode {
		return nil
	}
	
	snapshot := c.GetSnapshot()
	
	for addr, cachedState := range snapshot.Contracts {
		// Verify each raw slot
		for slot, cachedValue := range cachedState.RawSlots {
			canonicalValue := stateDB.GetState(addr, slot)
			
			if cachedValue != canonicalValue {
				c.stats.ValidationErrors.Add(1)
				return fmt.Errorf("%w: contract=%s slot=%s cached=%s canonical=%s",
					ErrInconsistentState,
					addr.Hex(),
					slot.Hex(),
					cachedValue.Hex(),
					canonicalValue.Hex())
			}
		}
	}
	
	log.Debug("Cache validation passed", "block", snapshot.BlockNumber)
	return nil
}

// ValidateContract validates a specific contract's cached state.
func (c *Cache) ValidateContract(addr common.Address, stateDB StateReader) error {
	if !c.config.ShadowMode {
		return nil
	}
	
	cachedState, err := c.GetContractState(addr)
	if err != nil {
		return err
	}
	
	for slot, cachedValue := range cachedState.RawSlots {
		canonicalValue := stateDB.GetState(addr, slot)
		
		if cachedValue != canonicalValue {
			c.stats.ValidationErrors.Add(1)
			return fmt.Errorf("%w: contract=%s slot=%s cached=%s canonical=%s",
				ErrInconsistentState,
				addr.Hex(),
				slot.Hex(),
				cachedValue.Hex(),
				canonicalValue.Hex())
		}
	}
	
	return nil
}

// cleanupOldSnapshots removes snapshots beyond the retention limit.
// Must be called with snapshotMu held.
func (c *Cache) cleanupOldSnapshots(currentBlock uint64) {
	if uint64(len(c.snapshots)) <= uint64(c.config.MaxSnapshots) {
		return
	}
	
	// Find snapshots to remove (older than currentBlock - MaxSnapshots)
	cutoff := uint64(0)
	if currentBlock > uint64(c.config.MaxSnapshots) {
		cutoff = currentBlock - uint64(c.config.MaxSnapshots)
	}
	
	for hash, snapshot := range c.snapshots {
		if snapshot.BlockNumber < cutoff {
			delete(c.snapshots, hash)
			log.Trace("Removed old snapshot", "block", snapshot.BlockNumber)
		}
	}
}

// HandleReorg handles a chain reorganization by rolling back to a common ancestor
// and replaying the new chain.
func (c *Cache) HandleReorg(oldChain, newChain []*types.Header, stateDB StateReader) error {
	if !c.config.Enabled {
		return nil
	}
	
	c.stats.ReorgCount.Add(1)
	
	log.Warn("Hot cache handling reorg",
		"oldBlocks", len(oldChain),
		"newBlocks", len(newChain))
	
	// Find common ancestor
	var commonHash common.Hash
	for i := len(oldChain) - 1; i >= 0; i-- {
		for j := len(newChain) - 1; j >= 0; j-- {
			if oldChain[i].Hash() == newChain[j].Hash() {
				commonHash = oldChain[i].Hash()
				break
			}
		}
		if commonHash != (common.Hash{}) {
			break
		}
	}
	
	// Roll back to common ancestor
	c.snapshotMu.RLock()
	commonSnapshot, ok := c.snapshots[commonHash]
	c.snapshotMu.RUnlock()
	
	if !ok {
		log.Error("Common ancestor snapshot not found, clearing cache",
			"commonHash", commonHash.Hex())
		// Clear cache and rebuild from current state
		return c.Update(newChain[len(newChain)-1], stateDB)
	}
	
	// Restore common ancestor as current
	c.current.Store(commonSnapshot)
	
	log.Info("Rolled back to common ancestor",
		"block", commonSnapshot.BlockNumber,
		"hash", commonHash.Hex()[:10])
	
	// Replay new chain
	for _, header := range newChain {
		if header.Number.Uint64() <= commonSnapshot.BlockNumber {
			continue
		}
		if err := c.Update(header, stateDB); err != nil {
			return fmt.Errorf("failed to replay block %d: %w", header.Number.Uint64(), err)
		}
	}
	
	log.Info("Replayed new chain",
		"blocks", len(newChain),
		"newHead", newChain[len(newChain)-1].Number.Uint64())
	
	return nil
}

// StateDBReader adapts state.StateDB to the StateReader interface.
type StateDBReader struct {
	db *state.StateDB
}

// NewStateDBReader creates a StateReader from a StateDB.
func NewStateDBReader(db *state.StateDB) StateReader {
	return &StateDBReader{db: db}
}

// GetState implements StateReader.
func (r *StateDBReader) GetState(addr common.Address, slot common.Hash) common.Hash {
	return r.db.GetState(addr, slot)
}

