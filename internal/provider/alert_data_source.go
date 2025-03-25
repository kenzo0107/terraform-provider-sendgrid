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
	_ datasource.DataSource              = &alertDataSource{}
	_ datasource.DataSourceWithConfigure = &alertDataSource{}
)

func newAlertDataSource() datasource.DataSource {
	return &alertDataSource{}
}

type alertDataSource struct {
	client *sendgrid.Client
}

type alertDataSourceModel struct {
	ID         types.String `tfsdk:"id"`
	EmailTo    types.String `tfsdk:"email_to"`
	Type       types.String `tfsdk:"type"`
	Frequency  types.String `tfsdk:"frequency"`
	Percentage types.Int64  `tfsdk:"percentage"`
}

func (d *alertDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_alert"
}

func (d *alertDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *alertDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
Twilio SendGrid's Alerts feature allows you to receive notifications regarding your usage or program statistics from SendGrid at an email address you specify.
		`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of alert",
				Required:            true,
			},
			"email_to": schema.StringAttribute{
				MarkdownDescription: "The email address the alert will be sent to. Example: test@example.com",
				Computed:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "The type of alert you want to create. Can be either usage_limit or stats_notification. Example: usage_limit",
				Computed:            true,
			},
			"frequency": schema.StringAttribute{
				MarkdownDescription: "If the alert is of type stats_notification, this indicates how frequently the stats notifications will be sent. For example, `daily`, `weekly`, or `monthly`.",
				Computed:            true,
			},
			"percentage": schema.Int64Attribute{
				MarkdownDescription: "If the alert is of type usage_limit, this indicates the percentage of email usage that must be reached before the alert will be sent.",
				Computed:            true,
			},
		},
	}
}

func (d *alertDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var s alertDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &s)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := s.ID.ValueString()
	idInt64, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Reading alert",
			fmt.Sprintf("Unable to read alert (id: %s), got error: %s", id, err),
		)
		return
	}

	o, err := d.client.GetAlert(ctx, idInt64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Reading alert",
			fmt.Sprintf("Unable to get alert by id: %s, err: %s", id, err.Error()),
		)
		return
	}

	if o == nil {
		resp.Diagnostics.AddError(
			"Reading alert",
			fmt.Sprintf("Not found alert (id: %s)", id),
		)
		return
	}

	u := alertDataSourceModel{
		ID:         types.StringValue(id),
		EmailTo:    types.StringValue(o.EmailTo),
		Type:       types.StringValue(o.Type),
		Frequency:  types.StringValue(o.Frequency),
		Percentage: types.Int64Value(o.Percentage),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &u)...)
	if resp.Diagnostics.HasError() {
		return
	}
}
