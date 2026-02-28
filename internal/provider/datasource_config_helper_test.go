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
					resource.TestCheckResourceAttr("data.natsjwt_config_helper.test", "resolver", "MEMORY"),
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
						if !strings.Contains(config, "resolver: MEMORY") {
							return fmt.Errorf("server_config missing resolver")
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

func TestAccConfigHelperDataSource_ResolverPreloadContents(t *testing.T) {
	opKP, _ := nkeys.CreatePair(nkeys.PrefixByteOperator)
	opPub, _ := opKP.PublicKey()

	acct1KP, _ := nkeys.CreatePair(nkeys.PrefixByteAccount)
	acct1Pub, _ := acct1KP.PublicKey()
	acct2KP, _ := nkeys.CreatePair(nkeys.PrefixByteAccount)
	acct2Pub, _ := acct2KP.PublicKey()

	opClaims := natsjwt.NewOperatorClaims(opPub)
	opClaims.Name = "op"
	opClaims.IssuedAt = 0
	opClaims.ID = ""
	opJWT, _ := opClaims.Encode(opKP)

	acct1Claims := natsjwt.NewAccountClaims(acct1Pub)
	acct1Claims.Name = "acct1"
	acct1Claims.IssuedAt = 0
	acct1Claims.ID = ""
	acct1JWT, _ := acct1Claims.Encode(opKP)

	acct2Claims := natsjwt.NewAccountClaims(acct2Pub)
	acct2Claims.Name = "acct2"
	acct2Claims.IssuedAt = 0
	acct2Claims.ID = ""
	acct2JWT, _ := acct2Claims.Encode(opKP)

	config := fmt.Sprintf(`
data "natsjwt_config_helper" "test" {
  operator_jwt = %q
  account_jwts = [%q, %q]
}
`, opJWT, acct1JWT, acct2JWT)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.natsjwt_config_helper.test", "resolver_preload."+acct1Pub, acct1JWT),
					resource.TestCheckResourceAttr("data.natsjwt_config_helper.test", "resolver_preload."+acct2Pub, acct2JWT),
				),
			},
		},
	})
}
