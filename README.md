# ChainForm

**Infrastructure as Code and Configuration Management for Blockchain Protocols**

ChainForm is a declarative configuration management tool for blockchain protocols.

It helps protocol teams manage smart contract configuration, governance parameters, access control, treasury settings, and protocol operations using workflows inspired by Terraform, Kubernetes reconciliation loops, and GitOps.

Instead of manually executing governance transactions, multisig actions, and operational scripts, teams define the desired protocol state and let ChainForm generate reviewable reconciliation plans.

## Why ChainForm?

Most protocol configuration today is managed through:

- ad hoc scripts
- multisig transactions
- governance proposals
- spreadsheets
- internal documentation

This creates several problems:

- no single source of truth
- difficult audits
- manual protocol operations
- configuration drift
- inconsistent settings across deployments
- poor change visibility

ChainForm continuously compares desired state with actual on-chain state, detects drift, and generates the minimal set of operations required to reconcile them.

## Features

- Infrastructure as Code for blockchain protocols
- Declarative protocol configuration
- Smart contract state reconciliation
- Configuration drift detection
- Governance transaction planning
- Safe transaction export
- Existing protocol import
- Reviewable execution plans
- GitOps-compatible workflows
- Multi-contract configuration management

## Who Is ChainForm For?

ChainForm is designed for:

- DeFi protocol teams
- DAO operators
- Protocol engineers
- Smart contract platform teams
- Governance contributors
- Multisig operators
- Blockchain DevOps teams

Typical use cases include:

- protocol parameter management
- fee configuration
- treasury administration
- role management
- access control management
- timelock administration
- multisig governance operations
- protocol upgrades

## How It Works

Define the desired state in HCL. Attribute names mirror the contract's
functions — `feeBps` is read via `feeBps()` and reconciled via `setFeeBps()`:

```hcl
resource "protocol" "main" {
  address = "0xF38D8Be3E0A7B3c94C00a25b4A443ca062f343f5"
  feeBps  = 30
  paused  = false
}
```

Inspect the current on-chain state (no diff):

```bash
chainform show -f protocol.hcl
```

Generate a reconciliation plan — the drift between desired and actual state:

```bash
chainform plan -f protocol.hcl
```

Need a machine-readable plan for CI or GitOps checks?

```bash
chainform plan -f protocol.hcl --json
```

Example CI gate (fail if any operations are proposed):

```bash
chainform plan -f protocol.hcl --json | jq -e '.summary.operationCount == 0'
```

JSON field-by-field reference is documented in
**[Plan JSON format](docs/plan-json.md)**.

Example output:

```
Plan: 2 operation(s)

  1. main.setFeeBps(30)
       drift: feeBps: 50 -> 30
  2. main.unpause()
       drift: paused: true -> false
```

Export the operations as a Safe transaction batch for multisig execution:

```bash
chainform export -f protocol.hcl -o batch.json
```

No RPC endpoint needed to try it — add `--mock` to use a built-in demo reader.
The repo ships runnable examples:

```bash
chainform plan -f examples/protocol.hcl --mock
chainform show -f examples/feed.hcl     --mock   # read-only Chainlink ETH/USD feed
```

## Import Existing Protocols

Most protocols already exist on-chain.

ChainForm can import an existing contract and generate an initial desired-state
definition from its current on-chain values — managed attributes plus read-only
`expect` assertions:

```bash
chainform import \
  --address 0xF38D8Be3E0A7B3c94C00a25b4A443ca062f343f5 \
  --abi protocol.abi.json --name main --chain-id 1 -o protocol.hcl
```

An immediate `plan` against the freshly imported config reports no drift, so you
can adopt ChainForm gradually without rebuilding existing operational workflows.
Add `--mock` to try it offline.

👉 See the full **[import → plan → export walkthrough](docs/walkthrough.md)** —
a copy-pasteable, offline end-to-end example.

## Commands

```bash
chainform validate   # check a config without contacting the chain
chainform import     # generate a config from a live contract
chainform show       # print actual on-chain state, without diffing
chainform plan       # detect drift and print the reconciliation plan
chainform plan --json # emit a machine-readable JSON plan (CI/GitOps friendly)
chainform export     # export the plan as a Safe transaction batch
chainform version    # print the version
```

## Why Not Scripts?

Scripts can change state.

Scripts do not tell you:

- what changed
- what drifted
- whether actual state matches expectations
- what still requires reconciliation

ChainForm focuses on continuously reconciling desired and actual protocol state.

## Is ChainForm Terraform for Blockchain?

Partially.

Terraform focuses on provisioning infrastructure.

ChainForm focuses on managing and reconciling protocol configuration after deployment.

A more accurate analogy is:

- Terraform + Kubernetes reconciliation
- GitOps for blockchain protocols
- Configuration management for smart contracts

## Current Roadmap

### Current

- Declarative protocol definitions
- ABI-driven contract resources (auto-derived getters/setters)
- State inspection (`show`)
- Reconciliation planning + drift detection
- Read-only assertions (`expect`)
- Safe transaction export
- Protocol import

### Next

- Richer attribute types (addresses, arrays, structs, enums)
- AccessControl resources
- Proxy resources
- Apply engine (explicit execution step, separate from planning)
- Multi-chain reconciliation

### Future

- Governance simulation
- GitOps workflows
- Drift monitoring
- Hosted control plane

## FAQ

### Does ChainForm deploy contracts?

No.

ChainForm manages protocol state after deployment.

### Does ChainForm manage private keys?

No.

ChainForm integrates with existing signing and governance systems.

### Does ChainForm replace Safe?

No.

ChainForm generates operations that can be executed through Safe.

### Is ChainForm a governance platform?

No.

ChainForm focuses on protocol configuration management and transaction planning.

### Can ChainForm be adopted incrementally?

Yes.

Existing protocols can be imported and managed gradually without changing governance processes.
