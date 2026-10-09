# Status groups are managed by Ravenna and cannot be created through the API.
data "ravenna_status_group" "open" {
  label = "Open"
}
