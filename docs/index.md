# NATS JWT Provider

A Terraform provider for managing [NATS](https://nats.io/) JWT credentials offline. Issuance does not need a running NATS server.

It covers the [`nsc`](https://github.com/nats-io/nsc) happy path: operator, account, user, and server config as Terraform. It is not a drop-in nsc replacement. See [Known limitations](#known-limitations).

## Features

- **Offline issuance.** Generates NKeys and signed JWTs without connecting to a NATS server.
- **Deterministic JWTs.** Same inputs produce the same JWT, so `terraform plan` stays quiet.
- **Operators, accounts, users.** JetStream limits on accounts. A separate system-account data source with default `$SYS` exports.
- **Server config.** Builds a NATS config snippet (`operator`, `system_account`, `resolver_preload`).
- **Seed type checks.** Rejects the wrong nkey prefix for each operation.
- **External seeds.** Pass seeds from Vault or similar, or generate them with `natsjwt_nkey`.
- **Ephemeral credentials.** With Terraform 1.10+, issue JWTs without writing seeds or results to plan or state.
- **Seed conversion.** `provider::natsjwt::seed_public_key(...)` derives a public key from a seed.

## Example Usage

```terraform
terraform {
  required_providers {
    natsjwt = {
      source  = "m3nowak/natsjwt"
      version = "~> 0.1"
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

## Provider Configuration

The provider supports the following optional arguments:

- `nats_url` — NATS server URL for resolver interactions (e.g. `nats://localhost:4222`). Required when using `natsjwt_resolver_account` resources.
- `creds` — Contents of a NATS credentials file (`.creds`) for authentication.

```terraform
provider "natsjwt" {
  nats_url = "nats://localhost:4222"
  creds    = file("/path/to/sys.creds")
}
```

## NATS-Based Resolver Account Resource

The `natsjwt_resolver_account` resource uploads and updates account JWTs on a running NATS server that uses the NATS-based resolver.

```terraform
resource "natsjwt_resolver_account" "app" {
  jwt = data.natsjwt_account.app.jwt
}
```

### Arguments

- `jwt` (**Required**) — The signed account JWT to push to the NATS resolver.
- `operator_seed` (Optional, Sensitive) — Operator seed used to sign deletion requests. If omitted, `terraform destroy` will only remove the resource from state and will **not** delete the account from the server.

### Deleting Accounts

To delete an account from the NATS resolver server, provide the `operator_seed`:

```terraform
resource "natsjwt_resolver_account" "app" {
  jwt           = data.natsjwt_account.app.jwt
  operator_seed = natsjwt_nkey.operator.seed
}
```

If `operator_seed` is omitted, Terraform will emit a warning during destroy and leave the account on the server.

## Security Notes

- **Sensitive is not ephemeral** — sensitive data-source attributes are redacted in CLI output but are still stored in Terraform plan and state
- **Ephemeral JWT generation** — with Terraform 1.10 or later, use the `natsjwt_operator`, `natsjwt_account`, `natsjwt_system_account`, and `natsjwt_user` ephemeral resources when seeds and generated credentials must not be persisted
- **Ephemeral result restrictions** — ephemeral results can only flow into other ephemeral contexts, provider configuration, provisioners, or write-only resource arguments
- **Regular data-source seeds are sensitive** — they are stored in Terraform state and marked as sensitive
- **State should be encrypted** — use remote state backends with encryption
- Consider using external seed management for production setups

See the [ephemeral resource guide](ephemeral-resources/natsjwt_user.md) for usage and migration details. Existing data sources remain available when persisted outputs are required.

## Known Limitations

**Signing keys are listed, not used.** `signing_keys` on operator and account JWTs is a public-key list. This provider still signs with the identity seed. `operator_seed` must be an operator seed (`SO`). `account_seed` must be an account seed (`SA`). In NATS, an operator signing key is an account nkey (`SA`/`A`) and an account signing key is a user nkey (`SU`/`U`). Those seeds are rejected. `issuer_account` is copied into the user JWT. It does not change the signer.

Do not set `strict_signing_key_usage = true` unless something else will sign account JWTs. NATS will then reject accounts signed with the operator identity key, which is the only signer this provider has.

**Import and export subjects are missing.** `account_limits.imports` and `account_limits.exports` are numeric caps. There is no way to declare stream or service import/export subjects on an account. Subject mappings and revocations are also absent.

## Compatibility

- NATS 2.11, 2.12, and 2.14
- Terraform >= 1.0 for regular resources and data sources; Terraform >= 1.10 for ephemeral resources
- Go 1.25 and 1.26
- Uses `github.com/nats-io/jwt/v2` and `github.com/nats-io/nkeys`

## Demo

The github repository contains a simple demo in `demo` folder. You can experiment with the provider in it.
