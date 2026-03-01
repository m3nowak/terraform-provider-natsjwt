terraform {
  required_providers {
    natsjwt = {
      source = "m3nowak/natsjwt"
    }
  }
}

# Generate NKeys for all entities
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

# Create system account (gets default $SYS.> exports)
data "natsjwt_system_account" "sys" {
  name          = "SYS"
  seed          = natsjwt_nkey.sys_account.seed
  operator_seed = natsjwt_nkey.operator.seed
}

# Create operator JWT (references system account)
data "natsjwt_operator" "main" {
  name           = "my-operator"
  seed           = natsjwt_nkey.operator.seed
  system_account = data.natsjwt_system_account.sys.public_key
}

# Create application account with JetStream limits
data "natsjwt_account" "app" {
  name          = "app-account"
  seed          = natsjwt_nkey.app_account.seed
  operator_seed = natsjwt_nkey.operator.seed

  jetstream_limits = [{
    mem_storage  = 1073741824  # 1 GB
    disk_storage = 10737418240 # 10 GB
    streams      = 10
    consumer     = 100
  }]
}

# Create a user with pub/sub permissions
data "natsjwt_user" "app_user" {
  name         = "app-user"
  seed         = natsjwt_nkey.app_user.seed
  account_seed = natsjwt_nkey.app_account.seed

  permissions = {
    pub_allow = ["app.>"]
    sub_allow = ["app.>", "_INBOX.>"]
  }
}

# Demonstration user with an already expired JWT
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
  value       = data.natsjwt_config_helper.server.server_config
  description = "NATS server configuration"
}

output "operator_jwt" {
  value       = data.natsjwt_operator.main.jwt
  description = "Operator JWT"
}

output "user_jwt" {
  value       = data.natsjwt_user.app_user.jwt
  description = "User JWT for app-user"
  sensitive   = true
}

output "user_creds" {
  value       = data.natsjwt_user.app_user.creds
  description = "User credentials file content for app-user"
  sensitive   = true
}

output "user_seed" {
  value       = natsjwt_nkey.app_user.seed
  description = "User seed for app-user"
  sensitive   = true
}
