package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccTagResource_lifecycle(t *testing.T) {
	srv := newFakeRavenna(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testTagConfig(srv.URL, "hardware", "blue", "Physical kit"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ravenna_tag.test", "name", "hardware"),
					resource.TestCheckResourceAttr("ravenna_tag.test", "color", "blue"),
					resource.TestCheckResourceAttr("ravenna_tag.test", "description", "Physical kit"),
					resource.TestCheckResourceAttrSet("ravenna_tag.test", "id"),
				),
			},
			{
				Config: testTagConfig(srv.URL, "hardware", "green", "Physical kit"),
				Check:  resource.TestCheckResourceAttr("ravenna_tag.test", "color", "green"),
			},
			{
				ResourceName:      "ravenna_tag.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccTagResource_rejectsUnknownColor(t *testing.T) {
	srv := newFakeRavenna(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testTagConfig(srv.URL, "bad", "chartreuse", "nope"),
				ExpectError: regexp.MustCompile(`Attribute color value must be one of`),
			},
		},
	})
}

func TestAccTagResource_clearsDescription(t *testing.T) {
	srv := newFakeRavenna(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testTagConfig(srv.URL, "hardware", "blue", "Physical kit"),
				Check:  resource.TestCheckResourceAttr("ravenna_tag.test", "description", "Physical kit"),
			},
			{
				Config: testTagConfigNoDescription(srv.URL, "hardware", "blue"),
				Check:  resource.TestCheckNoResourceAttr("ravenna_tag.test", "description"),
			},
		},
	})
}

func testTagConfig(baseURL, name, color, description string) string {
	return fmt.Sprintf(`
provider "ravenna" {
  api_token = "test-token"
  base_url  = %[1]q
}

resource "ravenna_tag" "test" {
  name        = %[2]q
  color       = %[3]q
  description = %[4]q
}
`, baseURL, name, color, description)
}

func testTagConfigNoDescription(baseURL, name, color string) string {
	return fmt.Sprintf(`
provider "ravenna" {
  api_token = "test-token"
  base_url  = %[1]q
}

resource "ravenna_tag" "test" {
  name  = %[2]q
  color = %[3]q
}
`, baseURL, name, color)
}
