// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/boolvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
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

type ssoSubuserAccessResourceModel struct {
	ID             types.Int64    `tfsdk:"id"`
	PermissionType types.String   `tfsdk:"permission_type"`
	Scopes         []types.String `tfsdk:"scopes"`
}

func toInputSubuserAccess(model ssoSubuserAccessResourceModel) sendgrid.InputSubuserAccess {
	input := sendgrid.InputSubuserAccess{
		ID:             model.ID.ValueInt64(),
		PermissionType: model.PermissionType.ValueString(),
		Scopes:         []string{},
	}

	for _, scope := range model.Scopes {
		input.Scopes = append(input.Scopes, scope.ValueString())
	}

	return input
}

func toInputSubuserAccessArray(subuserAccess []ssoSubuserAccessResourceModel) []sendgrid.InputSubuserAccess {
	saArrayInput := make([]sendgrid.InputSubuserAccess, 0, len(subuserAccess))
	for _, sa := range subuserAccess {
		saArrayInput = append(saArrayInput, toInputSubuserAccess(sa))
	}
	return saArrayInput
}

func fromOutputSubuserAccess(output sendgrid.OutputSubuserAccess) ssoSubuserAccessResourceModel {
	model := ssoSubuserAccessResourceModel{
		ID:             types.Int64Value(output.ID),
		PermissionType: types.StringValue(output.PermissionType),
		Scopes:         nil,
	}

	if len(output.Scopes) > 0 {
		model.Scopes = make([]types.String, 0, len(output.Scopes))
		for _, scope := range output.Scopes {
			model.Scopes = append(model.Scopes, types.StringValue(scope))
		}
	}

	return model
}

func fromOutputSubuserAccessArray(output []sendgrid.OutputSubuserAccess) []ssoSubuserAccessResourceModel {
	if len(output) == 0 {
		return nil
	}

	saArrayOutput := make([]ssoSubuserAccessResourceModel, 0, len(output))
	for _, sa := range output {
		saArrayOutput = append(saArrayOutput, fromOutputSubuserAccess(sa))
	}
	return saArrayOutput
}

func fromSendgridSubuserAccess(output sendgrid.SubuserAccess) ssoSubuserAccessResourceModel {
	model := ssoSubuserAccessResourceModel{
		ID:             types.Int64Value(output.ID),
		PermissionType: types.StringValue(output.PermissionType),
		Scopes:         nil,
	}

	if len(output.Scopes) > 0 {
		model.Scopes = make([]types.String, 0, len(output.Scopes))
		for _, scope := range output.Scopes {
			model.Scopes = append(model.Scopes, types.StringValue(scope))
		}
	}

	return model
}

func fromSendgridSubuserAccessArray(output []sendgrid.SubuserAccess) []ssoSubuserAccessResourceModel {
	if len(output) == 0 {
		return nil
	}

	saArrayOutput := make([]ssoSubuserAccessResourceModel, 0, len(output))
	for _, sa := range output {
		saArrayOutput = append(saArrayOutput, fromSendgridSubuserAccess(sa))
	}
	return saArrayOutput
}

type ssoTeammateResourceModel struct {
	ID            types.String                    `tfsdk:"id"`
	Email         types.String                    `tfsdk:"email"`
	IsAdmin       types.Bool                      `tfsdk:"is_admin"`
	Scopes        []types.String                  `tfsdk:"scopes"`
	Username      types.String                    `tfsdk:"username"`
	FirstName     types.String                    `tfsdk:"first_name"`
	LastName      types.String                    `tfsdk:"last_name"`
	SubuserAccess []ssoSubuserAccessResourceModel `tfsdk:"subuser_access"`
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
				Validators: []validator.Bool{
					boolvalidator.ConflictsWith(
						path.MatchRelative().AtParent().AtName("subuser_access"),
						path.MatchRelative().AtParent().AtName("scopes"),
					),
				},
			},
			"scopes": schema.SetAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "Add or remove permissions from a Teammate using this scopes property. See [Teammate Permissions](https://www.twilio.com/docs/sendgrid/ui/account-and-settings/teammate-permissions) for a complete list of available scopes. You should not include this propety in the request when setting the `is_admin` property to `true` or `subuser_access` property to a list of subuser accesses.",
				Optional:            true,
				Validators: []validator.Set{
					setvalidator.ConflictsWith(
						path.MatchRelative().AtParent().AtName("subuser_access"),
						path.MatchRelative().AtParent().AtName("is_admin"),
					),
				},
			},
			"first_name": schema.StringAttribute{
				MarkdownDescription: "Teammate's first name",
				Required:            true,
			},
			"last_name": schema.StringAttribute{
				MarkdownDescription: "Teammate's last name",
				Required:            true,
			},
			"subuser_access": schema.ListNestedAttribute{
				Optional:            true,
				MarkdownDescription: "Specify which Subusers the Teammate may access and act on behalf of.",
				Validators: []validator.List{
					listvalidator.ConflictsWith(
						path.MatchRelative().AtParent().AtName("scopes"),
						path.MatchRelative().AtParent().AtName("is_admin"),
					),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.Int64Attribute{
							MarkdownDescription: "Set this property to the ID of a Subuser to which the Teammate should have access.",
							Required:            true,
						},
						"permission_type": schema.StringAttribute{
							MarkdownDescription: "Grant the level of access the Teammate should have to the specified Subuser with this property. This property value may be either `admin` or `restricted`. When set to `restricted`, the Teammate has only the permissions assigned in the `scopes` property.",
							Required:            true,
							Validators: []validator.String{
								stringvalidator.OneOf("admin", "restricted"),
							},
						},
						"scopes": schema.SetAttribute{
							ElementType:         types.StringType,
							Optional:            true,
							MarkdownDescription: "Add or remove permissions that the Teammate can access on behalf of the Subuser. See [Teammate Permissions](https://www.twilio.com/docs/sendgrid/ui/account-and-settings/teammate-permissions) for a complete list of available scopes. You should not include this property in the request when the `permission_type` property is set to `admin` — administrators have full access to the specified Subuser.",
						},
					},
				},
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
		Email:                      data.Email.ValueString(),
		FirstName:                  data.FirstName.ValueString(),
		LastName:                   data.LastName.ValueString(),
		IsAdmin:                    data.IsAdmin.ValueBool(),
		HasRestrictedSubuserAccess: len(data.SubuserAccess) > 0,
		SubuserAccess:              toInputSubuserAccessArray(data.SubuserAccess),
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

	saArray := fromOutputSubuserAccessArray(o.SubuserAccess)
	// NOTE: The teammate read API returns subuser access with admin permissions to all subusers when the user is admin,
	//       causing a discrepancy with the subuser access specified in the resource and resulting in an error.
	if o.IsAdmin {
		saArray = nil
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
		Scopes:        data.Scopes,
		SubuserAccess: saArray,
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

	sa, err := r.client.GetTeammateSubuserAccess(
		ctx,
		email,
		&sendgrid.InputGetTeammateSubuserAccess{
			Username: email,
		},
	)

	if err != nil {
		resp.Diagnostics.AddError(
			"Reading SSO teammate subuser access",
			fmt.Sprintf("Unable to read SSO teammate subuser access, got error: %s", err),
		)
		return
	}

	scopes := []types.String{}
	if !data.IsAdmin.ValueBool() {
		for _, s := range o.Scopes {
			scopes = append(scopes, types.StringValue(s))
		}
	}
	saArray := fromSendgridSubuserAccessArray(sa.SubuserAccess)

	// NOTE: The teammate read API returns scopes even if the subuser access is set,
	//       causing a discrepancy with the scopes specified in the resource and resulting in an error.
	if len(sa.SubuserAccess) > 0 {
		scopes = nil
	}
	// NOTE: The teammate read API returns subuser access with admin permissions to all subusers when the user is admin,
	//       causing a discrepancy with the subuser access specified in the resource and resulting in an error.
	if o.IsAdmin {
		saArray = nil
		scopes = nil
	}

	data = ssoTeammateResourceModel{
		ID:            types.StringValue(o.Email),
		Email:         types.StringValue(o.Email),
		IsAdmin:       types.BoolValue(o.IsAdmin),
		Username:      types.StringValue(o.Username),
		FirstName:     types.StringValue(o.FirstName),
		LastName:      types.StringValue(o.LastName),
		Scopes:        scopes,
		SubuserAccess: saArray,
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
		IsAdmin:                    data.IsAdmin.ValueBool(),
		Scopes:                     scopes,
		FirstName:                  data.FirstName.ValueString(),
		LastName:                   data.LastName.ValueString(),
		HasRestrictedSubuserAccess: len(data.SubuserAccess) > 0,
		SubuserAccess:              toInputSubuserAccessArray(data.SubuserAccess),
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
	saArray := fromOutputSubuserAccessArray(o.SubuserAccess)

	// NOTE: The teammate read API returns scopes even if the subuser access is set,
	//       causing a discrepancy with the scopes specified in the resource and resulting in an error.
	if len(o.SubuserAccess) > 0 {
		scopesSet = nil
	}
	// NOTE: The teammate read API returns subuser access with admin permissions to all subusers when the user is admin,
	//       causing a discrepancy with the subuser access specified in the resource and resulting in an error.
	if o.IsAdmin {
		saArray = nil
		scopesSet = nil
	}

	data = ssoTeammateResourceModel{
		ID:            types.StringValue(o.Email),
		Email:         types.StringValue(o.Email),
		IsAdmin:       types.BoolValue(o.IsAdmin),
		Username:      types.StringValue(o.Username),
		Scopes:        scopesSet,
		FirstName:     types.StringValue(o.FirstName),
		LastName:      types.StringValue(o.LastName),
		SubuserAccess: saArray,
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

	_, err := retryOnRateLimit(ctx, func() (interface{}, error) {
		return nil, r.client.DeleteTeammate(ctx, email)
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Deleting SSO teammate",
			fmt.Sprintf(
				"Could not delete SSO teammate %s, unexpected error: %s",
				email,
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

	teammate, err := r.client.GetTeammate(ctx, email)
	if err != nil {
		resp.Diagnostics.AddError(
			"Importing SSO teammate",
			fmt.Sprintf("Unable to read SSO teammate, got error: %s", err),
		)
		return
	}

	sa, err := r.client.GetTeammateSubuserAccess(
		ctx,
		email,
		&sendgrid.InputGetTeammateSubuserAccess{
			Username: email,
		},
	)

	if err != nil {
		resp.Diagnostics.AddError(
			"Reading SSO teammate subuser access",
			fmt.Sprintf("Unable to read SSO teammate subuser access, got error: %s", err),
		)
		return
	}

	scopes := []types.String{}
	for _, s := range teammate.Scopes {
		scopes = append(scopes, types.StringValue(s))
	}
	saArray := fromSendgridSubuserAccessArray(sa.SubuserAccess)

	// NOTE: The teammate read API returns scopes even if the subuser access is set,
	//       causing a discrepancy with the scopes specified in the resource and resulting in an error.
	if len(sa.SubuserAccess) > 0 {
		scopes = nil
	}
	// NOTE: The teammate read API returns subuser access with admin permissions to all subusers when the user is admin,
	//       causing a discrepancy with the subuser access specified in the resource and resulting in an error.
	if teammate.IsAdmin {
		saArray = nil
	}

	data = ssoTeammateResourceModel{
		ID:            types.StringValue(teammate.Email),
		Email:         types.StringValue(teammate.Email),
		IsAdmin:       types.BoolValue(teammate.IsAdmin),
		Username:      types.StringValue(teammate.Username),
		Scopes:        scopes,
		FirstName:     types.StringValue(teammate.FirstName),
		LastName:      types.StringValue(teammate.LastName),
		SubuserAccess: saArray,
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}
