# Architecture

ChainForm is structured around a single idea borrowed from Kubernetes
controllers and Terraform: a **reconciliation loop**. Declared desired state is
compared against observed actual state, and the difference is turned into the
minimal set of operations needed to converge them.

```
            ┌─────────────────────── reconcile ───────────────────────┐
            │                                                          │
  Desired state (HCL)                                           Actual state (chain)
            │                                                          │
       config.Load ──► resource.Build ──► Resource.Refresh ◄── chain.Reader (eth_call)
            │                                   │
            │                            current State
            │                                   │
            └──────────────► Resource.Plan(current) ──► []Operation
                                                │
                                       chain.Pack (ABI encode)
                                                │
                                             plan.Plan
                                          ┌─────┴─────┐
                                     Render (review)  export.Safe (execute)
```

## Package layout

| Package | Responsibility | Depends on |
| --- | --- | --- |
| `internal/config` | Desired-state schema, HCL/JSON loader, validation | - |
| `internal/chain` | EVM reads (`Reader`), ABI encode/decode, type support, live + mock readers | go-ethereum |
| `internal/abi` | ABI loader; derives getters/setters/attributes for ABI-driven resources | `chain`, go-ethereum |
| `internal/resource` | `Resource` contract, `Operation`, type registry, `protocol` + ABI-driven `contract` | `config`, `chain`, `abi` |
| `internal/plan` | Reconciliation pass + human-readable rendering | `config`, `chain`, `resource` |
| `internal/export` | Render a plan into executable formats (Safe batch) | `plan` |
| `internal/cli` | Cobra command tree (`validate`, `import`, `show`, `plan`, `export`, `version`) | all of the above |
| `cmd/chainform` | Entrypoint | `cli` |

Dependencies point in one direction. `chain` knows nothing about resources or
config; resources depend only on the small `chain.Reader` interface, never on a
concrete client. That keeps the whole reconciliation path testable offline with
`chain.MockReader` / `chain.DemoReader`.

## The reconciliation pass

[`plan.Planner.Run`](../internal/plan/planner.go) is the loop body. For each
configured resource it:

1. **Builds** the resource from config via the type registry
   (`resource.Build`).
2. **Refreshes** actual state by issuing read-only calls through a
   `chain.Reader` (`Resource.Refresh`).
3. **Diffs** desired vs. actual and emits operations (`Resource.Plan`). A
   resource with no drift returns no operations.
4. **Encodes** each operation's calldata (`chain.Pack`).

The result is a `plan.Plan`: an ordered, chain-scoped list of operations that
can be rendered for review or exported for execution. Nothing in this path
sends a transaction - execution is intentionally a separate, explicit step
(today: export to a Safe batch; later: an apply engine).

## Why this shape

- **Providers are pluggable.** New resource types register themselves
  (`resource.Register`) in `init()`. The planner discovers them through the
  registry; no central switch statement to edit. See
  [adding-a-resource.md](adding-a-resource.md).
- **Reads are abstracted.** Anything implementing `chain.Reader` works:
  the live JSON-RPC client, a mock with canned values, or the demo reader used
  by `--mock`.
- **Encoding is centralized.** All ABI packing/decoding lives in
  [`internal/chain/abi.go`](../internal/chain/abi.go), so resources describe
  calls by name + types and never touch keccak or padding.
- **Types are described by string.** A call carries ABI type strings
  (`["address[]"]`), which keeps `ViewCall` and `Operation` free of go-ethereum
  types - at the cost of tuples, whose type string cannot be parsed back into a
  type. [`chain.SupportedType`](../internal/chain/types.go) is the single place
  that decides what ChainForm can encode or decode; unsupported getters and
  setters are filtered out when attributes are derived, and encoding rejects
  them, so nothing is ever silently mis-decoded.

## The value layer

Three representations of one attribute value meet in
[`internal/resource/value.go`](../internal/resource/value.go): what the config
loader produced (`bool`, `string`, `int`, `[]any`), what the chain decoded
(`*big.Int`, `common.Address`, `[32]byte`, `[]common.Address`, …), and the exact
Go type the ABI encoder wants. `canonical` folds the first two into one
comparable form so drift detection is decoder-agnostic; `setterArg` produces the
third. Adding an attribute type means teaching those two functions (plus
`chain.SupportedType`) about it - not touching resources.

See [concepts.md](concepts.md) for the vocabulary, [golden-path.md](golden-path.md)
for the full loop on a real mainnet contract, [mainnet-example.md](mainnet-example.md)
for live read-only monitoring, and [roadmap.md](roadmap.md) for where this is
heading.
