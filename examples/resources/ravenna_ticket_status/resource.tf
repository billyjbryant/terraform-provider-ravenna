data "ravenna_status_group" "pending" {
  label = "Pending"
}

resource "ravenna_ticket_status" "waiting_on_vendor" {
  label           = "Waiting on vendor"
  status_group_id = data.ravenna_status_group.pending.id
}
