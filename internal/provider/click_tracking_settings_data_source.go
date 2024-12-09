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
	_ datasource.DataSource              = &clickTrackingSettingsDataSource{}
	_ datasource.DataSourceWithConfigure = &clickTrackingSettingsDataSource{}
)

func newClickTrackingSettingsDataSource() datasource.DataSource {
	return &clickTrackingSettingsDataSource{}
}

type clickTrackingSettingsDataSource struct {
	client *sendgrid.Client
}

type clickTrackingSettingsDataSourceModel struct {
	Enabled    types.Bool `tfsdk:"enabled"`
	EnableText types.Bool `tfsdk:"enable_text"`
}

func (d *clickTrackingSettingsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_click_tracking_settings"
}

func (d *clickTrackingSettingsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *clickTrackingSettingsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
Click Tracking overrides all the links and URLs in your emails and points them to either SendGrid’s servers or the domain with which you branded your link. When a customer clicks a link, SendGrid tracks those [clicks](https://www.twilio.com/docs/sendgrid/glossary/clicks).

Click tracking helps you understand how users are engaging with your communications. SendGrid can track up to 1000 links per email
		`,
		Attributes: map[string]schema.Attribute{
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Indicates if click tracking is enabled or disabled.",
				Computed:            true,
			},
			"enable_text": schema.BoolAttribute{
				MarkdownDescription: "Indicates if click tracking is enabled for plain text emails.",
				Computed:            true,
			},
		},
	}
}

func (d *clickTrackingSettingsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data clickTrackingSettingsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	o, err := d.client.GetClickTrackingSettings(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Reading click tracking settings",
			fmt.Sprintf("Unable to get click tracking settings, err: %s", err.Error()),
		)
		return
	}

	data = clickTrackingSettingsDataSourceModel{
		Enabled:    types.BoolValue(o.Enabled),
		EnableText: types.BoolValue(o.EnableText),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}
