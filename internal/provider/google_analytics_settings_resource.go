// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kenzo0107/sendgrid"
)

var _ resource.Resource = &googleAnalyticsSettingsResource{}
var _ resource.ResourceWithImportState = &googleAnalyticsSettingsResource{}

func newGoogleAnalyticsSettingsResource() resource.Resource {
	return &googleAnalyticsSettingsResource{}
}

type googleAnalyticsSettingsResource struct {
	client *sendgrid.Client
}

type googleAnalyticsSettingsResourceModel struct {
	Enabled     types.Bool   `tfsdk:"enabled"`
	UTMSource   types.String `tfsdk:"utm_source"`
	UTMMedium   types.String `tfsdk:"utm_medium"`
	UTMTerm     types.String `tfsdk:"utm_term"`
	UTMContent  types.String `tfsdk:"utm_content"`
	UTMCampaign types.String `tfsdk:"utm_campaign"`
}

func newGoogleAnalyticsSettingsResourceModel(o *sendgrid.OutputGetGoogleAnalyticsSettings) googleAnalyticsSettingsResourceModel {
	return googleAnalyticsSettingsResourceModel{
		Enabled:     types.BoolValue(o.Enabled),
		UTMSource:   types.StringValue(o.UTMSource),
		UTMMedium:   types.StringValue(o.UTMMedium),
		UTMTerm:     types.StringValue(o.UTMTerm),
		UTMContent:  types.StringValue(o.UTMContent),
		UTMCampaign: types.StringValue(o.UTMCampaign),
	}
}

func (r *googleAnalyticsSettingsResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_google_analytics_settings"
}

func utmAttribute(name string) schema.StringAttribute {
	return schema.StringAttribute{
		MarkdownDescription: fmt.Sprintf("The value set as `%s` in the links. Defaults to `\"\"`, so omitting it clears the value on the account.", name),
		Optional:            true,
		Computed:            true,
		Default:             stringdefault.StaticString(""),
	}
}

func (r *googleAnalyticsSettingsResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
Google Analytics tracking adds [UTM parameters](https://www.twilio.com/docs/sendgrid/ui/account-and-settings/tracking#google-analytics) to the links in your emails, so you can see in Google Analytics which emails brought visitors to your site.

SendGrid stores one Google Analytics setting per account, so declaring more than one of this resource for the same account will make them fight over that single setting. Destroying this resource leaves the setting on the account as it is.
		`,
		Attributes: map[string]schema.Attribute{
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Indicates if Google Analytics tracking is enabled or disabled. Defaults to `false`, so omitting it turns Google Analytics tracking off for the account.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"utm_source":   utmAttribute("utm_source"),
			"utm_medium":   utmAttribute("utm_medium"),
			"utm_term":     utmAttribute("utm_term"),
			"utm_content":  utmAttribute("utm_content"),
			"utm_campaign": utmAttribute("utm_campaign"),
		},
	}
}

func (r *googleAnalyticsSettingsResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := clientFromProviderData(req.ProviderData)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *providerData, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

// Not sending only the changed attributes: a string left out of the request
// keeps its old value on SendGrid instead of being cleared.
func (r *googleAnalyticsSettingsResource) update(ctx context.Context, plan googleAnalyticsSettingsResourceModel) (googleAnalyticsSettingsResourceModel, error) {
	o, err := r.client.UpdateGoogleAnalyticsSettings(ctx, &sendgrid.InputUpdateGoogleAnalyticsSettings{
		Enabled:     plan.Enabled.ValueBool(),
		UTMSource:   sendgrid.String(plan.UTMSource.ValueString()),
		UTMMedium:   sendgrid.String(plan.UTMMedium.ValueString()),
		UTMTerm:     sendgrid.String(plan.UTMTerm.ValueString()),
		UTMContent:  sendgrid.String(plan.UTMContent.ValueString()),
		UTMCampaign: sendgrid.String(plan.UTMCampaign.ValueString()),
	})
	if err != nil {
		return googleAnalyticsSettingsResourceModel{}, err
	}

	return newGoogleAnalyticsSettingsResourceModel((*sendgrid.OutputGetGoogleAnalyticsSettings)(o)), nil
}

func (r *googleAnalyticsSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan googleAnalyticsSettingsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state, err := r.update(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Creating Google Analytics settings",
			fmt.Sprintf("Unable to update Google Analytics settings, got error: %s", err),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *googleAnalyticsSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	o, err := r.client.GetGoogleAnalyticsSettings(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Reading Google Analytics settings",
			fmt.Sprintf("Unable to read Google Analytics settings, got error: %s", err),
		)
		return
	}

	state := newGoogleAnalyticsSettingsResourceModel(o)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *googleAnalyticsSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan googleAnalyticsSettingsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state, err := r.update(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Updating Google Analytics settings",
			fmt.Sprintf("Unable to update Google Analytics settings, got error: %s", err),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Not disabling the setting on destroy: SendGrid has no "unset" state to go
// back to, and the previous value is unknown, so the account is left as is.
func (r *googleAnalyticsSettingsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
}

func (r *googleAnalyticsSettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	o, err := r.client.GetGoogleAnalyticsSettings(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Importing Google Analytics settings",
			fmt.Sprintf("Unable to read Google Analytics settings, got error: %s", err),
		)
		return
	}

	state := newGoogleAnalyticsSettingsResourceModel(o)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
