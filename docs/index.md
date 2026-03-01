# NATS JWT Provider

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

## Example Usage

```terraform
terraform {
  required_providers {
    natsjwt = {
      source  = "m3nowak/natsjwt"
      version = "~> 0.0"
    }
  }
}

provider "natsjwt" {}

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

resource "natsjwt_nkey" "expired_user" {
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

# Example expired user (for demonstration/testing)
data "natsjwt_user" "expired_user" {
  name         = "expired-user"
  seed         = natsjwt_nkey.expired_user.seed
  account_seed = natsjwt_nkey.app_account.seed
  expires      = 1
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

output "user_creds" {
  value     = data.natsjwt_user.app_user.creds
  sensitive = true
}
```

## Security Notes

- **Seeds are sensitive** — they are stored in Terraform state and marked as sensitive
- **State should be encrypted** — use remote state backends with encryption
- Consider using external seed management for production setups

## Compatibility

- NATS 2.11 and 2.12
- Terraform >= 1.0
- Uses `github.com/nats-io/jwt/v2` and `github.com/nats-io/nkeys`

## Demo

The github repository contains a simple demo in `demo` folder. You can experiment with the provider in it.
