# terraform-provider-natsjwt

A Terraform provider for managing [NATS](https://nats.io/) JWT credentials offline — no running NATS server required.

This provider is a Terraform-native replacement for the [`nsc`](https://github.com/nats-io/nsc) command-line tool, enabling you to manage operators, accounts, users, and server configuration as code.

## ⚠️ Warning ⚠️

Entire thing was vibe-coded. Use at your own risk.

## Features

- **Offline operation** — generates NKeys and signed JWTs without connecting to a NATS server
- **Deterministic JWTs** — same inputs always produce the same JWT output (stable `terraform plan`)
- **Full JWT support** — operators, accounts (with JetStream limits), system accounts, and users
- **Server config generation** — produces NATS server configuration with memory resolver
- **Seed validation** — validates that the correct key type is used for each operation
- **External seed support** — use NKeys from external sources (e.g., HashiCorp Vault) or generate them with the provider

## Installation

```hcl
terraform {
  required_providers {
    natsjwt = {
      source  = "m3nowak/natsjwt"
      version = "~> 0.1"
    }
  }
}
```

## Quick Start

```hcl
# Generate NKeys
resource "natsjwt_nkey" "operator" {
  type = "operator"
}

resource "natsjwt_nkey" "sys_account" {
  type = "account"
}

resource "natsjwt_nkey" "app_account" {
  type = "account"
}

resource "natsjwt_nkey" "app_user" {
  type = "user"
}

# System account with default $SYS exports
data "natsjwt_system_account" "sys" {
  name          = "SYS"
  seed          = natsjwt_nkey.sys_account.seed
  operator_seed = natsjwt_nkey.operator.seed
}

# Operator referencing system account
data "natsjwt_operator" "main" {
  name           = "my-operator"
  seed           = natsjwt_nkey.operator.seed
  system_account = data.natsjwt_system_account.sys.public_key
}

# Application account with JetStream
data "natsjwt_account" "app" {
  name          = "app"
  seed          = natsjwt_nkey.app_account.seed
  operator_seed = natsjwt_nkey.operator.seed

  jetstream_limits = [{
    mem_storage  = 1073741824
    disk_storage = 10737418240
    streams      = 10
    consumer     = 100
  }]
}

# User with permissions
data "natsjwt_user" "app_user" {
  name         = "app-user"
  seed         = natsjwt_nkey.app_user.seed
  account_seed = natsjwt_nkey.app_account.seed

  permissions = {
    pub_allow = ["app.>"]
    sub_allow = ["app.>", "_INBOX.>"]
  }
}

# Generate NATS server config
data "natsjwt_config_helper" "server" {
  operator_jwt       = data.natsjwt_operator.main.jwt
  system_account_jwt = data.natsjwt_system_account.sys.jwt
  account_jwts       = [data.natsjwt_account.app.jwt]
}

output "server_config" {
  value = data.natsjwt_config_helper.server.server_config
}
```

## Resources

### `natsjwt_nkey`

Generates an NKey pair (seed + public key).

| Attribute    | Type     | Description                                        |
| ------------ | -------- | -------------------------------------------------- |
| `type`       | Required | Key type: `operator`, `account`, or `user`         |
| `keepers`    | Optional | Map of values that trigger recreation when changed |
| `seed`       | Computed | Generated seed (sensitive)                         |
| `public_key` | Computed | Generated public key                               |

## Data Sources

### `natsjwt_operator`

Generates a signed operator JWT.

| Attribute                  | Type     | Description                        |
| -------------------------- | -------- | ---------------------------------- |
| `name`                     | Required | Operator name                      |
| `seed`                     | Required | Operator seed (sensitive)          |
| `signing_keys`             | Optional | Additional signing key public keys |
| `account_server_url`       | Optional | Account server URL                 |
| `operator_service_urls`    | Optional | Operator service URLs              |
| `system_account`           | Optional | System account public key          |
| `strict_signing_key_usage` | Optional | Require signing keys               |
| `tags`                     | Optional | Tags                               |
| `public_key`               | Computed | Operator public key                |
| `jwt`                      | Computed | Signed operator JWT                |

### `natsjwt_account` / `natsjwt_system_account`

Generates a signed account JWT. The `system_account` variant includes default `$SYS` exports.

| Attribute             | Type     | Description                                    |
| --------------------- | -------- | ---------------------------------------------- |
| `name`                | Required | Account name                                   |
| `seed`                | Required | Account seed (sensitive)                       |
| `operator_seed`       | Required | Operator seed for signing (sensitive)          |
| `signing_keys`        | Optional | Signing key public keys                        |
| `nats_limits`         | Optional | Connection limits (subs, data, payload)        |
| `account_limits`      | Optional | Account limits (imports, exports, connections) |
| `jetstream_limits`    | Optional | JetStream limits (global or tiered)            |
| `default_permissions` | Optional | Default user permissions                       |
| `trace`               | Optional | Message trace config                           |
| `public_key`          | Computed | Account public key                             |
| `jwt`                 | Computed | Signed account JWT                             |

### `natsjwt_user`

Generates a signed user JWT.

| Attribute                  | Type     | Description                                 |
| -------------------------- | -------- | ------------------------------------------- |
| `name`                     | Required | User name                                   |
| `seed`                     | Required | User seed (sensitive)                       |
| `account_seed`             | Required | Account seed for signing (sensitive)        |
| `issuer_account`           | Optional | Account public key (when using signing key) |
| `permissions`              | Optional | Pub/sub permissions                         |
| `limits`                   | Optional | Connection limits                           |
| `bearer_token`             | Optional | Allow bearer tokens                         |
| `allowed_connection_types` | Optional | STANDARD, WEBSOCKET, LEAFNODE, MQTT         |
| `source_networks`          | Optional | Allowed CIDRs                               |
| `time_restrictions`        | Optional | Time-based access                           |
| `locale`                   | Optional | Timezone for time restrictions              |
| `public_key`               | Computed | User public key                             |
| `jwt`                      | Computed | Signed user JWT                             |

### `natsjwt_config_helper`

Generates NATS server configuration for memory resolver.

| Attribute            | Type     | Description               |
| -------------------- | -------- | ------------------------- |
| `operator_jwt`       | Required | Operator JWT              |
| `account_jwts`       | Optional | Account JWTs              |
| `system_account_jwt` | Optional | System account JWT        |
| `resolver_type`      | Optional | Only `MEMORY` supported   |
| `server_config`      | Computed | Complete config snippet   |
| `operator`           | Computed | Operator JWT value        |
| `system_account`     | Computed | System account public key |
| `resolver`           | Computed | Resolver type             |
| `resolver_preload`   | Computed | Account pubkey → JWT map  |

## Security Notes

- **Seeds are sensitive** — they are stored in Terraform state and marked as sensitive
- **State should be encrypted** — use remote state backends with encryption
- Consider using external seed management (e.g., Vault) for production setups

## Compatibility

- NATS 2.11 and 2.12
- Terraform >= 1.0
- Uses `github.com/nats-io/jwt/v2` and `github.com/nats-io/nkeys`

## Development

```bash
# Build
mise run build

# Run tests
mise run testacc

# Lint
mise run lint
```

## License

See [LICENSE](LICENSE) for details.
