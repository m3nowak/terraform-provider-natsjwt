# seed_public_key Function

Converts an NATS seed (`SO...`, `SA...`, `SU...`) into its matching public key.

## Example Usage

```terraform
output "operator_public_key" {
  value = provider::natsjwt::seed_public_key(var.operator_seed)
}
```

If `var.operator_seed` is sensitive, Terraform will typically mark the function result as sensitive too.  
When you intentionally want a non-sensitive value (for example for display), wrap the call with `nonsensitive(...)`.

```terraform
output "operator_public_key_plain" {
  value = nonsensitive(provider::natsjwt::seed_public_key(var.operator_seed))
}
```

## Signature

```text
seed_public_key(seed string) string
```
