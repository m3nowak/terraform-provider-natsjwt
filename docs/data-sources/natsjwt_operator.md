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

  strict_signing_key_usage = true
  tags                     = ["prod", "us-west"]
}

# With additional signing keys
data "natsjwt_operator" "signing_keys" {
  name = "my-operator"
  seed = natsjwt_nkey.operator.seed

  signing_keys = [
    "OAJHB43CKFBNXQGVX2XYXQGZVDVFPVMXZEYQOZWKSLVN7CBJJ5HU2TCM",
    "OAKVLYKJX2SJ4C3XSVXO42W6T5LJVB45USHHDDBKZTL2M5VNH6ZQRQV4"
  ]
}
```

## Argument Reference

- `name` - (Required) Operator name.
- `seed` - (Required, sensitive) Operator seed (private key).
- `signing_keys` - (Optional) List of additional signing key public keys.
- `account_server_url` - (Optional) Account server URL.
- `operator_service_urls` - (Optional) List of operator service URLs.
- `system_account` - (Optional) System account public key.
- `strict_signing_key_usage` - (Optional) If true, require signing keys to be used. Default is false.
- `issued_at` - (Optional) JWT issued-at Unix timestamp. Defaults to `0` (Unix epoch).
- `expires` - (Optional) JWT expiration Unix timestamp. Defaults to no expiration.
- `not_before` - (Optional) JWT not-before Unix timestamp. Defaults to `issued_at`.
- `tags` - (Optional) List of tags to associate with the operator.

## Attributes Reference

- `public_key` - The operator public key (starts with `O`).
- `jwt` - The signed operator JWT (sensitive).

## Notes

- The JWT is deterministic and depends only on the seed and configuration parameters
- Changing any argument will result in a new JWT being generated
- The operator JWT is required for account and user JWT generation (as operator_seed)
