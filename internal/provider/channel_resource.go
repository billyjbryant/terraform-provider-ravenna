package provider

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RavennaHQ/terraform-provider-ravenna/internal/ravenna"
)

var (
	_ resource.Resource                = &channelResource{}
	_ resource.ResourceWithConfigure   = &channelResource{}
	_ resource.ResourceWithImportState = &channelResource{}
)

// channelPrefixPattern mirrors the API's own constraint on queue prefixes.
var channelPrefixPattern = regexp.MustCompile(`^[A-Z][A-Z0-9]{0,9}$`)

// NewChannelResource returns the ravenna_channel resource.
func NewChannelResource() resource.Resource {
	return &channelResource{}
}

type channelResource struct {
	data *providerData
}

type channelResourceModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	Prefix           types.String `tfsdk:"prefix"`
	Emoji            types.String `tfsdk:"emoji"`
	WorkspaceID      types.String `tfsdk:"workspace_id"`
	RequestChannelID types.String `tfsdk:"request_channel_id"`
	TriageChannelID  types.String `tfsdk:"triage_channel_id"`
	Type             types.String `tfsdk:"type"`
	System           types.Bool   `tfsdk:"system"`
}

func (r *channelResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_channel"
}

func (r *channelResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A Ravenna channel, referred to as a queue in the REST API. " +
			"Channels organise tickets.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Channel identifier.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Display name.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"prefix": schema.StringAttribute{
				MarkdownDescription: "Ticket key prefix, for example `IT`. Must start with an " +
					"uppercase letter and contain only uppercase letters and digits, up to 10 characters.",
				Optional: true,
				Computed: true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						channelPrefixPattern,
						"must start with an uppercase letter and contain only uppercase letters and digits (max 10)",
					),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"emoji": schema.StringAttribute{
				MarkdownDescription: "Emoji shown beside the channel.",
				Required:            true,
			},
			"workspace_id": schema.StringAttribute{
				MarkdownDescription: "Workspace that owns the channel. Defaults to the provider's " +
					"`workspace_id`. Changing this forces a new channel.",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"request_channel_id": schema.StringAttribute{
				MarkdownDescription: "Slack channel that requests arrive on. Changing this forces a new channel.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"triage_channel_id": schema.StringAttribute{
				MarkdownDescription: "Slack channel used for triage. Changing this forces a new channel.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "Channel type assigned by Ravenna: `DEFAULT`, `PORTAL`, `DM` or `PERSONAL_DM`.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"system": schema.BoolAttribute{
				MarkdownDescription: "Whether Ravenna manages this channel as a system channel.",
				Computed:            true,
			},
		},
	}
}

func (r *channelResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
	r.data = data
}

func (r *channelResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan channelResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := ravenna.ChannelCreateRequest{
		Name:             plan.Name.ValueString(),
		Emoji:            plan.Emoji.ValueString(),
		Prefix:           optionalString(plan.Prefix),
		RequestChannelID: optionalString(plan.RequestChannelID),
		TriageChannelID:  optionalString(plan.TriageChannelID),
	}
	if ws := r.workspaceFor(plan.WorkspaceID); ws != "" {
		createReq.WorkspaceID = &ws
	}

	ch, err := r.data.Client.CreateChannel(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create channel", err.Error())
		return
	}

	applyChannel(&plan, ch)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *channelResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state channelResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ch, err := r.data.Client.GetChannel(ctx, state.ID.ValueString())
	if err != nil {
		var apiErr *ravenna.APIError
		if errors.As(err, &apiErr) && apiErr.IsNotFound() {
			// Deleted outside Terraform: drop it so the next plan recreates it.
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read channel", err.Error())
		return
	}

	applyChannel(&state, ch)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *channelResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan channelResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state channelResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := plan.Name.ValueString()
	emoji := plan.Emoji.ValueString()
	updateReq := ravenna.ChannelUpdateRequest{
		Name:   &name,
		Emoji:  &emoji,
		Prefix: optionalString(plan.Prefix),
	}

	ch, err := r.data.Client.UpdateChannel(ctx, state.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update channel", err.Error())
		return
	}

	applyChannel(&plan, ch)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *channelResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state channelResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.data.Client.DeleteChannel(ctx, state.ID.ValueString())
	if err != nil {
		var apiErr *ravenna.APIError
		if errors.As(err, &apiErr) && apiErr.IsNotFound() {
			// Already gone; deleting is idempotent from Terraform's view.
			return
		}
		resp.Diagnostics.AddError("Unable to delete channel", err.Error())
	}
}

func (r *channelResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// workspaceFor resolves the effective workspace: the resource's own attribute
// when set, otherwise the provider default.
func (r *channelResource) workspaceFor(attr types.String) string {
	if !attr.IsNull() && !attr.IsUnknown() && attr.ValueString() != "" {
		return attr.ValueString()
	}
	return r.data.WorkspaceID
}

func applyChannel(m *channelResourceModel, ch *ravenna.Channel) {
	m.ID = types.StringValue(ch.ID)
	m.Name = types.StringValue(ch.Name)
	m.Prefix = types.StringValue(ch.Prefix)
	m.Emoji = types.StringValue(ch.Emoji)
	m.WorkspaceID = types.StringValue(ch.WorkspaceID)
	m.Type = types.StringValue(ch.Type)
	m.System = types.BoolValue(ch.System)
}

// optionalString converts a Terraform string into an optional API field,
// returning nil for null or unknown so the key is omitted from the payload.
func optionalString(v types.String) *string {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	s := v.ValueString()
	return &s
}
