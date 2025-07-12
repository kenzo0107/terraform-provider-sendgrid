package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kenzo0107/sendgrid"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &eventWebhookDataSource{}
	_ datasource.DataSourceWithConfigure = &eventWebhookDataSource{}
)

func newEventWebhookDataSource() datasource.DataSource {
	return &eventWebhookDataSource{}
}

type eventWebhookDataSource struct {
	client *sendgrid.Client
}

type eventWebhookDataSourceModel struct {
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
	Signed            types.Bool   `tfsdk:"signed"`
	PublicKey         types.String `tfsdk:"public_key"`
}

func (d *eventWebhookDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_event_webhook"
}

func (d *eventWebhookDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*sendgrid.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *sendgrid.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

func (d *eventWebhookDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
The SendGrid Event Webhook sends email event data as SendGrid processes it. This means you can receive data in nearly real-time, making it ideal to integrate with logging or monitoring systems.
Because the Event Webhook delivers data to your systems, it is also well-suited to backing up and storing event data within your infrastructure to meet your own data access and retention needs.
		`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of Event Webhook",
				Required:            true,
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Set this property to true to enable the Event Webhook or false to disable it.",
				Computed:            true,
			},
			"url": schema.StringAttribute{
				MarkdownDescription: "Set this property to the URL where you want the Event Webhook to send event data.",
				Computed:            true,
			},
			"group_resubscribe": schema.BoolAttribute{
				MarkdownDescription: "Set this property to true to receive group resubscribe events. Group resubscribes occur when recipients resubscribe to a specific unsubscribe group by updating their subscription preferences. You must enable Subscription Tracking to receive this type of event.",
				Computed:            true,
			},
			"delivered": schema.BoolAttribute{
				MarkdownDescription: "Set this property to true to receive delivered events. Delivered events occur when a message has been successfully delivered to the receiving server.",
				Computed:            true,
			},
			"group_unsubscribe": schema.BoolAttribute{
				MarkdownDescription: "Set this property to true to receive group unsubscribe events. Group unsubscribes occur when recipients unsubscribe from a specific unsubscribe group either by direct link or by updating their subscription preferences. You must enable Subscription Tracking to receive this type of event.",
				Computed:            true,
			},
			"spam_report": schema.BoolAttribute{
				MarkdownDescription: "Set this property to true to receive spam report events. Spam reports occur when recipients mark a message as spam.",
				Computed:            true,
			},
			"bounce": schema.BoolAttribute{
				MarkdownDescription: "Set this property to true to receive bounce events. A bounce occurs when a receiving server could not or would not accept a message.",
				Computed:            true,
			},
			"deferred": schema.BoolAttribute{
				MarkdownDescription: "Set this property to true to receive deferred events. Deferred events occur when a recipient's email server temporarily rejects a message.",
				Computed:            true,
			},
			"unsubscribe": schema.BoolAttribute{
				MarkdownDescription: "Set this property to true to receive unsubscribe events. Unsubscribes occur when recipients click on a message's subscription management link. You must enable Subscription Tracking to receive this type of event.",
				Computed:            true,
			},
			"processed": schema.BoolAttribute{
				MarkdownDescription: "Set this property to true to receive processed events. Processed events occur when a message has been received by Twilio SendGrid and the message is ready to be delivered.",
				Computed:            true,
			},
			"open": schema.BoolAttribute{
				MarkdownDescription: "Set this property to true to receive open events. Open events occur when a recipient has opened the HTML message. You must enable Open Tracking to receive this type of event.",
				Computed:            true,
			},
			"click": schema.BoolAttribute{
				MarkdownDescription: "Set this property to true to receive click events. Click events occur when a recipient clicks on a link within the message. You must enable Click Tracking to receive this type of event.",
				Computed:            true,
			},
			"dropped": schema.BoolAttribute{
				MarkdownDescription: "Set this property to true to receive dropped events. Dropped events occur when your message is not delivered by Twilio SendGrid. Dropped events are accompanied by a reason property, which indicates why the message was dropped. Reasons for a dropped message include: Invalid SMTPAPI header, Spam Content (if spam checker app enabled), Unsubscribed Address, Bounced Address, Spam Reporting Address, Invalid, Recipient List over Package Quota.",
				Computed:            true,
			},
			"friendly_name": schema.StringAttribute{
				MarkdownDescription: "Optionally set this property to a friendly name for the Event Webhook. A friendly name may be assigned to each of your webhooks to help you differentiate them. The friendly name is for convenience only. You should use the webhook id property for any programmatic tasks.",
				Computed:            true,
			},
			"oauth_client_id": schema.StringAttribute{
				MarkdownDescription: "Set this property to the OAuth client ID that SendGrid will pass to your OAuth server or service provider to generate an OAuth access token. When passing data in this property, you must also include the oauth_token_url property.",
				Computed:            true,
			},
			"oauth_client_secret": schema.StringAttribute{
				MarkdownDescription: "Set this property to the OAuth client secret that SendGrid will pass to your OAuth server or service provider to generate an OAuth access token. This secret is needed only once to create an access token. SendGrid will store the secret, allowing you to update your client ID and Token URL without passing the secret to SendGrid again. When passing data in this field, you must also include the oauth_client_id and oauth_token_url properties.",
				Computed:            true,
				Sensitive:           true,
			},
			"oauth_token_url": schema.StringAttribute{
				MarkdownDescription: "Set this property to the URL where SendGrid will send the OAuth client ID and client secret to generate an OAuth access token. This should be your OAuth server or service provider. When passing data in this field, you must also include the oauth_client_id property.",
				Computed:            true,
			},
			"signed": schema.BoolAttribute{
				MarkdownDescription: "Indicates whether signature verification is enabled for the Event Webhook. When enabled, SendGrid signs webhook payloads with a private key.",
				Computed:            true,
			},
			"public_key": schema.StringAttribute{
				MarkdownDescription: "The public key used to verify webhook signatures. This field is populated when signature verification is enabled.",
				Computed:            true,
			},
		},
	}
}

func (d *eventWebhookDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var s eventWebhookDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &s)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := s.ID.ValueString()

	o, err := d.client.GetEventWebhook(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Reading event webhook",
			fmt.Sprintf("Unable to get event webhook by id: %s, err: %s", id, err.Error()),
		)
		return
	}

	u := eventWebhookDataSourceModel{
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
		Signed:           types.BoolValue(o.PublicKey != ""),
		PublicKey:        types.StringValue(o.PublicKey),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &u)...)
	if resp.Diagnostics.HasError() {
		return
	}
}
