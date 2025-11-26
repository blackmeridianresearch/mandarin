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
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestUniswapV2Decoder(t *testing.T) {
	decoder := &UniswapV2Decoder{}
	
	// Test contract type
	if decoder.Type() != ContractTypeUniswapV2 {
		t.Errorf("Expected contract type %v, got %v", ContractTypeUniswapV2, decoder.Type())
	}
	
	// Test required slots
	slots := decoder.RequiredSlots()
	if len(slots) != 6 {
		t.Errorf("Expected 6 required slots, got %d", len(slots))
	}
}

func TestUniswapV2Decode(t *testing.T) {
	decoder := &UniswapV2Decoder{}
	
	// Create test data representing a Uniswap V2 pool
	token0 := common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48") // USDC
	token1 := common.HexToAddress("0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2") // WETH
	
	// Pack reserves: reserve0 (uint112) + reserve1 (uint112) + timestamp (uint32)
	// For simplicity, use small values:
	// reserve0 = 1000000 (1M USDC with 6 decimals)
	// reserve1 = 500 (500 WETH with 18 decimals)
	// timestamp = 1234567890
	
	reserve0 := big.NewInt(1000000)
	reserve1 := big.NewInt(500)
	timestamp := uint32(1234567890)
	
	// Pack into single 256-bit value:
	// [timestamp (32 bits)][reserve1 (112 bits)][reserve0 (112 bits)]
	packed := new(big.Int)
	packed.Or(packed, reserve0)                                      // Add reserve0
	packed.Or(packed, new(big.Int).Lsh(reserve1, 112))              // Add reserve1 shifted
	packed.Or(packed, new(big.Int).Lsh(big.NewInt(int64(timestamp)), 224)) // Add timestamp shifted
	
	slots := map[common.Hash]common.Hash{
		uniswapV2SlotToken0:   common.BytesToHash(token0.Bytes()),
		uniswapV2SlotToken1:   common.BytesToHash(token1.Bytes()),
		uniswapV2SlotReserves: common.BigToHash(packed),
		uniswapV2SlotPrice0Cumulative: common.BigToHash(big.NewInt(123456)),
		uniswapV2SlotPrice1Cumulative: common.BigToHash(big.NewInt(789012)),
		uniswapV2SlotKLast:            common.BigToHash(big.NewInt(999999)),
	}
	
	// Decode
	decoded, err := decoder.Decode(slots)
	if err != nil {
		t.Fatalf("Failed to decode: %v", err)
	}
	
	state, ok := decoded.(*UniswapV2State)
	if !ok {
		t.Fatal("Decoded value is not UniswapV2State")
	}
	
	// Verify token addresses
	if state.Token0 != token0 {
		t.Errorf("Expected token0 %s, got %s", token0.Hex(), state.Token0.Hex())
	}
	if state.Token1 != token1 {
		t.Errorf("Expected token1 %s, got %s", token1.Hex(), state.Token1.Hex())
	}
	
	// Verify reserves
	if state.Reserve0.Cmp(reserve0) != 0 {
		t.Errorf("Expected reserve0 %s, got %s", reserve0.String(), state.Reserve0.String())
	}
	if state.Reserve1.Cmp(reserve1) != 0 {
		t.Errorf("Expected reserve1 %s, got %s", reserve1.String(), state.Reserve1.String())
	}
	
	// Verify timestamp
	if state.BlockTimestampLast != timestamp {
		t.Errorf("Expected timestamp %d, got %d", timestamp, state.BlockTimestampLast)
	}
	
	// Verify price calculations (use threshold for floating point comparison)
	price := state.GetPrice()
	// Price should be reserve1 / reserve0 = 500 / 1000000 = 0.0005
	if price.Sign() == 0 {
		t.Error("Price should not be zero")
	}
	
	inversePrice := state.GetInversePrice()
	// Inverse price should be reserve0 / reserve1 = 1000000 / 500 = 2000
	if inversePrice.Sign() == 0 {
		t.Error("Inverse price should not be zero")
	}
}

func TestUniswapV2String(t *testing.T) {
	state := &UniswapV2State{
		Token0:   common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"),
		Token1:   common.HexToAddress("0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2"),
		Reserve0: big.NewInt(1000000),
		Reserve1: big.NewInt(500),
		BlockTimestampLast: 1234567890,
	}
	
	str := state.String()
	if str == "" {
		t.Error("String() returned empty string")
	}
	
	// Verify string contains key information
	if len(str) < 50 {
		t.Errorf("String() output seems too short: %s", str)
	}
}

func BenchmarkUniswapV2Decode(b *testing.B) {
	decoder := &UniswapV2Decoder{}
	
	// Prepare test data
	token0 := common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48")
	token1 := common.HexToAddress("0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2")
	
	reserve0 := big.NewInt(1000000)
	reserve1 := big.NewInt(500)
	timestamp := uint32(1234567890)
	
	packed := new(big.Int)
	packed.Or(packed, reserve0)
	packed.Or(packed, new(big.Int).Lsh(reserve1, 112))
	packed.Or(packed, new(big.Int).Lsh(big.NewInt(int64(timestamp)), 224))
	
	slots := map[common.Hash]common.Hash{
		uniswapV2SlotToken0:           common.BytesToHash(token0.Bytes()),
		uniswapV2SlotToken1:           common.BytesToHash(token1.Bytes()),
		uniswapV2SlotReserves:         common.BigToHash(packed),
		uniswapV2SlotPrice0Cumulative: common.BigToHash(big.NewInt(123456)),
		uniswapV2SlotPrice1Cumulative: common.BigToHash(big.NewInt(789012)),
		uniswapV2SlotKLast:            common.BigToHash(big.NewInt(999999)),
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := decoder.Decode(slots)
		if err != nil {
			b.Fatal(err)
		}
	}
}

