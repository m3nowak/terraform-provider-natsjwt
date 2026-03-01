package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	natsjwt "github.com/nats-io/jwt/v2"
)

func TestAccSystemAccountDataSource_Basic(t *testing.T) {
	opSeed := testOperatorSeed(t)
	acctSeed := testAccountSeed(t)

	config := fmt.Sprintf(`
data "natsjwt_system_account" "test" {
  name          = "SYS"
  seed          = %q
  operator_seed = %q
}
`, acctSeed, opSeed)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.natsjwt_system_account.test", "jwt"),
					resource.TestMatchResourceAttr("data.natsjwt_system_account.test", "public_key", regexp.MustCompile(`^A`)),
					testCheckJWTField("data.natsjwt_system_account.test", func(jwtStr string) error {
						claims, err := natsjwt.DecodeAccountClaims(jwtStr)
						if err != nil {
							return fmt.Errorf("failed to decode system account JWT: %w", err)
						}
						if claims.IssuedAt != 0 {
							return fmt.Errorf("expected default issued_at 0, got %d", claims.IssuedAt)
						}
						if claims.Expires != 0 {
							return fmt.Errorf("expected default expires 0, got %d", claims.Expires)
						}
						if claims.NotBefore != 0 {
							return fmt.Errorf("expected default not_before 0, got %d", claims.NotBefore)
						}
						return nil
					}),
				),
			},
		},
	})
}

func TestAccSystemAccountDataSource_HasDefaultExports(t *testing.T) {
	opSeed := testOperatorSeed(t)
	acctSeed := testAccountSeed(t)

	config := fmt.Sprintf(`
data "natsjwt_system_account" "test" {
  name          = "SYS"
  seed          = %q
  operator_seed = %q
}
`, acctSeed, opSeed)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckJWTField("data.natsjwt_system_account.test", func(jwtStr string) error {
						claims, err := natsjwt.DecodeAccountClaims(jwtStr)
						if err != nil {
							return fmt.Errorf("failed to decode system account JWT: %w", err)
						}
						if len(claims.Exports) < 2 {
							return fmt.Errorf("expected at least 2 default exports, got %d", len(claims.Exports))
						}
						hasServiceExport := false
						hasStreamExport := false
						for _, exp := range claims.Exports {
							if exp.Subject == "$SYS.REQ.ACCOUNT.*.*" && exp.Type == natsjwt.Service {
								hasServiceExport = true
							}
							if exp.Subject == "$SYS.ACCOUNT.*.>" && exp.Type == natsjwt.Stream {
								hasStreamExport = true
							}
						}
						if !hasServiceExport {
							return fmt.Errorf("missing $SYS.REQ.ACCOUNT.*.* service export")
						}
						if !hasStreamExport {
							return fmt.Errorf("missing $SYS.ACCOUNT.*.> stream export")
						}
						return nil
					}),
				),
			},
		},
	})
}

func TestAccSystemAccountDataSource_OverrideDefaults(t *testing.T) {
	opSeed := testOperatorSeed(t)
	acctSeed := testAccountSeed(t)

	config := fmt.Sprintf(`
data "natsjwt_system_account" "test" {
  name          = "SYS"
  seed          = %q
  operator_seed = %q
  issued_at     = 321
  expires       = 654
  tags          = ["system"]
}
`, acctSeed, opSeed)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckJWTField("data.natsjwt_system_account.test", func(jwtStr string) error {
						claims, err := natsjwt.DecodeAccountClaims(jwtStr)
						if err != nil {
							return fmt.Errorf("failed to decode system account JWT: %w", err)
						}
						if len(claims.Tags) != 1 || claims.Tags[0] != "system" {
							return fmt.Errorf("expected tags [system], got %v", claims.Tags)
						}
						if claims.IssuedAt != 321 {
							return fmt.Errorf("expected issued_at 321, got %d", claims.IssuedAt)
						}
						if claims.Expires != 654 {
							return fmt.Errorf("expected expires 654, got %d", claims.Expires)
						}
						if claims.NotBefore != 321 {
							return fmt.Errorf("expected not_before to default to issued_at (321), got %d", claims.NotBefore)
						}
						return nil
					}),
				),
			},
		},
	})
}
