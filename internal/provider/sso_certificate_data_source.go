// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

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
	_ datasource.DataSource              = &ssoCertificateDataSource{}
	_ datasource.DataSourceWithConfigure = &ssoCertificateDataSource{}
)

func newSSOCertificateDataSource() datasource.DataSource {
	return &ssoCertificateDataSource{}
}

type ssoCertificateDataSource struct {
	client *sendgrid.Client
}

type ssoCertificateDataSourceModel struct {
	ID                types.String `tfsdk:"id"`
	PublicCertificate types.String `tfsdk:"public_certificate"`
	IntegrationID     types.String `tfsdk:"integration_id"`
	NotBefore         types.Int64  `tfsdk:"not_before"`
	NotAfter          types.Int64  `tfsdk:"not_after"`
}

func (d *ssoCertificateDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sso_certificate"
}

func (d *ssoCertificateDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ssoCertificateDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
Provides a SSO Certificate resource.
		`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "A unique ID assigned to the certificate by SendGrid.",
				Required:            true,
			},
			"public_certificate": schema.StringAttribute{
				MarkdownDescription: "This public certificate allows SendGrid to verify that SAML requests it receives are signed by an IdP that it recognizes.",
				Computed:            true,
			},
			"integration_id": schema.StringAttribute{
				MarkdownDescription: "An ID that matches a certificate to a specific IdP integration. This is the id returned by the \"Get All SSO Integrations\" endpoint.",
				Computed:            true,
			},
			"not_before": schema.Int64Attribute{
				MarkdownDescription: "A unix timestamp (e.g., 1603915954) that indicates the time before which the certificate is not valid.",
				Computed:            true,
			},
			"not_after": schema.Int64Attribute{
				MarkdownDescription: "A unix timestamp (e.g., 1603915954) that indicates the time after which the certificate is no longer valid.",
				Computed:            true,
			},
		},
	}
}

func (d *ssoCertificateDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var s ssoCertificateDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &s)...)
	if resp.Diagnostics.HasError() {
		return
	}

	certificateId := s.ID.ValueString()
	id, _ := strconv.ParseInt(certificateId, 10, 64)

	o, err := d.client.GetSSOCertificate(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Reading sso certificate",
			fmt.Sprintf("Unable to get sso certificate, got error: %s", err),
		)
		return
	}

	s = ssoCertificateDataSourceModel{
		ID:                types.StringValue(strconv.FormatInt(o.ID, 10)),
		PublicCertificate: types.StringValue(o.PublicCertificate),
		IntegrationID:     types.StringValue(o.IntegrationID),
		NotBefore:         types.Int64Value(o.NotBefore),
		NotAfter:          types.Int64Value(o.NotAfter),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &s)...)
	if resp.Diagnostics.HasError() {
		return
	}
}
