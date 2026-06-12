# Configuration reference

A ChainForm configuration is an HCL document — the same language Terraform uses
— describing the target chain and the resources to manage. See
[`examples/protocol.hcl`](../examples/protocol.hcl) for a complete, runnable
example.

```hcl
version = "1"

chain {
  name     = "ethereum"          # human-readable label
  chain_id = 1                   # EIP-155 chain id (required)
  rpc      = env("RPC_URL")  # JSON-RPC endpoint; env() reads the environment
}

resource "protocol" "main" {     # resource "TYPE" "NAME"
  address = "0x..."              # contract address (required)

  feeBps = 30                    # type-specific desired attributes
  paused = false
}
```

## Top level

| Construct         | Required | Notes                            |
| ----------------- | -------- | -------------------------------- |
| `version`         | no       | Schema version. Currently `"1"`. |
| `chain` block     | yes      | Target network. Exactly one.     |
| `resource` blocks | yes      | At least one.                    |

## `chain` block

| Attribute  | Type   | Required          | Notes                                                                                |
| ---------- | ------ | ----------------- | ------------------------------------------------------------------------------------ |
| `name`     | string | no                | Display label only.                                                                  |
| `chain_id` | number | yes               | Must be non-zero.                                                                    |
| `rpc`      | string | yes for live runs | JSON-RPC URL. Use `env("VAR")` to keep secrets out of git. Not needed with `--mock`. |

## `resource "TYPE" "NAME"` blocks

The two labels are the resource **type** and a unique local **name**:

```hcl
resource "protocol" "main" { ... }
#         ^type      ^name
```

| Attribute  | Required        | Notes                                          |
| ---------- | --------------- | ---------------------------------------------- |
| `address`  | yes             | 0x-prefixed contract address.                  |
| _(others)_ | depends on type | Desired attributes, validated by the provider. |

### `protocol` (reference resource)

| Attribute | Type         | Operation when drifted  |
| --------- | ------------ | ----------------------- |
| `feeBps`  | number (≥ 0) | `setFeeBps(uint256)`    |
| `paused`  | bool         | `pause()` / `unpause()` |

Only declared attributes are managed; omit one to leave it untouched.

## Functions

| Function      | Returns | Notes                                                                       |
| ------------- | ------- | --------------------------------------------------------------------------- |
| `env("NAME")` | string  | The environment variable `NAME`, or empty if unset. Evaluated at load time. |

Keep RPC URLs and API keys in the environment or a local `.env` (gitignored),
not in the committed configuration.

## Validation

`chainform validate -f <file>` runs HCL parsing, schema-level checks (required
fields, non-zero chain id, unique resource names), and provider-level checks
(valid address, known attributes) without contacting the chain.
