# ChainForm configuration — ABI-driven resource.
#
# The "contract" resource needs no hand-written Go: point it at a contract ABI
# and declare the attributes you want managed. Each attribute X is read via the
# getter X() and reconciled via the setter setX(...), both derived from the ABI.
#
# Run against the offline demo reader (no RPC required):
#   chainform plan   -f examples/contract.hcl --mock
#   chainform export -f examples/contract.hcl --mock -o batch.json
#
# Run against a live network by setting RPC_URL and dropping --mock.

version = "1"

chain {
  name     = "ethereum sepolia"
  chain_id = 11155111

  rpc = env("RPC_URL")
}

resource "contract" "protocol" {
  address = "0xF38D8Be3E0A7B3c94C00a25b4A443ca062f343f5"

  # Path to the ABI JSON, resolved relative to the working directory.
  abi = "testdata/protocol.abi.json"

  # Managed attributes. Each must have a getter X() and setter setX(...) in the
  # ABI. Anything omitted is left untouched on-chain.
  feeBps = 30
  paused = false
}
