# natsjwt_config_helper Data Source

Generates NATS server configuration for the memory resolver. This data source combines operator, system account, and regular account JWTs into a complete server configuration snippet.

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

# Additional server configuration...
websocket {
  port: 8080
}

mqtt {
  port: 1883
}
EOT
}
```

## Argument Reference

- `operator_jwt` - (Required) Operator JWT.
- `account_jwts` - (Optional) List of account JWTs.
- `system_account_jwt` - (Optional) System account JWT.
- `resolver_type` - (Optional) Resolver type. Currently only `MEMORY` is supported. Defaults to `MEMORY`.

## Attributes Reference

- `server_config` - The complete NATS server configuration snippet.
- `operator` - The operator JWT value.
- `system_account` - The system account public key.
- `resolver` - The resolver type (currently `MEMORY`).
- `resolver_preload` - A map of account public keys to their JWTs for preloading in the resolver.

## Notes

- The `server_config` output can be directly embedded in your `nats-server.conf` file
- All JWTs should be signed by the operator specified in `operator_jwt`
- The system account JWT is required for full NATS server functionality
- Multiple accounts can be specified in `account_jwts`
- The memory resolver stores all account JWTs and can verify user JWTs on-the-fly

## Configuration Output Format

The generated configuration follows this format:

```
operator: "<operator-jwt>"
system_account: "<system-account-public-key>"
resolver: MEMORY
resolver_preload: {
  <account-public-key-1>: <account-jwt-1>
  <account-public-key-2>: <account-jwt-2>
}
```
