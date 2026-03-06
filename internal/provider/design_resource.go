// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kenzo0107/sendgrid"
	"github.com/kenzo0107/terraform-provider-sendgrid/flex"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &designResource{}
var _ resource.ResourceWithImportState = &designResource{}

func newDesignResource() resource.Resource {
	return &designResource{}
}

type designResource struct {
	client *sendgrid.Client
}

type designResourceModel struct {
	ID                   types.String `tfsdk:"id"`
	Name                 types.String `tfsdk:"name"`
	Editor               types.String `tfsdk:"editor"`
	HTMLContent          types.String `tfsdk:"html_content"`
	PlainContent         types.String `tfsdk:"plain_content"`
	Subject              types.String `tfsdk:"subject"`
	Categories           types.Set    `tfsdk:"categories"`
	GeneratePlainContent types.Bool   `tfsdk:"generate_plain_content"`
	ThumbnailURL         types.String `tfsdk:"thumbnail_url"`
	UpdatedAt            types.String `tfsdk:"updated_at"`
	CreatedAt            types.String `tfsdk:"created_at"`
}

func (r *designResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_design"
}

func (r *designResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
Provides a Design resource.

Designs are reusable email layouts that can be used to create marketing campaigns and single sends.

For more detailed information, please see the [SendGrid documentation](https://docs.sendgrid.com/api-reference/designs-api/).
		`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the design.",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the design.",
				Required:            true,
			},
			"editor": schema.StringAttribute{
				MarkdownDescription: "The editor used in the UI. Allowed values: `code`, `design`.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringOneOf("code", "design"),
				},
			},
			"html_content": schema.StringAttribute{
				MarkdownDescription: "The HTML content of the design.",
				Required:            true,
			},
			"plain_content": schema.StringAttribute{
				MarkdownDescription: "The plain text content of the design. When `generate_plain_content` is `true`, this field is auto-generated from `html_content`.",
				Optional:            true,
				Computed:            true,
			},
			"subject": schema.StringAttribute{
				MarkdownDescription: "The subject line of the design.",
				Optional:            true,
			},
			"categories": schema.SetAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "The list of categories applied to the design.",
				Optional:            true,
				Computed:            true,
			},
			"generate_plain_content": schema.BoolAttribute{
				MarkdownDescription: "If `true`, `plain_content` is always generated from `html_content`. If `false`, `plain_content` is not altered.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"thumbnail_url": schema.StringAttribute{
				MarkdownDescription: "The URL of the thumbnail for the design.",
				Computed:            true,
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "The date and time the design was last updated.",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "The date and time the design was created.",
				Computed:            true,
			},
		},
	}
}

func (r *designResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *designResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan designResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := &sendgrid.InputCreateDesign{
		Name:                 plan.Name.ValueString(),
		GeneratePlainContent: plan.GeneratePlainContent.ValueBool(),
	}

	if !plan.Editor.IsNull() && !plan.Editor.IsUnknown() {
		input.Editor = plan.Editor.ValueString()
	}
	if !plan.HTMLContent.IsNull() {
		input.HTMLContent = plan.HTMLContent.ValueString()
	}
	if !plan.PlainContent.IsNull() {
		input.PlainContent = plan.PlainContent.ValueString()
	}
	if !plan.Subject.IsNull() {
		input.Subject = plan.Subject.ValueString()
	}
	if !plan.Categories.IsNull() {
		input.Categories = flex.ExpandFrameworkStringSet(ctx, plan.Categories)
	}

	res, err := retryOnRateLimit(ctx, func() (interface{}, error) {
		return r.client.CreateDesign(ctx, input)
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Creating design",
			fmt.Sprintf("Unable to create design, got error: %s", err),
		)
		return
	}

	o, ok := res.(*sendgrid.OutputCreateDesign)
	if !ok {
		resp.Diagnostics.AddError(
			"Creating design",
			"Failed to assert type *sendgrid.OutputCreateDesign",
		)
		return
	}

	categories, d := types.SetValueFrom(ctx, types.StringType, o.Categories)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan = designResourceModel{
		ID:                   types.StringValue(o.ID),
		Name:                 types.StringValue(o.Name),
		Editor:               types.StringValue(o.Editor),
		HTMLContent:          types.StringValue(o.HTMLContent),
		PlainContent:         types.StringValue(o.PlainContent),
		Subject:              types.StringValue(o.Subject),
		Categories:           categories,
		GeneratePlainContent: types.BoolValue(o.GeneratePlainContent),
		ThumbnailURL:         types.StringValue(o.ThumbnailURL),
		UpdatedAt:            types.StringValue(o.UpdatedAt),
		CreatedAt:            types.StringValue(o.CreatedAt),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *designResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state designResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	if id == "" {
		resp.State.RemoveResource(ctx)
		return
	}

	o, err := r.client.GetDesign(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Reading design",
			fmt.Sprintf("Unable to read design (id: %s), got error: %s", id, err),
		)
		return
	}

	categories, d := types.SetValueFrom(ctx, types.StringType, o.Categories)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	state = designResourceModel{
		ID:                   types.StringValue(o.ID),
		Name:                 types.StringValue(o.Name),
		Editor:               types.StringValue(o.Editor),
		HTMLContent:          types.StringValue(o.HTMLContent),
		PlainContent:         types.StringValue(o.PlainContent),
		Subject:              types.StringValue(o.Subject),
		Categories:           categories,
		GeneratePlainContent: types.BoolValue(o.GeneratePlainContent),
		ThumbnailURL:         types.StringValue(o.ThumbnailURL),
		UpdatedAt:            types.StringValue(o.UpdatedAt),
		CreatedAt:            types.StringValue(o.CreatedAt),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *designResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, state designResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()

	input := &sendgrid.InputUpdateDesign{
		Name:                 data.Name.ValueString(),
		GeneratePlainContent: data.GeneratePlainContent.ValueBool(),
	}

	if !data.HTMLContent.IsNull() {
		input.HTMLContent = data.HTMLContent.ValueString()
	}
	if !data.PlainContent.IsNull() {
		input.PlainContent = data.PlainContent.ValueString()
	}
	if !data.Subject.IsNull() {
		input.Subject = data.Subject.ValueString()
	}
	if !data.Categories.IsNull() {
		input.Categories = flex.ExpandFrameworkStringSet(ctx, data.Categories)
	}

	o, err := r.client.UpdateDesign(ctx, id, input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Updating design",
			fmt.Sprintf("Unable to update design (id: %s), got error: %s", id, err),
		)
		return
	}

	categories, d := types.SetValueFrom(ctx, types.StringType, o.Categories)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	data = designResourceModel{
		ID:                   types.StringValue(o.ID),
		Name:                 types.StringValue(o.Name),
		Editor:               types.StringValue(o.Editor),
		HTMLContent:          types.StringValue(o.HTMLContent),
		PlainContent:         types.StringValue(o.PlainContent),
		Subject:              types.StringValue(o.Subject),
		Categories:           categories,
		GeneratePlainContent: types.BoolValue(o.GeneratePlainContent),
		ThumbnailURL:         types.StringValue(o.ThumbnailURL),
		UpdatedAt:            types.StringValue(o.UpdatedAt),
		CreatedAt:            types.StringValue(o.CreatedAt),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *designResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state designResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()

	_, err := retryOnRateLimit(ctx, func() (interface{}, error) {
		return nil, r.client.DeleteDesign(ctx, id)
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Deleting design",
			fmt.Sprintf("Unable to delete design (id: %s), got error: %s", id, err),
		)
		return
	}
}

func (r *designResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var data designResourceModel

	id := req.ID

	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)

	o, err := r.client.GetDesign(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Importing design",
			fmt.Sprintf("Unable to read design, got error: %s", err),
		)
		return
	}

	categories, d := types.SetValueFrom(ctx, types.StringType, o.Categories)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	data = designResourceModel{
		ID:                   types.StringValue(o.ID),
		Name:                 types.StringValue(o.Name),
		Editor:               types.StringValue(o.Editor),
		HTMLContent:          types.StringValue(o.HTMLContent),
		PlainContent:         types.StringValue(o.PlainContent),
		Subject:              types.StringValue(o.Subject),
		Categories:           categories,
		GeneratePlainContent: types.BoolValue(o.GeneratePlainContent),
		ThumbnailURL:         types.StringValue(o.ThumbnailURL),
		UpdatedAt:            types.StringValue(o.UpdatedAt),
		CreatedAt:            types.StringValue(o.CreatedAt),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
