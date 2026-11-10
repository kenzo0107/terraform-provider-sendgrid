// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kenzo0107/sendgrid"
)

var _ resource.Resource = &webhookSecurityPolicyResource{}
var _ resource.ResourceWithImportState = &webhookSecurityPolicyResource{}

func newWebhookSecurityPolicyResource() resource.Resource {
	return &webhookSecurityPolicyResource{}
}

type webhookSecurityPolicyResource struct {
	client *sendgrid.Client
}

type webhookSecurityPolicyResourceModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	OAuthClientID     types.String `tfsdk:"oauth_client_id"`
	OAuthClientSecret types.String `tfsdk:"oauth_client_secret"`
	OAuthTokenURL     types.String `tfsdk:"oauth_token_url"`
	OAuthScopes       types.List   `tfsdk:"oauth_scopes"`
	SignatureEnabled  types.Bool   `tfsdk:"signature_enabled"`
	PublicKey         types.String `tfsdk:"public_key"`
}

func (r *webhookSecurityPolicyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_webhook_security_policy"
}

func (r *webhookSecurityPolicyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	oauthAttrs := []string{"oauth_client_id", "oauth_client_secret", "oauth_token_url"}
	alsoRequires := func(self string) validator.String {
		var others []path.Expression
		for _, a := range oauthAttrs {
			if a != self {
				others = append(others, path.MatchRoot(a))
			}
		}
		return stringvalidator.AlsoRequires(others...)
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: `
Provides a webhook security policy. A policy can enable signature verification (ECDSA) and OAuth for webhooks such as the Inbound Parse Webhook.

Attach a policy to an Inbound Parse setting with the ` + "`security_policy`" + ` attribute of ` + "`sendgrid_inbound_parse_webhook`" + `. One policy can be attached to several Inbound Parse settings.
		`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the webhook security policy.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the webhook security policy.",
				Required:            true,
			},
			"oauth_client_id": schema.StringAttribute{
				MarkdownDescription: "The OAuth client ID that SendGrid passes to your OAuth server to generate an access token. Removing the OAuth settings recreates the policy, which also issues a new signature key pair.",
				Optional:            true,
				Validators:          []validator.String{alsoRequires("oauth_client_id")},
				PlanModifiers: []planmodifier.String{
					requiresReplaceIfRemovedString(),
				},
			},
			"oauth_client_secret": schema.StringAttribute{
				MarkdownDescription: "The OAuth client secret that SendGrid passes to your OAuth server. SendGrid does not return this value, so changes made outside Terraform are not detected.",
				Optional:            true,
				Sensitive:           true,
				Validators:          []validator.String{alsoRequires("oauth_client_secret")},
			},
			"oauth_token_url": schema.StringAttribute{
				MarkdownDescription: "The URL where SendGrid sends the OAuth client ID and client secret to generate an access token.",
				Optional:            true,
				Validators:          []validator.String{alsoRequires("oauth_token_url")},
			},
			"oauth_scopes": schema.ListAttribute{
				MarkdownDescription: "The OAuth scopes to request. Removing this attribute recreates the policy.",
				ElementType:         types.StringType,
				Optional:            true,
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
					listvalidator.AlsoRequires(path.MatchRoot("oauth_client_id")),
				},
				PlanModifiers: []planmodifier.List{
					requiresReplaceIfRemovedList(),
				},
			},
			"signature_enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether SendGrid signs webhook requests with ECDSA. Changing this updates the policy in place. Enabling it issues a new key pair. (Default: `false`)",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"public_key": schema.StringAttribute{
				MarkdownDescription: "The public key to verify the signature of webhook requests. Set only when `signature_enabled` is `true`.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					useStateForUnknownIfSignatureUnchanged(),
				},
			},
		},
	}
}

func (r *webhookSecurityPolicyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *webhookSecurityPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan webhookSecurityPolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	oauth, diags := webhookSecurityPolicyOAuthFromModel(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := &sendgrid.InputCreateWebhookSecurityPolicy{
		Name:  plan.Name.ValueString(),
		OAuth: oauth,
	}
	if plan.SignatureEnabled.ValueBool() {
		input.Signature = &sendgrid.WebhookSecurityPolicySignature{Enabled: true}
	}

	o, err := r.client.CreateWebhookSecurityPolicy(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Creating webhook security policy",
			fmt.Sprintf("Unable to create webhook security policy, got error: %s", err),
		)
		return
	}
	if o.Policy == nil {
		resp.Diagnostics.AddError(
			"Creating webhook security policy",
			"The create webhook security policy API returned no policy.",
		)
		return
	}

	state, diags := webhookSecurityPolicyModelFromAPI(ctx, o.Policy, plan.OAuthClientSecret)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *webhookSecurityPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state webhookSecurityPolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	o, err := r.client.GetWebhookSecurityPolicy(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Reading webhook security policy",
			fmt.Sprintf("Unable to read webhook security policy (id: %s), got error: %s", id, err),
		)
		return
	}
	if o.Policy == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	newState, diags := webhookSecurityPolicyModelFromAPI(ctx, o.Policy, state.OAuthClientSecret)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *webhookSecurityPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state webhookSecurityPolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	oauth, diags := webhookSecurityPolicyOAuthFromModel(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := &sendgrid.InputUpdateWebhookSecurityPolicy{
		Name:  plan.Name.ValueString(),
		OAuth: oauth,
	}
	// Sending enabled: true while the signature is already enabled is avoided,
	// because it is not known whether the API keeps the current key pair.
	if !plan.SignatureEnabled.Equal(state.SignatureEnabled) {
		input.Signature = &sendgrid.WebhookSecurityPolicySignature{Enabled: plan.SignatureEnabled.ValueBool()}
	}

	id := state.ID.ValueString()
	o, err := r.client.UpdateWebhookSecurityPolicy(ctx, id, input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Updating webhook security policy",
			fmt.Sprintf("Unable to update webhook security policy (id: %s), got error: %s", id, err),
		)
		return
	}
	if o.Policy == nil {
		resp.Diagnostics.AddError(
			"Updating webhook security policy",
			"The update webhook security policy API returned no policy.",
		)
		return
	}

	// Returning right after the update is not enough: GET keeps returning the
	// old values for a few seconds, and the next refresh would show a false diff.
	r.waitForPolicy(ctx, id, o.Policy)

	newState, diags := webhookSecurityPolicyModelFromAPI(ctx, o.Policy, plan.OAuthClientSecret)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *webhookSecurityPolicyResource) waitForPolicy(ctx context.Context, id string, want *sendgrid.WebhookSecurityPolicy) {
	// One up-to-date read is not enough: a later GET can still return the old values.
	const wantConsecutive = 3
	consecutive := 0
	deadline := time.Now().Add(90 * time.Second)
	for time.Now().Before(deadline) {
		o, err := r.client.GetWebhookSecurityPolicy(ctx, id)
		if err == nil && o.Policy != nil && webhookSecurityPolicyUpToDate(o.Policy, want) {
			consecutive++
			if consecutive == wantConsecutive {
				return
			}
		} else {
			consecutive = 0
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(2 * time.Second):
		}
	}
}

func webhookSecurityPolicyUpToDate(got, want *sendgrid.WebhookSecurityPolicy) bool {
	if got.Name != want.Name || (got.Signature != nil) != (want.Signature != nil) || (got.OAuth != nil) != (want.OAuth != nil) {
		return false
	}
	if want.OAuth == nil {
		return true
	}
	return got.OAuth.ClientID == want.OAuth.ClientID &&
		got.OAuth.TokenURL == want.OAuth.TokenURL &&
		slices.Equal(got.OAuth.Scopes, want.OAuth.Scopes)
}

func (r *webhookSecurityPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state webhookSecurityPolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	if err := r.client.DeleteWebhookSecurityPolicy(ctx, id); err != nil {
		resp.Diagnostics.AddError(
			"Deleting webhook security policy",
			fmt.Sprintf("Unable to delete webhook security policy (id: %s), got error: %s", id, err),
		)
	}
}

func (r *webhookSecurityPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func webhookSecurityPolicyOAuthFromModel(ctx context.Context, m webhookSecurityPolicyResourceModel) (*sendgrid.WebhookSecurityPolicyOAuth, diag.Diagnostics) {
	if m.OAuthClientID.IsNull() {
		return nil, nil
	}

	oauth := &sendgrid.WebhookSecurityPolicyOAuth{
		ClientID:     m.OAuthClientID.ValueString(),
		ClientSecret: m.OAuthClientSecret.ValueString(),
		TokenURL:     m.OAuthTokenURL.ValueString(),
	}
	var diags diag.Diagnostics
	if !m.OAuthScopes.IsNull() {
		diags = m.OAuthScopes.ElementsAs(ctx, &oauth.Scopes, false)
	}
	return oauth, diags
}

// The client secret is taken from the caller instead of p, because the API never returns it.
func webhookSecurityPolicyModelFromAPI(ctx context.Context, p *sendgrid.WebhookSecurityPolicy, clientSecret types.String) (webhookSecurityPolicyResourceModel, diag.Diagnostics) {
	m := webhookSecurityPolicyResourceModel{
		ID:                types.StringValue(p.ID),
		Name:              types.StringValue(p.Name),
		OAuthClientID:     types.StringNull(),
		OAuthClientSecret: types.StringNull(),
		OAuthTokenURL:     types.StringNull(),
		OAuthScopes:       types.ListNull(types.StringType),
		SignatureEnabled:  types.BoolValue(p.Signature != nil),
		PublicKey:         types.StringNull(),
	}

	var diags diag.Diagnostics
	if p.OAuth != nil {
		m.OAuthClientID = types.StringValue(p.OAuth.ClientID)
		m.OAuthClientSecret = clientSecret
		m.OAuthTokenURL = types.StringValue(p.OAuth.TokenURL)
		if len(p.OAuth.Scopes) > 0 {
			m.OAuthScopes, diags = types.ListValueFrom(ctx, types.StringType, p.OAuth.Scopes)
		}
	}
	if p.Signature != nil {
		m.PublicKey = types.StringValue(p.Signature.PublicKey)
	}
	return m, diags
}
