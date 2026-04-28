# natsjwt_resolver_account Resource

Uploads and updates an account JWT on a running NATS server using the NATS-based resolver.

## Example Usage

```terraform
# Push an account JWT to the NATS resolver
resource "natsjwt_resolver_account" "app" {
  jwt = data.natsjwt_account.app.jwt
}

# With operator seed to support deletion from the server
resource "natsjwt_resolver_account" "app" {
  jwt           = data.natsjwt_account.app.jwt
  operator_seed = natsjwt_nkey.operator.seed
}
```

## Argument Reference

- `jwt` - (Required) The signed account JWT to push to the NATS resolver.
- `operator_seed` - (Optional, Sensitive) Operator seed used to sign deletion requests. If omitted, `terraform destroy` will only remove the resource from state and will **not** delete the account from the server.

## Attributes Reference

This resource exports no additional attributes.

## Notes

- The provider must be configured with `nats_url` and `creds` for this resource to work.
- On create and update, the resource sends the JWT to `$SYS.REQ.ACCOUNT.<pubkey>.CLAIMS.UPDATE`.
- On read, it looks up the account via `$SYS.REQ.ACCOUNT.<pubkey>.CLAIMS.LOOKUP` and compares the returned JWT.
- On delete, if `operator_seed` is provided, it sends an operator-signed delete request to `$SYS.REQ.CLAIMS.DELETE`.
