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

// providerData is handed to every resource and data source through
// ConfigureResponse.ResourceData / DataSourceData.
type providerData struct {
	client *sendgrid.Client
	// teammates caches GET /v3/teammates and GET /v3/teammates/pending for the
	// lifetime of this provider instance. See teammateCache.
	teammates *teammateCache
}

// providerDataFrom returns the value set by sendgridProvider.Configure.
func providerDataFrom(v any) (*providerData, bool) {
	pd, ok := v.(*providerData)
	if !ok || pd == nil {
		return nil, false
	}
	return pd, true
}

// clientFromProviderData returns the SendGrid client set by sendgridProvider.Configure.
func clientFromProviderData(v any) (*sendgrid.Client, bool) {
	pd, ok := providerDataFrom(v)
	if !ok {
		return nil, false
	}
	return pd.client, true
}

// sendgridProviderModel describes the provider data model.
type sendgridProviderModel struct {
	APIKey  types.String `tfsdk:"api_key"`
	Subuser types.String `tfsdk:"subuser"`
	Region  types.String `tfsdk:"region"`
}

func (p *sendgridProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "sendgrid"
	resp.Version = p.version
}

func (p *sendgridProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
The SendGrid provider manages resources of a [Twilio SendGrid](https://sendgrid.com/) account.

## Rate limits

Every request is retried automatically when SendGrid answers with HTTP 429. The provider waits until the
window reported by the ` + "`X-RateLimit-Reset`" + ` header ends (up to 5 retries, at most 60 seconds per wait) before
failing. Only 429 responses are retried, so a request is never applied twice. Each attempt times out after 60 seconds.
`,
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
			"region": schema.StringAttribute{
				MarkdownDescription: "Region for Sendgrid API. May also be provided via SENDGRID_REGION environment variable. Valid values are `global` and `eu`, with `global` as the default.",
				Optional:            true,
			},
		},
	}
}

func (p *sendgridProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	// Check environment variables
	apiKey := os.Getenv("SENDGRID_API_KEY")
	subuser := os.Getenv("SENDGRID_SUBUSER")
	region := os.Getenv("SENDGRID_REGION")

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

	if !config.Region.IsNull() {
		region = config.Region.ValueString()
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

	opts := []sendgrid.Option{
		// Retry 429 responses and reuse connections across Terraform's parallel reads.
		sendgrid.OptionHTTPClient(newHTTPClient()),
	}
	if subuser != "" {
		opts = append(opts, sendgrid.OptionSubuser(subuser))
	}

	switch region {
	case "eu":
		opts = append(opts, sendgrid.OptionBaseURL(sendgrid.BaseURLEU))
	}

	client := sendgrid.New(apiKey, opts...)

	// Make the SendGrid client (and the per-instance teammate cache) available
	// during DataSource and Resource type Configure methods.
	pd := &providerData{
		client:    client,
		teammates: newTeammateCache(client),
	}
	resp.DataSourceData = pd
	resp.ResourceData = pd
}

func (p *sendgridProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		newTeammateResource,
		newAPIKeyResource,
		newSubuserResource,
		newSubuserWhitelabelDomainResource,
		newSenderAuthenticationResource,
		newLinkBrandingResource,
		newSenderVerificationResource,
		newUnsubscribeGroupResource,
		newTemplateResource,
		newTemplateVersionResource,
		newEnforceTLSResource,
		newReverseDNSResource,
		newSSOIntegrationResource,
		newSSOCertificateResource,
		newEventWebhookResource,
		newInboundParseWebhookResource,
		newSSOTeammateResource,
		newClickTrackingSettingsResource,
		newBounceSettingsResource,
		newAlertResource,
		newDesignResource,
		newIPPoolResource,
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
		newEnforceTLSDataSource,
		newReverseDNSDataSource,
		newSSOIntegrationDataSource,
		newSSOCertificateDataSource,
		newEventWebhookDataSource,
		newInboundParseWebhookDataSource,
		newClickTrackingSettingsDataSource,
		newBounceSettingsDataSource,
		newAlertDataSource,
		newDesignDataSource,
		newIPPoolDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &sendgridProvider{
			version: version,
		}
	}
}
