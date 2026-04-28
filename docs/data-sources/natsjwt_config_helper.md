# natsjwt_config_helper Data Source

Generates a NATS server configuration snippet from operator and account JWTs.

## Example Usage

```terraform
# Basic configuration
data "natsjwt_config_helper" "server" {
  operator_jwt       = data.natsjwt_operator.main.jwt
  system_account_jwt = data.natsjwt_system_account.sys.jwt
  account_jwts       = [data.natsjwt_account.app.jwt]
}

output "server_config" {
  value = data.natsjwt_config_helper.server.server_config
}

# Use in a complete NATS server configuration
resource "local_file" "nats_config" {
  filename = "${path.module}/nats-server.conf"
  content = <<-EOT
# NATS Server Configuration

server_name: "my-nats-server"
port: 4222
max_payload: 1MB

${data.natsjwt_config_helper.server.server_config}

# Choose your resolver type
resolver: MEMORY

EOT
}
```

## Argument Reference

- `operator_jwt` - (Required) Operator JWT.
- `account_jwts` - (Optional) List of account JWTs to include in the resolver preload.
- `system_account_jwt` - (Optional) System account JWT.

## Attributes Reference

- `server_config` - The NATS server configuration snippet containing `operator`, `system_account`, and `resolver_preload`.
- `operator` - The operator JWT value.
- `system_account` - The system account public key.
- `resolver_preload` - A map of account public keys to their JWTs for preloading in the resolver.

## Notes

- The `server_config` output includes `resolver_preload` but does **not** include a `resolver` line. You must add the resolver type yourself (e.g., `resolver: MEMORY` or configure a directory/NATS-based resolver).
- The `resolver_preload` map contains all accounts passed via `account_jwts` plus the system account.

## Configuration Output Format

The generated configuration follows this format:

```
operator: "<operator-jwt>"
system_account: "<system-account-public-key>"
resolver_preload: {
  <account-public-key-1>: <account-jwt-1>
  <account-public-key-2>: <account-jwt-2>
}
```

Note lack of `resolver` section. You'll need to add resolver configuration by yourself. [See here for NATS docs.](https://docs.nats.io/running-a-nats-service/configuration/securing_nats/auth_intro/jwt/resolver)