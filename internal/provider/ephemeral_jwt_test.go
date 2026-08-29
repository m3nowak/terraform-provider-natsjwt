package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

func TestProviderRegistersJWTEphemeralResources(t *testing.T) {
	t.Parallel()

	providerWithEphemeral, ok := New("test")().(provider.ProviderWithEphemeralResources)
	if !ok {
		t.Fatal("provider does not implement provider.ProviderWithEphemeralResources")
	}

	wantTypes := map[string]bool{
		"natsjwt_operator":       false,
		"natsjwt_account":        false,
		"natsjwt_system_account": false,
		"natsjwt_user":           false,
	}

	for _, factory := range providerWithEphemeral.EphemeralResources(context.Background()) {
		resource := factory()
		var metadata ephemeral.MetadataResponse
		resource.Metadata(context.Background(), ephemeral.MetadataRequest{ProviderTypeName: "natsjwt"}, &metadata)
		if _, ok := wantTypes[metadata.TypeName]; !ok {
			t.Fatalf("unexpected ephemeral resource type %q", metadata.TypeName)
		}
		wantTypes[metadata.TypeName] = true

		var schemaResponse ephemeral.SchemaResponse
		resource.Schema(context.Background(), ephemeral.SchemaRequest{}, &schemaResponse)
		if schemaResponse.Diagnostics.HasError() {
			t.Fatalf("schema for %s returned errors: %v", metadata.TypeName, schemaResponse.Diagnostics)
		}
		if diagnostics := schemaResponse.Schema.Validate(); diagnostics.HasError() {
			t.Fatalf("schema for %s is invalid: %v", metadata.TypeName, diagnostics)
		}
		if !schemaResponse.Schema.Attributes["seed"].IsSensitive() {
			t.Errorf("%s seed must be sensitive", metadata.TypeName)
		}
		if metadata.TypeName == "natsjwt_user" && !schemaResponse.Schema.Attributes["creds"].IsSensitive() {
			t.Error("natsjwt_user creds must be sensitive")
		}
		if metadata.TypeName == "natsjwt_user" && !schemaResponse.Schema.Attributes["account_seed"].IsSensitive() {
			t.Error("natsjwt_user account_seed must be sensitive")
		}
		if (metadata.TypeName == "natsjwt_account" || metadata.TypeName == "natsjwt_system_account") && !schemaResponse.Schema.Attributes["operator_seed"].IsSensitive() {
			t.Errorf("%s operator_seed must be sensitive", metadata.TypeName)
		}
	}

	for typeName, registered := range wantTypes {
		if !registered {
			t.Errorf("ephemeral resource %s is not registered", typeName)
		}
	}
}

func TestAccJWTEphemeralResourcesAreNotInArtifacts(t *testing.T) {
	opSeed := testOperatorSeed(t)
	accountSeed := testAccountSeed(t)
	userSeed := testUserSeed(t)
	userJWT, userCreds := expectedBasicUserEphemeralValues(t, userSeed, accountSeed)
	artifactValues := []string{opSeed, accountSeed, userSeed, userJWT, userCreds}

	config := fmt.Sprintf(`
ephemeral "natsjwt_operator" "test" {
  name = "operator"
  seed = %q
}

ephemeral "natsjwt_account" "test" {
  name          = "account"
  seed          = %q
  operator_seed = %q
}

ephemeral "natsjwt_system_account" "test" {
  name          = "SYS"
  seed          = %q
  operator_seed = %q
}

ephemeral "natsjwt_user" "test" {
  name         = "user"
  seed         = %q
  account_seed = %q
}
`, opSeed, accountSeed, opSeed, accountSeed, opSeed, userSeed, accountSeed)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(version.Must(version.NewVersion("1.10.0"))),
		},
		Steps: []resource.TestStep{{
			Config: config,
			ConfigPlanChecks: resource.ConfigPlanChecks{
				PreApply: []plancheck.PlanCheck{planDoesNotContain{values: artifactValues}},
			},
			Check: func(state *terraform.State) error {
				stateText := state.String()
				for _, value := range artifactValues {
					if strings.Contains(stateText, value) {
						return fmt.Errorf("ephemeral value was persisted in Terraform state")
					}
				}
				if strings.Contains(stateText, "natsjwt_user.test") {
					return fmt.Errorf("ephemeral resource was persisted in Terraform state")
				}
				return nil
			},
		}},
	})
}

func expectedBasicUserEphemeralValues(t *testing.T, userSeed, accountSeed string) (string, string) {
	t.Helper()
	data := UserDataSourceModel{
		Name:        types.StringValue("user"),
		Seed:        types.StringValue(userSeed),
		AccountSeed: types.StringValue(accountSeed),
	}
	var diagnostics diag.Diagnostics
	generateUser(context.Background(), &data, &diagnostics)
	if diagnostics.HasError() {
		t.Fatalf("generate expected user values: %v", diagnostics)
	}
	return data.JWT.ValueString(), data.Creds.ValueString()
}

func TestAccJWTEphemeralResourcesMatchDataSources(t *testing.T) {
	opSeed := testOperatorSeed(t)
	accountSeed := testAccountSeed(t)
	userSeed := testUserSeed(t)

	config := fmt.Sprintf(`
ephemeral "natsjwt_operator" "test" {
  name = "operator"
  seed = %q
}
data "natsjwt_operator" "test" {
  name = "operator"
  seed = %q
}

ephemeral "natsjwt_account" "test" {
  name          = "account"
  seed          = %q
  operator_seed = %q
}
data "natsjwt_account" "test" {
  name          = "account"
  seed          = %q
  operator_seed = %q
}

ephemeral "natsjwt_system_account" "test" {
  name          = "SYS"
  seed          = %q
  operator_seed = %q
}
data "natsjwt_system_account" "test" {
  name          = "SYS"
  seed          = %q
  operator_seed = %q
}

ephemeral "natsjwt_user" "test" {
  name         = "user"
  seed         = %q
  account_seed = %q
}
data "natsjwt_user" "test" {
  name         = "user"
  seed         = %q
  account_seed = %q
}

resource "terraform_data" "verify" {
  provisioner "local-exec" {
    command = <<-EOT
      test "$OP_EPHEMERAL" = "$OP_DATA" &&
      test "$OP_KEY_EPHEMERAL" = "$OP_KEY_DATA" &&
      test "$ACCOUNT_EPHEMERAL" = "$ACCOUNT_DATA" &&
      test "$ACCOUNT_KEY_EPHEMERAL" = "$ACCOUNT_KEY_DATA" &&
      test "$SYSTEM_EPHEMERAL" = "$SYSTEM_DATA" &&
      test "$SYSTEM_KEY_EPHEMERAL" = "$SYSTEM_KEY_DATA" &&
      test "$USER_EPHEMERAL" = "$USER_DATA" &&
      test "$USER_KEY_EPHEMERAL" = "$USER_KEY_DATA" &&
      test "$CREDS_EPHEMERAL" = "$CREDS_DATA"
    EOT
    environment = {
      OP_EPHEMERAL      = ephemeral.natsjwt_operator.test.jwt
      OP_DATA           = data.natsjwt_operator.test.jwt
      OP_KEY_EPHEMERAL  = ephemeral.natsjwt_operator.test.public_key
      OP_KEY_DATA       = data.natsjwt_operator.test.public_key
      ACCOUNT_EPHEMERAL = ephemeral.natsjwt_account.test.jwt
      ACCOUNT_DATA      = data.natsjwt_account.test.jwt
      ACCOUNT_KEY_EPHEMERAL = ephemeral.natsjwt_account.test.public_key
      ACCOUNT_KEY_DATA      = data.natsjwt_account.test.public_key
      SYSTEM_EPHEMERAL  = ephemeral.natsjwt_system_account.test.jwt
      SYSTEM_DATA       = data.natsjwt_system_account.test.jwt
      SYSTEM_KEY_EPHEMERAL = ephemeral.natsjwt_system_account.test.public_key
      SYSTEM_KEY_DATA      = data.natsjwt_system_account.test.public_key
      USER_EPHEMERAL    = ephemeral.natsjwt_user.test.jwt
      USER_DATA         = data.natsjwt_user.test.jwt
      USER_KEY_EPHEMERAL = ephemeral.natsjwt_user.test.public_key
      USER_KEY_DATA      = data.natsjwt_user.test.public_key
      CREDS_EPHEMERAL   = ephemeral.natsjwt_user.test.creds
      CREDS_DATA        = data.natsjwt_user.test.creds
    }
  }
}
`, opSeed, opSeed, accountSeed, opSeed, accountSeed, opSeed, accountSeed, opSeed, accountSeed, opSeed, userSeed, accountSeed, userSeed, accountSeed)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(version.Must(version.NewVersion("1.10.0"))),
		},
		Steps: []resource.TestStep{{Config: config}},
	})
}

type planDoesNotContain struct {
	values []string
}

func (check planDoesNotContain) CheckPlan(_ context.Context, req plancheck.CheckPlanRequest, resp *plancheck.CheckPlanResponse) {
	planJSON, err := json.Marshal(req.Plan)
	if err != nil {
		resp.Error = fmt.Errorf("marshal Terraform plan: %w", err)
		return
	}
	for _, value := range check.values {
		if strings.Contains(string(planJSON), value) {
			resp.Error = fmt.Errorf("ephemeral value was persisted in Terraform plan")
			return
		}
	}
}

func TestAccJWTEphemeralResourceRejectsWrongSeedType(t *testing.T) {
	opSeed := testOperatorSeed(t)
	accountSeed := testAccountSeed(t)
	userSeed := testUserSeed(t)
	tests := map[string]string{
		"operator":       fmt.Sprintf("ephemeral \"natsjwt_operator\" \"test\" {\nname = \"operator\"\nseed = %q\n}", accountSeed),
		"account":        fmt.Sprintf("ephemeral \"natsjwt_account\" \"test\" {\nname = \"account\"\nseed = %q\noperator_seed = %q\n}", userSeed, opSeed),
		"system_account": fmt.Sprintf("ephemeral \"natsjwt_system_account\" \"test\" {\nname = \"SYS\"\nseed = %q\noperator_seed = %q\n}", userSeed, opSeed),
		"user":           fmt.Sprintf("ephemeral \"natsjwt_user\" \"test\" {\nname = \"user\"\nseed = %q\naccount_seed = %q\n}", accountSeed, accountSeed),
	}

	for name, config := range tests {
		t.Run(name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				TerraformVersionChecks: []tfversion.TerraformVersionCheck{
					tfversion.SkipBelow(version.Must(version.NewVersion("1.10.0"))),
				},
				Steps: []resource.TestStep{{
					Config:      config,
					ExpectError: regexp.MustCompile(`Wrong NKey Seed Type|Expected (operator|account|user) seed`),
				}},
			})
		})
	}
}
