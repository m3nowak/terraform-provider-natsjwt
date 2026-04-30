package provider

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	natsjwt "github.com/nats-io/jwt/v2"
	"github.com/nats-io/nkeys"
)

func TestAccConfigHelperDataSource_Basic(t *testing.T) {
	// Create operator
	opKP, _ := nkeys.CreatePair(nkeys.PrefixByteOperator)
	opPub, _ := opKP.PublicKey()

	// Create system account
	sysKP, _ := nkeys.CreatePair(nkeys.PrefixByteAccount)
	sysPub, _ := sysKP.PublicKey()

	// Create regular account
	acctKP, _ := nkeys.CreatePair(nkeys.PrefixByteAccount)
	acctPub, _ := acctKP.PublicKey()

	// Build operator JWT
	opClaims := natsjwt.NewOperatorClaims(opPub)
	opClaims.Name = "test-op"
	opClaims.SystemAccount = sysPub
	opClaims.IssuedAt = 0
	opClaims.ID = ""
	opJWT, _ := opClaims.Encode(opKP)

	// Build system account JWT
	sysClaims := natsjwt.NewAccountClaims(sysPub)
	sysClaims.Name = "SYS"
	sysClaims.IssuedAt = 0
	sysClaims.ID = ""
	sysJWT, _ := sysClaims.Encode(opKP)

	// Build account JWT
	acctClaims := natsjwt.NewAccountClaims(acctPub)
	acctClaims.Name = "test-acct"
	acctClaims.IssuedAt = 0
	acctClaims.ID = ""
	acctJWT, _ := acctClaims.Encode(opKP)

	config := fmt.Sprintf(`
data "natsjwt_config_helper" "test" {
  operator_jwt       = %q
  system_account_jwt = %q
  account_jwts       = [%q]
}
`, opJWT, sysJWT, acctJWT)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.natsjwt_config_helper.test", "operator", opJWT),
					resource.TestCheckResourceAttr("data.natsjwt_config_helper.test", "system_account", sysPub),
					resource.TestCheckResourceAttr("data.natsjwt_config_helper.test", "resolver_preload."+sysPub, sysJWT),
					resource.TestCheckResourceAttr("data.natsjwt_config_helper.test", "resolver_preload."+acctPub, acctJWT),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["data.natsjwt_config_helper.test"]
						if !ok {
							return fmt.Errorf("not found")
						}
						config := rs.Primary.Attributes["server_config"]
						if !strings.Contains(config, "operator:") {
							return fmt.Errorf("server_config missing operator")
						}
						if !strings.Contains(config, "system_account:") {
							return fmt.Errorf("server_config missing system_account")
						}
						if strings.Contains(config, "resolver: MEMORY") {
							return fmt.Errorf("server_config should not contain resolver line")
						}
						if !strings.Contains(config, "resolver_preload:") {
							return fmt.Errorf("server_config missing resolver_preload")
						}
						if !strings.Contains(config, sysPub) {
							return fmt.Errorf("server_config missing system account key")
						}
						if !strings.Contains(config, acctPub) {
							return fmt.Errorf("server_config missing account key")
						}
						return nil
					},
				),
			},
		},
	})
}

func TestAccConfigHelperDataSource_NoSystemAccount(t *testing.T) {
	opKP, _ := nkeys.CreatePair(nkeys.PrefixByteOperator)
	opPub, _ := opKP.PublicKey()

	acctKP, _ := nkeys.CreatePair(nkeys.PrefixByteAccount)
	acctPub, _ := acctKP.PublicKey()

	opClaims := natsjwt.NewOperatorClaims(opPub)
	opClaims.Name = "op"
	opClaims.IssuedAt = 0
	opClaims.ID = ""
	opJWT, _ := opClaims.Encode(opKP)

	acctClaims := natsjwt.NewAccountClaims(acctPub)
	acctClaims.Name = "acct"
	acctClaims.IssuedAt = 0
	acctClaims.ID = ""
	acctJWT, _ := acctClaims.Encode(opKP)

	config := fmt.Sprintf(`
data "natsjwt_config_helper" "test" {
  operator_jwt = %q
  account_jwts = [%q]
}
`, opJWT, acctJWT)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.natsjwt_config_helper.test", "operator", opJWT),
					resource.TestCheckResourceAttr("data.natsjwt_config_helper.test", "system_account", ""),
					resource.TestCheckResourceAttr("data.natsjwt_config_helper.test", "resolver_preload."+acctPub, acctJWT),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["data.natsjwt_config_helper.test"]
						if !ok {
							return fmt.Errorf("not found")
						}
						config := rs.Primary.Attributes["server_config"]
						if !strings.Contains(config, "operator:") {
							return fmt.Errorf("server_config missing operator")
						}
						if strings.Contains(config, "system_account:") {
							return fmt.Errorf("server_config should not contain system_account when not provided")
						}
						if strings.Contains(config, "resolver: MEMORY") {
							return fmt.Errorf("server_config should not contain resolver line")
						}
						if !strings.Contains(config, "resolver_preload:") {
							return fmt.Errorf("server_config missing resolver_preload")
						}
						return nil
					},
				),
			},
		},
	})
}
