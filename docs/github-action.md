# GitHub Action - gate pull requests on drift

ChainForm ships a reusable action that runs `chainform plan` and fails the job
when live contract state no longer matches the committed configuration. It is a
thin, reviewable wrapper around the CLI: it installs a release binary, runs
`plan`, and turns the exit code (1 on drift) into a job result, a job summary,
and step outputs. Nothing is signed or sent - `plan` only issues read-only
calls.

Copy-paste template: [`examples/workflows/drift-check.yml`](../examples/workflows/drift-check.yml).

## Minimal usage

```yaml
- uses: actions/checkout@v4

- uses: aleksandarknezevic/chainform@v0.0.2
  with:
      file: chainform.hcl
      rpc-url: ${{ secrets.RPC_URL }}
```

The job fails if any managed attribute drifted or any `expect` assertion is
violated. Store the endpoint as a repository secret; the action exports it as
`RPC_URL`, which is what `rpc = env("RPC_URL")` in the config reads.

Offline (no endpoint, demo values - useful for trying the action out):

```yaml
- uses: aleksandarknezevic/chainform@v0.0.2
  with:
      file: examples/vault.hcl
      mock: "true"
```

## Inputs

| Input               | Default          | Notes                                                                                                                |
| ------------------- | ---------------- | -------------------------------------------------------------------------------------------------------------------- |
| `file`              | `chainform.hcl`  | Configuration file (HCL or JSON).                                                                                    |
| `version`           | `latest`         | Release tag (e.g. `v0.0.2`), `latest`, or `source` to build the checked-out repo (needs Go on the runner).            |
| `rpc-url`           | _empty_          | JSON-RPC endpoint, exported as `RPC_URL`. Pass a secret. Not needed with `mock: true`.                               |
| `mock`              | `false`          | Use the offline demo reader instead of an endpoint.                                                                   |
| `json`              | `false`          | Also write the machine-readable plan. Runs `plan` a second time, so the chain is read twice.                          |
| `fail-on-drift`     | `true`           | Set `false` to report drift through outputs without failing the job.                                                  |
| `summary`           | `true`           | Append the plan to the job summary.                                                                                  |
| `working-directory` | `.`              | Directory to run in; ABI paths in the config resolve relative to it.                                                  |

Pin a release tag in production. `latest` resolves the newest release on every
run, so a new version changes your gate without a commit.

## Outputs

| Output           | Notes                                                              |
| ---------------- | ------------------------------------------------------------------ |
| `drift`          | `"true"` when drift was detected, `"false"` otherwise.             |
| `exit-code`      | `chainform plan` exit code: `0` no drift, `1` drift.               |
| `plan-file`      | Path to the human-readable plan.                                   |
| `plan-json-file` | Path to the JSON plan; empty unless `json: true`.                  |
| `version`        | The ChainForm version that ran.                                    |

## Report without failing

To surface drift without blocking a merge - for example while adopting
ChainForm - turn the gate off and act on the output:

```yaml
- uses: aleksandarknezevic/chainform@v0.0.2
  id: plan
  with:
      file: chainform.hcl
      rpc-url: ${{ secrets.RPC_URL }}
      fail-on-drift: "false"
      json: "true"

- name: Count proposed operations
  run: jq '.summary.operationCount' "${{ steps.plan.outputs.plan-json-file }}"
```

The [plan JSON format](plan-json.md) documents every field, including
`summary.failedAssertionCount` for read-only `expect` violations.

## Scheduled monitoring

Drift usually appears without a commit: someone changes a parameter on chain.
Adding a `schedule:` trigger turns the same gate into periodic monitoring - the
supported approach until built-in scheduled reconciliation lands (see the
[roadmap](roadmap.md)).

```yaml
on:
    schedule:
        - cron: "0 * * * *"
```

## Multiple configs

Run the action once per file, or use a matrix:

```yaml
strategy:
    fail-fast: false
    matrix:
        config: [mainnet.hcl, arbitrum.hcl]
steps:
    - uses: actions/checkout@v4
    - uses: aleksandarknezevic/chainform@v0.0.2
      with:
          file: ${{ matrix.config }}
          rpc-url: ${{ secrets[format('RPC_URL_{0}', matrix.config)] }}
```

One config targets one chain today; a matrix is how you cover several.

## Notes and limits

- Tested on `ubuntu-latest` and `macos-latest` runners. Downloaded release
  binaries are checksum-verified against the release's `SHA256SUMS`.
- The action needs no write permissions: `permissions: contents: read` is
  enough.
- It does not comment on pull requests yet; the plan goes to the job summary and
  the step outputs. PR comments are on the [roadmap](roadmap.md).
- Without a checkout step the config and ABI files are not on disk, so start the
  job with `actions/checkout`.

Alternatively, skip the action and call the CLI (or
[Docker image](../README.md#docker)) directly - the exit code is the whole
contract:

```yaml
- run: |
      go install github.com/aleksandarknezevic/chainform/cmd/chainform@latest
      chainform plan -f chainform.hcl
  env:
      RPC_URL: ${{ secrets.RPC_URL }}
```
