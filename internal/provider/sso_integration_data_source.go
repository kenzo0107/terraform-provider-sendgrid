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
	_ datasource.DataSource              = &ssoIntegrationDataSource{}
	_ datasource.DataSourceWithConfigure = &ssoIntegrationDataSource{}
)

func newSSOIntegrationDataSource() datasource.DataSource {
	return &ssoIntegrationDataSource{}
}

type ssoIntegrationDataSource struct {
	client *sendgrid.Client
}

type ssoIntegrationDataSourceModel struct {
	ID                   types.String `tfsdk:"id"`
	Name                 types.String `tfsdk:"name"`
	Enabled              types.Bool   `tfsdk:"enabled"`
	SigninURL            types.String `tfsdk:"signin_url"`
	SignoutURL           types.String `tfsdk:"signout_url"`
	EntityID             types.String `tfsdk:"entity_id"`
	CompletedIntegration types.Bool   `tfsdk:"completed_integration"`
	SingleSignonURL      types.String `tfsdk:"single_signon_url"`
	AudienceURL          types.String `tfsdk:"audience_url"`
}

func (d *ssoIntegrationDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sso_integration"
}

func (d *ssoIntegrationDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ssoIntegrationDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
Provides a SSO Integration resource.
		`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "A unique ID assigned to the configuration by SendGrid.",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of your integration. This name can be anything that makes sense for your organization (eg. Twilio SendGrid)",
				Computed:            true,
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Indicates if the integration is enabled.",
				Computed:            true,
			},
			"signin_url": schema.StringAttribute{
				MarkdownDescription: "The IdP's SAML POST endpoint. This endpoint should receive requests and initiate an SSO login flow. This is called the \"Embed Link\" in the Twilio SendGrid UI.",
				Computed:            true,
			},
			"signout_url": schema.StringAttribute{
				MarkdownDescription: "This URL is relevant only for an IdP-initiated authentication flow. If a user authenticates from their IdP, this URL will return them to their IdP when logging out.",
				Computed:            true,
			},
			"entity_id": schema.StringAttribute{
				MarkdownDescription: "An identifier provided by your IdP to identify Twilio SendGrid in the SAML interaction. This is called the \"SAML Issuer ID\" in the Twilio SendGrid UI.",
				Computed:            true,
			},
			"completed_integration": schema.BoolAttribute{
				MarkdownDescription: "Indicates if the integration is complete.",
				Computed:            true,
			},
			"single_signon_url": schema.StringAttribute{
				MarkdownDescription: "The URL where your IdP should POST its SAML response. This is the Twilio SendGrid URL that is responsible for receiving and parsing a SAML assertion. This is the same URL as the Audience URL when using SendGrid.",
				Computed:            true,
			},
			"audience_url": schema.StringAttribute{
				MarkdownDescription: "The URL where your IdP should POST its SAML response. This is the Twilio SendGrid URL that is responsible for receiving and parsing a SAML assertion. This is the same URL as the Single Sign-On URL when using SendGrid.",
				Computed:            true,
			},
		},
	}
}

func (d *ssoIntegrationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var s ssoIntegrationDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &s)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := s.ID.ValueString()
	o, err := d.client.GetSSOIntegration(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Reading sso integration",
			fmt.Sprintf("Unable to get sso integration, got error: %s", err),
		)
		return
	}

	s = ssoIntegrationDataSourceModel{
		ID:                   types.StringValue(o.ID),
		Name:                 types.StringValue(o.Name),
		Enabled:              types.BoolValue(o.Enabled),
		SigninURL:            types.StringValue(o.SigninURL),
		SignoutURL:           types.StringValue(o.SignoutURL),
		EntityID:             types.StringValue(o.EntityID),
		CompletedIntegration: types.BoolValue(o.CompletedIntegration),
		SingleSignonURL:      types.StringValue(o.SingleSignonURL),
		AudienceURL:          types.StringValue(o.AudienceURL),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &s)...)
	if resp.Diagnostics.HasError() {
		return
	}
}
