# ChainForm

Infrastructure as Code for blockchain protocols.

ChainForm lets protocol teams define desired on-chain state in code, detect
configuration drift, and generate safe, reviewable operations before execution.

It behaves like a Kubernetes controller's reconciliation loop rather than a
deployment tool: continuously compare desired state with actual on-chain state
and produce the minimal set of operations needed to converge them.

```
Desired State (Git) → ChainForm → Plan → Drift Detection → Operations → Safe / Governance / Apply
```

## Example

Desired (config, HCL):

```hcl
resource "protocol" "main" {
  address = "0x..."
  feeBps  = 30
  paused  = false
}
```

Actual (chain): `feeBps = 50`, `paused = true`

Plan:

```
setFeeBps(30)
unpause()
```

## Quick start

No RPC endpoint required — the built-in demo reader supplies drifted state:

```bash
make build
./bin/chainform plan   -f examples/protocol.hcl --mock
./bin/chainform export -f examples/protocol.hcl --mock -o batch.json
```

Or inspect a contract's live state with no config to write — `examples/feed.hcl`
points at the real Chainlink ETH/USD price feed on Sepolia:

```bash
./bin/chainform show -f examples/feed.hcl --mock          # canned demo values
RPC_URL=<sepolia-rpc> ./bin/chainform show -f examples/feed.hcl   # live on-chain
```

That feed is read-only, so nothing can be *managed* — but an `expect` block
asserts what its getters should return and reports **read-only drift** (a
warning that never becomes a transaction) when they don't.

Run against a live network by setting `RPC_URL` and dropping `--mock`.

## Commands

| Command                                    | Description                                               |
| ------------------------------------------ | --------------------------------------------------------- |
| `chainform validate -f <file>`             | Validate a configuration without contacting the chain     |
| `chainform show -f <file>`                 | Print actual on-chain state, without diffing              |
| `chainform plan -f <file>`                 | Read actual state, detect drift, print the plan           |
| `chainform export -f <file> -o batch.json` | Generate a plan and export it as a Safe transaction batch |
| `chainform version`                        | Print the version                                         |

Add `--mock` to `show`/`plan`/`export` to use the offline demo reader.

## Project layout

```
cmd/chainform/      CLI entrypoint
internal/config/    desired-state schema + loader
internal/chain/     EVM reads, ABI encode/decode, live + mock readers
internal/abi/       ABI loader; derives getters/setters for ABI-driven resources
internal/resource/  Resource contract, registry, "protocol" + ABI-driven "contract"
internal/plan/      reconciliation (refresh → diff → plan) + rendering
internal/export/    Safe transaction batch export
examples/           runnable example configuration
docs/               architecture, concepts, configuration, roadmap
```

## Documentation

- [Architecture](docs/architecture.md) — package map and the reconciliation flow
- [Concepts](docs/concepts.md) — desired/actual state, drift, resources, plans
- [Configuration reference](docs/configuration.md) — the HCL schema
- [Adding a resource type](docs/adding-a-resource.md) — the main extension point
- [Roadmap](docs/roadmap.md) — what's implemented and what's next
- [Contributing](CONTRIBUTING.md) — dev workflow and conventions

## Scope

**Now:** EVM chains, read contract state, detect drift, generate plans, export
Safe transactions.

**Later:** apply engine, simulation, multi-chain reconciliation, AccessControl
and Proxy resources, GitOps integration. See the [roadmap](docs/roadmap.md).

ChainForm is **not** a smart-contract framework, a deployment tool, a wallet, a
key manager, or a block explorer. It manages the _configuration state_ of
already-deployed contracts.
