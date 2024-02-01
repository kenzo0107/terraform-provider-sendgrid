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
	_ datasource.DataSource              = &enforceTLSDataSource{}
	_ datasource.DataSourceWithConfigure = &enforceTLSDataSource{}
)

func newEnforceTLSDataSource() datasource.DataSource {
	return &enforceTLSDataSource{}
}

type enforceTLSDataSource struct {
	client *sendgrid.Client
}

type enforceTLSDataSourceModel struct {
	RequireTLS       types.Bool    `tfsdk:"require_tls"`
	RequireValidCert types.Bool    `tfsdk:"require_valid_cert"`
	Version          types.Float64 `tfsdk:"version"`
}

func (d *enforceTLSDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_enforce_tls"
}

func (d *enforceTLSDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *enforceTLSDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
The Enforced TLS settings specify whether or not the recipient of your send is required to support TLS or have a valid certificate. The Enforced TLS endpoint supports retrieving and updating TLS settings.

NOTE: Even if you run the current forced TLS settings acquisition API immediately after updating, the changes may not be reflected.
		`,
		Attributes: map[string]schema.Attribute{
			"require_tls": schema.BoolAttribute{
				MarkdownDescription: "Indicates if you want to require your recipients to support TLS.",
				Computed:            true,
			},
			"require_valid_cert": schema.BoolAttribute{
				MarkdownDescription: "Indicates if you want to require your recipients to have a valid certificate.",
				Computed:            true,
			},
			"version": schema.Float64Attribute{
				MarkdownDescription: "The minimum required TLS certificate version.",
				Computed:            true,
			},
		},
	}
}

func (d *enforceTLSDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data enforceTLSDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	o, err := d.client.GetEnforceTLS(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Reading enforceTLS",
			fmt.Sprintf("Unable to get enforce TLS, err: %s", err.Error()),
		)
		return
	}

	data = enforceTLSDataSourceModel{
		RequireTLS:       types.BoolValue(o.RequireTLS),
		RequireValidCert: types.BoolValue(o.RequireValidCert),
		Version:          types.Float64Value(o.Version),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}
