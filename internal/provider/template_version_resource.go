// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kenzo0107/sendgrid"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &templateVersionResource{}
var _ resource.ResourceWithImportState = &templateVersionResource{}

func newTemplateVersionResource() resource.Resource {
	return &templateVersionResource{}
}

type templateVersionResource struct {
	client *sendgrid.Client
}

type templateVersionResourceModel struct {
	ID                   types.String `tfsdk:"id"`
	TemplateID           types.String `tfsdk:"template_id"`
	Subject              types.String `tfsdk:"subject"`
	Active               types.Number `tfsdk:"active"`
	Name                 types.String `tfsdk:"name"`
	HTMLContent          types.String `tfsdk:"html_content"`
	PlainContent         types.String `tfsdk:"plain_content"`
	GeneratePlainContent types.Bool   `tfsdk:"generate_plain_content"`
	Editor               types.String `tfsdk:"editor"`
	TestData             types.String `tfsdk:"test_data"`
	ThumbnailURL         types.String `tfsdk:"thumbnail_url"`
}

func (r *templateVersionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_template_version"
}

func (r *templateVersionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
Provides a template version resource.

Represents the code for a particular transactional template. Each transactional template can have multiple versions, each version with its own subject and content. Each user can have up to 300 versions across across all templates.

For more information about transactional templates, please see our Transactional Templates documentation. You can also manage your Transactional Templates in the Dynamic Templates section of the Twilio SendGrid App.
		`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the transactional template version.",
				Computed:            true,
			},
			"template_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the transactional template.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name for the transactional template.",
				Required:            true,
			},
			"subject": schema.StringAttribute{
				MarkdownDescription: "Subject of the new transactional template version. maxLength: 255",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"active": schema.NumberAttribute{
				MarkdownDescription: "Set the version as the active version associated with the template (0 is inactive, 1 is active). Only one version of a template can be active. The first version created for a template will automatically be set to Active. Allowed Values: 0, 1",
				Optional:            true,
			},
			"html_content": schema.StringAttribute{
				MarkdownDescription: "The HTML content of the version. Maximum of 1048576 bytes allowed.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"plain_content": schema.StringAttribute{
				MarkdownDescription: "Text/plain content of the transactional template version. Maximum of 1048576 bytes allowed.",
				Optional:            true,
				Computed:            true,
			},
			"generate_plain_content": schema.BoolAttribute{
				MarkdownDescription: "If true, plain_content is always generated from html_content. If false, plain_content is not altered.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"editor": schema.StringAttribute{
				MarkdownDescription: "The editor used in the UI.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("code"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringOneOf("code", "design"),
				},
			},
			"test_data": schema.StringAttribute{
				MarkdownDescription: "For dynamic templates only, the mock json data that will be used for template preview and test sends.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"thumbnail_url": schema.StringAttribute{
				MarkdownDescription: "A Thumbnail preview of the template's html content.",
				Computed:            true,
			},
		},
	}
}

func (r *templateVersionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *templateVersionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan templateVersionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	templateID := plan.TemplateID.ValueString()

	active, _ := plan.Active.ValueBigFloat().Int64()

	o, err := r.client.CreateTemplateVersion(ctx, templateID, &sendgrid.InputCreateTemplateVersion{
		Active:               int(active),
		Name:                 plan.Name.ValueString(),
		HTMLContent:          plan.HTMLContent.ValueString(),
		PlainContent:         plan.PlainContent.ValueString(),
		GeneratePlainContent: plan.GeneratePlainContent.ValueBool(),
		Subject:              plan.Subject.ValueString(),
		Editor:               plan.Editor.ValueString(),
		TestData:             plan.TestData.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Creating template version",
			fmt.Sprintf("Unable to create template version (template id: %s), got error: %s", templateID, err),
		)
		return
	}

	plan = templateVersionResourceModel{
		ID:                   types.StringValue(o.ID),
		TemplateID:           types.StringValue(o.TemplateID),
		Subject:              types.StringValue(o.Subject),
		Active:               types.NumberValue(big.NewFloat(float64(o.Active))),
		Name:                 types.StringValue(o.Name),
		HTMLContent:          types.StringValue(o.HTMLContent),
		PlainContent:         types.StringValue(o.PlainContent),
		GeneratePlainContent: types.BoolValue(o.GeneratePlainContent),
		Editor:               types.StringValue(o.Editor),
		TestData:             types.StringValue(o.TestData),
		ThumbnailURL:         types.StringValue(o.ThumbnailURL),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *templateVersionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state templateVersionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	versionID := state.ID.ValueString()
	templateID := state.TemplateID.ValueString()
	o, err := r.client.GetTemplateVersion(ctx, templateID, versionID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Reading template version",
			fmt.Sprintf("Unable to read template version (template id: %s, version id: %s), got error: %s", templateID, versionID, err),
		)
		return
	}

	state = templateVersionResourceModel{
		ID:                   state.ID,
		TemplateID:           state.TemplateID,
		Subject:              types.StringValue(o.Subject),
		Active:               types.NumberValue(big.NewFloat(float64(o.Active))),
		Name:                 types.StringValue(o.Name),
		HTMLContent:          types.StringValue(o.HTMLContent),
		PlainContent:         types.StringValue(o.PlainContent),
		GeneratePlainContent: types.BoolValue(o.GeneratePlainContent),
		Editor:               types.StringValue(o.Editor),
		TestData:             types.StringValue(o.TestData),
		ThumbnailURL:         types.StringValue(o.ThumbnailURL),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *templateVersionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, state templateVersionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := &sendgrid.InputUpdateTemplateVersion{}

	active, _ := data.Active.ValueBigFloat().Int64()
	input.Active = int(active)
	input.GeneratePlainContent = data.GeneratePlainContent.ValueBool()
	if data.Name.ValueString() != "" && data.Name.ValueString() != state.Name.ValueString() {
		input.Name = data.Name.ValueString()
	}
	if data.Subject.ValueString() != "" && data.Subject.ValueString() != state.Subject.ValueString() {
		input.Subject = data.Subject.ValueString()
	}
	if data.HTMLContent.ValueString() != "" && data.HTMLContent.ValueString() != state.HTMLContent.ValueString() {
		input.HTMLContent = data.HTMLContent.ValueString()
	}
	if data.PlainContent.ValueString() != "" && data.PlainContent.ValueString() != state.PlainContent.ValueString() {
		input.PlainContent = data.PlainContent.ValueString()
	}
	// NOTE: Even if "code" is already set, if you try to update it with "code", an error will occur.
	if data.Editor.ValueString() != "" && data.Editor.ValueString() != state.Editor.ValueString() {
		input.Editor = data.Editor.ValueString()
	}
	if data.TestData.ValueString() != "" && data.TestData.ValueString() != state.TestData.ValueString() {
		input.TestData = data.TestData.ValueString()
	}

	versionID := state.ID.ValueString()
	templateID := data.TemplateID.ValueString()

	o, err := r.client.UpdateTemplateVersion(ctx, templateID, versionID, input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Updating template version",
			fmt.Sprintf("Unable to update template version (template id: %s, version id: %s), got error: %s", templateID, versionID, err),
		)
		return
	}

	data = templateVersionResourceModel{
		ID:                   state.ID,
		TemplateID:           state.TemplateID,
		Subject:              types.StringValue(o.Subject),
		Active:               types.NumberValue(big.NewFloat(float64(o.Active))),
		Name:                 types.StringValue(o.Name),
		HTMLContent:          types.StringValue(o.HTMLContent),
		PlainContent:         types.StringValue(o.PlainContent),
		GeneratePlainContent: types.BoolValue(o.GeneratePlainContent),
		Editor:               types.StringValue(o.Editor),
		TestData:             types.StringValue(o.TestData),
		ThumbnailURL:         types.StringValue(o.ThumbnailURL),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *templateVersionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state templateVersionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	versionID := state.ID.ValueString()
	templateID := state.TemplateID.ValueString()
	if err := r.client.DeleteTemplateVersion(ctx, templateID, versionID); err != nil {
		resp.Diagnostics.AddError(
			"Deleting template version",
			fmt.Sprintf("Unable to delete template version (template id: %s, version id: %s), got error: %s", templateID, versionID, err),
		)
		return
	}
}

func (r *templateVersionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var data templateVersionResourceModel

	id := req.ID
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)

	// id = templateID/versionID
	a := strings.Split(id, "/")
	if len(a) != 2 {
		resp.Diagnostics.AddError(
			"Importing template version",
			"Unable to import template version, id must be in the format of templateID/versionID",
		)
		return
	}
	templateID := a[0]
	versionID := a[1]

	o, err := r.client.GetTemplateVersion(ctx, templateID, versionID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Importing template version",
			fmt.Sprintf("Unable to read template version (template id: %s, version id: %s), got error: %s", templateID, versionID, err),
		)
		return
	}

	data = templateVersionResourceModel{
		ID:                   types.StringValue(o.ID),
		TemplateID:           types.StringValue(o.TemplateID),
		Subject:              types.StringValue(o.Subject),
		Active:               types.NumberValue(big.NewFloat(float64(o.Active))),
		Name:                 types.StringValue(o.Name),
		HTMLContent:          types.StringValue(o.HTMLContent),
		PlainContent:         types.StringValue(o.PlainContent),
		GeneratePlainContent: types.BoolValue(o.GeneratePlainContent),
		Editor:               types.StringValue(o.Editor),
		TestData:             types.StringValue(o.TestData),
		ThumbnailURL:         types.StringValue(o.ThumbnailURL),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}
