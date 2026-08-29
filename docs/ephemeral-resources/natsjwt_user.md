# natsjwt_user Ephemeral Resource

Generates a signed user JWT and `.creds` content without storing the configured seeds or generated values in Terraform plan or state. Terraform 1.10 or later is required.

```terraform
ephemeral "natsjwt_user" "app" {
  name         = "app-user"
  seed         = var.user_seed
  account_seed = var.account_seed

  permissions = {
    pub_allow = ["app.>"]
    sub_allow = ["app.>", "_INBOX.>"]
  }
}

# Pass ephemeral.natsjwt_user.app.creds only to an ephemeral context,
# provider configuration, provisioner, or write-only resource argument.
```

The arguments and results match the [`natsjwt_user` data source](../data-sources/natsjwt_user.md). Results are `public_key`, `jwt`, and sensitive `creds`; `seed` and `account_seed` are also sensitive.

Marking a value sensitive only redacts its display. Using this ephemeral resource is what prevents its configuration and results from being persisted by Terraform. Existing data sources remain appropriate when downstream configuration requires values stored in state.
