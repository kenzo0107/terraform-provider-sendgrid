package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kenzo0107/sendgrid"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &reverseDNSDataSource{}
	_ datasource.DataSourceWithConfigure = &reverseDNSDataSource{}
)

func newReverseDNSDataSource() datasource.DataSource {
	return &reverseDNSDataSource{}
}

type reverseDNSDataSource struct {
	client *sendgrid.Client
}

type reverseDNSDataSourceModel struct {
	ID                    types.String `tfsdk:"id"`
	IP                    types.String `tfsdk:"ip"`
	RDNS                  types.String `tfsdk:"rdns"`
	Users                 types.Set    `tfsdk:"users"`
	Subdomain             types.String `tfsdk:"subdomain"`
	Domain                types.String `tfsdk:"domain"`
	Valid                 types.Bool   `tfsdk:"valid"`
	Legacy                types.Bool   `tfsdk:"legacy"`
	LastValidationAttempt types.Int64  `tfsdk:"last_validation_attempt"`
	ARecord               types.Object `tfsdk:"a_record"`
}

func (d *reverseDNSDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_reverse_dns"
}

func (d *reverseDNSDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *reverseDNSDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
Provides reverseDNS resource.

Reverse DNS (formerly IP Whitelabel) allows mailbox providers to verify the sender of an email by performing a reverse DNS lookup upon receipt of the emails you send.

Reverse DNS is available for [dedicated IP addresses](https://sendgrid.com/docs/ui/account-and-settings/dedicated-ip-addresses/) only.

When setting up reverse DNS, Twilio SendGrid will provide an A Record (address record) for you to add to your DNS records. The A Record maps your sending domain to a dedicated Twilio SendGrid IP address.

A Reverse DNS consists of a subdomain and domain that will be used to generate a reverse DNS record for a given IP address. Once Twilio SendGrid has verified that the appropriate A record for the IP address has been created, the appropriate reverse DNS record for the IP address is generated.

You can also manage your reverse DNS settings in the [Sender Authentication setion of the Twilio SendGrid App](https://app.sendgrid.com/settings/sender_auth).

For more about Reverse DNS, see ["How to set up reverse DNS"](https://sendgrid.com/docs/ui/account-and-settings/how-to-set-up-reverse-dns/) in the Twilio SendGrid documentation.
		`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the Reverse DNS.",
				Required:            true,
			},
			"ip": schema.StringAttribute{
				MarkdownDescription: "The IP address that this Reverse DNS was created for.",
				Computed:            true,
			},
			"domain": schema.StringAttribute{
				MarkdownDescription: "The root, or sending, domain.",
				Computed:            true,
			},
			"rdns": schema.StringAttribute{
				MarkdownDescription: "The reverse DNS record for the IP address. This points to the Reverse DNS subdomain.",
				Computed:            true,
			},
			"users": schema.SetNestedAttribute{
				MarkdownDescription: "The users who are able to send mail from the IP address.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"username": schema.StringAttribute{
							MarkdownDescription: "The username of a user who can send mail from the IP address.",
							Computed:            true,
						},
						"user_id": schema.Int64Attribute{
							MarkdownDescription: "The ID of a user who can send mail from the IP address.",
							Computed:            true,
						},
					},
				},
			},
			"subdomain": schema.StringAttribute{
				MarkdownDescription: "The subdomain created for this reverse DNS. This is where the rDNS record points.",
				Computed:            true,
			},
			"valid": schema.BoolAttribute{
				MarkdownDescription: "Indicates if this is a valid Reverse DNS.",
				Computed:            true,
			},
			"legacy": schema.BoolAttribute{
				MarkdownDescription: "Indicates if this Reverse DNS was created using the legacy whitelabel tool. If it is a legacy whitelabel, it will still function, but you'll need to create a new Reverse DNS if you need to update it.",
				Computed:            true,
			},
			"last_validation_attempt": schema.Int64Attribute{
				MarkdownDescription: "A Unix epoch timestamp representing the last time of a validation attempt.",
				Computed:            true,
			},
			"a_record": schema.ObjectAttribute{
				Computed:       true,
				AttributeTypes: aRecordObjectAttribute,
			},
		},
	}
}

func (d *reverseDNSDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var s reverseDNSDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &s)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := s.ID.ValueString()
	reverseDNSId, _ := strconv.ParseInt(id, 10, 64)

	o, err := d.client.GetReverseDNS(ctx, reverseDNSId)
	if err != nil {
		resp.Diagnostics.AddError(
			"Reading reverseDNS",
			fmt.Sprintf("Unable to read reverseDNS (id: %v), got error: %s", id, err),
		)
		return
	}

	s = reverseDNSDataSourceModel{
		ID:                    types.StringValue(strconv.FormatInt(o.ID, 10)),
		IP:                    types.StringValue(o.IP),
		RDNS:                  types.StringValue(o.RDNS),
		Subdomain:             types.StringValue(o.Subdomain),
		Domain:                types.StringValue(o.Domain),
		Users:                 convertUsersToSetType(o.Users),
		Valid:                 types.BoolValue(o.Valid),
		Legacy:                types.BoolValue(o.Legacy),
		LastValidationAttempt: types.Int64Value(o.LastValidationAttemptAt),
		ARecord:               newARecord(o.ARecord),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &s)...)
	if resp.Diagnostics.HasError() {
		return
	}
}
