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
var _ resource.Resource = &eventWebhookResource{}
var _ resource.ResourceWithImportState = &eventWebhookResource{}

func newEventWebhookResource() resource.Resource {
	return &eventWebhookResource{}
}

type eventWebhookResource struct {
	client *sendgrid.Client
}

type eventWebhookResourceModel struct {
	ID                types.String `tfsdk:"id"`
	Enabled           types.Bool   `tfsdk:"enabled"`
	URL               types.String `tfsdk:"url"`
	GroupResubscribe  types.Bool   `tfsdk:"group_resubscribe"`
	Delivered         types.Bool   `tfsdk:"delivered"`
	GroupUnsubscribe  types.Bool   `tfsdk:"group_unsubscribe"`
	SpamReport        types.Bool   `tfsdk:"spam_report"`
	Bounce            types.Bool   `tfsdk:"bounce"`
	Deferred          types.Bool   `tfsdk:"deferred"`
	Unsubscribe       types.Bool   `tfsdk:"unsubscribe"`
	Processed         types.Bool   `tfsdk:"processed"`
	Open              types.Bool   `tfsdk:"open"`
	Click             types.Bool   `tfsdk:"click"`
	Dropped           types.Bool   `tfsdk:"dropped"`
	FriendlyName      types.String `tfsdk:"friendly_name"`
	OAuthClientID     types.String `tfsdk:"oauth_client_id"`
	OAuthClientSecret types.String `tfsdk:"oauth_client_secret"`
	OAuthTokenURL     types.String `tfsdk:"oauth_token_url"`
}

func (r *eventWebhookResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_event_webhook"
}

func (r *eventWebhookResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
The ​​SendGrid Event Webhook sends email event data as SendGrid processes it. This means you can receive data in nearly real-time, making it ideal to integrate with logging or monitoring systems.
Because the Event Webhook delivers data to your systems, it is also well-suited to backing up and storing event data within your infrastructure to meet your own data access and retention needs.
		`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of Event Webhook",
				Computed:            true,
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Set this property to true to enable the Event Webhook or false to disable it.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"url": schema.StringAttribute{
				MarkdownDescription: "Set this property to the URL where you want the Event Webhook to send event data.",
				Required:            true,
			},
			"group_resubscribe": schema.BoolAttribute{
				MarkdownDescription: "Set this property to true to receive group resubscribe events. Group resubscribes occur when recipients resubscribe to a specific unsubscribe group by updating their subscription preferences. You must enable Subscription Tracking to receive this type of event.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"delivered": schema.BoolAttribute{
				MarkdownDescription: "Set this property to true to receive delivered events. Delivered events occur when a message has been successfully delivered to the receiving server.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"group_unsubscribe": schema.BoolAttribute{
				MarkdownDescription: "Set this property to true to receive group unsubscribe events. Group unsubscribes occur when recipients unsubscribe from a specific unsubscribe group either by direct link or by updating their subscription preferences. You must enable Subscription Tracking to receive this type of event.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"spam_report": schema.BoolAttribute{
				MarkdownDescription: "Set this property to true to receive spam report events. Spam reports occur when recipients mark a message as spam.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"bounce": schema.BoolAttribute{
				MarkdownDescription: "Set this property to true to receive bounce events. A bounce occurs when a receiving server could not or would not accept a message.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"deferred": schema.BoolAttribute{
				MarkdownDescription: "Set this property to true to receive deferred events. Deferred events occur when a recipient's email server temporarily rejects a message.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"unsubscribe": schema.BoolAttribute{
				MarkdownDescription: "Set this property to true to receive unsubscribe events. Unsubscribes occur when recipients click on a message's subscription management link. You must enable Subscription Tracking to receive this type of event.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"processed": schema.BoolAttribute{
				MarkdownDescription: "Set this property to true to receive processed events. Processed events occur when a message has been received by Twilio SendGrid and the message is ready to be delivered.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"open": schema.BoolAttribute{
				MarkdownDescription: "Set this property to true to receive open events. Open events occur when a recipient has opened the HTML message. You must enable Open Tracking to receive this type of event.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"click": schema.BoolAttribute{
				MarkdownDescription: "Set this property to true to receive click events. Click events occur when a recipient clicks on a link within the message. You must enable Click Tracking to receive this type of event.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"dropped": schema.BoolAttribute{
				MarkdownDescription: "Set this property to true to receive dropped events. Dropped events occur when your message is not delivered by Twilio SendGrid. Dropped events are accompanied by a reason property, which indicates why the message was dropped. Reasons for a dropped message include: Invalid SMTPAPI header, Spam Content (if spam checker app enabled), Unsubscribed Address, Bounced Address, Spam Reporting Address, Invalid, Recipient List over Package Quota.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"friendly_name": schema.StringAttribute{
				MarkdownDescription: "Optionally set this property to a friendly name for the Event Webhook. A friendly name may be assigned to each of your webhooks to help you differentiate them. The friendly name is for convenience only. You should use the webhook id property for any programmatic tasks.",
				Optional:            true,
				Computed:            true,
			},
			"oauth_client_id": schema.StringAttribute{
				MarkdownDescription: "Set this property to the OAuth client ID that SendGrid will pass to your OAuth server or service provider to generate an OAuth access token. When passing data in this property, you must also include the oauth_token_url property.",
				Optional:            true,
				Computed:            true,
			},
			"oauth_client_secret": schema.StringAttribute{
				MarkdownDescription: "Set this property to the OAuth client secret that SendGrid will pass to your OAuth server or service provider to generate an OAuth access token. This secret is needed only once to create an access token. SendGrid will store the secret, allowing you to update your client ID and Token URL without passing the secret to SendGrid again. When passing data in this field, you must also include the oauth_client_id and oauth_token_url properties.",
				Optional:            true,
				Computed:            true,
				Sensitive:           true,
			},
			"oauth_token_url": schema.StringAttribute{
				MarkdownDescription: "Set this property to the URL where SendGrid will send the OAuth client ID and client secret to generate an OAuth access token. This should be your OAuth server or service provider. When passing data in this field, you must also include the oauth_client_id property.",
				Optional:            true,
				Computed:            true,
			},
		},
	}
}

func (r *eventWebhookResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *eventWebhookResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan eventWebhookResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := &sendgrid.InputCreateEventWebhook{
		Enabled:          plan.Enabled.ValueBool(),
		URL:              plan.URL.ValueString(),
		GroupResubscribe: plan.GroupResubscribe.ValueBool(),
		Delivered:        plan.Delivered.ValueBool(),
		GroupUnsubscribe: plan.GroupUnsubscribe.ValueBool(),
		SpamReport:       plan.SpamReport.ValueBool(),
		Bounce:           plan.Bounce.ValueBool(),
		Deferred:         plan.Deferred.ValueBool(),
		Unsubscribe:      plan.Unsubscribe.ValueBool(),
		Processed:        plan.Processed.ValueBool(),
		Open:             plan.Open.ValueBool(),
		Click:            plan.Click.ValueBool(),
		Dropped:          plan.Dropped.ValueBool(),
		FriendlyName:     plan.FriendlyName.ValueString(),
	}
	if !plan.OAuthClientID.IsNull() {
		input.OAuthClientID = plan.OAuthClientID.ValueString()
	}
	if !plan.OAuthClientSecret.IsNull() {
		input.OAuthClientSecret = plan.OAuthClientSecret.ValueString()
	}
	if !plan.OAuthTokenURL.IsNull() {
		input.OAuthTokenURL = plan.OAuthTokenURL.ValueString()
	}

	o, err := r.client.CreateEventWebhook(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Creating event webhook",
			fmt.Sprintf("Unable to create event webhook, got error: %s", err),
		)
		return
	}

	plan = eventWebhookResourceModel{
		ID:               types.StringValue(o.ID),
		Enabled:          types.BoolValue(o.Enabled),
		URL:              types.StringValue(o.URL),
		GroupResubscribe: types.BoolValue(o.GroupResubscribe),
		Delivered:        types.BoolValue(o.Delivered),
		GroupUnsubscribe: types.BoolValue(o.GroupUnsubscribe),
		SpamReport:       types.BoolValue(o.SpamReport),
		Bounce:           types.BoolValue(o.Bounce),
		Deferred:         types.BoolValue(o.Deferred),
		Unsubscribe:      types.BoolValue(o.Unsubscribe),
		Processed:        types.BoolValue(o.Processed),
		Open:             types.BoolValue(o.Open),
		Click:            types.BoolValue(o.Click),
		Dropped:          types.BoolValue(o.Dropped),
		FriendlyName:     types.StringValue(o.FriendlyName),
		OAuthClientID:    types.StringValue(o.OAuthClientID),
		OAuthTokenURL:    types.StringValue(o.OAuthTokenURL),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *eventWebhookResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state eventWebhookResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	o, err := r.client.GetEventWebhook(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Reading eventWebhook",
			fmt.Sprintf("Unable to read eventWebhook, got error: %s", err),
		)
		return
	}

	state = eventWebhookResourceModel{
		ID:               types.StringValue(o.ID),
		Enabled:          types.BoolValue(o.Enabled),
		URL:              types.StringValue(o.URL),
		GroupResubscribe: types.BoolValue(o.GroupResubscribe),
		Delivered:        types.BoolValue(o.Delivered),
		GroupUnsubscribe: types.BoolValue(o.GroupUnsubscribe),
		SpamReport:       types.BoolValue(o.SpamReport),
		Bounce:           types.BoolValue(o.Bounce),
		Deferred:         types.BoolValue(o.Deferred),
		Unsubscribe:      types.BoolValue(o.Unsubscribe),
		Processed:        types.BoolValue(o.Processed),
		Open:             types.BoolValue(o.Open),
		Click:            types.BoolValue(o.Click),
		Dropped:          types.BoolValue(o.Dropped),
		FriendlyName:     types.StringValue(o.FriendlyName),
		OAuthClientID:    types.StringValue(o.OAuthClientID),
		OAuthTokenURL:    types.StringValue(o.OAuthTokenURL),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *eventWebhookResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state eventWebhookResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := &sendgrid.InputUpdateEventWebhook{
		Enabled:          plan.Enabled.ValueBool(),
		URL:              plan.URL.ValueString(),
		GroupResubscribe: plan.GroupResubscribe.ValueBool(),
		Delivered:        plan.Delivered.ValueBool(),
		GroupUnsubscribe: plan.GroupUnsubscribe.ValueBool(),
		SpamReport:       plan.SpamReport.ValueBool(),
		Bounce:           plan.Bounce.ValueBool(),
		Deferred:         plan.Deferred.ValueBool(),
		Unsubscribe:      plan.Unsubscribe.ValueBool(),
		Processed:        plan.Processed.ValueBool(),
		Open:             plan.Open.ValueBool(),
		Click:            plan.Click.ValueBool(),
		Dropped:          plan.Dropped.ValueBool(),
		FriendlyName:     plan.FriendlyName.ValueString(),
	}
	if !plan.OAuthClientID.IsNull() {
		input.OAuthClientID = plan.OAuthClientID.ValueString()
	}
	if !plan.OAuthClientSecret.IsNull() {
		input.OAuthClientSecret = plan.OAuthClientSecret.ValueString()
	}
	if !plan.OAuthTokenURL.IsNull() {
		input.OAuthTokenURL = plan.OAuthTokenURL.ValueString()
	}

	id := state.ID.ValueString()
	o, err := r.client.UpdateEventWebhook(ctx, id, input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Updating event webhook",
			fmt.Sprintf("Unable to update event webhook, got error: %s", err),
		)
		return
	}

	data := eventWebhookResourceModel{
		ID:               types.StringValue(o.ID),
		Enabled:          types.BoolValue(o.Enabled),
		URL:              types.StringValue(o.URL),
		GroupResubscribe: types.BoolValue(o.GroupResubscribe),
		Delivered:        types.BoolValue(o.Delivered),
		GroupUnsubscribe: types.BoolValue(o.GroupUnsubscribe),
		SpamReport:       types.BoolValue(o.SpamReport),
		Bounce:           types.BoolValue(o.Bounce),
		Deferred:         types.BoolValue(o.Deferred),
		Unsubscribe:      types.BoolValue(o.Unsubscribe),
		Processed:        types.BoolValue(o.Processed),
		Open:             types.BoolValue(o.Open),
		Click:            types.BoolValue(o.Click),
		Dropped:          types.BoolValue(o.Dropped),
		FriendlyName:     types.StringValue(o.FriendlyName),
		OAuthClientID:    types.StringValue(o.OAuthClientID),
		OAuthTokenURL:    types.StringValue(o.OAuthTokenURL),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *eventWebhookResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state eventWebhookResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *eventWebhookResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id := req.ID
	o, err := r.client.GetEventWebhook(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Importing event webhook",
			fmt.Sprintf("Unable to read event webhook, got error: %s", err),
		)
		return
	}

	d := eventWebhookResourceModel{
		ID:               types.StringValue(o.ID),
		Enabled:          types.BoolValue(o.Enabled),
		URL:              types.StringValue(o.URL),
		GroupResubscribe: types.BoolValue(o.GroupResubscribe),
		Delivered:        types.BoolValue(o.Delivered),
		GroupUnsubscribe: types.BoolValue(o.GroupUnsubscribe),
		SpamReport:       types.BoolValue(o.SpamReport),
		Bounce:           types.BoolValue(o.Bounce),
		Deferred:         types.BoolValue(o.Deferred),
		Unsubscribe:      types.BoolValue(o.Unsubscribe),
		Processed:        types.BoolValue(o.Processed),
		Open:             types.BoolValue(o.Open),
		Click:            types.BoolValue(o.Click),
		Dropped:          types.BoolValue(o.Dropped),
		FriendlyName:     types.StringValue(o.FriendlyName),
		OAuthClientID:    types.StringValue(o.OAuthClientID),
		OAuthTokenURL:    types.StringValue(o.OAuthTokenURL),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &d)...)
	if resp.Diagnostics.HasError() {
		return
	}
}
