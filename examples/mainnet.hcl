# ChainForm — read-only Ethereum mainnet example (Lido + Chainlink).
#
# Monitors two production contracts with `expect` blocks:
#   • Lido stETH — protocol fee (getFee) and emergency stop flag (isStopped)
#   • Chainlink ETH/USD — oracle metadata (decimals, description)
#
# Nothing is managed (no setX setters) — `plan` never proposes transactions.
# Use `show` to inspect live values; `plan` checks invariants and reports drift.
#
# Full walkthrough: docs/mainnet-example.md

version = "1"

chain {
  name     = "ethereum"
  chain_id = 1
  rpc      = env("RPC_URL")
}

resource "contract" "lidoSteth" {
  # Lido stETH token — Ethereum mainnet (proxy)
  # https://docs.lido.fi/deployed-contracts/
  # https://etherscan.io/address/0xae7ab96520DE3A18E5e111B5EaAb095312D7fE84
  address = "0xae7ab96520DE3A18E5e111B5EaAb095312D7fE84"

  abi = "testdata/lido-steth.abi.json"

  expect {
    name      = "Liquid staked Ether 2.0"
    symbol    = "stETH"
    decimals  = 18
    getFee    = 999   # protocol fee in basis points; governance-updatable
    isStopped = false # emergency stop — analogous to a "paused" flag
  }
}

resource "contract" "ethUsdOracle" {
  # Chainlink ETH/USD price feed — Ethereum mainnet
  # https://docs.chain.link/data-feeds/price-feeds/addresses?network=ethereum
  # https://etherscan.io/address/0x5f4eC3Df9cbd43714FE2740f5E3616155c5b8419
  address = "0x5f4eC3Df9cbd43714FE2740f5E3616155c5b8419"

  abi = "testdata/chainlink-eth-usd.abi.json"

  expect {
    decimals    = 8
    description = "ETH / USD"
    version     = 6
  }
}
