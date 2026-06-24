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
- `args` (`any[]`): argument values in the same order as `inputs`.
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

### `summary`

- `operationCount` (`number`): length of `operations`.
- `assertionCount` (`number`): total read-only assertions evaluated.
- `failedAssertionCount` (`number`): number of unsatisfied assertions.
- `empty` (`boolean`): `true` only when `operationCount == 0`. Failed
  `expect` assertions do not affect this field; use `failedAssertionCount` or
  the `plan` process exit code (1 when any drift is present).

## CI examples

Fail when any drift is detected (`plan` exits 1):

```bash
chainform plan -f chainform.hcl
```

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
