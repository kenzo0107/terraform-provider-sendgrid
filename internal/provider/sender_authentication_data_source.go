// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

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
	_ datasource.DataSource              = &senderAuthenticationDataSource{}
	_ datasource.DataSourceWithConfigure = &senderAuthenticationDataSource{}
)

func newSenderAuthenticationDataSource() datasource.DataSource {
	return &senderAuthenticationDataSource{}
}

type senderAuthenticationDataSource struct {
	client *sendgrid.Client
}

type senderAuthenticationDataSourceModel struct {
	ID                 types.String   `tfsdk:"id"`
	UserID             types.Int64    `tfsdk:"user_id"`
	Domain             types.String   `tfsdk:"domain"`
	Subdomain          types.String   `tfsdk:"subdomain"`
	Username           types.String   `tfsdk:"username"`
	IPs                []types.String `tfsdk:"ips"`
	Default            types.Bool     `tfsdk:"default"`
	Legacy             types.Bool     `tfsdk:"legacy"`
	CustomDkimSelector types.String   `tfsdk:"custom_dkim_selector"`
	DNS                types.Set      `tfsdk:"dns"`
	Valid              types.Bool     `tfsdk:"valid"`
}

func (d *senderAuthenticationDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sender_authentication"
}

func (d *senderAuthenticationDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *senderAuthenticationDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
Provides a Sender Authentication resource.

Sender authentication refers to the process of showing email providers that SendGrid has your permission to send emails on your behalf.

For more detailed information, please see the [SendGrid documentation](https://docs.sendgrid.com/glossary/domain-authentication).
		`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the authenticated domain.",
				Required:            true,
			},
			"user_id": schema.Int64Attribute{
				MarkdownDescription: "The ID of the user that this domain is associated with.",
				Computed:            true,
			},
			"domain": schema.StringAttribute{
				MarkdownDescription: "Domain being authenticated.",
				Computed:            true,
			},
			"subdomain": schema.StringAttribute{
				MarkdownDescription: "The subdomain to use for this authenticated domain.",
				Computed:            true,
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "The username associated with this domain.",
				Computed:            true,
			},
			"ips": schema.SetAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "The IP addresses that will be included in the custom SPF record for this authenticated domain. NOTE: Even if you add the associated IPs when running the Domain Authentication API, the response returns an empty IP list. Also, even if you execute the IP association/disassociation API, the results may not be reflected immediately after executing the authentication domain acquisition API.",
				Computed:            true,
			},
			"default": schema.BoolAttribute{
				MarkdownDescription: "Whether to use this authenticated domain as the fallback if no authenticated domains match the sender's domain.",
				Computed:            true,
			},
			"legacy": schema.BoolAttribute{
				MarkdownDescription: "Whether to use this authenticated domain as the fallback if no authenticated domains match the sender's domain.",
				Computed:            true,
			},
			"custom_dkim_selector": schema.StringAttribute{
				MarkdownDescription: "Add a custom DKIM selector. Accepts three letters or numbers.",
				Computed:            true,
			},
			"valid": schema.BoolAttribute{
				MarkdownDescription: "Indicates if this is a valid authenticated domain.",
				Computed:            true,
			},
			"dns": schema.SetNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"valid": schema.BoolAttribute{
							MarkdownDescription: "Indicated whether the CName of the DNS is valid or not.",
							Computed:            true,
						},
						"type": schema.StringAttribute{
							MarkdownDescription: "The type of DNS record.",
							Computed:            true,
						},
						"host": schema.StringAttribute{
							MarkdownDescription: "The domain that this DNS record was created for.",
							Computed:            true,
						},
						"data": schema.StringAttribute{
							MarkdownDescription: "The DNS record.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *senderAuthenticationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var s senderAuthenticationDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &s)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := s.ID.ValueString()
	domainId, _ := strconv.ParseInt(id, 10, 64)
	o, err := d.client.GetAuthenticatedDomain(ctx, domainId)
	if err != nil {
		resp.Diagnostics.AddError(
			"Reading sender authentication",
			fmt.Sprintf("Unable to get authenticated domain, got error: %s", err),
		)
		return
	}

	ips := []types.String{}
	for _, ip := range o.IPs {
		ips = append(ips, types.StringValue(ip))
	}

	s.IPs = ips

	s.UserID = types.Int64Value(o.UserID)
	s.Domain = types.StringValue(o.Domain)
	s.Subdomain = types.StringValue(o.Subdomain)
	s.Username = types.StringValue(o.Username)
	s.Default = types.BoolValue(o.Default)
	s.Legacy = types.BoolValue(o.Legacy)
	s.Valid = types.BoolValue(o.Valid)
	s.DNS = convertDNSToSetType(o.DNS)

	resp.Diagnostics.Append(resp.State.Set(ctx, &s)...)
	if resp.Diagnostics.HasError() {
		return
	}
}
