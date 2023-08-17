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

var defaultScopes = []string{
	"user.profile.read",
}

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
	ID      types.String `tfsdk:"id"`
	Email   types.String `tfsdk:"email"`
	IsAdmin types.Bool   `tfsdk:"is_admin"`
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
			"is_admin": schema.BoolAttribute{
				MarkdownDescription: "Set to true if teammate has admin privileges",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
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

	// NOTE: There are cases where SendGrid automatically adds scopes,
	// 		 and they may not match the values specified in the terraform,
	//       so we do not manage them.
	// 		 We will assign appropriate values to the scopes.
	scopes := defaultScopes
	if data.IsAdmin.ValueBool() {
		scopes = nil
	}

	inviteTeammate, err := r.client.InviteTeammate(context.TODO(), &sendgrid.InputInviteTeammate{
		Email:   data.Email.ValueString(),
		IsAdmin: data.IsAdmin.ValueBool(),
		Scopes:  scopes,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Creating teammate",
			fmt.Sprintf("Unable to invite teammate, got error: %s", err),
		)
		return
	}

	data = teammateResourceModel{
		ID:      types.StringValue(inviteTeammate.Email),
		Email:   types.StringValue(inviteTeammate.Email),
		IsAdmin: types.BoolValue(inviteTeammate.IsAdmin),
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

	data = teammateResourceModel{
		ID:      types.StringValue(teammateByEmail.Email),
		Email:   types.StringValue(teammateByEmail.Email),
		IsAdmin: types.BoolValue(teammateByEmail.IsAdmin),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *teammateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data teammateResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
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
			//       pending teammates cannot update the administrator flag.
			//       In such cases, discrepancies arise between the Terraform code and the tfstate,
			//       leading to errors during the execution of terraform apply.
			//       For pending teammates, it update the is_admin value in the tfstate to prevent any discrepancies.
			//       While there might be differences from the actual code,
			//       not accommodating the above would hinder team member management, making it unavoidable.
			IsAdmin: data.IsAdmin,
		}
		resp.Diagnostics.Append(resp.State.Set(ctx, &p)...)
		return
	}

	teammateByEmail, err := getTeammateByEmail(ctx, r.client, email)
	if err != nil {
		resp.Diagnostics.AddError(
			"Updating teammate",
			fmt.Sprintf("Unable to read teammate username by email, got error: %s", err),
		)
		return
	}

	if teammateByEmail == nil {
		resp.Diagnostics.AddError(
			"Updating teammate",
			fmt.Sprintf("Not found teammates (%s)", email),
		)
		return
	}

	// NOTE: There are cases where SendGrid automatically adds scopes,
	// 		 and they may not match the values specified in the terraform,
	//       so we do not manage them.
	// 		 We will assign appropriate values to the scopes.
	scopes := defaultScopes
	if data.IsAdmin.ValueBool() {
		scopes = nil
	}

	out, err := r.client.UpdateTeammatePermissions(ctx, teammateByEmail.Username, &sendgrid.InputUpdateTeammatePermissions{
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

	// Save updated data into Terraform state
	data = teammateResourceModel{
		ID:      types.StringValue(out.Email),
		Email:   types.StringValue(out.Email),
		IsAdmin: types.BoolValue(out.IsAdmin),
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

	// Invited users are treated as pending users until they set up their profiles.
	pendingUser, err := pendingTeammateByEmail(ctx, r.client, email)
	if err != nil {
		resp.Diagnostics.AddError(
			"Deleting teammate",
			fmt.Sprintf("Unable to get pending teammates, got error: %s", err),
		)
		return
	}

	if pendingUser != nil {
		// If the teammate is in a pending state, execute the API to remove pending teammates.
		if err := r.client.DeletePendingTeammate(ctx, pendingUser.Token); err != nil {
			resp.Diagnostics.AddError(
				"Deleting teammate",
				fmt.Sprintf("Unable to delete pending teammate, got error: %s", err),
			)
		}
		return
	}

	teammateByEmail, err := getTeammateByEmail(ctx, r.client, email)
	if err != nil {
		resp.Diagnostics.AddError(
			"Deleting teammate",
			fmt.Sprintf("Unable to get teammates, got error: %s", err),
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

	if err := r.client.DeleteTeammate(ctx, teammateByEmail.Username); err != nil {
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
		data = teammateResourceModel{
			ID:      types.StringValue(email),
			Email:   types.StringValue(email),
			IsAdmin: types.BoolValue(pendingTeammate.IsAdmin),
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

	data = teammateResourceModel{
		ID:      types.StringValue(teammate.Email),
		Email:   types.StringValue(teammate.Email),
		IsAdmin: types.BoolValue(teammate.IsAdmin),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}
