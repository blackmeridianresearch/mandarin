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

package core

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state/hotcache"
)

var (
	ErrHotCacheDisabled = errors.New("hot cache is disabled")
	ErrHotCacheNotFound = errors.New("contract not in hot cache")
)

// HotCache returns the hot state cache instance.
// Returns nil if the cache is disabled.
func (bc *BlockChain) HotCache() *hotcache.Cache {
	if bc.hotCache == nil || !bc.hotCache.IsEnabled() {
		return nil
	}
	return bc.hotCache
}

// GetHotCacheSnapshot returns the current hot cache snapshot.
// This provides a consistent view of all cached contract states.
func (bc *BlockChain) GetHotCacheSnapshot() (*hotcache.Snapshot, error) {
	if bc.hotCache == nil || !bc.hotCache.IsEnabled() {
		return nil, ErrHotCacheDisabled
	}
	return bc.hotCache.GetSnapshot(), nil
}

// GetHotCachedContractState returns the cached state for a specific contract.
// This is significantly faster than state trie lookups for frequently-accessed contracts.
func (bc *BlockChain) GetHotCachedContractState(addr common.Address) (*hotcache.ContractState, error) {
	if bc.hotCache == nil || !bc.hotCache.IsEnabled() {
		return nil, ErrHotCacheDisabled
	}
	state, err := bc.hotCache.GetContractState(addr)
	if err != nil {
		if errors.Is(err, hotcache.ErrNotFound) {
			return nil, ErrHotCacheNotFound
		}
		return nil, err
	}
	return state, nil
}

// GetHotCachedUniswapV2State returns decoded Uniswap V2 pool state.
// Returns ErrHotCacheNotFound if the contract is not cached.
// Returns an error if the contract is cached but is not a Uniswap V2 pool.
func (bc *BlockChain) GetHotCachedUniswapV2State(addr common.Address) (*hotcache.UniswapV2State, error) {
	state, err := bc.GetHotCachedContractState(addr)
	if err != nil {
		return nil, err
	}
	
	if state.Type != hotcache.ContractTypeUniswapV2 {
		return nil, errors.New("contract is not a Uniswap V2 pool")
	}
	
	if state.Decoded == nil {
		return nil, errors.New("contract state not decoded")
	}
	
	v2State, ok := state.Decoded.(*hotcache.UniswapV2State)
	if !ok {
		return nil, errors.New("failed to cast to Uniswap V2 state")
	}
	
	return v2State, nil
}

// GetHotCacheStatistics returns performance statistics for the hot cache.
func (bc *BlockChain) GetHotCacheStatistics() (hotcache.Statistics, error) {
	if bc.hotCache == nil || !bc.hotCache.IsEnabled() {
		return hotcache.Statistics{}, ErrHotCacheDisabled
	}
	return bc.hotCache.GetStatistics(), nil
}

// RegisterHotCacheDecoder registers a decoder for a specific contract address.
// This allows the cache to decode contract-specific state automatically.
func (bc *BlockChain) RegisterHotCacheDecoder(addr common.Address, decoder hotcache.ContractDecoder) error {
	if bc.hotCache == nil {
		return ErrHotCacheDisabled
	}
	bc.hotCache.RegisterDecoder(addr, decoder)
	return nil
}

