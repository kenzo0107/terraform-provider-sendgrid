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

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &inboundParseWebhookResource{}
var _ resource.ResourceWithImportState = &inboundParseWebhookResource{}

func newInboundParseWebhookResource() resource.Resource {
	return &inboundParseWebhookResource{}
}

type inboundParseWebhookResource struct {
	client *sendgrid.Client
}

type inboundParseWebhookResourceModel struct {
	Hostname  types.String `tfsdk:"hostname"`
	URL       types.String `tfsdk:"url"`
	SpamCheck types.Bool   `tfsdk:"spam_check"`
	SendRaw   types.Bool   `tfsdk:"send_raw"`
}

func (r *inboundParseWebhookResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_inbound_parse_webhook"
}

func (r *inboundParseWebhookResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
Twilio SendGrid’s Inbound Parse Webhook allows you to receive emails as multipart/form-data at a URL of your choosing. SendGrid will grab the content, attachments, and the headers from any email it receives for your specified hostname.

See "Setting up the Inbound Parse Webhook" for help configuring the Webhook. You can also manage the Inbound Parse Webhook in the Twilio SendGrid App.

To begin processing email using SendGrid's Inbound Parse Webhook, you will have to setup MX Records, choose the hostname (or receiving domain) that will be receiving the emails you want to parse, and define the URL where you want to POST your parsed emails. If you do not have access to your domain's DNS records, you must work with someone in your organization who does.
		`,
		Attributes: map[string]schema.Attribute{
			"hostname": schema.StringAttribute{
				MarkdownDescription: "A specific and unique domain or subdomain that you have created to use exclusively to parse your incoming email. For example, `parse.yourdomain.com`.",
				Required:            true,
			},
			"url": schema.StringAttribute{
				MarkdownDescription: "The public URL where you would like SendGrid to POST the data parsed from your email. Any emails sent with the given hostname provided (whose MX records have been updated to point to SendGrid) will be parsed and POSTed to this URL.",
				Required:            true,
			},
			"spam_check": schema.BoolAttribute{
				MarkdownDescription: "Indicates if you would like SendGrid to check the content parsed from your emails for spam before POSTing them to your domain. (Default: `false`)",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"send_raw": schema.BoolAttribute{
				MarkdownDescription: "Indicates if you would like SendGrid to post the original MIME-type content of your parsed email. When this parameter is set to true, SendGrid will send a JSON payload of the content of your email. (Default: `false`)",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
		},
	}
}

func (r *inboundParseWebhookResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *inboundParseWebhookResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan inboundParseWebhookResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := &sendgrid.InputCreateInboundParseWebhook{
		Hostname:  plan.Hostname.ValueString(),
		URL:       plan.URL.ValueString(),
		SpamCheck: plan.SpamCheck.ValueBool(),
		SendRaw:   plan.SendRaw.ValueBool(),
	}

	res, err := retryOnRateLimit(ctx, func() (interface{}, error) {
		return r.client.CreateInboundParseWebhook(context.TODO(), input)
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Creating inbound parse webhook",
			fmt.Sprintf("Unable to create inbound parse webhook, got error: %s", err),
		)
		return
	}

	o, ok := res.(*sendgrid.OutputCreateInboundParseWebhook)
	if !ok {
		resp.Diagnostics.AddError(
			"Creating inbound parse webhook",
			"Failed to assert type *sendgrid.OutputCreateInboundParseWebhook",
		)
		return
	}

	plan = inboundParseWebhookResourceModel{
		Hostname:  types.StringValue(o.Hostname),
		SpamCheck: types.BoolValue(o.SpamCheck),
		SendRaw:   types.BoolValue(o.SendRaw),

		// NOTE: Immediately after creation, the URL cannot be obtained, but since it is actually set,
		//       the value set in plan will be used.
		//       The API documentation specifies that a URL be returned, but the implementation seems to be different.
		//       see: https://docs.sendgrid.com/api-reference/settings-inbound-parse/create-a-parse-setting
		URL: plan.URL,
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *inboundParseWebhookResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state inboundParseWebhookResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	hostname := state.Hostname.ValueString()
	o, err := r.client.GetInboundParseWebhook(ctx, hostname)
	if err != nil {
		resp.Diagnostics.AddError(
			"Reading inbound parse webhook",
			fmt.Sprintf("Unable to read inbound parse webhook, got error: %s", err),
		)
		return
	}

	state = inboundParseWebhookResourceModel{
		Hostname:  types.StringValue(o.Hostname),
		URL:       types.StringValue(o.URL),
		SpamCheck: types.BoolValue(o.SpamCheck),
		SendRaw:   types.BoolValue(o.SendRaw),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *inboundParseWebhookResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state inboundParseWebhookResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := &sendgrid.InputUpdateInboundParseWebhook{}
	if !plan.URL.IsNull() {
		input.URL = plan.URL.ValueString()
	}
	if !plan.SpamCheck.IsNull() {
		input.SpamCheck = plan.SpamCheck.ValueBool()
	}
	if !plan.SendRaw.IsNull() {
		input.SendRaw = plan.SendRaw.ValueBool()
	}

	hostname := state.Hostname.ValueString()
	o, err := r.client.UpdateInboundParseWebhook(ctx, hostname, input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Updating inbound parse webhook",
			fmt.Sprintf("Unable to update inbound parse webhook, got error: %s", err),
		)
		return
	}

	data := inboundParseWebhookResourceModel{
		Hostname:  types.StringValue(o.Hostname),
		URL:       types.StringValue(o.URL),
		SpamCheck: types.BoolValue(o.SpamCheck),
		SendRaw:   types.BoolValue(o.SendRaw),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *inboundParseWebhookResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data inboundParseWebhookResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	hostname := data.Hostname.ValueString()
	_, err := retryOnRateLimit(ctx, func() (interface{}, error) {
		return nil, r.client.DeleteInboundParseWebhook(ctx, hostname)
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Deleting inbound parse webhook",
			fmt.Sprintf("Unable to delete inbound parse webhook (hostname: %s), got error: %s", hostname, err),
		)
		return
	}
}

func (r *inboundParseWebhookResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	hostname := req.ID

	resource.ImportStatePassthroughID(ctx, path.Root("hostname"), req, resp)

	o, err := r.client.GetInboundParseWebhook(ctx, hostname)
	if err != nil {
		resp.Diagnostics.AddError(
			"Importing inbound parse webhook",
			fmt.Sprintf("Unable to read inbound parse webhook, got error: %s", err),
		)
		return
	}

	d := inboundParseWebhookResourceModel{
		Hostname:  types.StringValue(o.Hostname),
		URL:       types.StringValue(o.URL),
		SpamCheck: types.BoolValue(o.SpamCheck),
		SendRaw:   types.BoolValue(o.SendRaw),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &d)...)
	if resp.Diagnostics.HasError() {
		return
	}
}
