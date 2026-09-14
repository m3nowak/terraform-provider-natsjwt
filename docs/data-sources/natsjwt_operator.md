# natsjwt_operator Data Source

Generates a signed NATS operator JWT from the given seed and configuration. The operator JWT is the root credential that defines operator configuration and properties.

## Example Usage

```terraform
# Basic operator
data "natsjwt_operator" "main" {
  name = "my-operator"
  seed = natsjwt_nkey.operator.seed
}

# Operator with system account and service URLs
data "natsjwt_operator" "full" {
  name           = "my-operator"
  seed           = natsjwt_nkey.operator.seed
  system_account = data.natsjwt_system_account.sys.public_key

  account_server_url  = "https://accounts.example.com"
  operator_service_urls = [
    "nats://nats1.example.com:4222",
    "nats://nats2.example.com:4222"
  ]

  tags = ["prod", "us-west"]
}

# signing_keys are listed on the JWT only. This provider still signs
# account JWTs with seed (operator identity, SO).
data "natsjwt_operator" "signing_keys" {
  name = "my-operator"
  seed = natsjwt_nkey.operator.seed

  signing_keys = [
    "ADIQBFSAC24FGIHFI7TRILXU27HSLAG2PKMVPYVTTNCTOU3BDWK5GWK4",
    "ABADWACPHSQOSHCULO7NVPZGKRKXF3AQVU2KMTGDYAAJCAJ3AH2YOY24"
  ]
}
```

## Argument Reference

- `name` - (Required) Operator name.
- `seed` - (Required, sensitive) Operator seed (private key).
- `signing_keys` - (Optional) Public keys listed as extra signing keys on the operator JWT. This provider does not sign account JWTs with them.
- `account_server_url` - (Optional) Account server URL.
- `operator_service_urls` - (Optional) List of operator service URLs.
- `system_account` - (Optional) System account public key.
- `strict_signing_key_usage` - (Optional) When true, the operator JWT requires signing keys for account operations. Default is false. This provider still signs account JWTs with `seed`. Do not enable this unless something else will sign accounts.
- `issued_at` - (Optional) JWT issued-at Unix timestamp. Defaults to `0` (Unix epoch).
- `expires` - (Optional) JWT expiration Unix timestamp. Defaults to no expiration.
- `not_before` - (Optional) JWT not-before Unix timestamp. Defaults to `issued_at`.
- `tags` - (Optional) List of tags to associate with the operator.

## Attributes Reference

- `public_key` - The operator public key (starts with `O`).
- `jwt` - The signed operator JWT.

## Notes

- The JWT is deterministic and depends only on the seed and configuration parameters
- Changing any argument will result in a new JWT being generated
- Account JWTs from this provider are signed with the operator identity seed (`SO`), not with `signing_keys`
