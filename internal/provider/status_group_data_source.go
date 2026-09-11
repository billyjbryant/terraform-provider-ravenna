package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &statusGroupDataSource{}
	_ datasource.DataSourceWithConfigure = &statusGroupDataSource{}
)

// NewStatusGroupDataSource returns the ravenna_status_group data source.
// Status groups have no create, update or delete endpoint, so looking one up
// by label is the only way to reference it from configuration.
func NewStatusGroupDataSource() datasource.DataSource {
	return &statusGroupDataSource{}
}

type statusGroupDataSource struct {
	data *providerData
}

type statusGroupDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Label       types.String `tfsdk:"label"`
	Order       types.Int64  `tfsdk:"order"`
	Color       types.String `tfsdk:"color"`
	WorkspaceID types.String `tfsdk:"workspace_id"`
}

func (d *statusGroupDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_status_group"
}

func (d *statusGroupDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Look up a Ravenna status group by label. Status groups are " +
			"managed by Ravenna and cannot be created, changed or deleted through the API, " +
			"but `ravenna_ticket_status` requires one.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Status group identifier.",
				Computed:            true,
			},
			"label": schema.StringAttribute{
				MarkdownDescription: "Status group label to look up, for example `Open` or `Pending`.",
				Required:            true,
			},
			"order": schema.Int64Attribute{
				MarkdownDescription: "Display order.",
				Computed:            true,
			},
			"color": schema.StringAttribute{
				MarkdownDescription: "Status group colour.",
				Computed:            true,
			},
			"workspace_id": schema.StringAttribute{
				MarkdownDescription: "Workspace that owns the status group.",
				Computed:            true,
			},
		},
	}
}

func (d *statusGroupDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	data, ok := req.ProviderData.(*providerData)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected provider data",
			fmt.Sprintf("Expected *providerData, got %T. This is a bug in the provider.", req.ProviderData),
		)
		return
	}
	d.data = data
}

func (d *statusGroupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var cfg statusGroupDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, groups, err := d.data.Client.ListStatuses(ctx, d.data.WorkspaceID)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list status groups", err.Error())
		return
	}

	want := cfg.Label.ValueString()
	for _, g := range groups {
		if g.Label == want {
			cfg.ID = types.StringValue(g.ID)
			cfg.Order = types.Int64Value(int64(g.Order))
			cfg.Color = types.StringValue(g.Color)
			cfg.WorkspaceID = types.StringValue(g.WorkspaceID)
			resp.Diagnostics.Append(resp.State.Set(ctx, &cfg)...)
			return
		}
	}

	available := make([]string, 0, len(groups))
	for _, g := range groups {
		available = append(available, g.Label)
	}
	resp.Diagnostics.AddError(
		"Status group not found",
		fmt.Sprintf("No status group is labelled %q. Available labels: %v", want, available),
	)
}
