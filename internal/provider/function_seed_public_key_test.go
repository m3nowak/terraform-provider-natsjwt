package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/nats-io/nkeys"
)

func testSeedAndPublicKey(t *testing.T) (string, string) {
	t.Helper()

	kp, err := nkeys.CreatePair(nkeys.PrefixByteAccount)
	if err != nil {
		t.Fatal(err)
	}

	seed, err := kp.Seed()
	if err != nil {
		t.Fatal(err)
	}

	publicKey, err := kp.PublicKey()
	if err != nil {
		t.Fatal(err)
	}

	return string(seed), publicKey
}

func TestAccSeedPublicKeyFunction_Basic(t *testing.T) {
	seed, expectedPublicKey := testSeedAndPublicKey(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
output "public_key" {
  value = provider::natsjwt::seed_public_key(%q)
}
`, seed),
				Check: resource.TestCheckOutput("public_key", expectedPublicKey),
			},
		},
	})
}

func TestAccSeedPublicKeyFunction_InvalidSeed(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
output "public_key" {
  value = provider::natsjwt::seed_public_key("invalid-seed")
}
`,
				ExpectError: regexp.MustCompile(`failed to convert seed to public key`),
			},
		},
	})
}
