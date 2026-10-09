package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/billyjbryant/terraform-provider-ravenna/internal/ravenna"
)

var (
	_ resource.Resource                = &ticketStatusResource{}
	_ resource.ResourceWithConfigure   = &ticketStatusResource{}
	_ resource.ResourceWithImportState = &ticketStatusResource{}
)

// NewTicketStatusResource returns the ravenna_ticket_status resource.
func NewTicketStatusResource() resource.Resource {
	return &ticketStatusResource{}
}

type ticketStatusResource struct {
	data *providerData
}

type ticketStatusResourceModel struct {
	ID                   types.String `tfsdk:"id"`
	Label                types.String `tfsdk:"label"`
	StatusGroupID        types.String `tfsdk:"status_group_id"`
	RequestTypeID        types.String `tfsdk:"request_type_id"`
	Order                types.Int64  `tfsdk:"order"`
	System               types.Bool   `tfsdk:"system"`
	DeleteTargetStatusID types.String `tfsdk:"delete_target_status_id"`
}

func (r *ticketStatusResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ticket_status"
}

func (r *ticketStatusResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A ticket status. Every status belongs to a status group, which " +
			"Ravenna manages — use the `ravenna_status_group` data source to resolve one.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Status identifier.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"label": schema.StringAttribute{
				MarkdownDescription: "Status label shown on tickets.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"status_group_id": schema.StringAttribute{
				MarkdownDescription: "Status group this status belongs to. Changing it forces a new status, " +
					"because the API accepts the group only at creation.",
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"request_type_id": schema.StringAttribute{
				MarkdownDescription: "Restrict the status to a single request type. Write-only: the Ravenna " +
					"API accepts this at creation but does not return it as a scalar, so Terraform cannot " +
					"detect drift on it and an imported status will show it as null. Changing this forces " +
					"a new status.",
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"order": schema.Int64Attribute{
				MarkdownDescription: "Display order within the status group.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"system": schema.BoolAttribute{
				MarkdownDescription: "Whether Ravenna manages this status as a system status.",
				Computed:            true,
			},
			"delete_target_status_id": schema.StringAttribute{
				MarkdownDescription: "Status to move this status's tickets onto when it is destroyed. " +
					"Only used on destroy, and Terraform destroys with the value already in state, so " +
					"apply a change to this attribute before destroying the status. Not read back " +
					"from the API; an imported status shows it as null.",
				Optional: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
		},
	}
}

func (r *ticketStatusResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ticketStatusResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ticketStatusResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	st, err := r.data.Client.CreateStatus(ctx, ravenna.StatusCreateRequest{
		Label:         plan.Label.ValueString(),
		StatusGroupID: plan.StatusGroupID.ValueString(),
		RequestTypeID: optionalString(plan.RequestTypeID),
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to create ticket status", err.Error())
		return
	}

	// POST /statuses does not accept an order, so a user-specified order is
	// applied with a follow-up update rather than being silently ignored.
	if !plan.Order.IsNull() && !plan.Order.IsUnknown() {
		wanted := int(plan.Order.ValueInt64())
		if wanted != st.Order {
			reordered, err := r.data.Client.UpdateStatus(ctx, ravenna.StatusUpdateRequest{
				ID:    st.ID,
				Order: &wanted,
			})
			if err != nil {
				// The status exists, so persist it before failing — otherwise
				// Terraform loses track of it and the next apply orphans it.
				applyStatus(&plan, st)
				resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
				resp.Diagnostics.AddError(
					"Ticket status created but could not be ordered",
					fmt.Sprintf("The status was created with id %s but setting order=%d failed: %s. "+
						"The status is now under Terraform management; re-apply to retry the ordering.",
						st.ID, wanted, err),
				)
				return
			}
			st = reordered
		}
	}

	applyStatus(&plan, st)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ticketStatusResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ticketStatusResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	st, err := r.data.Client.GetStatus(ctx, state.ID.ValueString(), r.data.WorkspaceID)
	if err != nil {
		var apiErr *ravenna.APIError
		if errors.As(err, &apiErr) && apiErr.IsNotFound() {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read ticket status", err.Error())
		return
	}

	applyStatus(&state, st)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ticketStatusResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ticketStatusResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state ticketStatusResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	label := plan.Label.ValueString()
	updateReq := ravenna.StatusUpdateRequest{
		ID:    state.ID.ValueString(),
		Label: &label,
	}
	if !plan.Order.IsNull() && !plan.Order.IsUnknown() {
		order := int(plan.Order.ValueInt64())
		updateReq.Order = &order
	}

	st, err := r.data.Client.UpdateStatus(ctx, updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update ticket status", err.Error())
		return
	}

	applyStatus(&plan, st)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ticketStatusResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ticketStatusResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.data.Client.DeleteStatus(ctx, state.ID.ValueString(), state.DeleteTargetStatusID.ValueString())
	if err != nil {
		var apiErr *ravenna.APIError
		if errors.As(err, &apiErr) && apiErr.IsNotFound() {
			return
		}
		resp.Diagnostics.AddError("Unable to delete ticket status", err.Error())
	}
}

func (r *ticketStatusResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// applyStatus copies API state onto the model. request_type_id is
// deliberately absent: GET /statuses returns a requestTypes array on each
// status, not a requestTypeId scalar, so there is nothing to read back and
// populating it here would be a fabrication. Do not add it.
func applyStatus(m *ticketStatusResourceModel, st *ravenna.Status) {
	m.ID = types.StringValue(st.ID)
	m.Label = types.StringValue(st.Label)
	m.StatusGroupID = types.StringValue(st.StatusGroupID)
	m.Order = types.Int64Value(int64(st.Order))
	m.System = types.BoolValue(st.System)
}
