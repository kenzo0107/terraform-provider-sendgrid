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
	_ datasource.DataSource              = &teammateDataSource{}
	_ datasource.DataSourceWithConfigure = &teammateDataSource{}
)

func newTeammateDataSource() datasource.DataSource {
	return &teammateDataSource{}
}

type teammateDataSource struct {
	client *sendgrid.Client
}

type teammateDataSourceModel struct {
	ID        types.String   `tfsdk:"id"`
	Username  types.String   `tfsdk:"username"`
	Email     types.String   `tfsdk:"email"`
	FirstName types.String   `tfsdk:"first_name"`
	LastName  types.String   `tfsdk:"last_name"`
	Address   types.String   `tfsdk:"address"`
	Address2  types.String   `tfsdk:"address2"`
	City      types.String   `tfsdk:"city"`
	State     types.String   `tfsdk:"state"`
	Zip       types.String   `tfsdk:"zip"`
	Country   types.String   `tfsdk:"country"`
	Website   types.String   `tfsdk:"website"`
	Phone     types.String   `tfsdk:"phone"`
	IsAdmin   types.Bool     `tfsdk:"is_admin"`
	UserType  types.String   `tfsdk:"user_type"`
	Scopes    []types.String `tfsdk:"scopes"`
}

func (d *teammateDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_teammate"
}

func (d *teammateDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *teammateDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
Provides information about an existing pending teammate or a regular teammate.

For more detailed information, please see the [SendGrid documentation](https://docs.sendgrid.com/glossary/teammates).
		`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"email": schema.StringAttribute{
				MarkdownDescription: "Teammate's email",
				Required:            true,
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "Teammate's username",
				Computed:            true,
			},
			"first_name": schema.StringAttribute{
				MarkdownDescription: "Teammate's first name",
				Computed:            true,
			},
			"last_name": schema.StringAttribute{
				MarkdownDescription: "Teammate's last name",
				Computed:            true,
			},
			"address": schema.StringAttribute{
				MarkdownDescription: "Teammate's address",
				Computed:            true,
			},
			"address2": schema.StringAttribute{
				MarkdownDescription: "Teammate's address2",
				Computed:            true,
			},
			"city": schema.StringAttribute{
				MarkdownDescription: "Teammate's city",
				Computed:            true,
			},
			"state": schema.StringAttribute{
				MarkdownDescription: "Teammate's state",
				Computed:            true,
			},
			"zip": schema.StringAttribute{
				MarkdownDescription: "Teammate's zip",
				Computed:            true,
			},
			"country": schema.StringAttribute{
				MarkdownDescription: "Teammate's country",
				Computed:            true,
			},
			"website": schema.StringAttribute{
				MarkdownDescription: "Teammate's website",
				Computed:            true,
			},
			"phone": schema.StringAttribute{
				MarkdownDescription: "Teammate's phone",
				Computed:            true,
			},
			"is_admin": schema.BoolAttribute{
				MarkdownDescription: "Set to true if teammate has admin privileges",
				Computed:            true,
			},
			"user_type": schema.StringAttribute{
				MarkdownDescription: "Indicate the type of user: account owner, teammate admin user, or normal teammate. Allowed Values: admin, owner, teammate",
				Computed:            true,
			},
			"scopes": schema.SetAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "Scopes associated to teammate",
				Computed:            true,
			},
		},
	}
}

func (d *teammateDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var s teammateDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &s)...)
	if resp.Diagnostics.HasError() {
		return
	}

	email := s.Email.ValueString()

	pendingUser, err := pendingTeammateByEmail(ctx, d.client, email)
	if err != nil {
		resp.Diagnostics.AddError(
			"Reading teammate",
			fmt.Sprintf("Unable to Read teammate: %s, err: %s", email, err.Error()),
		)
		return
	}

	// If the teammate is in a pending state, return their data.
	if pendingUser != nil {
		scopes := []types.String{}
		for _, s := range pendingUser.Scopes {
			scopes = append(scopes, types.StringValue(s))
		}

		p := teammateDataSourceModel{
			ID:      types.StringValue(pendingUser.Email),
			Email:   types.StringValue(pendingUser.Email),
			IsAdmin: types.BoolValue(pendingUser.IsAdmin),
			Scopes:  scopes,
		}
		resp.Diagnostics.Append(resp.State.Set(ctx, &p)...)
		return
	}

	userByEmail, err := getTeammateByEmail(ctx, d.client, email)
	if err != nil {
		resp.Diagnostics.AddError(
			"Reading teammate",
			fmt.Sprintf("Unable to get teammate by email: %s, err: %s", email, err.Error()),
		)
		return
	}

	if userByEmail == nil {
		resp.Diagnostics.AddError(
			"Reading teammate",
			fmt.Sprintf("Not found teammate (%s)", email),
		)
		return
	}

	user, err := d.client.GetTeammate(ctx, userByEmail.Username)
	if err != nil {
		resp.Diagnostics.AddError(
			"Reading teammate",
			fmt.Sprintf("Unable to get teammate (%s), err: %s", email, err.Error()),
		)
		return
	}

	scopes := []types.String{}
	for _, s := range user.Scopes {
		scopes = append(scopes, types.StringValue(s))
	}

	u := teammateDataSourceModel{
		ID:        types.StringValue(user.Email),
		Username:  types.StringValue(user.Username),
		Email:     types.StringValue(user.Email),
		FirstName: types.StringValue(user.FirstName),
		LastName:  types.StringValue(user.LastName),
		Address:   types.StringValue(user.Address),
		Address2:  types.StringValue(user.Address2),
		City:      types.StringValue(user.City),
		State:     types.StringValue(user.State),
		Zip:       types.StringValue(user.Zip),
		Country:   types.StringValue(user.Country),
		Website:   types.StringValue(user.Website),
		Phone:     types.StringValue(user.Phone),
		IsAdmin:   types.BoolValue(user.IsAdmin),
		UserType:  types.StringValue(user.UserType),
		Scopes:    scopes,
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &u)...)
	if resp.Diagnostics.HasError() {
		return
	}
}
