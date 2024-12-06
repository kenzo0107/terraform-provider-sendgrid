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
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kenzo0107/sendgrid"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ssoTeammateResource{}
var _ resource.ResourceWithImportState = &ssoTeammateResource{}

func newSSOTeammateResource() resource.Resource {
	return &ssoTeammateResource{}
}

type ssoTeammateResource struct {
	client *sendgrid.Client
}

type ssoTeammateResourceModel struct {
	ID        types.String   `tfsdk:"id"`
	Email     types.String   `tfsdk:"email"`
	IsAdmin   types.Bool     `tfsdk:"is_admin"`
	Scopes    []types.String `tfsdk:"scopes"`
	Username  types.String   `tfsdk:"username"`
	FirstName types.String   `tfsdk:"first_name"`
	LastName  types.String   `tfsdk:"last_name"`
}

func (r *ssoTeammateResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sso_teammate"
}

func (r *ssoTeammateResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
Provides a SSO Teammate resource.

SSO Teammates is an account administration and security tool designed to help manage multiple users on a single SendGrid account. Teammates is built for groups of shared users, where each user has a different role and thus requires access to different SendGrid features.

For more detailed information, please see the [SendGrid documentation](https://docs.sendgrid.com/glossary/teammates).
		`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"email": schema.StringAttribute{
				MarkdownDescription: "Teammate's email",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "Teammate's username",
				Computed:            true,
			},
			"is_admin": schema.BoolAttribute{
				MarkdownDescription: "Set to true if teammate has admin privileges.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"scopes": schema.SetAttribute{
				ElementType: types.StringType,
				MarkdownDescription: `
				Add or remove permissions from a Teammate using this scopes property. See [Teammate Permissions](https://www.twilio.com/docs/sendgrid/ui/account-and-settings/teammate-permissions) for a complete list of available scopes. You should not include this propety in the request when using the persona property or when setting the is_admin property to true—assigning a persona or setting is_admin to true will allocate a group of permissions to the Teammate.
				`,
				Required: true,
			},
			"first_name": schema.StringAttribute{
				MarkdownDescription: "Teammate's first name",
				Required:            true,
			},
			"last_name": schema.StringAttribute{
				MarkdownDescription: "Teammate's last name",
				Required:            true,
			},
		},
	}
}

func (r *ssoTeammateResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ssoTeammateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ssoTeammateResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := &sendgrid.InputCreateSSOTeammate{
		Email:     data.Email.ValueString(),
		FirstName: data.FirstName.ValueString(),
		LastName:  data.LastName.ValueString(),
		IsAdmin:   data.IsAdmin.ValueBool(),
	}

	if !data.IsAdmin.ValueBool() {
		// Scopes are not required for admin users.
		var scopes []string
		for _, s := range data.Scopes {
			scopes = append(scopes, s.ValueString())
		}
		input.Scopes = scopes
	}

	// NOTE: Re-execute after the re-executable time has elapsed when a rate limit occurs
	res, err := retryOnRateLimit(ctx, func() (interface{}, error) {
		return r.client.CreateSSOTeammate(context.TODO(), input)
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Creating SSO teammate",
			fmt.Sprintf("Unable to invite SSO teammate, got error: %s", err),
		)
		return
	}

	o, ok := res.(*sendgrid.OutputCreateSSOTeammate)
	if !ok {
		resp.Diagnostics.AddError("Creating sso teammate", "Failed to assert type *sendgrid.OutputCreateSSOTeammate")
		return
	}

	data = ssoTeammateResourceModel{
		ID:        types.StringValue(o.Email),
		Email:     types.StringValue(o.Email),
		IsAdmin:   types.BoolValue(o.IsAdmin),
		FirstName: types.StringValue(o.FirstName),
		LastName:  types.StringValue(o.LastName),
		Username:  types.StringValue(o.Email),

		// NOTE: The teammate creation API returns an empty value for scopes,
		//       causing a discrepancy with the scopes specified in the resource and resulting in an error.
		//       To avoid this issue, we will adopt the specified scopes as-is.
		Scopes: data.Scopes,
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *ssoTeammateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ssoTeammateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	email := data.Email.ValueString()

	o, err := r.client.GetTeammate(ctx, email)
	if err != nil {
		resp.Diagnostics.AddError(
			"Reading SSO teammate",
			fmt.Sprintf("Unable to read SSO teammate, got error: %s", err),
		)
		return
	}

	scopes := []types.String{}
	if !data.IsAdmin.ValueBool() {
		for _, s := range o.Scopes {
			scopes = append(scopes, types.StringValue(s))
		}
	}

	data = ssoTeammateResourceModel{
		ID:        types.StringValue(o.Email),
		Email:     types.StringValue(o.Email),
		IsAdmin:   types.BoolValue(o.IsAdmin),
		Username:  types.StringValue(o.Username),
		FirstName: types.StringValue(o.FirstName),
		LastName:  types.StringValue(o.LastName),
		Scopes:    scopes,
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *ssoTeammateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, state ssoTeammateResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	email := data.Email.ValueString()

	scopes := []string{}
	for _, s := range data.Scopes {
		scopes = append(scopes, s.ValueString())
	}

	if data.IsAdmin.ValueBool() && len(scopes) > 0 {
		resp.Diagnostics.AddError(
			"Updating SSO teammate",
			"Unable to update SSO teammate, scopes must be empty for administors",
		)
		return
	}

	o, err := r.client.UpdateSSOTeammate(ctx, email, &sendgrid.InputUpdateSSOTeammate{
		IsAdmin:   data.IsAdmin.ValueBool(),
		Scopes:    scopes,
		FirstName: data.FirstName.ValueString(),
		LastName:  data.LastName.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Updating SSO teammate",
			fmt.Sprintf("Unable to update SSO teammate, got error: %s", err),
		)
		return
	}

	scopesSet := []types.String{}
	for _, s := range o.Scopes {
		scopesSet = append(scopesSet, types.StringValue(s))
	}
	data = ssoTeammateResourceModel{
		ID:        types.StringValue(o.Email),
		Email:     types.StringValue(o.Email),
		IsAdmin:   types.BoolValue(o.IsAdmin),
		Username:  types.StringValue(o.Username),
		Scopes:    scopesSet,
		FirstName: types.StringValue(o.FirstName),
		LastName:  types.StringValue(o.LastName),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *ssoTeammateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ssoTeammateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	email := data.Email.ValueString()

	res, err := retryOnRateLimit(ctx, func() (interface{}, error) {
		return getTeammateByEmail(ctx, r.client, email)
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Deleting SSO teammate",
			fmt.Sprintf("Unable to get SSO teammates, got error: %s", err),
		)
		return
	}

	teammateByEmail, ok := res.(*sendgrid.Teammate)
	if !ok {
		resp.Diagnostics.AddError(
			"Deleting SSO teammate",
			"Failed to assert type *sendgrid.Teammate",
		)
		return
	}

	if teammateByEmail == nil {
		resp.Diagnostics.AddError(
			"Deleting SSO teammate",
			fmt.Sprintf("Not found SSO teammate (%s)", email),
		)
		return
	}

	_, err = retryOnRateLimit(ctx, func() (interface{}, error) {
		return nil, r.client.DeleteTeammate(ctx, teammateByEmail.Username)
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Deleting SSO teammate",
			fmt.Sprintf(
				"Could not delete SSO teammate %s, unexpected error: %s",
				teammateByEmail.Username,
				err,
			),
		)
		return
	}
}

func (r *ssoTeammateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var data ssoTeammateResourceModel

	email := req.ID

	resource.ImportStatePassthroughID(ctx, path.Root("email"), req, resp)

	teammateByEmail, err := getTeammateByEmail(ctx, r.client, email)
	if err != nil {
		resp.Diagnostics.AddError(
			"Importing SSO teammate",
			fmt.Sprintf("Unable to read SSO teammate (%s), got error: %s", email, err),
		)
		return
	}

	if teammateByEmail == nil {
		resp.Diagnostics.AddError(
			"Importing SSO teammate",
			fmt.Sprintf("Not found SSO teammate (%s)", email),
		)
		return
	}

	teammate, err := r.client.GetTeammate(ctx, teammateByEmail.Username)
	if err != nil {
		resp.Diagnostics.AddError(
			"Importing SSO teammate",
			fmt.Sprintf("Unable to read SSO teammate, got error: %s", err),
		)
		return
	}

	scopes := []types.String{}
	for _, s := range teammate.Scopes {
		scopes = append(scopes, types.StringValue(s))
	}

	data = ssoTeammateResourceModel{
		ID:        types.StringValue(teammate.Email),
		Email:     types.StringValue(teammate.Email),
		IsAdmin:   types.BoolValue(teammate.IsAdmin),
		Username:  types.StringValue(teammate.Username),
		Scopes:    scopes,
		FirstName: types.StringValue(teammate.FirstName),
		LastName:  types.StringValue(teammate.LastName),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}
