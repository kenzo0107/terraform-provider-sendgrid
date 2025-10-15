// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

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
	_ datasource.DataSource              = &bounceSettingsDataSource{}
	_ datasource.DataSourceWithConfigure = &bounceSettingsDataSource{}
)

func newBounceSettingsDataSource() datasource.DataSource {
	return &bounceSettingsDataSource{}
}

type bounceSettingsDataSource struct {
	client *sendgrid.Client
}

type bounceSettingsDataSourceModel struct {
	Enabled     types.Bool  `tfsdk:"enabled"`
	SoftBounces types.Int64 `tfsdk:"soft_bounces"`
	HardBounces types.Int64 `tfsdk:"hard_bounces"`
}

func (d *bounceSettingsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_bounce_settings"
}

func (d *bounceSettingsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *bounceSettingsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
Retrieve bounce settings for your SendGrid account.

Bounce settings allow you to configure how long soft bounces are retained in your suppression list.
Soft bounces are temporary delivery failures, such as a full mailbox or temporary server issues.

The Soft Bounces setting specifies the number of days soft bounces will be kept in your soft bounces suppression list.
Any soft bounces older than this value will be purged.

For more information, see the [SendGrid Mail Settings Guide](https://support.sendgrid.com/hc/en-us/articles/9489871931803-Mail-Settings-Guide-within-a-SendGrid-Account).
		`,
		Attributes: map[string]schema.Attribute{
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Indicates if the Bounce Purge mail setting is enabled.",
				Computed:            true,
			},
			"soft_bounces": schema.Int64Attribute{
				MarkdownDescription: "The number of days after which SendGrid will purge all contacts from your soft bounces suppression lists. Must be between 1 and 3650 days.",
				Computed:            true,
			},
			"hard_bounces": schema.Int64Attribute{
				MarkdownDescription: "The number of days after which SendGrid will purge all contacts from your hard bounces suppression lists. Must be between 1 and 3650 days.",
				Computed:            true,
			},
		},
	}
}

func (d *bounceSettingsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state bounceSettingsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	res, err := retryOnRateLimit(ctx, func() (interface{}, error) {
		return d.client.GetBounceSettings(ctx)
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Reading bounce settings",
			fmt.Sprintf("Unable to get bounce settings, got error: %s", err),
		)
		return
	}

	o, ok := res.(*sendgrid.OutputGetBounceSettings)
	if !ok {
		resp.Diagnostics.AddError(
			"Reading bounce settings",
			"Failed to assert type *sendgrid.BounceSettings",
		)
		return
	}

	u := bounceSettingsDataSourceModel{
		Enabled:     types.BoolValue(o.Enabled),
		SoftBounces: types.Int64Value(o.SoftBounces),
		HardBounces: types.Int64Value(o.HardBounces),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &u)...)
	if resp.Diagnostics.HasError() {
		return
	}
}
