# natsjwt_account Data Source

Generates a signed account JWT. Accounts are the second tier of NATS security hierarchy and contain users. Accounts can have JetStream enabled with configurable limits.

## Example Usage

```terraform
# Simple account
data "natsjwt_account" "basic" {
  name          = "app"
  seed          = natsjwt_nkey.app_account.seed
  operator_seed = natsjwt_nkey.operator.seed
}

# Account with JetStream limits
data "natsjwt_account" "with_jetstream" {
  name          = "app"
  seed          = natsjwt_nkey.app_account.seed
  operator_seed = natsjwt_nkey.operator.seed

  jetstream_limits = [{
    mem_storage  = 1073741824  # 1GB
    disk_storage = 10737418240 # 10GB
    streams      = 10
    consumer     = 100
  }]
}

# Account with connection limits
data "natsjwt_account" "with_limits" {
  name          = "app"
  seed          = natsjwt_nkey.app_account.seed
  operator_seed = natsjwt_nkey.operator.seed

  account_limits = [{
    imports         = 100
    exports         = 100
    wildcard_exports = true
    conn            = 1000
    leaf_node_conn  = 50
  }]

  nats_limits = [{
    subs    = 10000
    data    = 10737418240  # 10GB
    payload = 1048576      # 1MB
  }]
}

# Account with default permissions
data "natsjwt_account" "with_permissions" {
  name          = "app"
  seed          = natsjwt_nkey.app_account.seed
  operator_seed = natsjwt_nkey.operator.seed

  default_permissions = [{
    pub_allow  = ["app.>"]
    pub_deny   = ["admin.>"]
    sub_allow  = ["app.>", "_INBOX.>"]
    sub_deny   = ["private.>"]
  }]
}
```

## Argument Reference

- `name` - (Required) Account name.
- `seed` - (Required, sensitive) Account seed (private key).
- `operator_seed` - (Required, sensitive) Operator seed for signing.
- `signing_keys` - (Optional) List of signing key public keys.
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

- `public_key` - The account public key (starts with `A`).
- `jwt` - The signed account JWT (sensitive).

## Notes

- Accounts must be signed with the operator's seed
- JetStream limits are optional; if not specified, JetStream is disabled
- Default permissions are inherited by users in the account
