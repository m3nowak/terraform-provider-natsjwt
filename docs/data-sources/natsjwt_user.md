# natsjwt_user Data Source

Generates a signed user JWT. Users are the final tier of NATS security hierarchy and are the credentials used by applications to connect to NATS.

## Example Usage

```terraform
# Simple user with full account access
data "natsjwt_user" "basic" {
  name         = "app-user"
  seed         = natsjwt_nkey.app_user.seed
  account_seed = natsjwt_nkey.app_account.seed
}

# User with specific permissions
data "natsjwt_user" "app_user" {
  name         = "app-user"
  seed         = natsjwt_nkey.app_user.seed
  account_seed = natsjwt_nkey.app_account.seed

  permissions = [{
    pub_allow = ["app.>", "admin.requests.app.>"]
    pub_deny  = ["admin.>"]
    sub_allow = ["app.>", "_INBOX.>"]
    sub_deny  = []
  }]
}

# User with connection limits
data "natsjwt_user" "limited" {
  name         = "consumer"
  seed         = natsjwt_nkey.consumer_user.seed
  account_seed = natsjwt_nkey.app_account.seed

  limits = [{
    subs    = 1000
    data    = 1073741824  # 1GB
    payload = 262144      # 256KB
  }]
}

# User with time restrictions
data "natsjwt_user" "time_restricted" {
  name         = "temp-user"
  seed         = natsjwt_nkey.temp_user.seed
  account_seed = natsjwt_nkey.app_account.seed

  time_restrictions = [{
    start = "09:00:00"
    end   = "17:00:00"
  }]
  locale = "America/New_York"
}

# User with connection type restrictions
data "natsjwt_user" "websocket_only" {
  name         = "web-client"
  seed         = natsjwt_nkey.web_user.seed
  account_seed = natsjwt_nkey.app_account.seed

  allowed_connection_types = ["WEBSOCKET"]
}

# User with source network restrictions
data "natsjwt_user" "restricted_network" {
  name         = "office-user"
  seed         = natsjwt_nkey.office_user.seed
  account_seed = natsjwt_nkey.app_account.seed

  source_networks = [
    "10.0.0.0/8",
    "192.168.0.0/16"
  ]
}

# User with an already expired JWT (demonstration)
data "natsjwt_user" "expired_demo" {
  name         = "expired-user"
  seed         = natsjwt_nkey.app_user.seed
  account_seed = natsjwt_nkey.app_account.seed
  expires      = 1
}
```

## Argument Reference

- `name` - (Required) User name.
- `seed` - (Required, sensitive) User seed (private key).
- `account_seed` - (Required, sensitive) Account seed for signing.
- `issuer_account` - (Optional) Account public key (when using a signing key).
- `issued_at` - (Optional) JWT issued-at Unix timestamp. Defaults to `0` (Unix epoch).
- `expires` - (Optional) JWT expiration Unix timestamp. Defaults to no expiration.
- `not_before` - (Optional) JWT not-before Unix timestamp. Defaults to `issued_at`.
- `permissions` - (Optional) Pub/sub permissions. See [Permissions](#permissions-1) below.
- `limits` - (Optional) Connection limits. See [Limits](#limits-1) below.
- `bearer_token` - (Optional) Allow bearer tokens.
- `allowed_connection_types` - (Optional) List of allowed connection types. Valid values: `STANDARD`, `WEBSOCKET`, `LEAFNODE`, `MQTT`.
- `source_networks` - (Optional) List of allowed CIDR blocks.
- `time_restrictions` - (Optional) Time-based access restrictions. See [Time Restrictions](#time-restrictions-1) below.
- `locale` - (Optional) Timezone for time restrictions (e.g., `America/New_York`).

### Permissions

- `pub_allow` - (Optional) List of allowed publish subjects.
- `pub_deny` - (Optional) List of denied publish subjects.
- `sub_allow` - (Optional) List of allowed subscribe subjects.
- `sub_deny` - (Optional) List of denied subscribe subjects.

### Limits

- `subs` - (Optional) Maximum number of subjects.
- `data` - (Optional) Maximum data in bytes.
- `payload` - (Optional) Maximum payload in bytes.

### Time Restrictions

- `start` - (Optional) Start time in HH:MM:SS format.
- `end` - (Optional) End time in HH:MM:SS format.

## Attributes Reference

- `public_key` - The user public key (starts with `U`).
- `jwt` - The signed user JWT (sensitive).
- `creds` - Full decorated NATS user credentials content (`.creds` format, includes JWT and user seed; sensitive).

## Notes

- Users inherit default permissions from their account if user-specific permissions are not set
- Connection type restrictions allow fine-grained control over connection protocols
- Source network restrictions are enforced at the NATS server level
- Time restrictions require a valid locale to be set
