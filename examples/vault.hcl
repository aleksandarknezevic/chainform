# ChainForm configuration — richer attribute types.
#
# Beyond scalars, a `contract` attribute can be a dynamic or fixed-size array,
# `bytes` / `bytesN`, or an enum (a `uint8` on the wire). Values are written the
# way you would expect in HCL: lists in brackets, byte values as 0x-hex.
#
# Offline (canned demo values, no RPC required):
#   chainform plan   -f examples/vault.hcl --mock
#   chainform show   -f examples/vault.hcl --mock
#   chainform export -f examples/vault.hcl --mock -o batch.json
#
# Against the demo state, `keepers` and `mode` have drifted; the remaining
# attributes match, so no operation is proposed for them.

version = "1"

chain {
  name     = "ethereum sepolia"
  chain_id = 11155111

  rpc = env("RPC_URL")
}

resource "contract" "vault" {
  address = "0x9a7f8e1c2b3d4e5f60718293a4b5c6d7e8f90123"

  abi = "testdata/vault.abi.json"

  # address[] — read via keepers(), reconciled via setKeepers(address[]).
  # Order matters: a list is compared element by element.
  keepers = [
    "0x1111111111111111111111111111111111111111",
    "0x2222222222222222222222222222222222222222",
    "0x4444444444444444444444444444444444444444",
  ]

  # bytes32 — a hex string of exactly 32 bytes.
  merkleRoot = "0xa1b2c3d4e5f60718293a4b5c6d7e8f90a1b2c3d4e5f60718293a4b5c6d7e8f90"

  # uint256[3] — a fixed-size array must have exactly 3 elements.
  tierCaps = [1000, 5000, 10000]

  # enum Mode { Idle, Active, Halted } — a uint8 on the wire.
  mode = 1

  # bytes — any length, as 0x-hex.
  extraData = "0xdeadbeef"

  # Getter-only values can be asserted but never managed, arrays included.
  expect {
    guardians = ["0x3333333333333333333333333333333333333333"]
  }
}
