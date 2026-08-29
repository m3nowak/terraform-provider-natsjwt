# natsjwt_user Ephemeral Resource

Generates a signed user JWT and `.creds` content without storing the configured seeds or generated values in Terraform plan or state. Terraform 1.10 or later is required.

```terraform
ephemeral "natsjwt_user" "app" {
  name         = "app-user"
  seed         = var.user_seed
  account_seed = var.account_seed

  permissions = {
    pub_allow = ["app.>"]
    sub_allow = ["app.>", "_INBOX.>"]
  }
}

# Pass ephemeral.natsjwt_user.app.creds only to an ephemeral context,
# provider configuration, provisioner, or write-only resource argument.
```

## Argument Reference

- `name` - (Required) User name.
- `seed` - (Required, sensitive) User seed (private key, starts with `SU`).
- `account_seed` - (Required, sensitive) Account seed used to sign the JWT (starts with `SA`).
- `issuer_account` - (Optional) Account public key. Set when `account_seed` belongs to a signing key rather than the account key.
- `issued_at` - (Optional) JWT issued-at Unix timestamp. Defaults to `0`.
- `expires` - (Optional) JWT expiration Unix timestamp. Defaults to no expiration.
- `not_before` - (Optional) JWT not-before Unix timestamp. Defaults to `issued_at`.
- `permissions` - (Optional) Publish, subscribe, and response permissions described below.
- `limits` - (Optional) User connection limits described below.
- `bearer_token` - (Optional) Allow bearer-token authentication. Defaults to `false`.
- `allowed_connection_types` - (Optional) Allowed connection types, such as `STANDARD`, `WEBSOCKET`, `LEAFNODE`, or `MQTT`.
- `source_networks` - (Optional) Allowed source networks in CIDR notation.
- `time_restrictions` - (Optional) List of allowed time ranges described below.
- `locale` - (Optional) Timezone used by time restrictions, such as `America/New_York`.
- `tags` - (Optional) List of tags associated with the user.

### Permissions

- `pub_allow` - (Optional) Subjects allowed for publishing.
- `pub_deny` - (Optional) Subjects denied for publishing.
- `sub_allow` - (Optional) Subjects allowed for subscribing.
- `sub_deny` - (Optional) Subjects denied for subscribing.
- `resp_max_msgs` - (Optional) Maximum response messages.
- `resp_ttl` - (Optional) Response permission TTL as a Go duration, such as `1m` or `5s`.

### Limits

- `subs` - (Optional) Maximum subscriptions. `-1` means unlimited.
- `data` - (Optional) Maximum data in bytes. `-1` means unlimited.
- `payload` - (Optional) Maximum payload in bytes. `-1` means unlimited.

### Time Restrictions

- `start` - (Required) Start time in `HH:MM:SS` format.
- `end` - (Required) End time in `HH:MM:SS` format.

## Result Reference

- `public_key` - User public key (starts with `U`).
- `jwt` - Signed user JWT.
- `creds` - Sensitive NATS `.creds` content containing the JWT and private user seed.

Marking a value sensitive only redacts its display. Using this ephemeral resource is what prevents its configuration and results from being persisted by Terraform. Existing data sources remain appropriate when downstream configuration requires values stored in state.
