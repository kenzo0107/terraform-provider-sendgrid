// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
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
	client         *sendgrid.Client
	extendedClient ClientWithBounceSettings
}

type bounceSettingsResourceModel struct {
	SoftBouncePurgeDays types.Int64 `tfsdk:"soft_bounce_purge_days"`
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
			"soft_bounce_purge_days": schema.Int64Attribute{
				MarkdownDescription: "The number of days soft bounces will be kept in your soft bounces suppression list. Any soft bounces older than this value will be purged. Must be between 1 and 3650 days.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(7), // Default to 7 days as a reasonable default
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

	// Get API key from environment for extended functionality
	apiKey := os.Getenv("SENDGRID_API_KEY")
	if apiKey == "" {
		resp.Diagnostics.AddError(
			"Missing SendGrid API Key",
			"The bounce settings resource requires the SENDGRID_API_KEY environment variable to be set.",
		)
		return
	}

	r.extendedClient = ExtendClient(client, apiKey)
}

func (r *bounceSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan bounceSettingsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate the value is within acceptable range
	days := plan.SoftBouncePurgeDays.ValueInt64()
	if days < 1 || days > 3650 {
		resp.Diagnostics.AddError(
			"Invalid soft_bounce_purge_days value",
			"soft_bounce_purge_days must be between 1 and 3650 days",
		)
		return
	}

	input := &InputUpdateBounceSettings{
		SoftBouncePurgeDays: plan.SoftBouncePurgeDays.ValueInt64(),
	}

	res, err := retryOnRateLimit(ctx, func() (interface{}, error) {
		return r.extendedClient.UpdateBounceSettings(ctx, input)
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Creating bounce settings",
			fmt.Sprintf("Unable to update bounce settings, got error: %s", err),
		)
		return
	}

	o, ok := res.(*OutputUpdateBounceSettings)
	if !ok {
		resp.Diagnostics.AddError(
			"Creating bounce settings",
			"Failed to assert type *OutputUpdateBounceSettings",
		)
		return
	}

	plan = bounceSettingsResourceModel{
		SoftBouncePurgeDays: types.Int64Value(o.SoftBouncePurgeDays),
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
		return r.extendedClient.GetBounceSettings(ctx)
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Reading bounce settings",
			fmt.Sprintf("Unable to read bounce settings, got error: %s", err),
		)
		return
	}

	o, ok := res.(*OutputGetBounceSettings)
	if !ok {
		resp.Diagnostics.AddError(
			"Reading bounce settings",
			"Failed to assert type *OutputGetBounceSettings",
		)
		return
	}

	state = bounceSettingsResourceModel{
		SoftBouncePurgeDays: types.Int64Value(o.SoftBouncePurgeDays),
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

	// Validate the value is within acceptable range
	days := data.SoftBouncePurgeDays.ValueInt64()
	if days < 1 || days > 3650 {
		resp.Diagnostics.AddError(
			"Invalid soft_bounce_purge_days value",
			"soft_bounce_purge_days must be between 1 and 3650 days",
		)
		return
	}

	input := &InputUpdateBounceSettings{}
	if !data.SoftBouncePurgeDays.IsNull() && data.SoftBouncePurgeDays.ValueInt64() != state.SoftBouncePurgeDays.ValueInt64() {
		input.SoftBouncePurgeDays = data.SoftBouncePurgeDays.ValueInt64()
	}

	res, err := retryOnRateLimit(ctx, func() (interface{}, error) {
		return r.extendedClient.UpdateBounceSettings(ctx, input)
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Updating bounce settings",
			fmt.Sprintf("Unable to update bounce settings, got error: %s", err),
		)
		return
	}

	o, ok := res.(*OutputUpdateBounceSettings)
	if !ok {
		resp.Diagnostics.AddError(
			"Updating bounce settings",
			"Failed to assert type *OutputUpdateBounceSettings",
		)
		return
	}

	data = bounceSettingsResourceModel{
		SoftBouncePurgeDays: types.Int64Value(o.SoftBouncePurgeDays),
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
	input := &InputUpdateBounceSettings{
		SoftBouncePurgeDays: 7,
	}

	_, err := retryOnRateLimit(ctx, func() (interface{}, error) {
		return r.extendedClient.UpdateBounceSettings(ctx, input)
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
		return r.extendedClient.GetBounceSettings(ctx)
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Importing bounce settings",
			fmt.Sprintf("Unable to read bounce settings, got error: %s", err),
		)
		return
	}

	o, ok := res.(*OutputGetBounceSettings)
	if !ok {
		resp.Diagnostics.AddError(
			"Importing bounce settings",
			"Failed to assert type *OutputGetBounceSettings",
		)
		return
	}

	data = bounceSettingsResourceModel{
		SoftBouncePurgeDays: types.Int64Value(o.SoftBouncePurgeDays),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}
