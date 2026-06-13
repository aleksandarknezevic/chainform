# ChainForm configuration — inspecting a real, read-only contract.
#
# This points at the live Chainlink ETH/USD price feed on Ethereum Sepolia
# (an EACAggregatorProxy). It exposes only getters — no setX setters — so there
# is nothing to manage: `plan` is always empty. Its value is in `show`, which
# prints the contract's current on-chain state derived entirely from its ABI.
#
# Offline (canned demo values, no RPC required):
#   chainform show -f examples/feed.hcl --mock
#
# Live Sepolia (real on-chain values):
#   RPC_URL=https://sepolia.infura.io/v3/<key> chainform show -f examples/feed.hcl

version = "1"

chain {
  name     = "ethereum sepolia"
  chain_id = 11155111

  rpc = env("RPC_URL")
}

resource "contract" "ethUsdFeed" {
  # Chainlink ETH/USD price feed on Sepolia.
  # https://docs.chain.link/data-feeds/price-feeds/addresses?network=ethereum
  address = "0x694AA1769357215DE4FAC081bf1f309aDC325306"

  abi = "testdata/aggregator.abi.json"

  # This contract is read-only (no setX setters), so nothing can be *managed*.
  # An `expect` block declares read-only invariants: ChainForm reads the getter
  # and reports drift, but never proposes a transaction (there is no setter).
  #
  # decimals is actually 8 on-chain, so `expect decimals = 9` is reported as
  # read-only drift by `plan` / `show`. Inspect full state with `show`.
  expect {
    decimals    = 9
    description = "ETH / USD"
  }
}
