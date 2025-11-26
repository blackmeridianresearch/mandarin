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
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestNewCache(t *testing.T) {
	config := Config{
		Enabled:      true,
		Watchlist:    []common.Address{common.HexToAddress("0x1")},
		ShadowMode:   true,
		MaxSnapshots: 64,
	}
	
	cache := New(config)
	if cache == nil {
		t.Fatal("New() returned nil")
	}
	
	if !cache.IsEnabled() {
		t.Error("Cache should be enabled")
	}
	
	if !cache.IsWatched(common.HexToAddress("0x1")) {
		t.Error("Address should be in watchlist")
	}
	
	if cache.IsWatched(common.HexToAddress("0x2")) {
		t.Error("Address should not be in watchlist")
	}
}

func TestCacheDisabledByDefault(t *testing.T) {
	config := DefaultConfig()
	cache := New(config)
	
	if cache.IsEnabled() {
		t.Error("Cache should be disabled by default")
	}
}

func TestGetSnapshot(t *testing.T) {
	config := Config{
		Enabled:   true,
		Watchlist: []common.Address{},
	}
	
	cache := New(config)
	snapshot := cache.GetSnapshot()
	
	if snapshot == nil {
		t.Fatal("GetSnapshot() returned nil")
	}
	
	if snapshot.BlockNumber != 0 {
		t.Errorf("Initial snapshot should have block number 0, got %d", snapshot.BlockNumber)
	}
	
	if len(snapshot.Contracts) != 0 {
		t.Errorf("Initial snapshot should have 0 contracts, got %d", len(snapshot.Contracts))
	}
}

func TestGetContractState(t *testing.T) {
	config := Config{
		Enabled:   true,
		Watchlist: []common.Address{},
	}
	
	cache := New(config)
	addr := common.HexToAddress("0x1")
	
	// Should return ErrNotFound for non-cached contract
	_, err := cache.GetContractState(addr)
	if err == nil {
		t.Error("Expected error for non-cached contract")
	}
	
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}
}

func TestRegisterDecoder(t *testing.T) {
	config := Config{
		Enabled:   true,
		Watchlist: []common.Address{},
	}
	
	cache := New(config)
	addr := common.HexToAddress("0x1")
	decoder := &UniswapV2Decoder{}
	
	cache.RegisterDecoder(addr, decoder)
	
	// Verify decoder was registered (internal state, can't directly test)
	// Just ensure no panic
}

func TestGetStatistics(t *testing.T) {
	config := Config{
		Enabled:   true,
		Watchlist: []common.Address{},
	}
	
	cache := New(config)
	stats := cache.GetStatistics()
	
	// Initially all stats should be 0
	if stats.Hits.Load() != 0 {
		t.Errorf("Expected 0 hits, got %d", stats.Hits.Load())
	}
	if stats.Misses.Load() != 0 {
		t.Errorf("Expected 0 misses, got %d", stats.Misses.Load())
	}
	if stats.Updates.Load() != 0 {
		t.Errorf("Expected 0 updates, got %d", stats.Updates.Load())
	}
}

func BenchmarkGetSnapshot(b *testing.B) {
	config := Config{
		Enabled:   true,
		Watchlist: make([]common.Address, 100),
	}
	
	// Add 100 addresses to watchlist
	for i := 0; i < 100; i++ {
		config.Watchlist[i] = common.BigToAddress(common.Big1)
	}
	
	cache := New(config)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cache.GetSnapshot()
	}
}

func BenchmarkGetContractState(b *testing.B) {
	config := Config{
		Enabled:   true,
		Watchlist: []common.Address{common.HexToAddress("0x1")},
	}
	
	cache := New(config)
	addr := common.HexToAddress("0x1")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = cache.GetContractState(addr)
	}
}

