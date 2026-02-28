package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"natsjwt": providerserver.NewProtocol6WithError(New("test")()),
}

func TestAccNkeyResource_Operator(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `resource "natsjwt_nkey" "test" { type = "operator" }`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("natsjwt_nkey.test", "seed", regexp.MustCompile(`^SO`)),
					resource.TestMatchResourceAttr("natsjwt_nkey.test", "public_key", regexp.MustCompile(`^O`)),
				),
			},
		},
	})
}

func TestAccNkeyResource_Account(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `resource "natsjwt_nkey" "test" { type = "account" }`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("natsjwt_nkey.test", "seed", regexp.MustCompile(`^SA`)),
					resource.TestMatchResourceAttr("natsjwt_nkey.test", "public_key", regexp.MustCompile(`^A`)),
				),
			},
		},
	})
}

func TestAccNkeyResource_User(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `resource "natsjwt_nkey" "test" { type = "user" }`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("natsjwt_nkey.test", "seed", regexp.MustCompile(`^SU`)),
					resource.TestMatchResourceAttr("natsjwt_nkey.test", "public_key", regexp.MustCompile(`^U`)),
				),
			},
		},
	})
}

func TestAccNkeyResource_InvalidType(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      `resource "natsjwt_nkey" "test" { type = "invalid" }`,
				ExpectError: regexp.MustCompile(`Must be one of: operator, account, user`),
			},
		},
	})
}

func TestAccNkeyResource_KeepersReplacement(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "natsjwt_nkey" "test" {
  type    = "account"
  keepers = { "v" = "1" }
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("natsjwt_nkey.test", "seed", regexp.MustCompile(`^SA`)),
				),
			},
			{
				Config: `
resource "natsjwt_nkey" "test" {
  type    = "account"
  keepers = { "v" = "2" }
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("natsjwt_nkey.test", "seed", regexp.MustCompile(`^SA`)),
				),
			},
		},
	})
}
