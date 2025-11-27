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
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// Uniswap V2 storage layout:
// slot 6: token0 (address)
// slot 7: token1 (address)  
// slot 8: reserve0 (uint112), reserve1 (uint112), blockTimestampLast (uint32) - packed
// slot 9: price0CumulativeLast (uint256)
// slot 10: price1CumulativeLast (uint256)
// slot 11: kLast (uint256)

var (
	// Standard storage slots for Uniswap V2
	uniswapV2SlotToken0           = common.BigToHash(big.NewInt(6))
	uniswapV2SlotToken1           = common.BigToHash(big.NewInt(7))
	uniswapV2SlotReserves         = common.BigToHash(big.NewInt(8))
	uniswapV2SlotPrice0Cumulative = common.BigToHash(big.NewInt(9))
	uniswapV2SlotPrice1Cumulative = common.BigToHash(big.NewInt(10))
	uniswapV2SlotKLast            = common.BigToHash(big.NewInt(11))
)

// UniswapV2State represents the decoded state of a Uniswap V2 pair.
type UniswapV2State struct {
	Token0             common.Address
	Token1             common.Address
	Reserve0           *big.Int // uint112
	Reserve1           *big.Int // uint112
	BlockTimestampLast uint32
	Price0Cumulative   *big.Int
	Price1Cumulative   *big.Int
	KLast              *big.Int
}

// String returns a human-readable representation of the pool state.
func (s *UniswapV2State) String() string {
	return fmt.Sprintf("UniswapV2{token0: %s, token1: %s, reserve0: %s, reserve1: %s, timestamp: %d}",
		s.Token0.Hex(), s.Token1.Hex(), s.Reserve0.String(), s.Reserve1.String(), s.BlockTimestampLast)
}

// UniswapV2Decoder decodes Uniswap V2 pair state from raw storage slots.
type UniswapV2Decoder struct{}

// Type returns the contract type.
func (d *UniswapV2Decoder) Type() ContractType {
	return ContractTypeUniswapV2
}

// RequiredSlots returns the storage slots needed for decoding.
func (d *UniswapV2Decoder) RequiredSlots() []common.Hash {
	return []common.Hash{
		uniswapV2SlotToken0,
		uniswapV2SlotToken1,
		uniswapV2SlotReserves,
		uniswapV2SlotPrice0Cumulative,
		uniswapV2SlotPrice1Cumulative,
		uniswapV2SlotKLast,
	}
}

// Decode decodes raw storage slots into UniswapV2State.
func (d *UniswapV2Decoder) Decode(slots map[common.Hash]common.Hash) (interface{}, error) {
	state := &UniswapV2State{
		Reserve0: new(big.Int),
		Reserve1: new(big.Int),
		Price0Cumulative: new(big.Int),
		Price1Cumulative: new(big.Int),
		KLast: new(big.Int),
	}
	
	// Decode token0 (slot 6)
	if token0Value, ok := slots[uniswapV2SlotToken0]; ok {
		state.Token0 = common.BytesToAddress(token0Value.Bytes())
	} else {
		return nil, fmt.Errorf("missing token0 slot")
	}
	
	// Decode token1 (slot 7)
	if token1Value, ok := slots[uniswapV2SlotToken1]; ok {
		state.Token1 = common.BytesToAddress(token1Value.Bytes())
	} else {
		return nil, fmt.Errorf("missing token1 slot")
	}
	
	// Decode reserves (slot 8) - packed: reserve0 (uint112), reserve1 (uint112), blockTimestampLast (uint32)
	if reservesValue, ok := slots[uniswapV2SlotReserves]; ok {
		// Extract from 32-byte slot:
		// bytes 0-13: reserve0 (14 bytes = 112 bits)
		// bytes 14-27: reserve1 (14 bytes = 112 bits)
		// bytes 28-31: blockTimestampLast (4 bytes = 32 bits)
		bytes := reservesValue.Bytes()
		if len(bytes) == 32 {
			// Reserve0: bytes 18-31 (last 14 bytes, since it's right-aligned)
			// Actually, Solidity packs from right to left:
			// [0-17: padding][18-31: reserve0 + reserve1 + timestamp]
			// More precisely: rightmost 14 bytes are reserve0, next 14 are reserve1, leftmost 4 are timestamp
			
			// Let's extract correctly:
			// Byte layout (right to left in storage):
			// [blockTimestampLast (4 bytes)][reserve1 (14 bytes)][reserve0 (14 bytes)]
			
			fullValue := new(big.Int).SetBytes(bytes)
			
			// Reserve0 is the rightmost 112 bits (14 bytes)
			mask112 := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 112), big.NewInt(1))
			state.Reserve0.And(fullValue, mask112)
			
			// Reserve1 is the next 112 bits
			shifted := new(big.Int).Rsh(fullValue, 112)
			state.Reserve1.And(shifted, mask112)
			
			// BlockTimestampLast is the next 32 bits
			timestampShifted := new(big.Int).Rsh(fullValue, 224) // 112 + 112
			state.BlockTimestampLast = uint32(timestampShifted.Uint64())
		}
	} else {
		return nil, fmt.Errorf("missing reserves slot")
	}
	
	// Decode price0CumulativeLast (slot 9)
	if price0Value, ok := slots[uniswapV2SlotPrice0Cumulative]; ok {
		state.Price0Cumulative.SetBytes(price0Value.Bytes())
	}
	
	// Decode price1CumulativeLast (slot 10)
	if price1Value, ok := slots[uniswapV2SlotPrice1Cumulative]; ok {
		state.Price1Cumulative.SetBytes(price1Value.Bytes())
	}
	
	// Decode kLast (slot 11)
	if kLastValue, ok := slots[uniswapV2SlotKLast]; ok {
		state.KLast.SetBytes(kLastValue.Bytes())
	}
	
	return state, nil
}

// GetPrice returns the current price of token0 in terms of token1.
// Price = reserve1 / reserve0
func (s *UniswapV2State) GetPrice() *big.Float {
	if s.Reserve0.Sign() == 0 {
		return big.NewFloat(0)
	}
	reserve0Float := new(big.Float).SetInt(s.Reserve0)
	reserve1Float := new(big.Float).SetInt(s.Reserve1)
	return new(big.Float).Quo(reserve1Float, reserve0Float)
}

// GetInversePrice returns the price of token1 in terms of token0.
// InversePrice = reserve0 / reserve1
func (s *UniswapV2State) GetInversePrice() *big.Float {
	if s.Reserve1.Sign() == 0 {
		return big.NewFloat(0)
	}
	reserve0Float := new(big.Float).SetInt(s.Reserve0)
	reserve1Float := new(big.Float).SetInt(s.Reserve1)
	return new(big.Float).Quo(reserve0Float, reserve1Float)
}

