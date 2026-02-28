package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	natsjwt "github.com/nats-io/jwt/v2"
)

func TestAccIntegration_FullChain(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
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

data "natsjwt_system_account" "sys" {
  name          = "SYS"
  seed          = natsjwt_nkey.sys_account.seed
  operator_seed = natsjwt_nkey.operator.seed
}

data "natsjwt_operator" "main" {
  name           = "test-operator"
  seed           = natsjwt_nkey.operator.seed
  system_account = data.natsjwt_system_account.sys.public_key
}

data "natsjwt_account" "app" {
  name          = "app-account"
  seed          = natsjwt_nkey.app_account.seed
  operator_seed = natsjwt_nkey.operator.seed
  jetstream_limits = [{
    mem_storage  = 1073741824
    disk_storage = 10737418240
    streams      = 10
    consumer     = 100
  }]
}

data "natsjwt_user" "app_user" {
  name         = "app-user"
  seed         = natsjwt_nkey.app_user.seed
  account_seed = natsjwt_nkey.app_account.seed
  permissions = {
    pub_allow = ["app.>"]
    sub_allow = ["app.>", "_INBOX.>"]
  }
}

data "natsjwt_config_helper" "server" {
  operator_jwt       = data.natsjwt_operator.main.jwt
  system_account_jwt = data.natsjwt_system_account.sys.jwt
  account_jwts       = [data.natsjwt_account.app.jwt]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// NKey resources created correctly
					resource.TestMatchResourceAttr("natsjwt_nkey.operator", "seed", regexp.MustCompile(`^SO`)),
					resource.TestMatchResourceAttr("natsjwt_nkey.sys_account", "seed", regexp.MustCompile(`^SA`)),
					resource.TestMatchResourceAttr("natsjwt_nkey.app_account", "seed", regexp.MustCompile(`^SA`)),
					resource.TestMatchResourceAttr("natsjwt_nkey.app_user", "seed", regexp.MustCompile(`^SU`)),

					// Operator JWT
					resource.TestCheckResourceAttrSet("data.natsjwt_operator.main", "jwt"),
					resource.TestMatchResourceAttr("data.natsjwt_operator.main", "public_key", regexp.MustCompile(`^O`)),

					// System account JWT with default exports
					resource.TestCheckResourceAttrSet("data.natsjwt_system_account.sys", "jwt"),

					// Account JWT
					resource.TestCheckResourceAttrSet("data.natsjwt_account.app", "jwt"),

					// User JWT
					resource.TestCheckResourceAttrSet("data.natsjwt_user.app_user", "jwt"),

					// Config helper
					resource.TestCheckResourceAttrSet("data.natsjwt_config_helper.server", "server_config"),
					resource.TestCheckResourceAttr("data.natsjwt_config_helper.server", "resolver", "MEMORY"),

					// Validate JWT chain
					func(s *terraform.State) error {
						// Get all JWTs
						opJWT := s.RootModule().Resources["data.natsjwt_operator.main"].Primary.Attributes["jwt"]
						sysJWT := s.RootModule().Resources["data.natsjwt_system_account.sys"].Primary.Attributes["jwt"]
						appJWT := s.RootModule().Resources["data.natsjwt_account.app"].Primary.Attributes["jwt"]
						userJWT := s.RootModule().Resources["data.natsjwt_user.app_user"].Primary.Attributes["jwt"]
						opPub := s.RootModule().Resources["data.natsjwt_operator.main"].Primary.Attributes["public_key"]

						// Decode and verify operator
						opClaims, err := natsjwt.DecodeOperatorClaims(opJWT)
						if err != nil {
							return fmt.Errorf("failed to decode operator JWT: %w", err)
						}
						if opClaims.Name != "test-operator" {
							return fmt.Errorf("operator name mismatch: %s", opClaims.Name)
						}

						// Verify system account reference
						sysClaims, err := natsjwt.DecodeAccountClaims(sysJWT)
						if err != nil {
							return fmt.Errorf("failed to decode sys account JWT: %w", err)
						}
						if opClaims.SystemAccount != sysClaims.Subject {
							return fmt.Errorf("operator system_account doesn't match sys account pubkey")
						}

						// Verify account is signed by operator
						appClaims, err := natsjwt.DecodeAccountClaims(appJWT)
						if err != nil {
							return fmt.Errorf("failed to decode app account JWT: %w", err)
						}
						if appClaims.Issuer != opPub {
							return fmt.Errorf("account not issued by operator")
						}
						if appClaims.Limits.MemoryStorage != 1073741824 {
							return fmt.Errorf("account JetStream mem_storage mismatch")
						}

						// Verify user JWT
						userClaims, err := natsjwt.DecodeUserClaims(userJWT)
						if err != nil {
							return fmt.Errorf("failed to decode user JWT: %w", err)
						}
						if userClaims.Name != "app-user" {
							return fmt.Errorf("user name mismatch")
						}
						if len(userClaims.Pub.Allow) != 1 || userClaims.Pub.Allow[0] != "app.>" {
							return fmt.Errorf("user pub_allow mismatch")
						}

						// Verify config contains all accounts
						serverConfig := s.RootModule().Resources["data.natsjwt_config_helper.server"].Primary.Attributes["server_config"]
						if serverConfig == "" {
							return fmt.Errorf("server_config is empty")
						}

						return nil
					},
				),
			},
		},
	})
}

func TestAccIntegration_ExternalSeeds(t *testing.T) {
	// Simulating seeds from an external source (e.g., vault)
	opSeed := testOperatorSeed(t)
	acctSeed := testAccountSeed(t)
	userSeed := testUserSeed(t)

	config := fmt.Sprintf(`
data "natsjwt_operator" "ext" {
  name = "external-op"
  seed = %q
}

data "natsjwt_account" "ext" {
  name          = "external-acct"
  seed          = %q
  operator_seed = %q
}

data "natsjwt_user" "ext" {
  name         = "external-user"
  seed         = %q
  account_seed = %q
}
`, opSeed, acctSeed, opSeed, userSeed, acctSeed)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.natsjwt_operator.ext", "jwt"),
					resource.TestCheckResourceAttrSet("data.natsjwt_account.ext", "jwt"),
					resource.TestCheckResourceAttrSet("data.natsjwt_user.ext", "jwt"),
				),
			},
		},
	})
}
