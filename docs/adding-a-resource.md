# Adding a resource type

Resource types are the primary extension point. A new type plugs into the
reconciliation loop by implementing one interface and registering itself — no
changes to the planner, CLI, or config schema are required.

Use [`internal/resource/protocol.go`](../internal/resource/protocol.go) as the
working template.

## 1. Implement `resource.Resource`

```go
type Resource interface {
    Type() string                                              // matches config "type"
    Name() string                                              // matches config "name"
    Address() common.Address
    Refresh(ctx context.Context, r chain.Reader) (State, error) // read actual state
    Plan(current State) ([]Operation, error)                    // diff -> operations
}
```

Guidelines:

- **`Refresh`** issues read-only calls through the `chain.Reader`. Build a
  `chain.ViewCall{To, Method, Inputs, Args, Outputs}` per attribute and decode
  the returned values. Only read attributes that are actually managed.
- **`Plan`** compares the desired state held on the struct against `current`
  and returns one `Operation` per drifted attribute. **Return no operations
  when there is no drift** — this invariant is what makes an empty plan
  meaningful. Describe each operation by `Method` + `Inputs` + `Args`; the
  planner encodes the calldata for you via `chain.Pack`.

## 2. Provide a factory and register it

```go
func init() {
    resource.Register("accessControl", newAccessControl)
}

func newAccessControl(cfg config.ResourceConfig) (resource.Resource, error) {
    // validate cfg.Address, parse cfg.Spec into typed desired state
}
```

`Register` is called at package init, so the type becomes available as soon as
the package is imported. The reference resource is imported transitively
through the planner; ensure your package is in the same import graph (it is, if
it lives under `internal/resource`).

## 3. Add tests

Drive the resource through `plan.NewPlanner` with a `chain.MockReader` that
returns the actual state you want to test against. Assert the operations
produced (and that no operations are produced when state matches). See
[`internal/plan/planner_test.go`](../internal/plan/planner_test.go).

## Notes on ABI handling

You never write keccak or padding by hand. `chain.ViewCall` and
`Operation{Method, Inputs, Args}` are encoded/decoded centrally in
[`internal/chain/abi.go`](../internal/chain/abi.go) using go-ethereum's ABI
package. Argument Go types must match the ABI types:

| ABI type | Go type |
| --- | --- |
| `uint256` / `uint*` | `*big.Int` |
| `address` | `common.Address` |
| `bool` | `bool` |
| `bytes` / `bytesN` | `[]byte` / `[N]byte` |
| `string` | `string` |

When a resource grows beyond a couple of attributes, prefer driving it from a
parsed contract ABI rather than hand-written getters/setters — see the roadmap.
