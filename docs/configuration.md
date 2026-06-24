# Configuration reference

A ChainForm configuration can be written as either HCL (the same language
Terraform uses) or JSON. Both formats map to the same schema describing the
target chain and the resources to manage. See
[`examples/protocol.hcl`](../examples/protocol.hcl) and
[`examples/protocol.json`](../examples/protocol.json) for complete, runnable
example. You can also generate one from a deployed contract with
`chainform import` (see [Commands](../README.md#commands)) instead of writing it
by hand.

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

Equivalent JSON form:

```json
{
  "version": "1",
  "chain": {
    "name": "ethereum",
    "chain_id": 1,
    "rpc": "https://rpc.example"
  },
  "resources": [
    {
      "type": "protocol",
      "name": "main",
      "address": "0x...",
      "feeBps": 30,
      "paused": false
    }
  ]
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

### `contract` (ABI-driven resource)

Manages any contract without hand-written Go. Point it at an ABI file and
declare the attributes you care about; each attribute `X` is read via the
getter `X()` and reconciled via the setter `setX(...)`, both derived from the
ABI. For bool `paused`, `pause()`/`unpause()` are preferred when present
(OpenZeppelin Pausable); otherwise `setPaused(bool)` is used. See
[`examples/contract.hcl`](../examples/contract.hcl).

```hcl
resource "contract" "protocol" {
  address = "0x..."
  abi     = "testdata/protocol.abi.json"  # path relative to the working dir

  feeBps = 30                              # read feeBps(), set via setFeeBps()
  paused = false                           # read paused(); pause()/unpause() or setPaused()
}
```

| Attribute  | Required | Notes                                                                                    |
| ---------- | -------- | ---------------------------------------------------------------------------------------- |
| `abi`      | yes      | Path to the contract ABI JSON, resolved relative to the working directory.               |
| _(others)_ | no       | Each must have a matching getter `X()` and setter `setX(T)` of the same type in the ABI. |

Supported attribute types: `bool`, `string`, `address`, and the integer types
`uintN` / `intN`. An attribute with a getter but no `setX` setter (a read-only
value) cannot be managed, and declaring it as a top-level attribute is an error
— use an [`expect` block](#expect-block--read-only-assertions) to assert it
instead. Only declared attributes are read and managed; omit one to leave it
untouched.

A `contract` with no managed attributes is valid — it is read-only and
produces no operations, but `chainform show` still prints every getter derived
from its ABI. [`examples/feed.hcl`](../examples/feed.hcl) does exactly this for
the live Chainlink ETH/USD price feed on Sepolia.

#### `expect` block — read-only assertions

Some values have a getter but no setter, so they cannot be managed — but you
may still want to assert what they should be and be warned if they drift. An
`expect` block declares those read-only invariants:

```hcl
resource "contract" "ethUsdFeed" {
  address = "0x694AA1769357215DE4FAC081bf1f309aDC325306"
  abi     = "testdata/aggregator.abi.json"

  expect {
    decimals    = 8
    description = "ETH / USD"
  }
}
```

Each `expect` attribute needs only a getter `X()` in the ABI (no setter). On
`plan`, ChainForm reads the getter and, if the value differs from the
expectation, reports it as **read-only drift** — a warning that is never turned
into a transaction (there is no setter to call), so `export` is unaffected. An
attribute cannot be both managed (top-level) and expected. Only the ABI-driven
`contract` resource supports `expect`.

## Functions

| Function      | Returns | Notes                                                                       |
| ------------- | ------- | --------------------------------------------------------------------------- |
| `env("NAME")` | string  | The environment variable `NAME`, or empty if unset. Evaluated at load time. |

Keep RPC URLs and API keys in the environment or a local `.env` (gitignored),
not in the committed configuration.

Note: `env("NAME")` is an HCL function. In JSON, use a plain string value
for `chain.rpc` (for example, inject it before running or keep JSON files
local/non-committed when they include secrets).

## Validation

`chainform validate -f <file>` runs parsing (HCL or JSON), schema-level checks
(required fields, non-zero chain id, unique resource names), and provider-level
checks without contacting the chain. Each resource is built (`resource.Build`):
unknown types, invalid addresses, missing ABI files, and attributes without a
matching getter/setter pair are rejected.

`chainform plan` exits with code **0** when there is no drift and **1** when
managed attributes differ or an `expect` assertion fails (the plan is still
printed to stdout). Use this for simple CI gates without parsing JSON.
