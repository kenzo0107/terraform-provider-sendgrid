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
var _ resource.Resource = &bounceSettingsResource{}
var _ resource.ResourceWithImportState = &bounceSettingsResource{}

func newBounceSettingsResource() resource.Resource {
	return &bounceSettingsResource{}
}

type bounceSettingsResource struct {
	client *sendgrid.Client
}

type bounceSettingsResourceModel struct {
	Enabled     types.Bool  `tfsdk:"enabled"`
	SoftBounces types.Int64 `tfsdk:"soft_bounces"`
	HardBounces types.Int64 `tfsdk:"hard_bounces"`
}

func (r *bounceSettingsResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_bounce_settings"
}

func (r *bounceSettingsResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
Manage bounce settings for your SendGrid account.

Bounce settings allow you to configure how long soft bounces are retained in your suppression list.
Soft bounces are temporary delivery failures, such as a full mailbox or temporary server issues.

The Soft Bounces setting specifies the number of days soft bounces will be kept in your soft bounces suppression list.
Any soft bounces older than this value will be purged.

For more information, see the [SendGrid Mail Settings Guide](https://support.sendgrid.com/hc/en-us/articles/9489871931803-Mail-Settings-Guide-within-a-SendGrid-Account).
		`,
		Attributes: map[string]schema.Attribute{
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Indicates if the Bounce Purge mail setting is enabled.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"soft_bounces": schema.Int64Attribute{
				MarkdownDescription: "The number of days after which SendGrid will purge all contacts from your soft bounces suppression lists. Must be between 1 and 3650 days.",
				Optional:            true,
				Computed:            true,
			},
			"hard_bounces": schema.Int64Attribute{
				MarkdownDescription: "The number of days after which SendGrid will purge all contacts from your hard bounces suppression lists. Must be between 1 and 3650 days.",
				Optional:            true,
				Computed:            true,
			},
		},
	}
}

func (r *bounceSettingsResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *bounceSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan bounceSettingsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := &sendgrid.InputUpdateBounceSettings{
		Enabled:     plan.Enabled.ValueBool(),
		SoftBounces: plan.SoftBounces.ValueInt64(),
		HardBounces: plan.HardBounces.ValueInt64(),
	}

	res, err := retryOnRateLimit(ctx, func() (interface{}, error) {
		return r.client.UpdateBounceSettings(ctx, input)
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Creating bounce settings",
			fmt.Sprintf("Unable to update bounce settings, got error: %s", err),
		)
		return
	}

	o, ok := res.(*sendgrid.OutputUpdateBounceSettings)
	if !ok {
		resp.Diagnostics.AddError(
			"Creating bounce settings",
			"Failed to assert type *sendgrid.OutputUpdateBounceSettings",
		)
		return
	}

	plan = bounceSettingsResourceModel{
		Enabled:     types.BoolValue(o.Enabled),
		SoftBounces: types.Int64Value(o.SoftBounces),
		HardBounces: types.Int64Value(o.HardBounces),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *bounceSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state bounceSettingsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	res, err := retryOnRateLimit(ctx, func() (interface{}, error) {
		return r.client.GetBounceSettings(ctx)
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Reading bounce settings",
			fmt.Sprintf("Unable to read bounce settings, got error: %s", err),
		)
		return
	}

	o, ok := res.(*sendgrid.OutputGetBounceSettings)
	if !ok {
		resp.Diagnostics.AddError(
			"Reading bounce settings",
			"Failed to assert type *sendgrid.OutputGetBounceSettings",
		)
		return
	}

	state = bounceSettingsResourceModel{
		SoftBounces: types.Int64Value(o.SoftBounces),
		HardBounces: types.Int64Value(o.HardBounces),
		Enabled:     types.BoolValue(o.Enabled),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *bounceSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, state bounceSettingsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := &sendgrid.InputUpdateBounceSettings{
		Enabled:     data.Enabled.ValueBool(),
		SoftBounces: data.SoftBounces.ValueInt64(),
		HardBounces: data.HardBounces.ValueInt64(),
	}
	res, err := retryOnRateLimit(ctx, func() (interface{}, error) {
		return r.client.UpdateBounceSettings(ctx, input)
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Updating bounce settings",
			fmt.Sprintf("Unable to update bounce settings, got error: %s", err),
		)
		return
	}

	o, ok := res.(*sendgrid.OutputUpdateBounceSettings)
	if !ok {
		resp.Diagnostics.AddError(
			"Updating bounce settings",
			"Failed to assert type *sendgrid.OutputUpdateBounceSettings",
		)
		return
	}

	data = bounceSettingsResourceModel{
		Enabled:     types.BoolValue(o.Enabled),
		SoftBounces: types.Int64Value(o.SoftBounces),
		HardBounces: types.Int64Value(o.HardBounces),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *bounceSettingsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state bounceSettingsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// For bounce settings, we can't really "delete" them, but we can reset to a reasonable default
	// Let's reset to 7 days as a default value
	input := &sendgrid.InputUpdateBounceSettings{
		Enabled: false,
	}
	_, err := retryOnRateLimit(ctx, func() (interface{}, error) {
		return r.client.UpdateBounceSettings(ctx, input)
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Deleting bounce settings",
			fmt.Sprintf("Unable to reset bounce settings, got error: %s", err),
		)
		return
	}
}

func (r *bounceSettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var data bounceSettingsResourceModel

	res, err := retryOnRateLimit(ctx, func() (interface{}, error) {
		return r.client.GetBounceSettings(ctx)
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Importing bounce settings",
			fmt.Sprintf("Unable to read bounce settings, got error: %s", err),
		)
		return
	}

	o, ok := res.(*sendgrid.OutputGetBounceSettings)
	if !ok {
		resp.Diagnostics.AddError(
			"Importing bounce settings",
			"Failed to assert type *sendgrid.OutputGetBounceSettings",
		)
		return
	}

	data = bounceSettingsResourceModel{
		Enabled:     types.BoolValue(o.Enabled),
		SoftBounces: types.Int64Value(o.SoftBounces),
		HardBounces: types.Int64Value(o.HardBounces),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}
