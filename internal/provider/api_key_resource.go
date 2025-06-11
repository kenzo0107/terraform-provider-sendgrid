// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kenzo0107/sendgrid"
	"github.com/kenzo0107/terraform-provider-sendgrid/flex"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &apiKeyResource{}
var _ resource.ResourceWithImportState = &apiKeyResource{}

func newAPIKeyResource() resource.Resource {
	return &apiKeyResource{}
}

type apiKeyResource struct {
	client *sendgrid.Client
}

type apiKeyResourceModel struct {
	ID     types.String `tfsdk:"id"`
	Name   types.String `tfsdk:"name"`
	Scopes types.Set    `tfsdk:"scopes"`
	APIKey types.String `tfsdk:"api_key"`
}

func (r *apiKeyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_key"
}

func (r *apiKeyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
Your application, mail client, or website can all use API (Application Programming Interface) keys to authenticate access to SendGrid services. You can revoke an API key at any time without having to change your username and password, and an API key can be scoped to perform a limited number of actions.

Provides a API Key resource.

For more detailed information, please see the [SendGrid documentation](https://docs.sendgrid.com/ui/account-and-settings/api-keys).
		`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of API Key",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of API Key",
				Required:            true,
			},
			"scopes": schema.SetAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "The permissions API Key has access to",
				Optional:            true,
			},
			"api_key": schema.StringAttribute{
				MarkdownDescription: "API Key. NOTE: If imported, you cannot set the value of the API key. This is because the API key is issued only during the creation process.",
				Computed:            true,
				Sensitive:           true,
			},
		},
	}
}

func (r *apiKeyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *apiKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan apiKeyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	scopes := flex.ExpandFrameworkStringSet(ctx, plan.Scopes)

	// NOTE: Re-execute after the re-executable time has elapsed when a rate limit occurs
	res, err := retryOnRateLimit(ctx, func() (interface{}, error) {
		return r.client.CreateAPIKey(ctx, &sendgrid.InputCreateAPIKey{
			Name:   plan.Name.ValueString(),
			Scopes: scopes,
		})
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Creating api key",
			fmt.Sprintf("Unable to create api key, got error: %s", err),
		)
		return
	}

	o, ok := res.(*sendgrid.OutputCreateAPIKey)
	if !ok {
		resp.Diagnostics.AddError(
			"Creating api key",
			"Failed to assert type *sendgrid.OutputCreateAPIKey",
		)
		return
	}

	scopesSet, d := types.SetValueFrom(ctx, types.StringType, o.Scopes)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan = apiKeyResourceModel{
		ID:     types.StringValue(o.ApiKeyId),
		Name:   types.StringValue(o.Name),
		Scopes: scopesSet,
		APIKey: types.StringValue(o.ApiKey),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *apiKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state apiKeyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	if id == "" {
		resp.State.RemoveResource(ctx)
		return
	}

	o, err := r.client.GetAPIKey(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Reading api key",
			fmt.Sprintf("Unable to read api key (id: %s), got error: %s", id, err),
		)
		return
	}

	scopes, d := types.SetValueFrom(ctx, types.StringType, o.Scopes)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.ID = types.StringValue(o.ApiKeyId)
	state.Name = types.StringValue(o.Name)
	state.Scopes = scopes
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *apiKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, state apiKeyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()

	var scopes []string
	if !reflect.DeepEqual(data.Scopes, state.Scopes) {
		scopes = flex.ExpandFrameworkStringSet(ctx, data.Scopes)
	}

	data.ID = types.StringValue(id)
	data.APIKey = state.APIKey

	if len(scopes) > 0 {
		// update name and scopes
		o, err := r.client.UpdateAPIKeyNameAndScopes(ctx, id, &sendgrid.InputUpdateAPIKeyNameAndScopes{
			Name:   data.Name.ValueString(),
			Scopes: scopes,
		})
		if err != nil {
			resp.Diagnostics.AddError(
				"Updating api key",
				fmt.Sprintf("Unable to update api key's permissions and name, got error: %s", err),
			)
			return
		}
		data.Name = types.StringValue(o.Name)
		s, d := types.SetValueFrom(ctx, types.StringType, scopes)
		data.Scopes = s
		resp.Diagnostics.Append(d...)
	} else {
		// update name only
		o, err := r.client.UpdateAPIKeyName(ctx, id, &sendgrid.InputUpdateAPIKeyName{
			Name: data.Name.ValueString(),
		})
		if err != nil {
			resp.Diagnostics.AddError(
				"Updating api key",
				fmt.Sprintf("Unable to update api key's name, got error: %s", err),
			)
			return
		}
		data.Name = types.StringValue(o.Name)
		data.Scopes = state.Scopes
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *apiKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state apiKeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()

	// NOTE: Re-execute after the re-executable time has elapsed when a rate limit occurs
	_, err := retryOnRateLimit(ctx, func() (interface{}, error) {
		return nil, r.client.DeleteAPIKey(ctx, id)
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Deleting api key",
			fmt.Sprintf("Unable to delete api key (id: %s), got error: %s", id, err),
		)
		return
	}
}

func (r *apiKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var data apiKeyResourceModel

	id := req.ID

	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)

	o, err := r.client.GetAPIKey(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Importing api key",
			fmt.Sprintf("Unable to read api key, got error: %s", err),
		)
		return
	}

	scopes, d := types.SetValueFrom(ctx, types.StringType, o.Scopes)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	// NOTE: cannot set ApiKey because sendgrid api cannot get api key
	data = apiKeyResourceModel{
		ID:     types.StringValue(o.ApiKeyId),
		Name:   types.StringValue(o.Name),
		Scopes: scopes,
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}
