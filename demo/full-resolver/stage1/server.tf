terraform {
  required_providers {
    natsjwt = {
      source  = "m3nowak/natsjwt"
      version = "~> 0.0"
    }
    local = {
      source  = "hashicorp/local"
      version = "2.7.0"
    }
  }
}

provider "natsjwt" {}

provider "local" {}

resource "natsjwt_nkey" "operator" {
  type = "operator"
}

resource "natsjwt_nkey" "sys_account" {
  type = "account"
}

resource "natsjwt_nkey" "sys_user" {
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


# Generate NATS server config
data "natsjwt_config_helper" "server" {
  operator_jwt       = data.natsjwt_operator.main.jwt
  system_account_jwt = data.natsjwt_system_account.sys.jwt
  account_jwts       = []
}

output "server_config" {
  value = data.natsjwt_config_helper.server.operator
}

# User with permissions
data "natsjwt_user" "sys_user" {
  name         = "sys-user"
  seed         = natsjwt_nkey.sys_user.seed
  account_seed = natsjwt_nkey.sys_account.seed

  permissions = {
    pub_allow = [">"]
    sub_allow = [">"]
  }
}

resource "local_file" "sys_user_creds" {
  filename = "${path.module}/../sys-user.creds"
  content  = data.natsjwt_user.sys_user.creds
}

resource "local_sensitive_file" "operator_seed" {
  filename = "${path.module}/../operator.nk"
  content  = natsjwt_nkey.operator.seed
}

resource "local_file" "nats_config" {
  filename = "${path.module}/../nats-server.conf"
  content  = <<-EOT
# NATS Server Configuration

server_name: "my-nats-server"
port: 4222
max_payload: 1MB

jetstream {
    store_dir: jetstream
    max_file: 100G
}

${data.natsjwt_config_helper.server.server_config}

resolver: {
    type: full
    dir: './jwt'
    allow_delete: false
    interval: "10m"
    limit: 1000
}

# Additional server configuration...
websocket {
  port: 8080
  no_tls: true
}

EOT
}
