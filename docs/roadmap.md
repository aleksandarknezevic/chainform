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
- [x] ABI-driven `contract` resource: derive getters/setters from a loaded ABI
      (`internal/abi` + `internal/resource/contract.go`)
- [x] `show` / state inspection: print actual on-chain state without diffing
      (`chainform show`, over the `resource.Inspector` capability)
- [x] Read-only assertions: `expect` blocks check getter-only values and report
      read-only drift as warnings, never as operations (`resource.Asserter`)
- [x] `import`: bootstrap a config from a live contract's current state — managed
      attributes + `expect` assertions, round-trips to a no-drift plan
      (`chainform import`, `config.WriteResource`)
- [x] Provider-level `validate`: builds each resource (ABI paths, known
      attributes, setter/getter pairs) without contacting the chain.
- [x] Bool toggle patterns for `contract` resources: `pause()`/`unpause()` for
      `paused` when present in the ABI (preferred over `setPaused(bool)`).
- [x] `plan` exits with code 1 when drift is detected (managed operations or
      failed `expect` assertions), so CI can gate without parsing JSON.
- [x] Multi-arch Docker images on release (`ghcr.io/<owner>/chainform`, linux
      amd64/arm64).

## Adoption & onboarding (highest priority)

The fastest path to real users is reducing time-to-first-value, not more
features. These are ordered so each unlocks the next.

- [ ] **Real, copy-paste example against mainnet.** A read-only `contract` +
      `expect` config for a well-known protocol (e.g. Lido / AAVE): fee
      parameters, oracle values, paused flags. Goal: anyone with an RPC URL
      sees value in under 5 minutes — no mock, no hand-written ABI. Ships as
      a runnable file under `examples/` plus its ABI in `testdata/`.
- [ ] **Golden-path doc.** A single end-to-end walkthrough on one well-known
      protocol: `import → edit → plan → export → Safe`, linking the real
      contract on a block explorer. Builds directly on the example above so the
      addresses and ABI are already in the repo.
- [ ] **Reusable GitHub Action for `chainform plan`.** A ready-to-use workflow
      in the repo so teams gate PRs on drift without writing CI from scratch.
      Wraps the `plan` exit code (1 on drift) and `--json`; documented with a
      copy-paste `uses:` snippet.

## Next (priority order)

- [x] **Plan output formats.** Machine-readable JSON plan (`--json`) alongside
      the human renderer, for CI gating and GitOps workflows.
- [ ] **Richer attribute types.** Arrays, structs, enums in specs; typed
      coercion from HCL. Extends spec parsing + `chain` type handling.
      (Addresses and integers are supported today.)

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
- [ ] **Scheduled drift detection.** Reconciliation is on-demand today (`chainform
      plan`). Periodic checks via cron or a Kubernetes CronJob are a supported
      workaround; a built-in watch loop or daemon is not implemented yet.
- [ ] **GitOps PR integration.** Post plan output on pull requests and gate merges.
      Today: `plan` exit codes, `--json`, and shell/`jq` scripting in CI.
- [ ] **Selective import.** `import` reads every ABI getter in one pass. Filters
      (`--include`/`--exclude`), batching, and graceful skip on revert are needed
      for large production contracts.

## Non-goals

ChainForm is not a smart-contract framework, a deployment tool, a wallet, a key
manager, or a block explorer. It manages _configuration state_ of already
deployed contracts.

## Design invariants to preserve

- Planning never sends transactions. Execution is always a separate, explicit
  step.
- Resources depend only on `chain.Reader`, never on a concrete client.
- A resource with no drift produces no operations.
- Read-only drift (`expect` assertions) is reported as a warning, never turned
  into an operation — there is no setter to execute.
- ABI encoding stays centralized in `internal/chain`.
