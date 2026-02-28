package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	natsjwt "github.com/nats-io/jwt/v2"
	"github.com/nats-io/nkeys"
)

func testOperatorSeed(t *testing.T) string {
	t.Helper()
	kp, err := nkeys.CreatePair(nkeys.PrefixByteOperator)
	if err != nil {
		t.Fatal(err)
	}
	seed, err := kp.Seed()
	if err != nil {
		t.Fatal(err)
	}
	return string(seed)
}

func TestAccOperatorDataSource_Basic(t *testing.T) {
	seed := testOperatorSeed(t)
	config := fmt.Sprintf(`
data "natsjwt_operator" "test" {
  name = "test-op"
  seed = %q
}
`, seed)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.natsjwt_operator.test", "jwt"),
					resource.TestMatchResourceAttr("data.natsjwt_operator.test", "public_key", regexp.MustCompile(`^O`)),
				),
			},
		},
	})
}

func TestAccOperatorDataSource_AllFields(t *testing.T) {
	seed := testOperatorSeed(t)

	// Create a signing key
	sigKP, err := nkeys.CreatePair(nkeys.PrefixByteOperator)
	if err != nil {
		t.Fatal(err)
	}
	sigPub, _ := sigKP.PublicKey()

	// Create a system account key
	sysKP, err := nkeys.CreatePair(nkeys.PrefixByteAccount)
	if err != nil {
		t.Fatal(err)
	}
	sysPub, _ := sysKP.PublicKey()

	config := fmt.Sprintf(`
data "natsjwt_operator" "test" {
  name                    = "full-op"
  seed                    = %q
  signing_keys            = [%q]
  account_server_url      = "nats://localhost:4222"
  operator_service_urls   = ["nats://localhost:4222"]
  system_account          = %q
  strict_signing_key_usage = true
  tags                    = ["env:test"]
}
`, seed, sigPub, sysPub)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.natsjwt_operator.test", "jwt"),
					testCheckJWTField("data.natsjwt_operator.test", func(jwtStr string) error {
						claims, err := natsjwt.DecodeOperatorClaims(jwtStr)
						if err != nil {
							return fmt.Errorf("failed to decode operator JWT: %w", err)
						}
						if claims.Name != "full-op" {
							return fmt.Errorf("expected name 'full-op', got %q", claims.Name)
						}
						if claims.AccountServerURL != "nats://localhost:4222" {
							return fmt.Errorf("expected account_server_url, got %q", claims.AccountServerURL)
						}
						if !claims.StrictSigningKeyUsage {
							return fmt.Errorf("expected strict_signing_key_usage to be true")
						}
						if claims.SystemAccount != sysPub {
							return fmt.Errorf("expected system_account %q, got %q", sysPub, claims.SystemAccount)
						}
						return nil
					}),
				),
			},
		},
	})
}

func TestAccOperatorDataSource_WrongSeedType(t *testing.T) {
	// Use an account seed instead of operator seed
	kp, err := nkeys.CreatePair(nkeys.PrefixByteAccount)
	if err != nil {
		t.Fatal(err)
	}
	seed, _ := kp.Seed()

	config := fmt.Sprintf(`
data "natsjwt_operator" "test" {
  name = "test-op"
  seed = %q
}
`, string(seed))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      config,
				ExpectError: regexp.MustCompile(`Wrong NKey Seed Type|Expected operator seed`),
			},
		},
	})
}

func TestAccOperatorDataSource_Stability(t *testing.T) {
	seed := testOperatorSeed(t)
	config := fmt.Sprintf(`
data "natsjwt_operator" "test" {
  name = "stable-op"
  seed = %q
}
`, seed)

	var firstJWT string
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					captureJWT("data.natsjwt_operator.test", &firstJWT),
				),
			},
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					compareJWT("data.natsjwt_operator.test", &firstJWT),
				),
			},
		},
	})
}

// Helper to capture JWT value from state
func captureJWT(resourceName string, target *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}
		*target = rs.Primary.Attributes["jwt"]
		return nil
	}
}

// Helper to compare JWT value with previously captured
func compareJWT(resourceName string, expected *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}
		if rs.Primary.Attributes["jwt"] != *expected {
			return fmt.Errorf("JWT changed between reads")
		}
		return nil
	}
}

// Helper to decode and check JWT fields
func testCheckJWTField(resourceName string, check func(string) error) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}
		jwtStr := rs.Primary.Attributes["jwt"]
		if jwtStr == "" {
			return fmt.Errorf("jwt attribute is empty")
		}
		return check(jwtStr)
	}
}
