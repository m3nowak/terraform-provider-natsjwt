# seed_public_key Function

Converts an NATS seed (`SO...`, `SA...`, `SU...`) into its matching public key.

## Example Usage

```terraform
output "operator_public_key" {
  value = provider::natsjwt::seed_public_key(var.operator_seed)
}
```

## Signature

```text
seed_public_key(seed string) string
```
