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
	_ datasource.DataSource              = &googleAnalyticsSettingsDataSource{}
	_ datasource.DataSourceWithConfigure = &googleAnalyticsSettingsDataSource{}
)

func newGoogleAnalyticsSettingsDataSource() datasource.DataSource {
	return &googleAnalyticsSettingsDataSource{}
}

type googleAnalyticsSettingsDataSource struct {
	client *sendgrid.Client
}

func (d *googleAnalyticsSettingsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_google_analytics_settings"
}

func (d *googleAnalyticsSettingsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *googleAnalyticsSettingsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	utm := func(name string) schema.StringAttribute {
		return schema.StringAttribute{
			MarkdownDescription: fmt.Sprintf("The value set as `%s` in the links.", name),
			Computed:            true,
		}
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: `
Google Analytics tracking adds [UTM parameters](https://www.twilio.com/docs/sendgrid/ui/account-and-settings/tracking#google-analytics) to the links in your emails, so you can see in Google Analytics which emails brought visitors to your site.

SendGrid stores one Google Analytics setting per account, so this data source always reads that single setting.
		`,
		Attributes: map[string]schema.Attribute{
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Indicates if Google Analytics tracking is enabled or disabled.",
				Computed:            true,
			},
			"utm_source":   utm("utm_source"),
			"utm_medium":   utm("utm_medium"),
			"utm_term":     utm("utm_term"),
			"utm_content":  utm("utm_content"),
			"utm_campaign": utm("utm_campaign"),
		},
	}
}

func (d *googleAnalyticsSettingsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	o, err := d.client.GetGoogleAnalyticsSettings(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Reading Google Analytics settings",
			fmt.Sprintf("Unable to get Google Analytics settings, err: %s", err.Error()),
		)
		return
	}

	data := newGoogleAnalyticsSettingsResourceModel(o)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
