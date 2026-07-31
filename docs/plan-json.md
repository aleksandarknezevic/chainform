# Plan JSON Format (`chainform plan --json`)

`chainform plan --json` emits a machine-readable document suitable for CI
gating and GitOps pipelines.

## Top-level structure

```json
{
  "chain": {
    "name": "ethereum",
    "chainId": 1,
    "rpc": ""
  },
  "operations": [],
  "assertions": [],
  "summary": {
    "operationCount": 0,
    "assertionCount": 0,
    "failedAssertionCount": 0,
    "empty": true
  }
}
```

## Field reference

### `chain`

- `name` (`string`): configured chain label.
- `chainId` (`number`): EIP-155 chain id.
- `rpc` (`string`): configured RPC URL string (may be empty).

### `operations[]`

- `resource` (`string`): local resource name.
- `to` (`string`): target contract address (`0x...`).
- `method` (`string`): function name to call.
- `inputs` (`string[]`): ABI input types in order.
- `args` (`any[]`): argument values in the same order as `inputs`. Numbers are
  JSON numbers, addresses hex strings, `bytes`/`bytesN` `0x...` hex strings, and
  array arguments JSON arrays of the same. `null` for a zero-argument call such
  as `unpause()`.
- `valueWei` (`string`): wei amount as a base-10 string.
- `reason` (`string`, optional): human drift explanation.
- `calldata` (`string`): ABI-encoded calldata as `0x...` hex.

### `assertions[]`

- `resource` (`string`): local resource name.
- `attr` (`string`): asserted attribute name.
- `type` (`string`): ABI type of the asserted value.
- `expected` (`any`): expected canonical value.
- `actual` (`any`): actual canonical value.
- `satisfied` (`boolean`): `true` when `actual == expected`.

Notes on canonical encoding:

- Integer-like values are strings in assertions (`"30"`, `"50"`).
- Address values are hex strings (`0x...`).
- `bytes` and `bytesN` values are `0x...` hex strings.
- Array values are JSON arrays whose elements follow the same rules.

### `summary`

- `operationCount` (`number`): length of `operations`.
- `assertionCount` (`number`): total read-only assertions evaluated.
- `failedAssertionCount` (`number`): number of unsatisfied assertions.
- `empty` (`boolean`): `true` only when `operationCount == 0`. Failed
  `expect` assertions do not affect this field; use `failedAssertionCount` or
  the `plan` process exit code (1 when any drift is present).

## Exit codes

| Code | Meaning                                                                     |
| ---- | --------------------------------------------------------------------------- |
| `0`  | No drift.                                                                    |
| `1`  | Drift: managed operations and/or failed `expect` assertions. Plan is printed. |
| `2`  | The command failed (bad config, RPC error). No plan is produced.              |

Distinguishing `1` from `2` is what lets a CI job fail loudly on a broken
endpoint instead of reporting it as drift.

## CI examples

Fail when any drift is detected (`plan` exits 1):

```bash
chainform plan -f chainform.hcl
```

Or use the [reusable GitHub Action](github-action.md), which wraps the exit code
and `--json` output.

Fail when any operation is proposed (JSON inspection):

```bash
chainform plan -f chainform.hcl --json | jq -e '.summary.operationCount == 0'
```

Fail when any read-only expectation is violated:

```bash
chainform plan -f chainform.hcl --json | jq -e '.summary.failedAssertionCount == 0'
```

Or rely on the process exit code (covers both operations and failed assertions):

```bash
chainform plan -f chainform.hcl
```
