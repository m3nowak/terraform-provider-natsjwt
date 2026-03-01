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

func testUserSeed(t *testing.T) string {
	t.Helper()
	kp, err := nkeys.CreatePair(nkeys.PrefixByteUser)
	if err != nil {
		t.Fatal(err)
	}
	seed, err := kp.Seed()
	if err != nil {
		t.Fatal(err)
	}
	return string(seed)
}

func TestAccUserDataSource_Basic(t *testing.T) {
	acctSeed := testAccountSeed(t)
	userSeed := testUserSeed(t)

	config := fmt.Sprintf(`
data "natsjwt_user" "test" {
  name         = "test-user"
  seed         = %q
  account_seed = %q
}
`, userSeed, acctSeed)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.natsjwt_user.test", "jwt"),
					resource.TestCheckResourceAttrSet("data.natsjwt_user.test", "creds"),
					resource.TestMatchResourceAttr("data.natsjwt_user.test", "public_key", regexp.MustCompile(`^U`)),
					testCheckUserCredsConsistency("data.natsjwt_user.test", userSeed),
					testCheckJWTField("data.natsjwt_user.test", func(jwtStr string) error {
						claims, err := natsjwt.DecodeUserClaims(jwtStr)
						if err != nil {
							return fmt.Errorf("failed to decode user JWT: %w", err)
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

func TestAccUserDataSource_Permissions(t *testing.T) {
	acctSeed := testAccountSeed(t)
	userSeed := testUserSeed(t)

	config := fmt.Sprintf(`
data "natsjwt_user" "test" {
  name         = "perm-user"
  seed         = %q
  account_seed = %q
  permissions = {
    pub_allow = ["orders.>", "events.>"]
    pub_deny  = ["admin.>"]
    sub_allow = ["_INBOX.>"]
    sub_deny  = ["secret.>"]
  }
}
`, userSeed, acctSeed)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckJWTField("data.natsjwt_user.test", func(jwtStr string) error {
						claims, err := natsjwt.DecodeUserClaims(jwtStr)
						if err != nil {
							return fmt.Errorf("failed to decode user JWT: %w", err)
						}
						if len(claims.Pub.Allow) != 2 {
							return fmt.Errorf("expected 2 pub_allow, got %d", len(claims.Pub.Allow))
						}
						if len(claims.Pub.Deny) != 1 {
							return fmt.Errorf("expected 1 pub_deny, got %d", len(claims.Pub.Deny))
						}
						if len(claims.Sub.Allow) != 1 {
							return fmt.Errorf("expected 1 sub_allow, got %d", len(claims.Sub.Allow))
						}
						if len(claims.Sub.Deny) != 1 {
							return fmt.Errorf("expected 1 sub_deny, got %d", len(claims.Sub.Deny))
						}
						return nil
					}),
				),
			},
		},
	})
}

func TestAccUserDataSource_ConnectionTypes(t *testing.T) {
	acctSeed := testAccountSeed(t)
	userSeed := testUserSeed(t)

	config := fmt.Sprintf(`
data "natsjwt_user" "test" {
  name                     = "conn-user"
  seed                     = %q
  account_seed             = %q
  allowed_connection_types = ["STANDARD", "WEBSOCKET"]
}
`, userSeed, acctSeed)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckJWTField("data.natsjwt_user.test", func(jwtStr string) error {
						claims, err := natsjwt.DecodeUserClaims(jwtStr)
						if err != nil {
							return fmt.Errorf("failed to decode user JWT: %w", err)
						}
						if len(claims.AllowedConnectionTypes) != 2 {
							return fmt.Errorf("expected 2 connection types, got %d", len(claims.AllowedConnectionTypes))
						}
						return nil
					}),
				),
			},
		},
	})
}

func TestAccUserDataSource_TimeRestrictions(t *testing.T) {
	acctSeed := testAccountSeed(t)
	userSeed := testUserSeed(t)

	config := fmt.Sprintf(`
data "natsjwt_user" "test" {
  name         = "time-user"
  seed         = %q
  account_seed = %q
  issued_at    = 10
  not_before   = 15
  expires      = 20
  time_restrictions = [{
    start = "08:00:00"
    end   = "17:00:00"
  }]
  locale = "America/New_York"
}
`, userSeed, acctSeed)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckJWTField("data.natsjwt_user.test", func(jwtStr string) error {
						claims, err := natsjwt.DecodeUserClaims(jwtStr)
						if err != nil {
							return fmt.Errorf("failed to decode user JWT: %w", err)
						}
						if len(claims.Times) != 1 {
							return fmt.Errorf("expected 1 time restriction, got %d", len(claims.Times))
						}
						if claims.Times[0].Start != "08:00:00" || claims.Times[0].End != "17:00:00" {
							return fmt.Errorf("time restriction mismatch: %+v", claims.Times[0])
						}
						if claims.Locale != "America/New_York" {
							return fmt.Errorf("expected locale America/New_York, got %q", claims.Locale)
						}
						if claims.IssuedAt != 10 {
							return fmt.Errorf("expected issued_at 10, got %d", claims.IssuedAt)
						}
						if claims.NotBefore != 15 {
							return fmt.Errorf("expected not_before 15, got %d", claims.NotBefore)
						}
						if claims.Expires != 20 {
							return fmt.Errorf("expected expires 20, got %d", claims.Expires)
						}
						return nil
					}),
				),
			},
		},
	})
}

func TestAccUserDataSource_WrongSeedType(t *testing.T) {
	acctSeed := testAccountSeed(t)

	config := fmt.Sprintf(`
data "natsjwt_user" "test" {
  name         = "bad-user"
  seed         = %q
  account_seed = %q
}
`, acctSeed, acctSeed)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      config,
				ExpectError: regexp.MustCompile(`Wrong NKey Seed Type|Expected user seed`),
			},
		},
	})
}

func TestAccUserDataSource_SourceNetworks(t *testing.T) {
	acctSeed := testAccountSeed(t)
	userSeed := testUserSeed(t)

	config := fmt.Sprintf(`
data "natsjwt_user" "test" {
  name            = "net-user"
  seed            = %q
  account_seed    = %q
  source_networks = ["10.0.0.0/8", "192.168.0.0/16"]
}
`, userSeed, acctSeed)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckJWTField("data.natsjwt_user.test", func(jwtStr string) error {
						claims, err := natsjwt.DecodeUserClaims(jwtStr)
						if err != nil {
							return fmt.Errorf("failed to decode user JWT: %w", err)
						}
						if len(claims.Src) != 2 {
							return fmt.Errorf("expected 2 source networks, got %d", len(claims.Src))
						}
						return nil
					}),
				),
			},
		},
	})
}

func testCheckUserCredsConsistency(resourceName, expectedSeed string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}
		creds := rs.Primary.Attributes["creds"]
		if creds == "" {
			return fmt.Errorf("creds attribute is empty")
		}

		parsedJWT, err := natsjwt.ParseDecoratedJWT([]byte(creds))
		if err != nil {
			return fmt.Errorf("failed to parse decorated JWT from creds: %w", err)
		}
		if parsedJWT != rs.Primary.Attributes["jwt"] {
			return fmt.Errorf("creds JWT does not match jwt attribute")
		}

		kp, err := natsjwt.ParseDecoratedUserNKey([]byte(creds))
		if err != nil {
			return fmt.Errorf("failed to parse decorated user nkey from creds: %w", err)
		}
		seed, err := kp.Seed()
		if err != nil {
			return fmt.Errorf("failed to read seed from creds: %w", err)
		}
		if string(seed) != expectedSeed {
			return fmt.Errorf("creds seed mismatch")
		}
		publicKey, err := kp.PublicKey()
		if err != nil {
			return fmt.Errorf("failed to read public key from creds seed: %w", err)
		}
		if publicKey != rs.Primary.Attributes["public_key"] {
			return fmt.Errorf("creds seed public key mismatch")
		}
		return nil
	}
}
