# ChainForm

[![Tests](https://github.com/aleksandarknezevic/chainform/actions/workflows/test.yml/badge.svg)](https://github.com/aleksandarknezevic/chainform/actions/workflows/test.yml)
[![Release](https://img.shields.io/github/v/release/aleksandarknezevic/chainform)](https://github.com/aleksandarknezevic/chainform/releases)
[![Go Version](https://img.shields.io/github/go-mod/go-version/aleksandarknezevic/chainform)](https://github.com/aleksandarknezevic/chainform/blob/main/go.mod)
[![License](https://img.shields.io/github/license/aleksandarknezevic/chainform)](https://github.com/aleksandarknezevic/chainform/blob/main/LICENSE)

**ChainForm is an early-stage CLI** that compares declared on-chain configuration
(HCL/JSON) against live contract state, reports drift, and encodes the resulting
calls as reviewable operations. **It does not execute transactions** - you review
the plan and run it through your existing multisig or governance flow.

Install:

```bash
go install github.com/aleksandarknezevic/chainform/cmd/chainform@latest
```

Or use a [release binary](https://github.com/aleksandarknezevic/chainform/releases) /
the [Docker image](#docker).

## What works today

| Capability | Status |
| --- | --- |
| Declarative config (HCL + JSON), one EVM chain per file | **Yes** |
| `validate` - schema + resource/ABI checks (no RPC) | **Yes** |
| `show` - read on-chain state (`eth_call`) | **Yes** |
| `plan` - drift detection, human + `--json` output, exit 1 on drift | **Yes** |
| `import` - snapshot a contract into config (getter/setter + `expect`) | **Yes** |
| `export` - Safe Transaction Builder JSON batch (import manually in Safe app) | **Yes** |
| ABI-driven `contract` resource (`X()` / `setX`, `pause`/`unpause`) | **Yes** |
| Read-only monitoring via `expect` blocks | **Yes** |
| Mainnet example (Lido + Chainlink) | **Yes** - [docs/mainnet-example.md](docs/mainnet-example.md) |
| Offline demo (`--mock`) | **Yes** |

**Supported today:** simple scalar types (bool, string, address, integers),
single-argument setters following `setX` naming (plus `pause`/`unpause` for
`paused`), one chain per config, manual `chainform plan` runs.

## What is not built yet

Do not expect these today - they are on the [roadmap](docs/roadmap.md):

| Capability | Status |
| --- | --- |
| Execute / apply plans (sign and send txs) | **No** - plan and export only |
| AccessControl / `grantRole` resources | **No** |
| Proxy / upgrade admin resources | **No** |
| Governance proposal export (Tally, OZ Governor, …) | **No** - Safe batch JSON only |
| Multi-chain in one config | **No** |
| Continuous or scheduled drift monitoring | **No** - run `plan` yourself (cron/K8s works) |
| GitHub App / PR plan comments | **No** - use `plan` exit code or `--json` in your CI |
| Hosted control plane / SaaS | **No** |
| Complex ABI types (structs, arrays, enums) | **Limited** - scalars only |

ChainForm is **not** a governance platform, wallet, deployment tool, or block
explorer. It does **not** hold private keys.

## Try it

**Mainnet (read-only, ~5 min)** - needs `RPC_URL`:

```bash
export RPC_URL=https://your-mainnet-rpc.example
chainform validate -f examples/mainnet.hcl
chainform show   -f examples/mainnet.hcl
chainform plan   -f examples/mainnet.hcl
```

**Offline (managed drift + Safe export)** - no RPC:

```bash
chainform plan   -f examples/protocol.hcl --mock
chainform export -f examples/protocol.hcl --mock -o batch.json
```

See [mainnet-example.md](docs/mainnet-example.md) and the offline
[walkthrough](docs/walkthrough.md).

## The problem (and the direction)

Protocol teams still coordinate parameter changes through scripts, multisig
queues, governance votes, and spreadsheets. That makes drift hard to see and
changes hard to review.

ChainForm's **current** loop is intentionally narrow:

```
config → plan (read chain, diff) → review → export to Safe batch → humans execute
```

The longer-term direction is GitOps-style protocol ops (PR checks, richer
resources, optional apply) - see [roadmap](docs/roadmap.md). Today you get the
**read and plan** half of that loop, not the full ArgoCD-style control plane.

## How it works

Declare desired state. Attribute names mirror contract getters/setters - e.g.
`feeBps` is read via `feeBps()` and reconciled via `setFeeBps()` when drifted:

```hcl
resource "protocol" "main" {
  address = "0xF38D8Be3E0A7B3c94C00a25b4A443ca062f343f5"
  feeBps  = 30
  paused  = false
}
```

```bash
chainform show -f protocol.hcl          # actual state, no diff
chainform plan -f protocol.hcl          # drift → operations (exit 1 if drift)
chainform plan -f protocol.hcl --json   # machine-readable plan
chainform export -f protocol.hcl -o batch.json   # Safe Transaction Builder JSON
```

Example plan output:

```
Plan: 2 operation(s)

  1. main.setFeeBps(30)
       drift: feeBps: 50 -> 30
  2. main.unpause()
       drift: paused: true -> false
```

CI gate (fails when drift is detected):

```bash
chainform plan -f protocol.hcl --mock
```

JSON reference: [docs/plan-json.md](docs/plan-json.md).

### Import an existing contract

Bootstrap config from live state (managed attrs + read-only `expect`):

```bash
chainform import \
  --address 0xF38D8Be3E0A7B3c94C00a25b4A443ca062f343f5 \
  --abi protocol.abi.json --name main --chain-id 1 -o protocol.hcl
```

A plan against the imported snapshot should report no drift until you change
desired values. `import` reads every ABI getter - large contracts may need
care; see roadmap for selective import.

## Commands

```bash
chainform validate    # config + resources, no RPC
chainform import      # snapshot live contract → HCL
chainform show        # on-chain state, no diff
chainform plan        # drift + operations (exit 1 on drift)
chainform plan --json
chainform export      # Safe Transaction Builder batch JSON
chainform version
```

## Who is this for (today)

Best fit **right now**:

- Engineers who want **config-as-code** for a few contracts with simple
  `getter` / `setX` (or `pause`/`unpause`) pairs
- Multisig operators who want a **reviewable calldata batch** before signing in
  Safe
- Teams experimenting with **drift checks** in CI (`plan` exit code) or
  **read-only invariants** (`expect`) on mainnet

**Not a fit yet** if you need role graphs, proxy upgrades, governor proposals,
automatic execution, or multi-chain reconciliation - track
[roadmap](docs/roadmap.md) instead.

## Docker

```bash
docker run --rm -v "$(pwd):/work" -w /work \
  ghcr.io/aleksandarknezevic/chainform:latest \
  plan -f examples/protocol.hcl --mock
```

Multi-arch (`linux/amd64`, `linux/arm64`) on each release. Pin a version tag in
production; pass `-e RPC_URL=...` for live runs.

## Roadmap

Full detail: **[docs/roadmap.md](docs/roadmap.md)**.

**Next up:** golden-path doc on a real protocol, reusable GitHub Action for
`plan`, richer ABI types.

**Later:** apply engine, AccessControl/proxy resources, governance export targets,
multi-chain, scheduled monitoring, GitOps PR integration.

## FAQ

### Does ChainForm deploy contracts?

No. It manages configuration of contracts that are already deployed.

### Does ChainForm send transactions?

No. `plan` and `export` are read-only with respect to the chain. Execution is
manual (e.g. import the Safe batch and sign).

### Does ChainForm replace Safe or governance tools?

No. It produces calldata batches and plans for humans to execute in existing
workflows.

### Is this production-ready?

It is early software with a focused feature set (see tables above). Use it where
the current scope matches your contracts; verify plans before any mainnet
execution.

### Can I adopt it incrementally?

Yes. `import` captures current state; `plan` shows only declared attributes;
`expect` adds monitoring without write paths.
