package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccTicketStatusResource_lifecycle(t *testing.T) {
	srv := newFakeRavenna(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testStatusConfig(srv.URL, "Waiting on vendor"),
				Check: resource.ComposeAggregateTestCheckFunc(
					// The data source must resolve the group by label, since
					// status groups cannot be created through the API.
					resource.TestCheckResourceAttr("data.ravenna_status_group.pending", "id", "sg_pending"),
					resource.TestCheckResourceAttr("data.ravenna_status_group.pending", "color", "amber"),
					resource.TestCheckResourceAttr("ravenna_ticket_status.test", "label", "Waiting on vendor"),
					resource.TestCheckResourceAttr("ravenna_ticket_status.test", "status_group_id", "sg_pending"),
					resource.TestCheckResourceAttr("ravenna_ticket_status.test", "system", "false"),
					resource.TestCheckResourceAttrSet("ravenna_ticket_status.test", "id"),
				),
			},
			{
				Config: testStatusConfig(srv.URL, "Waiting on supplier"),
				Check:  resource.TestCheckResourceAttr("ravenna_ticket_status.test", "label", "Waiting on supplier"),
			},
			{
				ResourceName:      "ravenna_ticket_status.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testStatusConfig(baseURL, label string) string {
	return fmt.Sprintf(`
provider "ravenna" {
  api_token = "test-token"
  base_url  = %[1]q
}

data "ravenna_status_group" "pending" {
  label = "Pending"
}

resource "ravenna_ticket_status" "test" {
  label           = %[2]q
  status_group_id = data.ravenna_status_group.pending.id
}
`, baseURL, label)
}

func TestAccTicketStatusResource_honoursExplicitOrder(t *testing.T) {
	srv := newFakeRavenna(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testStatusConfigWithOrder(srv.URL, "Waiting on vendor", 7),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ravenna_ticket_status.test", "order", "7"),
					resource.TestCheckResourceAttr("ravenna_ticket_status.test", "label", "Waiting on vendor"),
				),
			},
		},
	})
}

func testStatusConfigWithOrder(baseURL, label string, order int) string {
	return fmt.Sprintf(`
provider "ravenna" {
  api_token = "test-token"
  base_url  = %[1]q
}

data "ravenna_status_group" "pending" {
  label = "Pending"
}

resource "ravenna_ticket_status" "test" {
  label           = %[2]q
  status_group_id = data.ravenna_status_group.pending.id
  order           = %[3]d
}
`, baseURL, label, order)
}
