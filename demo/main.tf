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

provider "local" {

}

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

resource "natsjwt_nkey" "app_user2" {
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
    pub_allow = [">"]
    sub_allow = [">"]
  }
}

# User with permissions
data "natsjwt_user" "app_user2" {
  name         = "app-user2"
  seed         = natsjwt_nkey.app_user2.seed
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
  value = data.natsjwt_config_helper.server.operator
}

resource "local_file" "user_creds" {
  filename = "${path.module}/app-user.creds"
  content  = data.natsjwt_user.app_user.creds
}

resource "local_file" "user2_creds" {
  filename = "${path.module}/app-user2.creds"
  content  = data.natsjwt_user.app_user2.creds
}

resource "local_file" "nats_config" {
  filename = "${path.module}/nats-server.conf"
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

# Additional server configuration...
websocket {
  port: 8080
  no_tls: true
}

EOT
}
