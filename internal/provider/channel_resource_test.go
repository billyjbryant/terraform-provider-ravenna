package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccChannelResource_lifecycle(t *testing.T) {
	srv := newFakeRavenna(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testChannelConfig(srv.URL, "IT Helpdesk", "IT", "🎧"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ravenna_channel.test", "name", "IT Helpdesk"),
					resource.TestCheckResourceAttr("ravenna_channel.test", "prefix", "IT"),
					resource.TestCheckResourceAttr("ravenna_channel.test", "emoji", "🎧"),
					resource.TestCheckResourceAttr("ravenna_channel.test", "type", "DEFAULT"),
					resource.TestCheckResourceAttr("ravenna_channel.test", "system", "false"),
					resource.TestCheckResourceAttrSet("ravenna_channel.test", "id"),
				),
			},
			{
				Config: testChannelConfig(srv.URL, "IT Support", "IT", "🎧"),
				Check:  resource.TestCheckResourceAttr("ravenna_channel.test", "name", "IT Support"),
			},
			{
				ResourceName:      "ravenna_channel.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccChannelResource_invalidPrefix(t *testing.T) {
	srv := newFakeRavenna(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testChannelConfig(srv.URL, "Bad", "lowercase", "🎧"),
				ExpectError: regexp.MustCompile(`must start with an uppercase letter`),
			},
		},
	})
}

func testChannelConfig(baseURL, name, prefix, emoji string) string {
	return fmt.Sprintf(`
provider "ravenna" {
  api_token = "test-token"
  base_url  = %[1]q
}

resource "ravenna_channel" "test" {
  name   = %[2]q
  prefix = %[3]q
  emoji  = %[4]q
}
`, baseURL, name, prefix, emoji)
}
