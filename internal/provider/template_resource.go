// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kenzo0107/sendgrid"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &templateResource{}
var _ resource.ResourceWithImportState = &templateResource{}

func newTemplateResource() resource.Resource {
	return &templateResource{}
}

type templateResource struct {
	client *sendgrid.Client
}

type templateResourceModel struct {
	ID         types.String `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	Generation types.String `tfsdk:"generation"`
}

func (r *templateResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_template"
}

func (r *templateResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
Provides template resource.

An HTML template that can establish a consistent design for [transactional emails](https://sendgrid.com/use-cases/transactional-email/).

Each parent account, as well as each Subuser, can create up to 300 different transactional templates. Templates are specific to the parent account or Subuser, meaning templates created on a parent account will not be accessible from the parent's Subuser accounts.

Transactional templates are templates created specifically for transactional email and are not to be confused with [Marketing Campaigns designs](https://sendgrid.com/docs/ui/sending-email/working-with-marketing-campaigns-email-designs/). For more information about transactional templates, please see our [Dynamic Transactional Templates documentation](https://sendgrid.com/docs/ui/sending-email/how-to-send-an-email-with-dynamic-transactional-templates/).
		`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the transactional template.",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name for the transactional template. maxLength: 100",
				Required:            true,
			},
			"generation": schema.StringAttribute{
				MarkdownDescription: "Defines the generation of the template. Allowed Values: `legacy`, `dynamic`",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringOneOf("legacy", "dynamic"),
				},
			},
		},
	}
}

func (r *templateResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*sendgrid.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *sendgrid.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *templateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan templateResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	o, err := r.client.CreateTemplate(ctx, &sendgrid.InputCreateTemplate{
		Name:       plan.Name.ValueString(),
		Generation: plan.Generation.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Creating template",
			fmt.Sprintf("Unable to create template, got error: %s", err),
		)
		return
	}

	plan = templateResourceModel{
		ID:         types.StringValue(o.ID),
		Name:       types.StringValue(o.Name),
		Generation: types.StringValue(o.Generation),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *templateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state templateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	o, err := r.client.GetTemplate(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Reading template",
			fmt.Sprintf("Unable to read template (id: %v), got error: %s", id, err),
		)
		return
	}

	state = templateResourceModel{
		ID:         types.StringValue(o.ID),
		Name:       types.StringValue(o.Name),
		Generation: types.StringValue(o.Generation),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *templateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, state templateResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	o, err := r.client.UpdateTemplate(ctx, id, &sendgrid.InputUpdateTemplate{
		Name: data.Name.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Updating template",
			fmt.Sprintf("Unable to update template (id: %v), got error: %s", id, err),
		)
		return
	}

	data = templateResourceModel{
		ID:         state.ID,
		Name:       types.StringValue(o.Name),
		Generation: types.StringValue(o.Generation),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *templateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state templateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	if err := r.client.DeleteTemplate(ctx, id); err != nil {
		resp.Diagnostics.AddError(
			"Deleting template",
			fmt.Sprintf("Unable to delete template (id: %v), got error: %s", id, err),
		)
		return
	}
}

func (r *templateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var data templateResourceModel

	id := req.ID
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)

	o, err := r.client.GetTemplate(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Importing template",
			fmt.Sprintf("Unable to read template, got error: %s", err),
		)
		return
	}

	data = templateResourceModel{
		ID:         types.StringValue(o.ID),
		Name:       types.StringValue(o.Name),
		Generation: types.StringValue(o.Generation),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}
