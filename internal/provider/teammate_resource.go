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
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kenzo0107/sendgrid"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &teammateResource{}
var _ resource.ResourceWithImportState = &teammateResource{}

func newTeammateResource() resource.Resource {
	return &teammateResource{}
}

type teammateResource struct {
	client *sendgrid.Client
}

type teammateResourceModel struct {
	ID       types.String   `tfsdk:"id"`
	Email    types.String   `tfsdk:"email"`
	IsAdmin  types.Bool     `tfsdk:"is_admin"`
	Scopes   []types.String `tfsdk:"scopes"`
	Username types.String   `tfsdk:"username"`
}

func (r *teammateResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_teammate"
}

func (r *teammateResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
Provides a Teammate resource.

Teammates is an account administration and security tool designed to help manage multiple users on a single SendGrid account. Teammates is built for groups of shared users, where each user has a different role and thus requires access to different SendGrid features.

For more detailed information, please see the [SendGrid documentation](https://docs.sendgrid.com/glossary/teammates).
		`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"email": schema.StringAttribute{
				MarkdownDescription: "Teammate's email",
				Required:            true,
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
The permissions API Key has access to.

For more detailed information, please see the [SendGrid documentation](https://docs.sendgrid.com/ui/account-and-settings/teammate-permissions#persona-scopes)

Note:
Since scopes are automatically added after registering the usernames of invited teammates, there's a possibility that they might differ from the values defined in the Terraform code.
As some scopes that cannot be defined are unclear, the current approach is to use ignore_changes to avoid modifications and instead set permissions on the dashboard, which is preferable.
				`,
				Required: true,
			},
		},
	}
}

func (r *teammateResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *teammateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data teammateResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := &sendgrid.InputInviteTeammate{
		Email:   data.Email.ValueString(),
		IsAdmin: data.IsAdmin.ValueBool(),
	}

	if !data.IsAdmin.ValueBool() {
		var scopes []string
		for _, s := range data.Scopes {
			scopes = append(scopes, s.ValueString())
		}
		input.Scopes = scopes
	}

	// NOTE: Re-execute after the re-executable time has elapsed when a rate limit occurs
	res, err := retryOnRateLimit(ctx, func() (interface{}, error) {
		return r.client.InviteTeammate(context.TODO(), input)
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Creating teammate",
			fmt.Sprintf("Unable to invite teammate, got error: %s", err),
		)
		return
	}

	inviteTeammate, ok := res.(*sendgrid.OutputInviteTeammate)
	if !ok {
		resp.Diagnostics.AddError(
			"Creating teammate",
			"Failed to assert type *sendgrid.OutputInviteTeammate",
		)
		return
	}

	scopes := []types.String{}
	for _, s := range inviteTeammate.Scopes {
		scopes = append(scopes, types.StringValue(s))
	}

	// pending user does not have an username.
	data = teammateResourceModel{
		ID:      types.StringValue(inviteTeammate.Email),
		Email:   types.StringValue(inviteTeammate.Email),
		IsAdmin: types.BoolValue(inviteTeammate.IsAdmin),
		Scopes:  scopes,
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *teammateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data teammateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	email := data.Email.ValueString()

	pendingTeammate, err := pendingTeammateByEmail(ctx, r.client, email)
	if err != nil {
		resp.Diagnostics.AddError(
			"Reading teammate",
			fmt.Sprintf("Unable to get pending teammates, got error: %s", err),
		)
		return
	}

	// If the teammate is in a pending state, return their data.
	if pendingTeammate != nil {
		scopes := []types.String{}
		for _, s := range pendingTeammate.Scopes {
			scopes = append(scopes, types.StringValue(s))
		}
		data = teammateResourceModel{
			ID:    types.StringValue(pendingTeammate.Email),
			Email: types.StringValue(pendingTeammate.Email),
			// NOTE: As per the SendGrid API specifications,
			//       pending teammates cannot update the administrator flag.
			//       In such cases, discrepancies arise between the Terraform code and the tfstate,
			//       leading to errors during the execution of terraform apply.
			//       For pending teammates, it update the is_admin value in the tfstate to prevent any discrepancies.
			//       While there might be differences from the actual code,
			//       not accommodating the above would hinder team member management, making it unavoidable.
			IsAdmin: data.IsAdmin,
			Scopes:  scopes,
		}

		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}

	teammateByEmail, err := getTeammateByEmail(ctx, r.client, email)
	if err != nil {
		resp.Diagnostics.AddError(
			"Reading teammate",
			fmt.Sprintf("Unable to read teammate (%s), got error: %s", email, err),
		)
		return
	}

	// If you are unable to retrieve your teammate's information using their email address,
	// it removes the resource information from the state.
	if teammateByEmail == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	o, err := r.client.GetTeammate(ctx, teammateByEmail.Username)
	if err != nil {
		resp.Diagnostics.AddError(
			"Reading teammate",
			fmt.Sprintf("Unable to read teammate (username: %s), got error: %s", teammateByEmail.Username, err),
		)
		return
	}

	scopes := []types.String{}
	for _, s := range o.Scopes {
		scopes = append(scopes, types.StringValue(s))
	}

	data = teammateResourceModel{
		ID:       types.StringValue(o.Email),
		Email:    types.StringValue(o.Email),
		IsAdmin:  types.BoolValue(o.IsAdmin),
		Username: types.StringValue(o.Username),
		Scopes:   scopes,
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *teammateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, state teammateResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	email := data.Email.ValueString()

	pendingTeammate, err := pendingTeammateByEmail(ctx, r.client, email)
	if err != nil {
		resp.Diagnostics.AddError(
			"Updating teammate",
			fmt.Sprintf("Unable to get pending teammates, got error: %s", err),
		)
		return
	}

	// If the teammate is in a pending state, it is not possible to update the permissions.
	if pendingTeammate != nil {
		p := teammateResourceModel{
			ID:    types.StringValue(pendingTeammate.Email),
			Email: types.StringValue(pendingTeammate.Email),
			// NOTE: As per the SendGrid API specifications,
			//       pending teammates cannot update the administrator flag and scopes.
			//       In such cases, discrepancies arise between the Terraform code and the tfstate,
			//       leading to errors during the execution of terraform apply.
			//       For pending teammates, it update the is_admin value in the tfstate to prevent any discrepancies.
			//       While there might be differences from the actual code,
			//       not accommodating the above would hinder team member management, making it unavoidable.
			IsAdmin: data.IsAdmin,
			Scopes:  data.Scopes,
		}
		resp.Diagnostics.Append(resp.State.Set(ctx, &p)...)
		return
	}

	// get username from tfstate
	username := state.Username.ValueString()

	scopes := []string{}
	for _, s := range data.Scopes {
		scopes = append(scopes, s.ValueString())
	}

	o, err := r.client.UpdateTeammatePermissions(ctx, username, &sendgrid.InputUpdateTeammatePermissions{
		IsAdmin: data.IsAdmin.ValueBool(),
		Scopes:  scopes,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Updating teammate",
			fmt.Sprintf("Unable to update teammate permissions, got error: %s", err),
		)
		return
	}

	scopesSet := []types.String{}
	for _, s := range o.Scopes {
		scopesSet = append(scopesSet, types.StringValue(s))
	}

	// Save updated data into Terraform state
	data = teammateResourceModel{
		ID:       types.StringValue(o.Email),
		Email:    types.StringValue(o.Email),
		IsAdmin:  types.BoolValue(o.IsAdmin),
		Username: types.StringValue(o.Username),
		Scopes:   scopesSet,
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *teammateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data teammateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	email := data.Email.ValueString()

	res, err := retryOnRateLimit(ctx, func() (interface{}, error) {
		// Invited users are treated as pending users until they set up their profiles.
		return pendingTeammateByEmail(ctx, r.client, email)
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Deleting teammate",
			fmt.Sprintf("Unable to get pending teammates, got error: %s", err),
		)
		return
	}

	pendingUser, ok := res.(*sendgrid.PendingTeammate)
	if !ok {
		resp.Diagnostics.AddError(
			"Deleting teammate",
			"Failed to assert type *sendgrid.PendingTeammate",
		)
		return
	}

	if pendingUser != nil {
		_, err = retryOnRateLimit(ctx, func() (interface{}, error) {
			return nil, r.client.DeletePendingTeammate(ctx, pendingUser.Token)
		})
		// If the teammate is in a pending state, execute the API to remove pending teammates.
		if err != nil {
			resp.Diagnostics.AddError(
				"Deleting teammate",
				fmt.Sprintf("Unable to delete pending teammate, got error: %s", err),
			)
		}
		return
	}

	res, err = retryOnRateLimit(ctx, func() (interface{}, error) {
		return getTeammateByEmail(ctx, r.client, email)
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Deleting teammate",
			fmt.Sprintf("Unable to get teammates, got error: %s", err),
		)
		return
	}

	teammateByEmail, ok := res.(*sendgrid.Teammate)
	if !ok {
		resp.Diagnostics.AddError(
			"Deleting teammate",
			"Failed to assert type *sendgrid.Teammate",
		)
		return
	}

	if teammateByEmail == nil {
		resp.Diagnostics.AddError(
			"Deleting teammate",
			fmt.Sprintf("Not found teammate (%s)", email),
		)
		return
	}

	_, err = retryOnRateLimit(ctx, func() (interface{}, error) {
		return nil, r.client.DeleteTeammate(ctx, teammateByEmail.Username)
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Deleting teammate",
			fmt.Sprintf(
				"Could not delete teammate %s, unexpected error: %s",
				teammateByEmail.Username,
				err,
			),
		)
		return
	}
}

func (r *teammateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var data teammateResourceModel

	email := req.ID

	resource.ImportStatePassthroughID(ctx, path.Root("email"), req, resp)

	pendingTeammate, err := pendingTeammateByEmail(ctx, r.client, email)
	if err != nil {
		resp.Diagnostics.AddError(
			"Importing teammate",
			fmt.Sprintf("Unable to get pending teammates, got error: %s", err),
		)
		return
	}

	// If the teammate is in a pending state, return their data.
	if pendingTeammate != nil {
		scopes := []types.String{}
		for _, s := range pendingTeammate.Scopes {
			scopes = append(scopes, types.StringValue(s))
		}
		data = teammateResourceModel{
			ID:      types.StringValue(email),
			Email:   types.StringValue(email),
			IsAdmin: types.BoolValue(pendingTeammate.IsAdmin),
			Scopes:  scopes,
		}

		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}

	teammateByEmail, err := getTeammateByEmail(ctx, r.client, email)
	if err != nil {
		resp.Diagnostics.AddError(
			"Importing teammate",
			fmt.Sprintf("Unable to read teammate (%s), got error: %s", email, err),
		)
		return
	}

	if teammateByEmail == nil {
		resp.Diagnostics.AddError(
			"Importing teammate",
			fmt.Sprintf("Not found teammate (%s)", email),
		)
		return
	}

	teammate, err := r.client.GetTeammate(ctx, teammateByEmail.Username)
	if err != nil {
		resp.Diagnostics.AddError(
			"Importing teammate",
			fmt.Sprintf("Unable to read teammate, got error: %s", err),
		)
		return
	}

	scopes := []types.String{}
	for _, s := range teammate.Scopes {
		scopes = append(scopes, types.StringValue(s))
	}

	data = teammateResourceModel{
		ID:       types.StringValue(teammate.Email),
		Email:    types.StringValue(teammate.Email),
		IsAdmin:  types.BoolValue(teammate.IsAdmin),
		Username: types.StringValue(teammate.Username),
		Scopes:   scopes,
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}
