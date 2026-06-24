# Concepts

ChainForm's vocabulary maps closely onto Terraform and Kubernetes. If you know
either, this will feel familiar. For what is implemented today versus planned,
see the [README](../README.md#what-works-today) and [roadmap](roadmap.md).

## Desired state

The configuration file is the source of truth for how a protocol *should* be
configured. It is declarative: you describe the end state, not the steps. Only
attributes you declare are managed — anything omitted is left untouched on
chain. See [configuration.md](configuration.md) for the schema.

## Actual state

The real, current configuration of the contracts, read from the chain at plan
time via read-only `eth_call`s. ChainForm never assumes or caches actual state;
it always observes it fresh.

## Drift

The difference between desired and actual state. When `feeBps` is `30` in
config but `50` on chain, that attribute has drifted. Drift detection is per
attribute, and resources only report drift for attributes that are declared.

## Resource

A managed on-chain entity, analogous to a Terraform resource. A resource type
(e.g. `protocol`) knows how to:

- **Refresh** — read its current state from the chain.
- **Plan** — compare desired vs. actual and emit the operations to converge.

Resources implement the [`resource.Resource`](../internal/resource/resource.go)
interface. Two types ship today: `protocol`, a hand-written reference type
(manages `feeBps` and `paused`) that demonstrates the contract end to end; and
`contract`, an ABI-driven type that derives its getters and setters from a
loaded ABI so arbitrary contracts can be managed without writing Go. For bool
`paused`, the `contract` resource prefers `pause()`/`unpause()` when the ABI
exposes them (OpenZeppelin Pausable); otherwise it falls back to `setPaused(bool)`.

## Operation

A single contract call required to reduce drift — for example
`setFeeBps(30)`. An operation carries its target address, method, ABI input
types, arguments, a human-readable drift reason, and (after planning) the
encoded calldata. Operations are the atoms of a plan.

## Plan

The ordered set of operations that, when executed, converge actual state to
desired state. A plan is read-only output: it is reviewed by humans and/or
exported. An empty plan means no drift.

## Export target

A rendering of a plan into a format some external system can execute. The
initial target is a **Safe Transaction Builder batch**, suitable for multisig
review and execution. Future targets may include direct apply and governance
proposals.

## Reader

The abstraction over reading chain state
([`chain.Reader`](../internal/chain/reader.go)). Implementations:

- `chain.Client` — live JSON-RPC via go-ethereum.
- `chain.MockReader` — programmable canned values, for tests.
- `chain.DemoReader` — fixed drifted values, powering `--mock` for offline demos.
