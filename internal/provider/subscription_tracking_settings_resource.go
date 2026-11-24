// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kenzo0107/sendgrid"
)

var _ resource.Resource = &subscriptionTrackingSettingsResource{}
var _ resource.ResourceWithImportState = &subscriptionTrackingSettingsResource{}

func newSubscriptionTrackingSettingsResource() resource.Resource {
	return &subscriptionTrackingSettingsResource{}
}

type subscriptionTrackingSettingsResource struct {
	client *sendgrid.Client
}

type subscriptionTrackingSettingsResourceModel struct {
	Enabled      types.Bool   `tfsdk:"enabled"`
	HTMLContent  types.String `tfsdk:"html_content"`
	PlainContent types.String `tfsdk:"plain_content"`
	Landing      types.String `tfsdk:"landing"`
	Replace      types.String `tfsdk:"replace"`
	URL          types.String `tfsdk:"url"`
}

func newSubscriptionTrackingSettingsResourceModel(o *sendgrid.OutputGetSubscriptionTrackingSettings) subscriptionTrackingSettingsResourceModel {
	return subscriptionTrackingSettingsResourceModel{
		Enabled:      types.BoolValue(o.Enabled),
		HTMLContent:  types.StringValue(o.HTMLContent),
		PlainContent: types.StringValue(o.PlainContent),
		Landing:      types.StringValue(o.Landing),
		Replace:      types.StringValue(o.Replace),
		URL:          types.StringValue(o.URL),
	}
}

var subscriptionTrackingSettingsDescriptions = map[string]string{
	"enabled":       "Indicates if subscription tracking is enabled or disabled.",
	"html_content":  "The HTML of the unsubscribe link added to HTML emails. Put the link text inside `<% %>`, for example `<% Unsubscribe %>`.",
	"plain_content": "The text of the unsubscribe link added to plain text emails. Include `<% %>`, which is replaced with the unsubscribe URL.",
	"landing":       "The HTML of the page that SendGrid shows after a recipient unsubscribes. Used when `url` is empty.",
	"replace":       "A tag in your email content to replace with the unsubscribe link, so you can place it anywhere instead of at the end of the email.",
	"url":           "The URL of your own unsubscribe page. When it is set, recipients are sent there instead of the `landing` page.",
}

func (r *subscriptionTrackingSettingsResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_subscription_tracking_settings"
}

func (r *subscriptionTrackingSettingsResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	str := func(name string) schema.StringAttribute {
		return schema.StringAttribute{
			MarkdownDescription: subscriptionTrackingSettingsDescriptions[name] + " Defaults to `\"\"`, so omitting it clears the value on the account.",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(""),
		}
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: `
Subscription tracking adds an unsubscribe link to the end of your emails, so recipients can unsubscribe from all of your emails. See [Subscription Tracking](https://www.twilio.com/docs/sendgrid/ui/account-and-settings/tracking#subscription-tracking).

SendGrid stores one subscription tracking setting per account, so declaring more than one of this resource for the same account will make them fight over that single setting. Destroying this resource leaves the setting on the account as it is.
		`,
		Attributes: map[string]schema.Attribute{
			"enabled": schema.BoolAttribute{
				MarkdownDescription: subscriptionTrackingSettingsDescriptions["enabled"] + " Defaults to `false`, so omitting it turns subscription tracking off for the account.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"html_content":  str("html_content"),
			"plain_content": str("plain_content"),
			"landing":       str("landing"),
			"replace":       str("replace"),
			"url":           str("url"),
		},
	}
}

func (r *subscriptionTrackingSettingsResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// Not sending only the changed attributes: a string left out of the request
// keeps its old value on SendGrid instead of being cleared.
func (r *subscriptionTrackingSettingsResource) update(ctx context.Context, plan subscriptionTrackingSettingsResourceModel) (subscriptionTrackingSettingsResourceModel, error) {
	o, err := r.client.UpdateSubscriptionTrackingSettings(ctx, &sendgrid.InputUpdateSubscriptionTrackingSettings{
		Enabled:      plan.Enabled.ValueBool(),
		HTMLContent:  sendgrid.String(plan.HTMLContent.ValueString()),
		PlainContent: sendgrid.String(plan.PlainContent.ValueString()),
		Landing:      sendgrid.String(plan.Landing.ValueString()),
		Replace:      sendgrid.String(plan.Replace.ValueString()),
		URL:          sendgrid.String(plan.URL.ValueString()),
	})
	if err != nil {
		return subscriptionTrackingSettingsResourceModel{}, err
	}

	return newSubscriptionTrackingSettingsResourceModel((*sendgrid.OutputGetSubscriptionTrackingSettings)(o)), nil
}

func (r *subscriptionTrackingSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan subscriptionTrackingSettingsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state, err := r.update(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Creating subscription tracking settings",
			fmt.Sprintf("Unable to update subscription tracking settings, got error: %s", err),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *subscriptionTrackingSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	o, err := r.client.GetSubscriptionTrackingSettings(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Reading subscription tracking settings",
			fmt.Sprintf("Unable to read subscription tracking settings, got error: %s", err),
		)
		return
	}

	state := newSubscriptionTrackingSettingsResourceModel(o)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *subscriptionTrackingSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan subscriptionTrackingSettingsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state, err := r.update(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Updating subscription tracking settings",
			fmt.Sprintf("Unable to update subscription tracking settings, got error: %s", err),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Not disabling the setting on destroy: SendGrid has no "unset" state to go
// back to, and the previous value is unknown, so the account is left as is.
func (r *subscriptionTrackingSettingsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
}

func (r *subscriptionTrackingSettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	o, err := r.client.GetSubscriptionTrackingSettings(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Importing subscription tracking settings",
			fmt.Sprintf("Unable to read subscription tracking settings, got error: %s", err),
		)
		return
	}

	state := newSubscriptionTrackingSettingsResourceModel(o)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
