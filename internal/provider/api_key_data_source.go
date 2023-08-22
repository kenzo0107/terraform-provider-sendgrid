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
	_ datasource.DataSource              = &apiKeyDataSource{}
	_ datasource.DataSourceWithConfigure = &apiKeyDataSource{}
)

func newAPIKeyDataSource() datasource.DataSource {
	return &apiKeyDataSource{}
}

type apiKeyDataSource struct {
	client *sendgrid.Client
}

type apiKeyDataSourceModel struct {
	ID     types.String `tfsdk:"id"`
	Name   types.String `tfsdk:"name"`
	Scopes types.Set    `tfsdk:"scopes"`
}

func (d *apiKeyDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_key"
}

func (d *apiKeyDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *apiKeyDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
Your application, mail client, or website can all use API (Application Programming Interface) keys to authenticate access to SendGrid services. You can revoke an API key at any time without having to change your username and password, and an API key can be scoped to perform a limited number of actions.

Provides a API Key resource.

For more detailed information, please see the [SendGrid documentation](https://docs.sendgrid.com/ui/account-and-settings/api-keys).
		`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of API Key",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "API key's email",
				Computed:            true,
			},
			"scopes": schema.SetAttribute{
				MarkdownDescription: "The permissions API Key has access to",
				ElementType:         types.StringType,
				Computed:            true,
			},
		},
	}
}

func (d *apiKeyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var s apiKeyDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &s)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := s.ID.ValueString()

	o, err := d.client.GetAPIKey(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Reading api key",
			fmt.Sprintf("Unable to get api key by id: %s, err: %s", id, err.Error()),
		)
		return
	}

	if o == nil {
		resp.Diagnostics.AddError(
			"Reading api key",
			fmt.Sprintf("Not found api key (id: %s)", id),
		)
		return
	}

	scopes, diags := types.SetValueFrom(ctx, types.StringType, o.Scopes)
	resp.Diagnostics.Append(diags...)

	u := apiKeyDataSourceModel{
		ID:     types.StringValue(id),
		Name:   types.StringValue(o.Name),
		Scopes: scopes,
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &u)...)
	if resp.Diagnostics.HasError() {
		return
	}
}
