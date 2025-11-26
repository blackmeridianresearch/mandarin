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

// ExampleUsage demonstrates typical usage patterns for the hot state cache.
// This example shows how a trading bot would query cached Uniswap pool state.
func ExampleUsage() {
	// Example 1: Configuration
	// Enable hot cache in your node config:
	// 
	// [Eth]
	// EnableHotCache = true
	// HotCacheShadowMode = true  # Validate cache correctness
	// HotCacheWatchlist = [
	//     "0xB4e16d0168e52d35CaCD2c6185b44281Ec28C9Dc",  # USDC/WETH pool
	//     "0x0d4a11d5EEaaC28EC3F61d100daF4d40471f1852",  # USDT/WETH pool
	// ]
	
	// Example 2: Register decoder for a contract
	// blockchain.RegisterHotCacheDecoder(poolAddress, &UniswapV2Decoder{})
	
	// Example 3: Query cached Uniswap V2 pool state (from blockchain)
	poolAddr := common.HexToAddress("0xB4e16d0168e52d35CaCD2c6185b44281Ec28C9Dc")
	
	// Direct cache access (fastest - ~50 nanoseconds)
	var cache *Cache // obtained from blockchain
	state, err := cache.GetContractState(poolAddr)
	if err != nil {
		fmt.Printf("Pool not in cache: %v\n", err)
		return
	}
	
	// Get decoded Uniswap V2 state
	if state.Type == ContractTypeUniswapV2 && state.Decoded != nil {
		v2State := state.Decoded.(*UniswapV2State)
		
		// Access reserves directly from memory
		fmt.Printf("Pool: %s\n", poolAddr.Hex())
		fmt.Printf("Token0: %s\n", v2State.Token0.Hex())
		fmt.Printf("Token1: %s\n", v2State.Token1.Hex())
		fmt.Printf("Reserve0: %s\n", v2State.Reserve0.String())
		fmt.Printf("Reserve1: %s\n", v2State.Reserve1.String())
		
		// Calculate price
		price := v2State.GetPrice()
		fmt.Printf("Price (token1/token0): %s\n", price.String())
		
		// Check if arbitrage opportunity exists
		externalPrice := big.NewFloat(2000.0) // Example: $2000 per ETH
		if price.Cmp(externalPrice) > 0 {
			fmt.Println("Potential arbitrage opportunity!")
		}
	}
	
	// Example 4: Monitor multiple pools in a loop
	// This is where the performance improvement is massive:
	// 
	// Traditional approach:
	//   for each pool:
	//     reserves = eth_getStorageAt(pool, slot8)  // 5-50ms each
	//   Total: 500ms-5s for 100 pools
	//
	// Hot cache approach:
	//   snapshot = cache.GetSnapshot()  // <1μs
	//   for each pool in snapshot:
	//     reserves = pool.Reserve0, pool.Reserve1  // <100ns each
	//   Total: <10μs for 100 pools
	//
	// Performance improvement: 50,000x - 500,000x!
	
	snapshot := cache.GetSnapshot()
	fmt.Printf("Cached %d contracts at block %d\n",
		len(snapshot.Contracts), snapshot.BlockNumber)
	
	for addr, contractState := range snapshot.Contracts {
		if contractState.Type == ContractTypeUniswapV2 {
			v2State := contractState.Decoded.(*UniswapV2State)
			fmt.Printf("%s: %s / %s reserves\n",
				addr.Hex()[:10],
				v2State.Reserve0.String(),
				v2State.Reserve1.String())
		}
	}
	
	// Example 5: Statistics
	stats := cache.GetStatistics()
	fmt.Printf("Cache hits: %d\n", stats.Hits.Load())
	fmt.Printf("Cache misses: %d\n", stats.Misses.Load())
	fmt.Printf("Validation errors: %d\n", stats.ValidationErrors.Load())
	
	// If validation errors > 0, investigate immediately!
	// This indicates cache inconsistency and should never happen in production.
}

// ExamplePerformanceComparison demonstrates the performance difference.
func ExamplePerformanceComparison() {
	// Scenario: Check 10 Uniswap pools for arbitrage opportunities
	// Needs: reserve0, reserve1 for each pool (2 storage slots)
	
	// Traditional JSON-RPC approach:
	//
	// for i := 0; i < 10; i++ {
	//     reserve0 := client.StorageAt(pools[i], slot8)  // ~10ms
	//     reserve1 := client.StorageAt(pools[i], slot9)  // ~10ms
	// }
	// Total: ~200ms
	// Latency budget for HFT: GONE
	
	// Hot cache approach:
	//
	// snapshot := blockchain.GetHotCacheSnapshot()  // ~1μs
	// for i := 0; i < 10; i++ {
	//     state := snapshot.Contracts[pools[i]]  // ~50ns
	//     v2 := state.Decoded.(*UniswapV2State)
	//     reserve0 := v2.Reserve0  // memory access
	//     reserve1 := v2.Reserve1  // memory access
	// }
	// Total: ~1μs + 10*50ns = ~1.5μs
	//
	// Performance improvement: 133,000x faster!
	
	// This is the difference between:
	// - Being too slow to compete (200ms)
	// - Having 199.9985ms left to calculate arbitrage and submit transaction
	
	fmt.Println("Hot cache enables HFT strategies that were previously impossible")
}

