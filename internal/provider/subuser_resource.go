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
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kenzo0107/sendgrid"
	"github.com/kenzo0107/terraform-provider-sendgrid/flex"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &subuserResource{}
var _ resource.ResourceWithImportState = &subuserResource{}

func newSubuserResource() resource.Resource {
	return &subuserResource{}
}

type subuserResource struct {
	client *sendgrid.Client
}

type subuserResourceModel struct {
	ID       types.Int64  `tfsdk:"id"`
	Username types.String `tfsdk:"username"`
	Email    types.String `tfsdk:"email"`
	Password types.String `tfsdk:"password"`
	Ips      types.Set    `tfsdk:"ips"`
}

func (r *subuserResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_subuser"
}

func (r *subuserResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
Provides a subuser resource.

Subusers help you segment your email sending and API activity. You assign permissions and credit limits when you create the subuser. We recommend creating subusers for each of the different types of emails you send - one subuser for transactional emails, and another for marketing emails. Breaking your sending up this way allows you to get separate statistics for each type of email you send.

For more detailed information, please see the [SendGrid documentation](https://docs.sendgrid.com/ui/account-and-settings/api-keys).
		`,
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "The user ID of the subuser.",
				Computed:            true,
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "The username of the subuser.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"email": schema.StringAttribute{
				MarkdownDescription: "The email of the subuser.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "The password of the subuser. NOTE: The password will only be saved in the tfstate during the execution of the creation.",
				Required:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"ips": schema.SetAttribute{
				MarkdownDescription: "The IP addresses that should be assigned to this subuser.",
				ElementType:         types.StringType,
				Required:            true,
			},
		},
	}
}

func (r *subuserResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *subuserResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan subuserResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ips := flex.ExpandFrameworkStringSet(ctx, plan.Ips)

	// NOTE: Re-execute after the re-executable time has elapsed when a rate limit occurs
	res, err := retryOnRateLimit(ctx, func() (interface{}, error) {
		return r.client.CreateSubuser(ctx, &sendgrid.InputCreateSubuser{
			Username: plan.Username.ValueString(),
			Email:    plan.Email.ValueString(),
			Password: plan.Password.ValueString(),
			Ips:      ips,
		})
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Creating subuser",
			fmt.Sprintf("Unable to create subuser, got error: %s", err),
		)
		return
	}

	o, ok := res.(*sendgrid.OutputCreateSubuser)
	if !ok {
		resp.Diagnostics.AddError(
			"Creating subuser",
			"Failed to assert type *sendgrid.OutputCreateSubuser",
		)
		return
	}

	plan.ID = types.Int64Value(o.UserID)
	plan.Username = types.StringValue(o.Username)
	plan.Email = types.StringValue(o.Email)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *subuserResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state subuserResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	username := state.Username.ValueString()

	subusers, err := r.client.GetSubusers(ctx, &sendgrid.InputGetSubusers{
		Username: username,
		Limit:    1,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Reading subuser",
			fmt.Sprintf("Unable to read subuser (username: %s), got error: %s", username, err),
		)
		return
	}

	if len(subusers) == 0 {
		resp.Diagnostics.AddError(
			"Reading subuser",
			fmt.Sprintf("Unable to read subuser (username: %s)", username),
		)
		return
	}

	if state.Ips.IsNull() {
		state.Ips = types.SetNull(types.StringType)
	}

	subuser := subusers[0]
	state.ID = types.Int64Value(subuser.ID)
	state.Email = types.StringValue(subuser.Email)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *subuserResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, state subuserResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	username := data.Username.ValueString()
	ips := flex.ExpandFrameworkStringSet(ctx, data.Ips)

	if err := r.client.UpdateSubuserIps(ctx, username, ips); err != nil {
		resp.Diagnostics.AddError(
			"Updating subuser",
			fmt.Sprintf("Unable to update subuser's ips (username: %s), got error: %s", username, err),
		)
		return
	}

	data.ID = state.ID
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *subuserResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state subuserResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	username := state.Username.ValueString()

	// NOTE: Re-execute after the re-executable time has elapsed when a rate limit occurs
	_, err := retryOnRateLimit(ctx, func() (interface{}, error) {
		return nil, r.client.DeleteSubuser(ctx, username)
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Deleting subuser",
			fmt.Sprintf("Unable to delete subuser (username: %s), got error: %s", username, err),
		)
		return
	}
}

func (r *subuserResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var data subuserResourceModel

	username := req.ID

	resource.ImportStatePassthroughID(ctx, path.Root("username"), req, resp)

	subusers, err := r.client.GetSubusers(ctx, &sendgrid.InputGetSubusers{
		Username: username,
		Limit:    1,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Importing subuser",
			fmt.Sprintf("Unable to read subuser, got error: %s", err),
		)
		return
	}

	if len(subusers) == 0 {
		resp.Diagnostics.AddError(
			"Importing subuser",
			fmt.Sprintf("Unable to read subuser (username: %s)", username),
		)
		return
	}

	subuser := subusers[0]

	data = subuserResourceModel{
		ID:       types.Int64Value(subuser.ID),
		Username: types.StringValue(subuser.Username),
		Email:    types.StringValue(subuser.Email),
		// NOTE: set ips to null because sendgrid api cannot get ips associated with subuser
		Ips: types.SetNull(types.StringType),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}
