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
var _ resource.Resource = &clickTrackingSettingsResource{}
var _ resource.ResourceWithImportState = &clickTrackingSettingsResource{}

func newClickTrackingSettingsResource() resource.Resource {
	return &clickTrackingSettingsResource{}
}

type clickTrackingSettingsResource struct {
	client *sendgrid.Client
}

type clickTrackingSettingsResourceModel struct {
	Enabled    types.Bool `tfsdk:"enabled"`
	EnableText types.Bool `tfsdk:"enable_text"`
}

func (r *clickTrackingSettingsResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_click_tracking_settings"
}

func (r *clickTrackingSettingsResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
Click Tracking overrides all the links and URLs in your emails and points them to either SendGrid’s servers or the domain with which you branded your link. When a customer clicks a link, SendGrid tracks those [clicks](https://www.twilio.com/docs/sendgrid/glossary/clicks).

Click tracking helps you understand how users are engaging with your communications. SendGrid can track up to 1000 links per email
		`,
		Attributes: map[string]schema.Attribute{
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Indicates if click tracking is enabled or disabled.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"enable_text": schema.BoolAttribute{
				MarkdownDescription: "Indicates if click tracking is enabled for plain text emails.",
				Computed:            true,
			},
		},
	}
}

func (r *clickTrackingSettingsResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *clickTrackingSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan clickTrackingSettingsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := &sendgrid.InputUpdateClickTrackingSettings{
		Enabled: plan.Enabled.ValueBool(),
	}
	o, err := r.client.UpdateClickTrackingSettings(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Creating click tracking settings",
			fmt.Sprintf("Unable to update click tracking settings, got error: %s", err),
		)
		return
	}

	plan = clickTrackingSettingsResourceModel{
		Enabled:    types.BoolValue(o.Enabled),
		EnableText: types.BoolValue(o.EnableText),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *clickTrackingSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state clickTrackingSettingsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	o, err := r.client.GetClickTrackingSettings(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Reading click tracking settings",
			fmt.Sprintf("Unable to read click tracking settings, got error: %s", err),
		)
		return
	}

	state = clickTrackingSettingsResourceModel{
		Enabled:    types.BoolValue(o.Enabled),
		EnableText: types.BoolValue(o.EnableText),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *clickTrackingSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, state clickTrackingSettingsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := &sendgrid.InputUpdateClickTrackingSettings{}
	if !data.Enabled.IsNull() && data.Enabled.ValueBool() != state.Enabled.ValueBool() {
		input.Enabled = data.Enabled.ValueBool()
	}

	o, err := r.client.UpdateClickTrackingSettings(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Updating click tracking settings",
			fmt.Sprintf("Unable to update click tracking settings, got error: %s", err),
		)
		return
	}

	data = clickTrackingSettingsResourceModel{
		Enabled:    types.BoolValue(o.Enabled),
		EnableText: types.BoolValue(o.EnableText),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *clickTrackingSettingsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state clickTrackingSettingsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *clickTrackingSettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var data clickTrackingSettingsResourceModel

	o, err := r.client.GetClickTrackingSettings(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Importing click tracking settings",
			fmt.Sprintf("Unable to read click tracking settings, got error: %s", err),
		)
		return
	}

	data = clickTrackingSettingsResourceModel{
		Enabled:    types.BoolValue(o.Enabled),
		EnableText: types.BoolValue(o.EnableText),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}
