// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/kenzo0107/sendgrid"
)

var (
	_ datasource.DataSource              = &subscriptionTrackingSettingsDataSource{}
	_ datasource.DataSourceWithConfigure = &subscriptionTrackingSettingsDataSource{}
)

func newSubscriptionTrackingSettingsDataSource() datasource.DataSource {
	return &subscriptionTrackingSettingsDataSource{}
}

type subscriptionTrackingSettingsDataSource struct {
	client *sendgrid.Client
}

func (d *subscriptionTrackingSettingsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_subscription_tracking_settings"
}

func (d *subscriptionTrackingSettingsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *subscriptionTrackingSettingsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attrs := map[string]schema.Attribute{
		"enabled": schema.BoolAttribute{
			MarkdownDescription: subscriptionTrackingSettingsDescriptions["enabled"],
			Computed:            true,
		},
	}
	for _, name := range []string{"html_content", "plain_content", "landing", "replace", "url"} {
		attrs[name] = schema.StringAttribute{
			MarkdownDescription: subscriptionTrackingSettingsDescriptions[name],
			Computed:            true,
		}
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: `
Subscription tracking adds an unsubscribe link to the end of your emails, so recipients can unsubscribe from all of your emails. See [Subscription Tracking](https://www.twilio.com/docs/sendgrid/ui/account-and-settings/tracking#subscription-tracking).

SendGrid stores one subscription tracking setting per account, so this data source always reads that single setting.
		`,
		Attributes: attrs,
	}
}

func (d *subscriptionTrackingSettingsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	o, err := d.client.GetSubscriptionTrackingSettings(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Reading subscription tracking settings",
			fmt.Sprintf("Unable to get subscription tracking settings, err: %s", err.Error()),
		)
		return
	}

	data := newSubscriptionTrackingSettingsResourceModel(o)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
