# natsjwt_system_account Data Source

Generates a signed system account JWT with default `$SYS` exports. The system account is a special account used by NATS for internal monitoring and management.

## Example Usage

```terraform
data "natsjwt_system_account" "sys" {
  name          = "SYS"
  seed          = natsjwt_nkey.sys_account.seed
  operator_seed = natsjwt_nkey.operator.seed
}

# Reference in operator configuration
data "natsjwt_operator" "main" {
  name           = "my-operator"
  seed           = natsjwt_nkey.operator.seed
  system_account = data.natsjwt_system_account.sys.public_key
}
```

## Argument Reference

- `name` - (Required) Account name. Typically `SYS` for the system account.
- `seed` - (Required, sensitive) Account seed (private key).
- `operator_seed` - (Required, sensitive) Operator seed for signing.
- `signing_keys` - (Optional) List of signing key public keys.
- `issued_at` - (Optional) JWT issued-at Unix timestamp. Defaults to `0` (Unix epoch).
- `expires` - (Optional) JWT expiration Unix timestamp. Defaults to no expiration.
- `not_before` - (Optional) JWT not-before Unix timestamp. Defaults to `issued_at`.
- `nats_limits` - (Optional) Connection limits. See [NATS Limits](#nats-limits-1) below.
- `account_limits` - (Optional) Account limits. See [Account Limits](#account-limits-1) below.
- `jetstream_limits` - (Optional) JetStream limits. See [JetStream Limits](#jetstream-limits-1) below.
- `default_permissions` - (Optional) Default user permissions. See [Default Permissions](#default-permissions-1) below.
- `trace` - (Optional) Message trace configuration.

### NATS Limits

- `subs` - (Optional) Maximum number of subjects.
- `data` - (Optional) Maximum data in bytes.
- `payload` - (Optional) Maximum payload in bytes.

### Account Limits

- `imports` - (Optional) Maximum number of imports.
- `exports` - (Optional) Maximum number of exports.
- `wildcard_exports` - (Optional) Allow wildcard exports.
- `disallow_bearer` - (Optional) Disallow bearer tokens.
- `conn` - (Optional) Maximum connections.
- `leaf_node_conn` - (Optional) Maximum leaf node connections.

### JetStream Limits

- `tier` - (Optional) Tier name (for tiered configuration).
- `mem_storage` - (Optional) Maximum memory storage in bytes.
- `disk_storage` - (Optional) Maximum disk storage in bytes.
- `streams` - (Optional) Maximum number of streams.
- `consumer` - (Optional) Maximum number of consumers.
- `max_ack_pending` - (Optional) Maximum acknowledgments pending.
- `mem_max_stream_bytes` - (Optional) Maximum memory per stream in bytes.
- `disk_max_stream_bytes` - (Optional) Maximum disk per stream in bytes.
- `max_bytes_required` - (Optional) Require max bytes to be set.

### Default Permissions

- `pub_allow` - (Optional) Allowed publish subjects.
- `pub_deny` - (Optional) Denied publish subjects.
- `sub_allow` - (Optional) Allowed subscribe subjects.
- `sub_deny` - (Optional) Denied subscribe subjects.

## Attributes Reference

- `public_key` - The system account public key (starts with `A`).
- `jwt` - The signed system account JWT (sensitive).

## Differences from natsjwt_account

The `natsjwt_system_account` data source includes default `$SYS` exports, which allow NATS to publish system-level metrics and events. This is the recommended way to create a system account for NATS servers.

## Notes

- The system account is required in operator configuration for full NATS server functionality
- Changing any argument will result in a new JWT being generated
