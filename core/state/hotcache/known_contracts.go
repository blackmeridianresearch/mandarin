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

import "github.com/ethereum/go-ethereum/common"

// KnownContracts contains addresses of well-known DeFi protocols for testing and validation.
// These addresses can be used to populate the hot cache watchlist.

// Uniswap V2 Factory and Router addresses
var (
	// Mainnet Uniswap V2
	UniswapV2FactoryMainnet = common.HexToAddress("0x5C69bEe701ef814a2B6a3EDD4B1652CB9cc5aA6f")
	UniswapV2Router02Mainnet = common.HexToAddress("0x7a250d5630B4cF539739dF2C5dAcb4c659F2488D")
	
	// Sepolia Uniswap V2
	UniswapV2FactorySepolia = common.HexToAddress("0xF62c03E08ada871A0bEb309762E260a7a6a880E6")
	UniswapV2Router02Sepolia = common.HexToAddress("0xeE567Fe1712Faf6149d80dA1E6934E354124CfE3")
)

// Common mainnet Uniswap V2 pairs (high-value pools for testing)
var UniswapV2PairsMainnet = map[string]common.Address{
	"USDC/WETH": common.HexToAddress("0xB4e16d0168e52d35CaCD2c6185b44281Ec28C9Dc"),
	"USDT/WETH": common.HexToAddress("0x0d4a11d5EEaaC28EC3F61d100daF4d40471f1852"),
	"DAI/WETH":  common.HexToAddress("0xA478c2975Ab1Ea89e8196811F51A7B7Ade33eB11"),
	"WBTC/WETH": common.HexToAddress("0xBb2b8038a1640196FbE3e38816F3e67Cba72D940"),
	"USDC/USDT": common.HexToAddress("0x3041CbD36888bECc7bbCBc0045E3B1f144466f5f"),
}

// Token addresses for reference
var TokenAddresses = struct {
	Mainnet struct {
		WETH common.Address
		USDC common.Address
		USDT common.Address
		DAI  common.Address
		WBTC common.Address
	}
	Sepolia struct {
		WETH common.Address
		// Add Sepolia token addresses as they're discovered
	}
}{
	Mainnet: struct {
		WETH common.Address
		USDC common.Address
		USDT common.Address
		DAI  common.Address
		WBTC common.Address
	}{
		WETH: common.HexToAddress("0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2"),
		USDC: common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"),
		USDT: common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7"),
		DAI:  common.HexToAddress("0x6B175474E89094C44Da98b954EedeAC495271d0F"),
		WBTC: common.HexToAddress("0x2260FAC5E5542a773Aa44fBCfeDf7C193bc2C599"),
	},
	Sepolia: struct {
		WETH common.Address
	}{
		WETH: common.HexToAddress("0x7b79995e5f793A07Bc00c21412e50Ecae098E7f9"),
	},
}

// GetDefaultWatchlist returns a recommended watchlist for the given network.
func GetDefaultWatchlist(chainID uint64) []common.Address {
	switch chainID {
	case 1: // Mainnet
		return []common.Address{
			UniswapV2PairsMainnet["USDC/WETH"],
			UniswapV2PairsMainnet["USDT/WETH"],
			UniswapV2PairsMainnet["DAI/WETH"],
			UniswapV2PairsMainnet["WBTC/WETH"],
		}
	case 11155111: // Sepolia
		// Return empty for now - users should discover pools on Sepolia
		// and add them manually after deployment
		return []common.Address{}
	default:
		return []common.Address{}
	}
}

// RegisterDefaultDecoders registers decoders for all known Uniswap V2 pairs.
func RegisterDefaultDecoders(cache *Cache, chainID uint64) {
	decoder := &UniswapV2Decoder{}
	
	switch chainID {
	case 1: // Mainnet
		for _, addr := range UniswapV2PairsMainnet {
			cache.RegisterDecoder(addr, decoder)
		}
	case 11155111: // Sepolia
		// Register decoders for Sepolia pairs when discovered
	}
}

