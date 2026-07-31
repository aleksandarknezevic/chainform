# Roadmap

ChainForm today is a **CLI for read → plan → export**. It does not apply
transactions, run governance, or host a control plane. The sections below
separate what ships now from what is planned.

This repo implements the read-only half of the loop end to end:
desired-state config → drift detection → plan → Safe export.

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
- [x] `import`: bootstrap a config from a live contract's current state - managed
      attributes + `expect` assertions, round-trips to a no-drift plan
      (`chainform import`, `config.WriteResource`)
- [x] Provider-level `validate`: builds each resource (ABI paths, known
      attributes, setter/getter pairs) without contacting the chain.
- [x] Bool toggle patterns for `contract` resources: `pause()`/`unpause()` for
      `paused` when present in the ABI (preferred over `setPaused(bool)`).
- [x] `plan` exits with code 1 when drift is detected (managed operations or
      failed `expect` assertions), so CI can gate without parsing JSON. A
      command that cannot run exits 2, so a broken endpoint is never mistaken
      for drift.
- [x] Multi-arch Docker images on release (`ghcr.io/<owner>/chainform`, linux
      amd64/arm64).
- [x] Reusable GitHub Action ([`action.yml`](../action.yml)): installs a release
      binary, runs `plan`, posts the plan to the job summary, and exposes
      `drift` / `exit-code` / plan file outputs
      ([docs](github-action.md), [template](../examples/workflows/drift-check.yml)).
- [x] Richer attribute types: `bytes`, `bytesN`, enums, and dynamic or
      fixed-size arrays of any supported element type, end to end (config
      coercion, drift comparison, setter encoding, `show`, `import`, JSON plan)
      - [`examples/vault.hcl`](../examples/vault.hcl).

## Adoption & onboarding (highest priority)

The fastest path to real users is reducing time-to-first-value, not more
features. These are ordered so each unlocks the next.

- [x] **Real, copy-paste example against mainnet.** Read-only `contract` +
      `expect` config for Lido stETH (fee, emergency stop) and Chainlink ETH/USD
      (oracle metadata) on Ethereum mainnet - [`examples/mainnet.hcl`](../examples/mainnet.hcl),
      ABIs in `testdata/`, walkthrough in [mainnet-example.md](mainnet-example.md).
- [x] **Golden-path doc.** [golden-path.md](golden-path.md) - one end-to-end
      walkthrough on the Uniswap V3 Factory (`owner()`/`setOwner`) on Ethereum
      mainnet: `import → plan → edit → plan → export → Safe`, linked to
      Etherscan, ABI in `testdata/`, with an offline (`--mock`) variant. The
      flow is covered by a test so the doc cannot silently rot.
- [x] **Reusable GitHub Action for `chainform plan`.** [`action.yml`](../action.yml)
      at the repository root, documented in [github-action.md](github-action.md)
      with a copy-paste
      [workflow template](../examples/workflows/drift-check.yml). Wraps the
      `plan` exit code and `--json`, writes the plan to the job summary, and is
      itself tested on every push (`.github/workflows/action.yml`).

## Next (priority order)

- [x] **Plan output formats.** Machine-readable JSON plan (`--json`) alongside
      the human renderer, for CI gating and GitOps workflows.
- [x] **Richer attribute types.** `bytes`, `bytesN`, enums, and dynamic or
      fixed-size arrays, with typed coercion from HCL and JSON. Lists compare
      element by element; `bytesN` is length-checked at load time.
- [ ] **Struct (tuple) attributes.** Still unsupported, and deliberately so: a
      tuple's ABI type string (`(uint256,bool)`) cannot be parsed back into a
      type, and calls are described by type string throughout
      (`chain.ViewCall`, `resource.Operation`). Struct-valued getters and
      setters are therefore skipped rather than mis-decoded
      (`chain.SupportedType`). Supporting them means carrying resolved ABI
      types (or component lists) through the call description instead of
      strings - a deliberate interface change, not a value-layer patch.
- [ ] **PR plan comments.** The action posts to the job summary today; commenting
      the plan on the pull request needs `pull-requests: write` and comment
      de-duplication. First step of GitOps PR integration below.

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
      plan`). Periodic checks via cron, a Kubernetes CronJob, or a `schedule:`
      trigger on the shipped action are supported workarounds; a built-in watch
      loop or daemon is not implemented yet.
- [ ] **GitOps PR integration.** Gate merges and post plan output on pull
      requests. Today: the action (job summary, exit code, outputs), `--json`,
      and shell/`jq` scripting in CI.
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
- A type ChainForm cannot encode or decode is never guessed at: unsupported
  getters and setters are filtered out of attribute derivation, and encoding
  rejects them (`chain.SupportedType`).
- Read-only drift (`expect` assertions) is reported as a warning, never turned
  into an operation - there is no setter to execute.
- ABI encoding stays centralized in `internal/chain`.
