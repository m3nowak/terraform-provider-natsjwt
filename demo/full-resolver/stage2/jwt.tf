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
provider "local" {}

data "local_sensitive_file" "creds" {
  filename = "${path.module}/../sys-user.creds"
}

provider "natsjwt" {
  nats_url = "nats://localhost:4222"
  creds    = data.local_sensitive_file.creds.content
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

data "local_sensitive_file" "operator_seed" {
  filename = "${path.module}/../operator.nk"
}

# Application account with JetStream
data "natsjwt_account" "app" {
  name          = "app"
  seed          = natsjwt_nkey.app_account.seed
  operator_seed = data.local_sensitive_file.operator_seed.content

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

resource "local_file" "user_creds" {
  filename = "${path.module}/../app-user.creds"
  content  = data.natsjwt_user.app_user.creds
}

resource "local_file" "user2_creds" {
  filename = "${path.module}/../app-user2.creds"
  content  = data.natsjwt_user.app_user2.creds
}

resource "natsjwt_resolver_account" "app" {
  jwt = data.natsjwt_account.app.jwt
  # operator_seed - omitted
}
