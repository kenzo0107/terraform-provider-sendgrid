// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/kenzo0107/sendgrid"
)

var (
	_ datasource.DataSource              = &marketingSenderDataSource{}
	_ datasource.DataSourceWithConfigure = &marketingSenderDataSource{}
)

func newMarketingSenderDataSource() datasource.DataSource {
	return &marketingSenderDataSource{}
}

type marketingSenderDataSource struct {
	client *sendgrid.Client
}

func (d *marketingSenderDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_marketing_sender"
}

func (d *marketingSenderDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := clientFromProviderData(req.ProviderData)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *providerData, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

func (d *marketingSenderDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
Provides a Marketing Campaigns Sender data source.

For more detailed information, please see the [SendGrid documentation](https://www.twilio.com/docs/sendgrid/api-reference/marketing-campaigns-senders).
		`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the sender.",
				Required:            true,
			},
			"nickname": schema.StringAttribute{
				MarkdownDescription: "A nickname for the sender identity. Not used for sending.",
				Computed:            true,
			},
			"from_email": schema.StringAttribute{
				MarkdownDescription: "The email address from which your recipient will receive emails.",
				Computed:            true,
			},
			"from_name": schema.StringAttribute{
				MarkdownDescription: "The name appended to the from email field.",
				Computed:            true,
			},
			"reply_to": schema.StringAttribute{
				MarkdownDescription: "The email address to which your recipient will reply.",
				Computed:            true,
			},
			"reply_to_name": schema.StringAttribute{
				MarkdownDescription: "The name appended to the reply to email field.",
				Computed:            true,
			},
			"address": schema.StringAttribute{
				MarkdownDescription: "The physical address of the sender identity.",
				Computed:            true,
			},
			"address2": schema.StringAttribute{
				MarkdownDescription: "Additional sender identity address information.",
				Computed:            true,
			},
			"state": schema.StringAttribute{
				MarkdownDescription: "The state of the sender identity. SendGrid stores only the first two characters.",
				Computed:            true,
			},
			"city": schema.StringAttribute{
				MarkdownDescription: "The city of the sender identity.",
				Computed:            true,
			},
			"zip": schema.StringAttribute{
				MarkdownDescription: "The zipcode of the sender identity.",
				Computed:            true,
			},
			"country": schema.StringAttribute{
				MarkdownDescription: "The country of the sender identity.",
				Computed:            true,
			},
			"verified": schema.BoolAttribute{
				MarkdownDescription: "Whether the sender identity has been verified.",
				Computed:            true,
			},
			"locked": schema.BoolAttribute{
				MarkdownDescription: "Whether the sender is used in a campaign that is in progress.",
				Computed:            true,
			},
		},
	}
}

func (d *marketingSenderDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var s marketingSenderResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &s)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := s.ID.ValueString()
	idInt64, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Reading marketing sender",
			fmt.Sprintf("Unable to parse marketing sender id (id: %s), got error: %s", id, err),
		)
		return
	}

	o, err := d.client.GetMarketingSender(ctx, idInt64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Reading marketing sender",
			fmt.Sprintf("Unable to read marketing sender (id: %s), got error: %s", id, err),
		)
		return
	}

	s = newMarketingSenderResourceModel((*sendgrid.MarketingSender)(o))
	resp.Diagnostics.Append(resp.State.Set(ctx, &s)...)
}
