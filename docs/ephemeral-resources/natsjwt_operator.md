# natsjwt_operator Ephemeral Resource

Generates a signed operator JWT without storing the configured seed or generated values in Terraform plan or state. Terraform 1.10 or later is required.

```terraform
ephemeral "natsjwt_operator" "main" {
  name = "main"
  seed = var.operator_seed
}
```

The arguments and `public_key` and `jwt` results match the [`natsjwt_operator` data source](../data-sources/natsjwt_operator.md). `seed` is sensitive. Ephemeral results can only be consumed by other ephemeral contexts, provider configuration, provisioners, or write-only resource arguments.
