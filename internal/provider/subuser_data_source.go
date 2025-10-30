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
	_ datasource.DataSource              = &subuserDataSource{}
	_ datasource.DataSourceWithConfigure = &subuserDataSource{}
)

func newSubuserDataSource() datasource.DataSource {
	return &subuserDataSource{}
}

type subuserDataSource struct {
	client *sendgrid.Client
}

type subuserDataSourceModel struct {
	ID       types.Int64  `tfsdk:"id"`
	Username types.String `tfsdk:"username"`
	Email    types.String `tfsdk:"email"`
}

func (d *subuserDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_subuser"
}

func (d *subuserDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *subuserDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
Provides a subuser resource.

Subusers help you segment your email sending and API activity. You assign permissions and credit limits when you create the subuser. We recommend creating subusers for each of the different types of emails you send - one subuser for transactional emails, and another for marketing emails. Breaking your sending up this way allows you to get separate statistics for each type of email you send.

For more detailed information, please see the [SendGrid documentation](https://docs.sendgrid.com/ui/account-and-settings/api-keys).
		`,
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "The user ID of the subuser.",
				Computed:            true,
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "The username of the subuser.",
				Required:            true,
			},
			"email": schema.StringAttribute{
				MarkdownDescription: "The email of the subuser.",
				Computed:            true,
			},
		},
	}
}

func (d *subuserDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var s subuserDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &s)...)
	if resp.Diagnostics.HasError() {
		return
	}

	username := s.Username.ValueString()

	subuser, err := d.client.GetSubuser(ctx, username)
	if err != nil {
		resp.Diagnostics.AddError(
			"Reading subuser",
			fmt.Sprintf("Unable to get subuser by username: %s, err: %s", username, err.Error()),
		)
		return
	}

	s = subuserDataSourceModel{
		ID:       types.Int64Value(subuser.ID),
		Username: types.StringValue(subuser.Username),
		Email:    types.StringValue(subuser.Email),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &s)...)
	if resp.Diagnostics.HasError() {
		return
	}
}
