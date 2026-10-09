package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
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

func TestAccTicketStatusResource_deleteMovesTicketsToTarget(t *testing.T) {
	srv, fake := newFakeRavennaWithState(t)
	var retiredID, fallbackID string

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testStatusConfigWithDeleteTarget(srv.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"ravenna_ticket_status.retired", "delete_target_status_id",
						"ravenna_ticket_status.fallback", "id",
					),
					func(s *terraform.State) error {
						retiredID = s.RootModule().Resources["ravenna_ticket_status.retired"].Primary.ID
						fallbackID = s.RootModule().Resources["ravenna_ticket_status.fallback"].Primary.ID
						return nil
					},
				),
			},
			{
				ResourceName:            "ravenna_ticket_status.retired",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"delete_target_status_id"},
			},
		},
		CheckDestroy: func(_ *terraform.State) error {
			fake.mu.Lock()
			defer fake.mu.Unlock()
			if got := fake.deleteTargets[retiredID]; got != fallbackID {
				return fmt.Errorf("status %s deleted with targetStatusId %q, want %q", retiredID, got, fallbackID)
			}
			if got, ok := fake.deleteTargets[fallbackID]; ok {
				return fmt.Errorf("status %s deleted with targetStatusId %q, want none", fallbackID, got)
			}
			return nil
		},
	})
}

func testStatusConfigWithDeleteTarget(baseURL string) string {
	return fmt.Sprintf(`
provider "ravenna" {
  api_token = "test-token"
  base_url  = %[1]q
}

data "ravenna_status_group" "pending" {
  label = "Pending"
}

resource "ravenna_ticket_status" "fallback" {
  label           = "Waiting"
  status_group_id = data.ravenna_status_group.pending.id
}

resource "ravenna_ticket_status" "retired" {
  label                   = "Waiting on vendor"
  status_group_id         = data.ravenna_status_group.pending.id
  delete_target_status_id = ravenna_ticket_status.fallback.id
}
`, baseURL)
}
