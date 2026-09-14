# natsjwt_operator Ephemeral Resource

Generates a signed operator JWT without storing the configured seed or generated values in Terraform plan or state. Terraform 1.10 or later is required.

```terraform
ephemeral "natsjwt_operator" "main" {
  name = "main"
  seed = var.operator_seed
}
```

## Argument Reference

- `name` - (Required) Operator name.
- `seed` - (Required, sensitive) Operator seed (private key, starts with `SO`).
- `signing_keys` - (Optional) Public keys listed as extra signing keys on the operator JWT. This provider does not sign account JWTs with them.
- `account_server_url` - (Optional) Account server URL.
- `operator_service_urls` - (Optional) List of operator service URLs.
- `system_account` - (Optional) System account public key.
- `strict_signing_key_usage` - (Optional) When true, the operator JWT requires signing keys for account operations. Defaults to `false`. This provider still signs account JWTs with `seed`. Do not enable this unless something else will sign accounts.
- `issued_at` - (Optional) JWT issued-at Unix timestamp. Defaults to `0`.
- `expires` - (Optional) JWT expiration Unix timestamp. Defaults to no expiration.
- `not_before` - (Optional) JWT not-before Unix timestamp. Defaults to `issued_at`.
- `tags` - (Optional) List of tags associated with the operator.

## Result Reference

- `public_key` - Operator public key (starts with `O`).
- `jwt` - Signed operator JWT.

Ephemeral results can only be consumed by other ephemeral contexts, provider configuration, provisioners, or write-only resource arguments.

Account JWTs from this provider are signed with the operator identity seed (`SO`), not with `signing_keys`.
