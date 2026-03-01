package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	natsjwt "github.com/nats-io/jwt/v2"
	"github.com/nats-io/nkeys"
)

func testAccountSeed(t *testing.T) string {
	t.Helper()
	kp, err := nkeys.CreatePair(nkeys.PrefixByteAccount)
	if err != nil {
		t.Fatal(err)
	}
	seed, err := kp.Seed()
	if err != nil {
		t.Fatal(err)
	}
	return string(seed)
}

func TestAccAccountDataSource_Basic(t *testing.T) {
	opSeed := testOperatorSeed(t)
	acctSeed := testAccountSeed(t)

	config := fmt.Sprintf(`
data "natsjwt_account" "test" {
  name          = "test-acct"
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
					resource.TestCheckResourceAttrSet("data.natsjwt_account.test", "jwt"),
					resource.TestMatchResourceAttr("data.natsjwt_account.test", "public_key", regexp.MustCompile(`^A`)),
					testCheckJWTField("data.natsjwt_account.test", func(jwtStr string) error {
						claims, err := natsjwt.DecodeAccountClaims(jwtStr)
						if err != nil {
							return fmt.Errorf("failed to decode account JWT: %w", err)
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

func TestAccAccountDataSource_JetStreamGlobal(t *testing.T) {
	opSeed := testOperatorSeed(t)
	acctSeed := testAccountSeed(t)

	config := fmt.Sprintf(`
data "natsjwt_account" "test" {
  name          = "js-acct"
  seed          = %q
  operator_seed = %q
  jetstream_limits = [{
    mem_storage  = 1073741824
    disk_storage = 10737418240
    streams      = 10
    consumer     = 100
  }]
}
`, acctSeed, opSeed)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckJWTField("data.natsjwt_account.test", func(jwtStr string) error {
						claims, err := natsjwt.DecodeAccountClaims(jwtStr)
						if err != nil {
							return fmt.Errorf("failed to decode account JWT: %w", err)
						}
						if claims.Limits.MemoryStorage != 1073741824 {
							return fmt.Errorf("expected mem_storage 1073741824, got %d", claims.Limits.MemoryStorage)
						}
						if claims.Limits.DiskStorage != 10737418240 {
							return fmt.Errorf("expected disk_storage 10737418240, got %d", claims.Limits.DiskStorage)
						}
						if claims.Limits.Streams != 10 {
							return fmt.Errorf("expected streams 10, got %d", claims.Limits.Streams)
						}
						return nil
					}),
				),
			},
		},
	})
}

func TestAccAccountDataSource_JetStreamTiered(t *testing.T) {
	opSeed := testOperatorSeed(t)
	acctSeed := testAccountSeed(t)

	config := fmt.Sprintf(`
data "natsjwt_account" "test" {
  name          = "tiered-acct"
  seed          = %q
  operator_seed = %q
  jetstream_limits = [
    {
      tier         = "R1"
      mem_storage  = 1073741824
      disk_storage = 5368709120
      streams      = 5
      consumer     = 50
    },
    {
      tier         = "R3"
      mem_storage  = 2147483648
      disk_storage = 10737418240
      streams      = 10
      consumer     = 100
    }
  ]
}
`, acctSeed, opSeed)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckJWTField("data.natsjwt_account.test", func(jwtStr string) error {
						claims, err := natsjwt.DecodeAccountClaims(jwtStr)
						if err != nil {
							return fmt.Errorf("failed to decode account JWT: %w", err)
						}
						r1, ok := claims.Limits.JetStreamTieredLimits["R1"]
						if !ok {
							return fmt.Errorf("R1 tier not found")
						}
						if r1.MemoryStorage != 1073741824 {
							return fmt.Errorf("R1 mem_storage mismatch")
						}
						r3, ok := claims.Limits.JetStreamTieredLimits["R3"]
						if !ok {
							return fmt.Errorf("R3 tier not found")
						}
						if r3.DiskStorage != 10737418240 {
							return fmt.Errorf("R3 disk_storage mismatch")
						}
						return nil
					}),
				),
			},
		},
	})
}

func TestAccAccountDataSource_DefaultPermissions(t *testing.T) {
	opSeed := testOperatorSeed(t)
	acctSeed := testAccountSeed(t)

	config := fmt.Sprintf(`
data "natsjwt_account" "test" {
  name          = "perm-acct"
  seed          = %q
  operator_seed = %q
  issued_at     = 123
  expires       = 456
  default_permissions = {
    pub_allow = ["orders.>"]
    pub_deny  = ["admin.>"]
    sub_allow = ["_INBOX.>"]
  }
}
`, acctSeed, opSeed)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckJWTField("data.natsjwt_account.test", func(jwtStr string) error {
						claims, err := natsjwt.DecodeAccountClaims(jwtStr)
						if err != nil {
							return fmt.Errorf("failed to decode account JWT: %w", err)
						}
						if len(claims.DefaultPermissions.Pub.Allow) != 1 || claims.DefaultPermissions.Pub.Allow[0] != "orders.>" {
							return fmt.Errorf("pub_allow mismatch: %v", claims.DefaultPermissions.Pub.Allow)
						}
						if len(claims.DefaultPermissions.Pub.Deny) != 1 || claims.DefaultPermissions.Pub.Deny[0] != "admin.>" {
							return fmt.Errorf("pub_deny mismatch: %v", claims.DefaultPermissions.Pub.Deny)
						}
						if claims.IssuedAt != 123 {
							return fmt.Errorf("expected issued_at 123, got %d", claims.IssuedAt)
						}
						if claims.Expires != 456 {
							return fmt.Errorf("expected expires 456, got %d", claims.Expires)
						}
						if claims.NotBefore != 123 {
							return fmt.Errorf("expected not_before to default to issued_at (123), got %d", claims.NotBefore)
						}
						return nil
					}),
				),
			},
		},
	})
}

func TestAccAccountDataSource_WrongSeedType(t *testing.T) {
	opSeed := testOperatorSeed(t)
	userKP, _ := nkeys.CreatePair(nkeys.PrefixByteUser)
	userSeed, _ := userKP.Seed()

	config := fmt.Sprintf(`
data "natsjwt_account" "test" {
  name          = "bad-acct"
  seed          = %q
  operator_seed = %q
}
`, string(userSeed), opSeed)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      config,
				ExpectError: regexp.MustCompile(`Wrong NKey Seed Type|Expected account seed`),
			},
		},
	})
}

func TestAccAccountDataSource_Stability(t *testing.T) {
	opSeed := testOperatorSeed(t)
	acctSeed := testAccountSeed(t)

	config := fmt.Sprintf(`
data "natsjwt_account" "test" {
  name          = "stable-acct"
  seed          = %q
  operator_seed = %q
}
`, acctSeed, opSeed)

	var firstJWT string
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check:  captureJWT("data.natsjwt_account.test", &firstJWT),
			},
			{
				Config: config,
				Check:  compareJWT("data.natsjwt_account.test", &firstJWT),
			},
		},
	})
}
