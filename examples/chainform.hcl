# ChainForm configuration — desired on-chain protocol state.
#
# Run against the offline demo reader (no RPC required):
#   chainform plan   -f examples/chainform.hcl --mock
#   chainform export -f examples/chainform.hcl --mock -o batch.json
#
# Run against a live network by setting RPC_URL and dropping --mock.

version = "1"

chain {
  name     = "ethereum sepolia"
  chain_id = 11155111

  # env(...) reads from the process environment, keeping secrets out of git.
  rpc = env("RPC_URL")
}

resource "protocol" "protocol" {
  address = "0xF38D8Be3E0A7B3c94C00a25b4A443ca062f343f5"

  # Only declared attributes are managed; anything omitted is left as-is.
  feeBps = 501
  paused = true
}
