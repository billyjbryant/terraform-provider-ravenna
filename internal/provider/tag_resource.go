package provider

import (
	"context"
	"errors"
	"fmt"

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
	_ resource.Resource                = &tagResource{}
	_ resource.ResourceWithConfigure   = &tagResource{}
	_ resource.ResourceWithImportState = &tagResource{}
)

// NewTagResource returns the ravenna_tag resource.
func NewTagResource() resource.Resource {
	return &tagResource{}
}

type tagResource struct {
	data *providerData
}

type tagResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Color       types.String `tfsdk:"color"`
	WorkspaceID types.String `tfsdk:"workspace_id"`
}

func (r *tagResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tag"
}

func (r *tagResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A tag applied to Ravenna tickets.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Tag identifier.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Tag name.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Optional description.",
				Optional:            true,
			},
			"color": schema.StringAttribute{
				MarkdownDescription: "Tag colour. One of the Ravenna palette values.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf(ravenna.TagColors...),
				},
			},
			"workspace_id": schema.StringAttribute{
				MarkdownDescription: "Workspace that owns the tag.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *tagResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *tagResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan tagResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tag, err := r.data.Client.CreateTag(ctx, ravenna.TagCreateRequest{
		Name:        plan.Name.ValueString(),
		Color:       plan.Color.ValueString(),
		Description: optionalString(plan.Description),
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to create tag", err.Error())
		return
	}

	applyTag(&plan, tag)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *tagResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state tagResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tag, err := r.data.Client.GetTag(ctx, state.ID.ValueString(), r.data.WorkspaceID)
	if err != nil {
		var apiErr *ravenna.APIError
		if errors.As(err, &apiErr) && apiErr.IsNotFound() {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read tag", err.Error())
		return
	}

	applyTag(&state, tag)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *tagResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan tagResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state tagResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := plan.Name.ValueString()
	color := plan.Color.ValueString()

	tag, err := r.data.Client.UpdateTag(ctx, state.ID.ValueString(), ravenna.TagUpdateRequest{
		Name:        &name,
		Color:       &color,
		Description: optionalString(plan.Description),
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to update tag", err.Error())
		return
	}

	applyTag(&plan, tag)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *tagResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state tagResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.data.Client.DeleteTag(ctx, state.ID.ValueString())
	if err != nil {
		var apiErr *ravenna.APIError
		if errors.As(err, &apiErr) && apiErr.IsNotFound() {
			return
		}
		resp.Diagnostics.AddError("Unable to delete tag", err.Error())
	}
}

func (r *tagResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func applyTag(m *tagResourceModel, tag *ravenna.Tag) {
	m.ID = types.StringValue(tag.ID)
	m.Name = types.StringValue(tag.Name)
	m.Color = types.StringValue(tag.Color)
	m.WorkspaceID = types.StringValue(tag.WorkspaceID)
	if tag.Description == nil {
		m.Description = types.StringNull()
	} else {
		m.Description = types.StringValue(*tag.Description)
	}
}
