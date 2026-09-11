// Package provider implements the Terraform provider for Ravenna.
package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RavennaHQ/terraform-provider-ravenna/internal/ravenna"
)

const (
	envAPIToken  = "RAVENNA_API_TOKEN"
	envWorkspace = "RAVENNA_WORKSPACE_ID"
	envBaseURL   = "RAVENNA_BASE_URL"
)

// providerData is handed to every resource and data source via Configure.
type providerData struct {
	Client *ravenna.Client
	// WorkspaceID is the provider-level default. Resources override it with
	// their own workspace_id attribute when set.
	WorkspaceID string
}

type ravennaProvider struct {
	version string
}

// New returns a provider factory for the given build version.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &ravennaProvider{version: version}
	}
}

type providerModel struct {
	APIToken    types.String `tfsdk:"api_token"`
	WorkspaceID types.String `tfsdk:"workspace_id"`
	BaseURL     types.String `tfsdk:"base_url"`
}

func (p *ravennaProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "ravenna"
	resp.Version = p.version
}

func (p *ravennaProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage Ravenna service management configuration.",
		Attributes: map[string]schema.Attribute{
			"api_token": schema.StringAttribute{
				MarkdownDescription: "Ravenna API token. Prefer the `" + envAPIToken +
					"` environment variable over setting this in configuration.",
				Optional:  true,
				Sensitive: true,
			},
			"workspace_id": schema.StringAttribute{
				MarkdownDescription: "Default workspace for resources that are workspace-scoped. " +
					"May also be set with `" + envWorkspace + "`. Individual resources can override it.",
				Optional: true,
			},
			"base_url": schema.StringAttribute{
				MarkdownDescription: "Ravenna API base URL. Defaults to `" + ravenna.DefaultBaseURL +
					"`. May also be set with `" + envBaseURL + "`.",
				Optional: true,
			},
		},
	}
}

func (p *ravennaProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var cfg providerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if cfg.APIToken.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_token"),
			"Unknown Ravenna API token",
			"The API token cannot be a value that is unknown at plan time. Set it "+
				"statically or export "+envAPIToken+".",
		)
		return
	}

	token := firstNonEmpty(cfg.APIToken.ValueString(), os.Getenv(envAPIToken))
	workspaceID := firstNonEmpty(cfg.WorkspaceID.ValueString(), os.Getenv(envWorkspace))
	baseURL := firstNonEmpty(cfg.BaseURL.ValueString(), os.Getenv(envBaseURL), ravenna.DefaultBaseURL)

	if token == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_token"),
			"Missing Ravenna API token",
			"Set the api_token provider attribute or export "+envAPIToken+".",
		)
		return
	}

	client, err := ravenna.New(baseURL, token, ravenna.WithUserAgent("terraform-provider-ravenna/"+p.version))
	if err != nil {
		resp.Diagnostics.AddError("Unable to create Ravenna client", err.Error())
		return
	}

	data := &providerData{Client: client, WorkspaceID: workspaceID}
	resp.ResourceData = data
	resp.DataSourceData = data
}

func (p *ravennaProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewChannelResource,
		NewTagResource,
	}
}

func (p *ravennaProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
