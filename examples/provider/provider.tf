terraform {
  required_providers {
    ravenna = {
      source = "ravennahq/ravenna"
    }
  }
}

# The API token is read from the RAVENNA_API_TOKEN environment variable.
# Never commit a token to version control.
provider "ravenna" {
  workspace_id = var.ravenna_workspace_id
}
