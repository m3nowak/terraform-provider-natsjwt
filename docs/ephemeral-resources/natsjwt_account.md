# natsjwt_account Ephemeral Resource

Generates a signed account JWT without storing the configured seeds or generated values in Terraform plan or state. Terraform 1.10 or later is required.

```terraform
ephemeral "natsjwt_account" "app" {
  name          = "app"
  seed          = var.account_seed
  operator_seed = var.operator_seed
}
```

## Argument Reference

- `name` - (Required) Account name.
- `seed` - (Required, sensitive) Account seed (private key, starts with `SA`).
- `operator_seed` - (Required, sensitive) Operator seed used to sign the JWT (starts with `SO`).
- `signing_keys` - (Optional) List of additional signing key public keys.
- `issued_at` - (Optional) JWT issued-at Unix timestamp. Defaults to `0`.
- `expires` - (Optional) JWT expiration Unix timestamp. Defaults to no expiration.
- `not_before` - (Optional) JWT not-before Unix timestamp. Defaults to `issued_at`.
- `description` - (Optional) Account description.
- `info_url` - (Optional) Link to external information about the account.
- `tags` - (Optional) List of tags associated with the account.
- `nats_limits` - (Optional) NATS connection limits described below.
- `account_limits` - (Optional) Account-level limits described below.
- `jetstream_limits` - (Optional) List of global or tiered JetStream limits described below.
- `default_permissions` - (Optional) Default user permissions described below.
- `trace` - (Optional) Message trace configuration described below.

### NATS Limits

- `subs` - (Optional) Maximum subscriptions. `-1` means unlimited.
- `data` - (Optional) Maximum data in bytes. `-1` means unlimited.
- `payload` - (Optional) Maximum payload in bytes. `-1` means unlimited.

### Account Limits

- `imports` - (Optional) Maximum imports. `-1` means unlimited.
- `exports` - (Optional) Maximum exports. `-1` means unlimited.
- `wildcard_exports` - (Optional) Allow wildcard exports. Defaults to `true`.
- `disallow_bearer` - (Optional) Disallow bearer tokens. Defaults to `false`.
- `conn` - (Optional) Maximum connections. `-1` means unlimited.
- `leaf_node_conn` - (Optional) Maximum leaf-node connections. `-1` means unlimited.

### JetStream Limits

- `tier` - (Optional) Replication tier such as `R1` or `R3`. Omit for global limits.
- `mem_storage` - (Optional) Maximum memory storage in bytes. `0` disables memory storage.
- `disk_storage` - (Optional) Maximum disk storage in bytes. `0` disables disk storage.
- `streams` - (Optional) Maximum streams. `-1` means unlimited.
- `consumer` - (Optional) Maximum consumers. `-1` means unlimited.
- `max_ack_pending` - (Optional) Maximum pending acknowledgements. `-1` means unlimited.
- `mem_max_stream_bytes` - (Optional) Maximum bytes per memory stream. `0` means unlimited.
- `disk_max_stream_bytes` - (Optional) Maximum bytes per disk stream. `0` means unlimited.
- `max_bytes_required` - (Optional) Require streams to set `max_bytes`. Defaults to `false`.

### Default Permissions

- `pub_allow` - (Optional) Subjects allowed for publishing.
- `pub_deny` - (Optional) Subjects denied for publishing.
- `sub_allow` - (Optional) Subjects allowed for subscribing.
- `sub_deny` - (Optional) Subjects denied for subscribing.

### Trace

- `destination` - (Optional) Trace destination subject.
- `sampling` - (Optional) Sampling percentage from 0 to 100.

## Result Reference

- `public_key` - Account public key (starts with `A`).
- `jwt` - Signed account JWT.

Ephemeral results can only be consumed by other ephemeral contexts, provider configuration, provisioners, or write-only resource arguments.
