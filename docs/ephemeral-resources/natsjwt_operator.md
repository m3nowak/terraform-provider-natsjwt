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
- `signing_keys` - (Optional) List of additional signing key public keys.
- `account_server_url` - (Optional) Account server URL.
- `operator_service_urls` - (Optional) List of operator service URLs.
- `system_account` - (Optional) System account public key.
- `strict_signing_key_usage` - (Optional) Require signing keys for all operations. Defaults to `false`.
- `issued_at` - (Optional) JWT issued-at Unix timestamp. Defaults to `0`.
- `expires` - (Optional) JWT expiration Unix timestamp. Defaults to no expiration.
- `not_before` - (Optional) JWT not-before Unix timestamp. Defaults to `issued_at`.
- `tags` - (Optional) List of tags associated with the operator.

## Result Reference

- `public_key` - Operator public key (starts with `O`).
- `jwt` - Signed operator JWT.

Ephemeral results can only be consumed by other ephemeral contexts, provider configuration, provisioners, or write-only resource arguments.
