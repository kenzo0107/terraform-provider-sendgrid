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

var (
	_ datasource.DataSource              = &webhookSecurityPolicyDataSource{}
	_ datasource.DataSourceWithConfigure = &webhookSecurityPolicyDataSource{}
)

func newWebhookSecurityPolicyDataSource() datasource.DataSource {
	return &webhookSecurityPolicyDataSource{}
}

type webhookSecurityPolicyDataSource struct {
	client *sendgrid.Client
}

type webhookSecurityPolicyDataSourceModel struct {
	ID        types.String                               `tfsdk:"id"`
	Name      types.String                               `tfsdk:"name"`
	Signature *webhookSecurityPolicySignatureModel       `tfsdk:"signature"`
	OAuth     *webhookSecurityPolicyDataSourceOAuthModel `tfsdk:"oauth"`
}

type webhookSecurityPolicyDataSourceOAuthModel struct {
	ClientID types.String `tfsdk:"client_id"`
	TokenURL types.String `tfsdk:"token_url"`
	Scopes   types.List   `tfsdk:"scopes"`
}

func (d *webhookSecurityPolicyDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_webhook_security_policy"
}

func (d *webhookSecurityPolicyDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *webhookSecurityPolicyDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Provides a webhook security policy. The OAuth client secret is not returned by SendGrid.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the webhook security policy.",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the webhook security policy.",
				Computed:            true,
			},
			"signature": schema.SingleNestedAttribute{
				MarkdownDescription: "The signature verification (ECDSA) settings.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"enabled": schema.BoolAttribute{
						MarkdownDescription: "Whether SendGrid signs webhook requests with ECDSA.",
						Computed:            true,
					},
					"public_key": schema.StringAttribute{
						MarkdownDescription: "The public key to verify the signature of webhook requests.",
						Computed:            true,
					},
				},
			},
			"oauth": schema.SingleNestedAttribute{
				MarkdownDescription: "The OAuth settings. The client secret is not returned by SendGrid.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"client_id": schema.StringAttribute{
						MarkdownDescription: "The OAuth client ID that SendGrid passes to your OAuth server to generate an access token.",
						Computed:            true,
					},
					"token_url": schema.StringAttribute{
						MarkdownDescription: "The URL where SendGrid sends the OAuth client ID and client secret to generate an access token.",
						Computed:            true,
					},
					"scopes": schema.ListAttribute{
						MarkdownDescription: "The OAuth scopes to request.",
						ElementType:         types.StringType,
						Computed:            true,
					},
				},
			},
		},
	}
}

func (d *webhookSecurityPolicyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config webhookSecurityPolicyDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := config.ID.ValueString()
	o, err := d.client.GetWebhookSecurityPolicy(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Reading webhook security policy",
			fmt.Sprintf("Unable to read webhook security policy (id: %s), got error: %s", id, err),
		)
		return
	}
	if o.Policy == nil {
		resp.Diagnostics.AddError(
			"Reading webhook security policy",
			fmt.Sprintf("Webhook security policy (id: %s) was not found.", id),
		)
		return
	}

	m, diags := webhookSecurityPolicyModelFromAPI(ctx, o.Policy, types.StringNull())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state := webhookSecurityPolicyDataSourceModel{
		ID:        m.ID,
		Name:      m.Name,
		Signature: m.Signature,
	}
	if m.OAuth != nil {
		state.OAuth = &webhookSecurityPolicyDataSourceOAuthModel{
			ClientID: m.OAuth.ClientID,
			TokenURL: m.OAuth.TokenURL,
			Scopes:   m.OAuth.Scopes,
		}
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
