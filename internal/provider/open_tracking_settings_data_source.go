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
	_ datasource.DataSource              = &openTrackingSettingsDataSource{}
	_ datasource.DataSourceWithConfigure = &openTrackingSettingsDataSource{}
)

func newOpenTrackingSettingsDataSource() datasource.DataSource {
	return &openTrackingSettingsDataSource{}
}

type openTrackingSettingsDataSource struct {
	client *sendgrid.Client
}

type openTrackingSettingsDataSourceModel struct {
	Enabled types.Bool `tfsdk:"enabled"`
}

func (d *openTrackingSettingsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_open_tracking_settings"
}

func (d *openTrackingSettingsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
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

func (d *openTrackingSettingsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
Open Tracking adds an invisible, single pixel image at the end of the body of your emails. When the pixel is loaded by a recipient's email client, SendGrid records that as an [open](https://www.twilio.com/docs/sendgrid/glossary/opens).

SendGrid stores one open tracking setting per account, so this data source always reads that single setting.
		`,
		Attributes: map[string]schema.Attribute{
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Indicates if open tracking is enabled or disabled.",
				Computed:            true,
			},
		},
	}
}

func (d *openTrackingSettingsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data openTrackingSettingsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	o, err := d.client.GetOpenTrackingSettings(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Reading open tracking settings",
			fmt.Sprintf("Unable to get open tracking settings, err: %s", err.Error()),
		)
		return
	}

	data = openTrackingSettingsDataSourceModel{
		Enabled: types.BoolValue(o.Enabled),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}
