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
	_ datasource.DataSource              = &inboundParseWebhookDataSource{}
	_ datasource.DataSourceWithConfigure = &inboundParseWebhookDataSource{}
)

func newInboundParseWebhookDataSource() datasource.DataSource {
	return &inboundParseWebhookDataSource{}
}

type inboundParseWebhookDataSource struct {
	client *sendgrid.Client
}

type inboundParseWebhookDataSourceModel struct {
	Hostname  types.String `tfsdk:"hostname"`
	URL       types.String `tfsdk:"url"`
	SpamCheck types.Bool   `tfsdk:"spam_check"`
	SendRaw   types.Bool   `tfsdk:"send_raw"`
}

func (d *inboundParseWebhookDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_inbound_parse_webhook"
}

func (d *inboundParseWebhookDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *inboundParseWebhookDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
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
				Computed:            true,
			},
			"spam_check": schema.BoolAttribute{
				MarkdownDescription: "Indicates if you would like SendGrid to check the content parsed from your emails for spam before POSTing them to your domain. (Default: `false`)",
				Computed:            true,
			},
			"send_raw": schema.BoolAttribute{
				MarkdownDescription: "Indicates if you would like SendGrid to post the original MIME-type content of your parsed email. When this parameter is set to true, SendGrid will send a JSON payload of the content of your email. (Default: `false`)",
				Computed:            true,
			},
		},
	}
}

func (d *inboundParseWebhookDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var s inboundParseWebhookDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &s)...)
	if resp.Diagnostics.HasError() {
		return
	}

	hostname := s.Hostname.ValueString()
	o, err := d.client.GetInboundParseWebhook(ctx, hostname)
	if err != nil {
		resp.Diagnostics.AddError(
			"Reading inbound parse webhook",
			fmt.Sprintf("Unable to get inbound parse webhook by hostname: %s, err: %s", hostname, err.Error()),
		)
		return
	}

	u := inboundParseWebhookResourceModel{
		Hostname:  types.StringValue(o.Hostname),
		URL:       types.StringValue(o.URL),
		SpamCheck: types.BoolValue(o.SpamCheck),
		SendRaw:   types.BoolValue(o.SendRaw),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &u)...)
	if resp.Diagnostics.HasError() {
		return
	}
}
