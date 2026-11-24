// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kenzo0107/sendgrid"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &openTrackingSettingsResource{}
var _ resource.ResourceWithImportState = &openTrackingSettingsResource{}

func newOpenTrackingSettingsResource() resource.Resource {
	return &openTrackingSettingsResource{}
}

type openTrackingSettingsResource struct {
	client *sendgrid.Client
}

type openTrackingSettingsResourceModel struct {
	Enabled types.Bool `tfsdk:"enabled"`
}

func (r *openTrackingSettingsResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_open_tracking_settings"
}

func (r *openTrackingSettingsResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
Open Tracking adds an invisible, single pixel image at the end of the body of your emails. When the pixel is loaded by a recipient's email client, SendGrid records that as an [open](https://www.twilio.com/docs/sendgrid/glossary/opens).

SendGrid stores one open tracking setting per account, so declaring more than one of this resource for the same account will make them fight over that single setting. Destroying this resource leaves the setting on the account as it is.
		`,
		Attributes: map[string]schema.Attribute{
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Indicates if open tracking is enabled or disabled.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
		},
	}
}

func (r *openTrackingSettingsResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := clientFromProviderData(req.ProviderData)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *providerData, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *openTrackingSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan openTrackingSettingsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := &sendgrid.InputUpdateOpenTrackingSettings{
		Enabled: plan.Enabled.ValueBool(),
	}
	o, err := r.client.UpdateOpenTrackingSettings(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Creating open tracking settings",
			fmt.Sprintf("Unable to update open tracking settings, got error: %s", err),
		)
		return
	}

	plan = openTrackingSettingsResourceModel{
		Enabled: types.BoolValue(o.Enabled),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *openTrackingSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state openTrackingSettingsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	o, err := r.client.GetOpenTrackingSettings(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Reading open tracking settings",
			fmt.Sprintf("Unable to read open tracking settings, got error: %s", err),
		)
		return
	}

	state = openTrackingSettingsResourceModel{
		Enabled: types.BoolValue(o.Enabled),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *openTrackingSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data openTrackingSettingsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// enabled is always sent (no omitempty), so the request is built from the
	// planned value rather than only from what changed.
	input := &sendgrid.InputUpdateOpenTrackingSettings{
		Enabled: data.Enabled.ValueBool(),
	}

	o, err := r.client.UpdateOpenTrackingSettings(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Updating open tracking settings",
			fmt.Sprintf("Unable to update open tracking settings, got error: %s", err),
		)
		return
	}

	data = openTrackingSettingsResourceModel{
		Enabled: types.BoolValue(o.Enabled),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *openTrackingSettingsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state openTrackingSettingsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *openTrackingSettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var data openTrackingSettingsResourceModel

	o, err := r.client.GetOpenTrackingSettings(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Importing open tracking settings",
			fmt.Sprintf("Unable to read open tracking settings, got error: %s", err),
		)
		return
	}

	data = openTrackingSettingsResourceModel{
		Enabled: types.BoolValue(o.Enabled),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}
