// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kenzo0107/sendgrid"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ssoIntegrationResource{}
var _ resource.ResourceWithImportState = &ssoIntegrationResource{}

func newSSOIntegrationResource() resource.Resource {
	return &ssoIntegrationResource{}
}

type ssoIntegrationResource struct {
	client *sendgrid.Client
}

type ssoIntegrationResourceModel struct {
	ID                   types.String `tfsdk:"id"`
	Name                 types.String `tfsdk:"name"`
	Enabled              types.Bool   `tfsdk:"enabled"`
	SigninURL            types.String `tfsdk:"signin_url"`
	SignoutURL           types.String `tfsdk:"signout_url"`
	EntityID             types.String `tfsdk:"entity_id"`
	CompletedIntegration types.Bool   `tfsdk:"completed_integration"`
	SingleSignonURL      types.String `tfsdk:"single_signon_url"`
	AudienceURL          types.String `tfsdk:"audience_url"`
}

func (r *ssoIntegrationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sso_integration"
}

func (r *ssoIntegrationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
Provides SSO Integration resource.
		`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "A unique ID assigned to the configuration by SendGrid.",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of your integration. This name can be anything that makes sense for your organization (eg. Twilio SendGrid)",
				Required:            true,
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Indicates if the integration is enabled.",
				Required:            true,
			},
			"signin_url": schema.StringAttribute{
				MarkdownDescription: "The IdP's SAML POST endpoint. This endpoint should receive requests and initiate an SSO login flow. This is called the \"Embed Link\" in the Twilio SendGrid UI.",
				Required:            true,
			},
			"signout_url": schema.StringAttribute{
				MarkdownDescription: "This URL is relevant only for an IdP-initiated authentication flow. If a user authenticates from their IdP, this URL will return them to their IdP when logging out.",
				Required:            true,
			},
			"entity_id": schema.StringAttribute{
				MarkdownDescription: "An identifier provided by your IdP to identify Twilio SendGrid in the SAML interaction. This is called the \"SAML Issuer ID\" in the Twilio SendGrid UI.",
				Required:            true,
			},
			"completed_integration": schema.BoolAttribute{
				MarkdownDescription: "Indicates if the integration is complete.",
				Computed:            true,
			},
			"single_signon_url": schema.StringAttribute{
				MarkdownDescription: "The URL where your IdP should POST its SAML response. This is the Twilio SendGrid URL that is responsible for receiving and parsing a SAML assertion. This is the same URL as the Audience URL when using SendGrid.",
				Computed:            true,
			},
			"audience_url": schema.StringAttribute{
				MarkdownDescription: "The URL where your IdP should POST its SAML response. This is the Twilio SendGrid URL that is responsible for receiving and parsing a SAML assertion. This is the same URL as the Single Sign-On URL when using SendGrid.",
				Computed:            true,
			},
		},
	}
}

func (r *ssoIntegrationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ssoIntegrationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ssoIntegrationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := &sendgrid.InputCreateSSOIntegration{
		Name:       plan.Name.ValueString(),
		Enabled:    plan.Enabled.ValueBool(),
		SigninURL:  plan.SigninURL.ValueString(),
		SignoutURL: plan.SignoutURL.ValueString(),
		EntityID:   plan.EntityID.ValueString(),
	}

	if !plan.CompletedIntegration.IsNull() {
		input.CompletedIntegration = plan.CompletedIntegration.ValueBool()
	}

	o, err := r.client.CreateSSOIntegration(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Creating SSO Integration",
			fmt.Sprintf("Unable to create sso integration, got error: %s", err),
		)
		return
	}

	plan = ssoIntegrationResourceModel{
		ID:                   types.StringValue(o.ID),
		Name:                 types.StringValue(o.Name),
		Enabled:              types.BoolValue(o.Enabled),
		SigninURL:            types.StringValue(o.SigninURL),
		SignoutURL:           types.StringValue(o.SignoutURL),
		EntityID:             types.StringValue(o.EntityID),
		CompletedIntegration: types.BoolValue(o.CompletedIntegration),
		SingleSignonURL:      types.StringValue(o.SingleSignonURL),
		AudienceURL:          types.StringValue(o.AudienceURL),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *ssoIntegrationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ssoIntegrationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()

	o, err := r.client.GetSSOIntegration(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Reading sso integration",
			fmt.Sprintf("Unable to read sso integration (id: %v), got error: %s", id, err),
		)
		return
	}

	state = ssoIntegrationResourceModel{
		ID:                   types.StringValue(o.ID),
		Name:                 types.StringValue(o.Name),
		Enabled:              types.BoolValue(o.Enabled),
		SigninURL:            types.StringValue(o.SigninURL),
		SignoutURL:           types.StringValue(o.SignoutURL),
		EntityID:             types.StringValue(o.EntityID),
		CompletedIntegration: types.BoolValue(o.CompletedIntegration),
		SingleSignonURL:      types.StringValue(o.SingleSignonURL),
		AudienceURL:          types.StringValue(o.AudienceURL),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *ssoIntegrationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, state ssoIntegrationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := &sendgrid.InputUpdateSSOIntegration{}
	if !data.Name.IsNull() && data.Name != state.Name {
		input.Name = data.Name.ValueString()
	}
	if !data.Enabled.IsNull() && data.Enabled != state.Enabled {
		input.Enabled = data.Enabled.ValueBool()
	}
	if !data.SigninURL.IsNull() && data.SigninURL != state.SigninURL {
		input.SigninURL = data.SigninURL.ValueString()
	}
	if !data.SignoutURL.IsNull() && data.SignoutURL != state.SignoutURL {
		input.SignoutURL = data.SignoutURL.ValueString()
	}
	if !data.EntityID.IsNull() && data.EntityID != state.EntityID {
		input.EntityID = data.EntityID.ValueString()
	}
	if !data.CompletedIntegration.IsNull() && data.CompletedIntegration != state.CompletedIntegration {
		input.CompletedIntegration = data.CompletedIntegration.ValueBool()
	}

	id := data.ID.ValueString()
	o, err := r.client.UpdateSSOIntegration(ctx, id, input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Updating sso integration",
			fmt.Sprintf("Unable to sso integration (id: %s), got error: %s", id, err),
		)
		return
	}

	data = ssoIntegrationResourceModel{
		ID:                   types.StringValue(o.ID),
		Name:                 types.StringValue(o.Name),
		Enabled:              types.BoolValue(o.Enabled),
		SigninURL:            types.StringValue(o.SigninURL),
		SignoutURL:           types.StringValue(o.SignoutURL),
		EntityID:             types.StringValue(o.EntityID),
		CompletedIntegration: types.BoolValue(o.CompletedIntegration),
		SingleSignonURL:      types.StringValue(o.SingleSignonURL),
		AudienceURL:          types.StringValue(o.AudienceURL),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *ssoIntegrationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ssoIntegrationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()

	if err := r.client.DeleteSSOIntegration(ctx, id); err != nil {
		resp.Diagnostics.AddError(
			"Deleting sso integration",
			fmt.Sprintf("Unable to delete sso integration (id: %v), got error: %s", id, err),
		)
		return
	}
}

func (r *ssoIntegrationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var data ssoIntegrationResourceModel

	id := req.ID
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)

	o, err := r.client.GetSSOIntegration(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Importing sso integration",
			fmt.Sprintf("Unable to read sso integration, got error: %s", err),
		)
		return
	}

	data = ssoIntegrationResourceModel{
		ID:                   types.StringValue(o.ID),
		Name:                 types.StringValue(o.Name),
		Enabled:              types.BoolValue(o.Enabled),
		SigninURL:            types.StringValue(o.SigninURL),
		SignoutURL:           types.StringValue(o.SignoutURL),
		EntityID:             types.StringValue(o.EntityID),
		CompletedIntegration: types.BoolValue(o.CompletedIntegration),
		SingleSignonURL:      types.StringValue(o.SingleSignonURL),
		AudienceURL:          types.StringValue(o.AudienceURL),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}
