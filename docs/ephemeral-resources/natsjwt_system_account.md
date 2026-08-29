# natsjwt_system_account Ephemeral Resource

Generates a signed system-account JWT without storing the configured seeds or generated values in Terraform plan or state. Terraform 1.10 or later is required.

```terraform
ephemeral "natsjwt_system_account" "sys" {
  name          = "SYS"
  seed          = var.system_account_seed
  operator_seed = var.operator_seed
}
```

The arguments and `public_key` and `jwt` results match the [`natsjwt_system_account` data source](../data-sources/natsjwt_system_account.md), including its default system exports and override behavior. `seed` and `operator_seed` are sensitive. Ephemeral results can only be consumed by other ephemeral contexts, provider configuration, provisioners, or write-only resource arguments.
