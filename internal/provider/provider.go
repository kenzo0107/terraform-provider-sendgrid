// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kenzo0107/sendgrid"
)

// Ensure sendgridProvider satisfies various provider interfaces.
var _ provider.Provider = &sendgridProvider{}

// sendgridProvider defines the provider implementation.
type sendgridProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// sendgridProviderModel describes the provider data model.
type sendgridProviderModel struct {
	APIKey  types.String `tfsdk:"api_key"`
	Subuser types.String `tfsdk:"subuser"`
}

func (p *sendgridProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "sendgrid"
	resp.Version = p.version
}

func (p *sendgridProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"api_key": schema.StringAttribute{
				MarkdownDescription: "API Key for Sendgrid API. May also be provided via SENDGRID_API_KEY environment variable.",
				Optional:            true,
				Sensitive:           true,
			},
			"subuser": schema.StringAttribute{
				MarkdownDescription: "Subuser for Sendgrid API. May also be provided via SENDGRID_SUBUSER environment variable.",
				Optional:            true,
			},
		},
	}
}

func (p *sendgridProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	// Check environment variables
	apiKey := os.Getenv("SENDGRID_API_KEY")
	subuser := os.Getenv("SENDGRID_SUBUSER")

	// Retrieve provider data from configuration
	var config sendgridProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)

	// Default values to environment variables, but override
	// with Terraform configuration value if set.

	if !config.APIKey.IsNull() {
		apiKey = config.APIKey.ValueString()
	}

	if !config.Subuser.IsNull() {
		subuser = config.Subuser.ValueString()
	}

	// If any of the expected configurations are missing, return
	// errors with provider-specific guidance.

	if apiKey == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_key"),
			"Missing SendGrid API Key",
			"The provider cannot create the SendGrid API client as there is a missing or empty value for the SendGrid API Key. "+
				"Set the host value in the configuration or use the SENDGRID_API_KEY environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	// If practitioner provided a configuration value for any of the
	// attributes, it must be a known value.

	if config.APIKey.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_key"),
			"Unknown SendGrid API Key",
			"The provider cannot create the SendGrid API client as there is an unknown configuration value for the SendGrid API Key. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the SENDGRID_API_KEY environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	var client *sendgrid.Client
	if subuser != "" {
		client = sendgrid.New(apiKey, sendgrid.OptionSubuser(subuser))
	} else {
		client = sendgrid.New(apiKey)
	}

	// Make the SendGrid api key available during DataSource and Resource
	// type Configure methods.
	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *sendgridProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		newTeammateResource,
		newAPIKeyResource,
		newSubuserResource,
		newSenderAuthenticationResource,
		newLinkBrandingResource,
		newSenderVerificationResource,
		newUnsubscribeGroupResource,
		newTemplateResource,
		newTemplateVersionResource,
	}
}

func (p *sendgridProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		newTeammateDataSource,
		newAPIKeyDataSource,
		newSubuserDataSource,
		newSenderAuthenticationDataSource,
		newLinkBrandingDataSource,
		newSenderVerificationDataSource,
		newUnsubscribeGroupDataSource,
		newTemplateDataSource,
		newTemplateVersionDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &sendgridProvider{
			version: version,
		}
	}
}
