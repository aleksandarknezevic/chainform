# Roadmap

This boilerplate implements the read-only half of the loop end to end:
desired-state config → drift detection → plan → Safe export. The items below
are roughly ordered and mapped to where they land in the codebase.

## Now (implemented)

- [x] Desired-state HCL schema + loader (`internal/config`)
- [x] EVM read path + ABI encode/decode (`internal/chain`)
- [x] Resource contract + registry + reference `protocol` resource (`internal/resource`)
- [x] Reconciliation pass + plan rendering (`internal/plan`)
- [x] Safe Transaction Builder export (`internal/export`)
- [x] CLI: `validate`, `plan`, `export`, `version` (`internal/cli`)
- [x] Offline demo/mock readers for tests and `--mock`

## Next

- [ ] **ABI-driven resources.** Load a contract ABI and derive getters/setters
      instead of hand-writing them. Removes per-resource boilerplate and makes
      arbitrary contracts manageable. Lands in `internal/resource` + a new
      `internal/abi` loader.
- [ ] **Richer attribute types.** Addresses, arrays, structs, enums in specs;
      typed coercion from HCL. Extends spec parsing + `chain` type handling.
- [ ] **`show` / state inspection.** Print actual on-chain state without
      diffing, for debugging. New CLI command over `Resource.Refresh`.
- [ ] **Plan output formats.** Machine-readable JSON plan (`--json`) alongside
      the human renderer, for CI gating.

## Later

- [ ] **Apply engine.** Execute a plan directly with a signer, with
      confirmation and per-op gating. New `internal/apply` package; keep it
      strictly separate from planning.
- [ ] **Simulation.** Dry-run operations (eth_call / state override / fork) to
      validate a plan before execution.
- [ ] **AccessControl resources.** Manage roles/grants
      (`grantRole`/`revokeRole`) as a resource type.
- [ ] **Proxy resources.** Manage upgradeable proxy implementation/admin.
- [ ] **Governance export targets.** Emit proposals (e.g. OZ Governor,
      Tally-compatible) as an alternative to Safe batches.
- [ ] **Multi-chain reconciliation.** One config spanning several chains, with
      per-chain plans.
- [ ] **GitOps integration.** Run `plan` on PRs and post the diff; gate merges
      on no-unexpected-drift.

## Non-goals

ChainForm is not a smart-contract framework, a deployment tool, a wallet, a key
manager, or a block explorer. It manages *configuration state* of already
deployed contracts.

## Design invariants to preserve

- Planning never sends transactions. Execution is always a separate, explicit
  step.
- Resources depend only on `chain.Reader`, never on a concrete client.
- A resource with no drift produces no operations.
- ABI encoding stays centralized in `internal/chain`.
