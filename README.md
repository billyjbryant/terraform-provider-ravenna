# Terraform Provider for Ravenna

Manage [Ravenna](https://ravenna.ai) service management configuration as code.

## Status

Early development. Currently manages channels, tags, and ticket statuses.
Categories, custom fields, request types, snippets, business schedules,
applications, access levels, and access policies are in progress.

This provider is built for the Ravenna team and will move to Ravenna's GitHub
organisation. It is not yet published to the Terraform Registry; it will be
published from its permanent home under the `ravennahq` namespace, so early
adopters never have to migrate provider addresses.

## Installation

Until it is published, build the provider and point Terraform at the binary
with a [development override][dev-overrides] in `~/.terraformrc`:

```shell
go install github.com/billyjbryant/terraform-provider-ravenna@latest
```

```hcl
provider_installation {
  dev_overrides {
    "billyjbryant/ravenna" = "/path/to/your/GOBIN"
  }
  direct {}
}
```

```hcl
terraform {
  required_providers {
    ravenna = {
      source = "billyjbryant/ravenna"
    }
  }
}
```

With a development override in place, skip `terraform init` and run
`terraform plan` directly.

[dev-overrides]: https://developer.hashicorp.com/terraform/cli/config/config-file#development-overrides-for-provider-developers

## Authentication

The provider reads your API token from the `RAVENNA_API_TOKEN` environment
variable:

```shell
export RAVENNA_API_TOKEN="..."
```

**Never commit a token to version control**, and prefer a secrets manager or
your CI provider's secret store over a `.tfvars` file. Ravenna API keys support
CIDR restriction — scope the key used by CI to your runners' egress addresses.

Setting `api_token` directly in provider configuration is supported but
discouraged, because the value lands in plan output and state.

## Example

```hcl
provider "ravenna" {
  workspace_id = var.ravenna_workspace_id
}

resource "ravenna_channel" "it_helpdesk" {
  name   = "IT Helpdesk"
  prefix = "IT"
  emoji  = "🎧"
}

resource "ravenna_tag" "hardware" {
  name        = "hardware"
  color       = "blue"
  description = "Requests involving physical equipment"
}

data "ravenna_status_group" "pending" {
  label = "Pending"
}

resource "ravenna_ticket_status" "waiting_on_vendor" {
  label           = "Waiting on vendor"
  status_group_id = data.ravenna_status_group.pending.id
}
```

## Adopting an existing workspace

Every resource supports import, so an existing Ravenna workspace can be brought
under Terraform without recreating anything:

```shell
terraform import ravenna_channel.it_helpdesk q_01HXYZ
```

## Destroy behaviour

Resources that Ravenna can archive — applications, access levels, access
policies, and request types — will expose `archive_on_destroy`, defaulting to
`true`. Archiving is reversible; deletion is not, and many Ravenna
installations have only a production workspace. Set `archive_on_destroy = false`
for conventional destroy semantics.

Resources with no archive support are deleted outright. Use
`lifecycle { prevent_destroy = true }` on anything load-bearing.

## Development

```shell
mise install    # Go, Terraform, linters, goreleaser
make build
make test       # needs no Ravenna credentials
make lint
make docs
```

Tests run against an in-memory fake Ravenna server, so the full suite passes
offline and in CI without a token. Acceptance tests against a live workspace are
gated behind `TF_ACC=1`, `RAVENNA_API_TOKEN`, and `RAVENNA_ACC_ALLOW_PROD=1`;
they create and destroy real objects and are never run in CI.

`make spec-diff` compares the vendored `api/openapi.json` against the live
Ravenna spec and reports changes to endpoints this provider depends on.

## License

Apache-2.0. See [LICENSE](LICENSE).
