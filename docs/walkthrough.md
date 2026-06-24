# Walkthrough: import → plan → export

This is the full **managed-reconciliation** loop end to end, entirely offline.
Every command below uses `--mock` (a built-in demo reader) so you can follow
along without an RPC endpoint or a real contract.

For **live mainnet** read-only monitoring (Lido + Chainlink, no `--mock`), see
**[mainnet-example.md](mainnet-example.md)**.

Drop `--mock` and set `RPC_URL` to run it for real.

Build the binary first:

```bash
make build
```

## 1. Import a contract into a config

You don't start from a blank file - point `import` at a deployed contract and
its ABI, and ChainForm snapshots the current on-chain state into HCL:

```bash
./bin/chainform import \
  --address 0xF38D8Be3E0A7B3c94C00a25b4A443ca062f343f5 \
  --abi testdata/protocol.abi.json \
  --name main --chain-id 1 --chain-name ethereum \
  --mock -o protocol.hcl
```

`protocol.hcl`:

```hcl
version = "1"

chain {
  name     = "ethereum"
  chain_id = 1
  rpc      = env("RPC_URL")
}

resource "contract" "main" {
  address = "0xF38D8Be3E0A7B3c94C00a25b4A443ca062f343f5"
  abi     = "testdata/protocol.abi.json"

  feeBps = 50
  owner  = "0x21f73D42Eb58Ba49dDB685dc29D3bF5c0f0373CA"
  paused = true

  expect {
    name = "Demo Protocol"
  }
}
```

Two kinds of attributes are derived from the ABI:

- **Managed** (top-level): values with both a getter and a `setX` setter -
  `feeBps`/`setFeeBps`, `owner`/`setOwner`, `paused`/`pause`+`unpause` (or
  `setPaused` when toggle methods are absent). ChainForm can reconcile these.
- **`expect`** (read-only): values with a getter but no setter - `name`. These
  can drift but can never be changed, so they are asserted, not managed.

## 2. Plan - a faithful snapshot has no drift

Because `import` captured the current values, planning against the same state
proposes nothing:

```bash
./bin/chainform plan -f protocol.hcl --mock
```

```
No drift. Actual on-chain state matches desired state.
```

This is the point of import: you adopt ChainForm without changing anything.

## 3. Change the desired state, then plan again

Now edit `protocol.hcl` and set the desired fee to `30`:

```hcl
  feeBps = 30
```

Planning now shows exactly the operation needed to converge - and the ABI-encoded
calldata for it:

```bash
./bin/chainform plan -f protocol.hcl --mock
```

For automation (CI / GitOps), emit the same plan as JSON:

```bash
./bin/chainform plan -f protocol.hcl --mock --json
```

```
{
  "chain": {
    "name": "ethereum",
    "chainId": 1,
    "rpc": ""
  },
  "operations": [
    {
      "resource": "main",
      "to": "0xF38D8Be3E0A7B3c94C00a25b4A443ca062f343f5",
      "method": "setFeeBps",
      "inputs": ["uint256"],
      "args": [30],
      "valueWei": "0",
      "reason": "feeBps: 50 -> 30",
      "calldata": "0x72c27b62000000000000000000000000000000000000000000000000000000000000001e"
    }
  ],
  "assertions": [],
  "summary": {
    "operationCount": 1,
    "assertionCount": 0,
    "failedAssertionCount": 0,
    "empty": false
  }
}
```

For the complete JSON field reference, see
**[Plan JSON format](plan-json.md)**.

## 4. Export for execution

Turn the plan into a Safe (Gnosis Safe) Transaction Builder batch you can import
into the Safe app for multisig review and execution:

```bash
./bin/chainform export -f protocol.hcl --mock -o batch.json
```

Only operations become transactions - read-only `expect` assertions never do.

## 5. Show - inspect state any time

`show` prints the current on-chain values without diffing, including the
read-only ones:

```bash
./bin/chainform show -f protocol.hcl --mock
```

```
contract.main @ 0xF38D8Be3E0A7B3c94C00a25b4A443ca062f343f5
  feeBps = 50
  name   = "Demo Protocol"
  owner  = 0x21f73D42Eb58Ba49dDB685dc29D3bF5c0f0373CA
  paused = true
```

## Going live

Everything above used `--mock`. To run against a real network, drop `--mock`
and provide an endpoint:

```bash
export RPC_URL=https://sepolia.infura.io/v3/<key>
./bin/chainform show -f protocol.hcl
./bin/chainform plan -f protocol.hcl
```
