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
)

func TestKnownContracts(t *testing.T) {
	// Verify Uniswap V2 Factory addresses are set
	if UniswapV2FactoryMainnet.Hex() == "0x0000000000000000000000000000000000000000" {
		t.Error("UniswapV2FactoryMainnet should be set")
	}
	
	if UniswapV2FactorySepolia.Hex() == "0x0000000000000000000000000000000000000000" {
		t.Error("UniswapV2FactorySepolia should be set")
	}
	
	// Verify expected mainnet factory address
	expectedMainnet := "0x5C69bEe701ef814a2B6a3EDD4B1652CB9cc5aA6f"
	if UniswapV2FactoryMainnet.Hex() != expectedMainnet {
		t.Errorf("Expected mainnet factory %s, got %s", expectedMainnet, UniswapV2FactoryMainnet.Hex())
	}
	
	// Verify expected Sepolia factory address
	expectedSepolia := "0xF62c03E08ada871A0bEb309762E260a7a6a880E6"
	if UniswapV2FactorySepolia.Hex() != expectedSepolia {
		t.Errorf("Expected Sepolia factory %s, got %s", expectedSepolia, UniswapV2FactorySepolia.Hex())
	}
}

func TestUniswapV2Pairs(t *testing.T) {
	// Verify we have mainnet pairs defined
	if len(UniswapV2PairsMainnet) == 0 {
		t.Error("UniswapV2PairsMainnet should contain pairs")
	}
	
	// Verify specific pairs exist
	pairs := []string{"USDC/WETH", "USDT/WETH", "DAI/WETH", "WBTC/WETH"}
	for _, pair := range pairs {
		addr, ok := UniswapV2PairsMainnet[pair]
		if !ok {
			t.Errorf("Missing pair: %s", pair)
		}
		if addr.Hex() == "0x0000000000000000000000000000000000000000" {
			t.Errorf("Pair %s has zero address", pair)
		}
	}
}

func TestGetDefaultWatchlist(t *testing.T) {
	// Test mainnet
	mainnetList := GetDefaultWatchlist(1)
	if len(mainnetList) == 0 {
		t.Error("Mainnet default watchlist should not be empty")
	}
	
	// Test Sepolia (may be empty until pools are discovered)
	sepoliaList := GetDefaultWatchlist(11155111)
	_ = sepoliaList // May be empty for now
	
	// Test unknown chain
	unknownList := GetDefaultWatchlist(999999)
	if len(unknownList) != 0 {
		t.Error("Unknown chain should return empty watchlist")
	}
}

func TestRegisterDefaultDecoders(t *testing.T) {
	config := Config{
		Enabled:   true,
		Watchlist: GetDefaultWatchlist(1), // Mainnet
	}
	
	cache := New(config)
	RegisterDefaultDecoders(cache, 1)
	
	// Verify decoders were registered (internal state, can't directly test)
	// Just ensure no panic
}

// TestTokenAddresses verifies token addresses are set correctly
func TestTokenAddresses(t *testing.T) {
	// Mainnet tokens
	if TokenAddresses.Mainnet.WETH.Hex() == "0x0000000000000000000000000000000000000000" {
		t.Error("Mainnet WETH address should be set")
	}
	if TokenAddresses.Mainnet.USDC.Hex() == "0x0000000000000000000000000000000000000000" {
		t.Error("Mainnet USDC address should be set")
	}
	
	// Verify WETH address is correct
	expectedWETH := "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2"
	if TokenAddresses.Mainnet.WETH.Hex() != expectedWETH {
		t.Errorf("Expected WETH %s, got %s", expectedWETH, TokenAddresses.Mainnet.WETH.Hex())
	}
	
	// Sepolia WETH
	if TokenAddresses.Sepolia.WETH.Hex() == "0x0000000000000000000000000000000000000000" {
		t.Error("Sepolia WETH address should be set")
	}
}

