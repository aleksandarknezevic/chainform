# Contributing

## Prerequisites

- Go 1.23+ (developed against 1.26)
- `make`

## Common tasks

```bash
make build      # build ./bin/chainform
make test       # run all tests
make vet        # go vet
make fmt        # gofmt -w .
make run-plan   # plan against examples/protocol.hcl with the offline demo reader
```

## Try it offline

No RPC endpoint is needed to exercise the full pipeline — the demo reader
supplies intentionally-drifted state:

```bash
./bin/chainform plan   -f examples/protocol.hcl --mock
./bin/chainform export -f examples/protocol.hcl --mock -o batch.json
```

Drop `--mock` and set `ETH_RPC_URL` to run against a live network.

## Layout

See [docs/architecture.md](docs/architecture.md) for the package map and the
reconciliation flow. The most common contribution — a new resource type — is
documented in [docs/adding-a-resource.md](docs/adding-a-resource.md).

## Conventions

- Keep dependency direction one-way (see architecture doc). In particular,
  resources depend on the `chain.Reader` interface, not on a concrete client.
- Planning is read-only: never send a transaction from the plan path.
- A resource with no drift must produce no operations.
- New behaviour ships with tests. Reconciliation logic is testable offline via
  `chain.MockReader`.
- Run `make fmt vet test` before opening a PR.

## Commit / PR

Small, focused commits with imperative subjects. PRs should describe the
behaviour change and include test output where relevant.
