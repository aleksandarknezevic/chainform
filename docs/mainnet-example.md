# Mainnet example - Lido + Chainlink

This is the fastest way to see ChainForm against **real** Ethereum mainnet
contracts. No `--mock`, no demo addresses - just an RPC URL and five minutes.

The example file is [`examples/mainnet.hcl`](../examples/mainnet.hcl). It
monitors two production contracts with read-only `expect` blocks:

| Resource | Contract | What it checks |
| --- | --- | --- |
| `lidoSteth` | [Lido stETH](https://etherscan.io/address/0xae7ab96520DE3A18E5e111B5EaAb095312D7fE84) | Protocol fee (`getFee`), emergency stop (`isStopped`), token metadata |
| `ethUsdOracle` | [Chainlink ETH/USD](https://etherscan.io/address/0x5f4eC3Df9cbd43714FE2740f5E3616155c5b8419) | Oracle decimals, description, proxy version |

ABIs live in `testdata/lido-steth.abi.json` and
`testdata/chainlink-eth-usd.abi.json` - trimmed to the getters this example
uses, not full contract ABIs from Etherscan.

Nothing in this config is **managed** (no `setX` setters). `plan` never proposes
transactions; it only reports **read-only drift** when an `expect` value differs
from on-chain state. That is the right starting point for monitoring protocol
parameters you do not control directly.

For a fully offline demo with managed attributes and Safe export, see
[walkthrough.md](walkthrough.md).

## Prerequisites

- ChainForm built or installed (`make build`, a [release binary](https://github.com/aleksandarknezevic/chainform/releases), or [Docker](../README.md#docker))
- A mainnet JSON-RPC URL (Infura, Alchemy, your own node, or a public endpoint)
- Run commands from the **repository root** so relative ABI paths resolve

Copy the environment template and set your endpoint:

```bash
cp .env.example .env
# edit .env - set RPC_URL to your mainnet endpoint
export RPC_URL
```

## Quick start

```bash
make build

# 1. Validate config + ABIs (no RPC)
chainform validate -f examples/mainnet.hcl

# 2. Inspect live on-chain state
chainform show -f examples/mainnet.hcl

# 3. Check invariants - exit 0 when everything matches
chainform plan -f examples/mainnet.hcl
```

Expected `show` output (values such as `getTotalPooledEther` and `latestAnswer`
change over time; only `expect` fields are compared on `plan`):

```
contract.lidoSteth @ 0xae7ab96520DE3A18E5e111B5EaAb095312D7fE84
  decimals            = 18
  getFee              = 999
  getTotalPooledEther = ...
  isStopped           = false
  name                = Liquid staked Ether 2.0
  symbol              = stETH

contract.ethUsdOracle @ 0x5f4eC3Df9cbd43714FE2740f5E3616155c5b8419
  aggregator      = 0x...
  decimals        = 8
  description     = ETH / USD
  latestAnswer    = ...
  version         = 6
  ...
```

When all `expect` values match, `plan` prints:

```
No drift. Actual on-chain state matches desired state.
```

and exits **0**.

## Docker

```bash
docker run --rm -v "$(pwd):/work" -w /work -e RPC_URL \
  ghcr.io/aleksandarknezevic/chainform:latest \
  show -f examples/mainnet.hcl
```

## What each `expect` field means

### Lido stETH (`lidoSteth`)

```hcl
expect {
  name      = "Liquid staked Ether 2.0"
  symbol    = "stETH"
  decimals  = 18
  getFee    = 999   # protocol fee in basis points (9.99%)
  isStopped = false # emergency stop; false = protocol accepting deposits
}
```

- **`getFee`** - Lido protocol fee in basis points (999 = 9.99%). Can change via
  governance; if `plan` reports drift here, run `show` and update the value or
  investigate a governance vote.
- **`isStopped`** - emergency stop flag. When `true`, the protocol is halted
  (similar in spirit to a `paused` flag on other contracts).

`getTotalPooledEther` is readable via `show` but intentionally **not** in
`expect` - it changes every block as rewards accrue.

### Chainlink ETH/USD (`ethUsdOracle`)

```hcl
expect {
  decimals    = 8
  description = "ETH / USD"
  version     = 6
}
```

- **`decimals`** / **`description`** - stable feed metadata.
- **`version`** - aggregator proxy version; changes only on feed upgrades.

`latestAnswer` and `latestTimestamp` are visible in `show` but omitted from
`expect` because the price updates continuously.

## Simulating drift

Change one `expect` value and run `plan` again - for example set `getFee = 1000`
when on-chain is `999`:

```bash
chainform plan -f examples/mainnet.hcl
```

```
Read-only drift: 1 expectation(s) not met - no setter, cannot be changed:

  ! lidoSteth.getFee (uint256): on-chain 999, expected 1000
```

`plan` exits **1**. No transaction is proposed because there is no setter to call.

## CI gate

Fail the job when any invariant breaks:

```bash
chainform plan -f examples/mainnet.hcl
```

Requires `RPC_URL` in the environment. For PR checks against your own contracts,
commit a config under your repo and point the workflow at it.

## Adapting this pattern

1. Pick a contract and download its ABI from Etherscan (or the protocol docs).
2. Trim the ABI to zero-arg view getters you care about, or start from
   `chainform import` (see [walkthrough.md](walkthrough.md)).
3. Add a `resource "contract"` block with `expect { ... }` for read-only
   invariants, or top-level attributes for values with `setX` setters you can
   reconcile.

Related docs:

- [Configuration reference](configuration.md) - `expect` blocks and schema
- [Concepts](concepts.md) - desired vs actual state, drift, read-only assertions
- [Offline walkthrough](walkthrough.md) - import → plan → export → Safe with `--mock`
