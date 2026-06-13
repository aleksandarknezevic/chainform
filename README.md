# ChainForm

**Declarative, GitOps-friendly management of on-chain smart-contract state for
EVM protocols.** Infrastructure as Code (IaC) for blockchain - Terraform for
deployed contracts.

ChainForm lets protocol teams declare the desired _configuration state_ of their
deployed contracts in version-controlled HCL, read the actual state from the
chain, detect configuration drift, and review a Terraform-style plan before
anything executes. Approved changes are exported as a Safe (Gnosis Safe)
multisig transaction batch.

Think Terraform / Kubernetes reconciliation, but for the configuration state of
already-deployed EVM contracts — fees, roles, owners, pause switches, treasury
and admin addresses.

```
Desired State (Git) → ChainForm → Plan → Drift Detection → Operations → Safe / Governance / Apply
```

## Why ChainForm

- **Declarative desired state** — your protocol's intended configuration lives in
  Git as HCL, not in ad-hoc scripts, Etherscan tabs, or someone's memory.
- **Drift detection** — compare on-chain reality against desired state and see
  exactly what changed; read-only values (no setter) are reported as drift but
  never turned into transactions.
- **Reviewable diffs** — every change is a human-readable, Terraform-style plan
  you can review in a pull request and approve before execution.
- **GitOps workflow** — commit desired state → run `chainform plan` in CI →
  review the diff on the PR → export a Safe batch for multisig execution.
  Planning never touches keys and never sends transactions.
- **ABI-driven** — point it at any contract ABI and it derives the getters and
  setters automatically; no per-contract Go code to manage arbitrary contracts.

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

That feed is read-only, so nothing can be _managed_ — but an `expect` block
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

**Now:** EVM chains; declarative HCL config; read on-chain contract state;
ABI-driven resources (auto-derived getters/setters); drift detection and
Terraform-style plans; read-only assertions (`expect`); state inspection
(`show`); Safe / Gnosis Safe transaction-batch export.

**Later:** apply engine, governance simulation, multi-chain reconciliation,
config bootstrap from a live contract (`import`), continuous drift monitoring
(`watch`/alerts), AccessControl / Proxy / Timelock resources, deeper GitOps
integration. See the [roadmap](docs/roadmap.md).

ChainForm is **not** a smart-contract framework, a deployment tool, a wallet, a
key manager, or a block explorer. It is a declarative configuration-management
and drift-detection tool for the _state_ of already-deployed contracts.

---

<sub>Keywords: smart contract configuration management · on-chain state drift
detection · declarative infrastructure as code for EVM / Ethereum protocols ·
GitOps for smart contracts · Terraform-style plan & diff · Gnosis Safe multisig
transaction batches.</sub>
