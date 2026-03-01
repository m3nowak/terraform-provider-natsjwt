# natsjwt_nkey Resource

Generates an NKey pair (seed + public key) for NATS authentication. NKeys are the foundation of NATS security and can be used for operators, accounts, and users.

## Example Usage

```terraform
# Generate operator NKey
resource "natsjwt_nkey" "operator" {
  type = "operator"
}

# Generate account NKey
resource "natsjwt_nkey" "account" {
  type = "account"
}

# Generate user NKey
resource "natsjwt_nkey" "user" {
  type = "user"
}

# With keepers to trigger recreation
resource "natsjwt_nkey" "app_account" {
  type = "account"
  keepers = {
    environment = "production"
  }
}
```

## Argument Reference

- `type` - (Required) Type of NKey to generate. Must be one of `operator`, `account`, or `user`.
- `keepers` - (Optional) Arbitrary map of values that, when changed, will trigger recreation of the resource. Similar to the random provider's keepers.

## Attributes Reference

- `seed` - The generated NKey seed (private key). This is sensitive and should be protected. Starts with `SO` (operator), `SA` (account), or `SU` (user).
- `public_key` - The NKey public key. Starts with `O` (operator), `A` (account), or `U` (user).

## Import

Import is unnecessary for this resource. Data sources in this provider only require seeds as inputs, so existing externally managed seeds can be passed directly to data source arguments.

## Production Recommendation

For production environments, prefer generating NKeys outside Terraform and storing seeds in an external secrets manager (for example, Azure Key Vault or HashiCorp Vault).
