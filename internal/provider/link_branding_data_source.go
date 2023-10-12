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
	_ datasource.DataSource              = &linkBrandingDataSource{}
	_ datasource.DataSourceWithConfigure = &linkBrandingDataSource{}
)

func newLinkBrandingDataSource() datasource.DataSource {
	return &linkBrandingDataSource{}
}

type linkBrandingDataSource struct {
	client *sendgrid.Client
}

type linkBrandingDataSourceModel struct {
	ID        types.String `tfsdk:"id"`
	UserID    types.Int64  `tfsdk:"user_id"`
	Domain    types.String `tfsdk:"domain"`
	Subdomain types.String `tfsdk:"subdomain"`
	Username  types.String `tfsdk:"username"`
	Default   types.Bool   `tfsdk:"default"`
	Legacy    types.Bool   `tfsdk:"legacy"`
	Valid     types.Bool   `tfsdk:"valid"`
	DNS       types.Set    `tfsdk:"dns"`
}

func (d *linkBrandingDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_link_branding"
}

func (d *linkBrandingDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *linkBrandingDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
Provides a Link Branding resource.

Email link branding (formerly "Link Whitelabel") allows all of the click-tracked links, opens, and images in your emails to be served from your domain rather than sendgrid.net. Spam filters and recipient servers look at the links within emails to determine whether the email looks trustworthy. They use the reputation of the root domain to determine whether the links can be trusted.

For more detailed information, please see the [SendGrid documentation](https://docs.sendgrid.com/glossary/link-branding).
		`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the branded link.",
				Required:            true,
			},
			"domain": schema.StringAttribute{
				MarkdownDescription: "The root domain of the branded link.",
				Computed:            true,
			},
			"subdomain": schema.StringAttribute{
				MarkdownDescription: "The subdomain used to generate the DNS records for this link branding. This subdomain must be different from the subdomain used for your authenticated domain.",
				Computed:            true,
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "The username of the account that this link branding is associated with.",
				Computed:            true,
			},
			"user_id": schema.Int64Attribute{
				MarkdownDescription: "The ID of the user that this link branding is associated with.",
				Computed:            true,
			},
			"default": schema.BoolAttribute{
				MarkdownDescription: "Indicates if this is the default link branding.",
				Computed:            true,
			},
			"legacy": schema.BoolAttribute{
				MarkdownDescription: "Indicates if this link branding was created using the legacy whitelabel tool. If it is a legacy whitelabel, it will still function, but you'll need to create new link branding if you need to update it.",
				Computed:            true,
			},
			"valid": schema.BoolAttribute{
				MarkdownDescription: "Indicates if this link branding is valid.",
				Computed:            true,
			},
			"dns": schema.SetNestedAttribute{
				MarkdownDescription: "The DNS records generated for this link branding.",
				Computed:            true,
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

func (d *linkBrandingDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var s linkBrandingDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &s)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := s.ID.ValueString()
	linkId, _ := strconv.ParseInt(id, 10, 64)
	o, err := d.client.GetBrandedLink(ctx, linkId)
	if err != nil {
		resp.Diagnostics.AddError(
			"Reading link branding",
			fmt.Sprintf("Unable to get branded link, got error: %s", err),
		)
		return
	}

	s.UserID = types.Int64Value(o.UserID)
	s.Domain = types.StringValue(o.Domain)
	s.Subdomain = types.StringValue(o.Subdomain)
	s.Username = types.StringValue(o.Username)
	s.Default = types.BoolValue(o.Default)
	s.Legacy = types.BoolValue(o.Legacy)
	s.Valid = types.BoolValue(o.Valid)
	s.DNS = convertDNSBrandedLinkToSetType(o.DNS)

	resp.Diagnostics.Append(resp.State.Set(ctx, &s)...)
	if resp.Diagnostics.HasError() {
		return
	}
}
