// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kenzo0107/sendgrid"
)

var _ resource.Resource = &webhookSecurityPolicyResource{}
var _ resource.ResourceWithImportState = &webhookSecurityPolicyResource{}
var _ resource.ResourceWithValidateConfig = &webhookSecurityPolicyResource{}

func newWebhookSecurityPolicyResource() resource.Resource {
	return &webhookSecurityPolicyResource{}
}

type webhookSecurityPolicyResource struct {
	client *sendgrid.Client
}

type webhookSecurityPolicyResourceModel struct {
	ID        types.String                         `tfsdk:"id"`
	Name      types.String                         `tfsdk:"name"`
	Signature *webhookSecurityPolicySignatureModel `tfsdk:"signature"`
	OAuth     *webhookSecurityPolicyOAuthModel     `tfsdk:"oauth"`
}

type webhookSecurityPolicySignatureModel struct {
	Enabled   types.Bool   `tfsdk:"enabled"`
	PublicKey types.String `tfsdk:"public_key"`
}

type webhookSecurityPolicyOAuthModel struct {
	ClientID     types.String `tfsdk:"client_id"`
	ClientSecret types.String `tfsdk:"client_secret"`
	TokenURL     types.String `tfsdk:"token_url"`
	Scopes       types.List   `tfsdk:"scopes"`
}

var webhookSecurityPolicySignatureAttrTypes = map[string]attr.Type{
	"enabled":    types.BoolType,
	"public_key": types.StringType,
}

func (r *webhookSecurityPolicyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_webhook_security_policy"
}

func (r *webhookSecurityPolicyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
Provides a webhook security policy. A policy can enable signature verification (ECDSA) and OAuth for webhooks such as the Inbound Parse Webhook.

Attach a policy to an Inbound Parse setting with the ` + "`security_policy`" + ` attribute of ` + "`sendgrid_inbound_parse_webhook`" + `. One policy can be attached to several Inbound Parse settings.

SendGrid rejects deleting a policy that is still attached to an Inbound Parse setting. When an attached policy is recreated (for example, by removing its OAuth settings), set ` + "`create_before_destroy = true`" + ` in its ` + "`lifecycle`" + ` block so that the Inbound Parse settings move to the new policy before the old one is deleted.
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
			"signature": schema.SingleNestedAttribute{
				MarkdownDescription: "The signature verification (ECDSA) settings. (Default: `{ enabled = false }`)",
				Optional:            true,
				Computed:            true,
				Default: objectdefault.StaticValue(types.ObjectValueMust(webhookSecurityPolicySignatureAttrTypes, map[string]attr.Value{
					"enabled":    types.BoolValue(false),
					"public_key": types.StringNull(),
				})),
				Attributes: map[string]schema.Attribute{
					"enabled": schema.BoolAttribute{
						MarkdownDescription: "Whether SendGrid signs webhook requests with ECDSA. Changing this updates the policy in place. Enabling it issues a new key pair.",
						Required:            true,
					},
					"public_key": schema.StringAttribute{
						MarkdownDescription: "The public key to verify the signature of webhook requests. Set only when `enabled` is `true`.",
						Computed:            true,
						PlanModifiers: []planmodifier.String{
							useStateForUnknownIfSignatureUnchanged(),
						},
					},
				},
			},
			"oauth": schema.SingleNestedAttribute{
				MarkdownDescription: "The OAuth settings. Removing this attribute recreates the policy, which also issues a new signature key pair.",
				Optional:            true,
				PlanModifiers: []planmodifier.Object{
					requiresReplaceIfRemovedObject(),
				},
				Attributes: map[string]schema.Attribute{
					"client_id": schema.StringAttribute{
						MarkdownDescription: "The OAuth client ID that SendGrid passes to your OAuth server to generate an access token.",
						Required:            true,
					},
					"client_secret": schema.StringAttribute{
						MarkdownDescription: "The OAuth client secret that SendGrid passes to your OAuth server. SendGrid does not return this value, so changes made outside Terraform are not detected.",
						Required:            true,
						Sensitive:           true,
					},
					"token_url": schema.StringAttribute{
						MarkdownDescription: "The URL where SendGrid sends the OAuth client ID and client secret to generate an access token.",
						Required:            true,
					},
					"scopes": schema.ListAttribute{
						MarkdownDescription: "The OAuth scopes to request. Removing this attribute recreates the policy.",
						ElementType:         types.StringType,
						Optional:            true,
						Validators: []validator.List{
							listvalidator.SizeAtLeast(1),
						},
						PlanModifiers: []planmodifier.List{
							requiresReplaceIfRemovedList(),
						},
					},
				},
			},
		},
	}
}

func (r *webhookSecurityPolicyResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var oauth types.Object
	var enabled types.Bool
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("oauth"), &oauth)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("signature").AtName("enabled"), &enabled)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if oauth.IsNull() && !enabled.IsUnknown() && !enabled.ValueBool() {
		resp.Diagnostics.AddAttributeError(
			path.Root("oauth"),
			"Missing webhook security policy settings",
			"SendGrid requires at least one of oauth or signature. Set oauth, or set signature.enabled to true.",
		)
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

	oauth, diags := webhookSecurityPolicyOAuthFromModel(ctx, plan.OAuth)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := &sendgrid.InputCreateWebhookSecurityPolicy{
		Name:  plan.Name.ValueString(),
		OAuth: oauth,
	}
	if plan.Signature.Enabled.ValueBool() {
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

	r.waitForPolicy(ctx, o.Policy.ID, o.Policy)

	state, diags := webhookSecurityPolicyModelFromAPI(ctx, o.Policy, oauthClientSecret(plan.OAuth))
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

	newState, diags := webhookSecurityPolicyModelFromAPI(ctx, o.Policy, oauthClientSecret(state.OAuth))
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

	oauth, diags := webhookSecurityPolicyOAuthFromModel(ctx, plan.OAuth)
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
	if !plan.Signature.Enabled.Equal(state.Signature.Enabled) {
		input.Signature = &sendgrid.WebhookSecurityPolicySignature{Enabled: plan.Signature.Enabled.ValueBool()}
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

	newState, diags := webhookSecurityPolicyModelFromAPI(ctx, o.Policy, oauthClientSecret(plan.OAuth))
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

func webhookSecurityPolicyOAuthFromModel(ctx context.Context, m *webhookSecurityPolicyOAuthModel) (*sendgrid.WebhookSecurityPolicyOAuth, diag.Diagnostics) {
	if m == nil {
		return nil, nil
	}

	oauth := &sendgrid.WebhookSecurityPolicyOAuth{
		ClientID:     m.ClientID.ValueString(),
		ClientSecret: m.ClientSecret.ValueString(),
		TokenURL:     m.TokenURL.ValueString(),
	}
	var diags diag.Diagnostics
	if !m.Scopes.IsNull() {
		diags = m.Scopes.ElementsAs(ctx, &oauth.Scopes, false)
	}
	return oauth, diags
}

func oauthClientSecret(m *webhookSecurityPolicyOAuthModel) types.String {
	if m == nil {
		return types.StringNull()
	}
	return m.ClientSecret
}

// The client secret is taken from the caller instead of p, because the API never returns it.
func webhookSecurityPolicyModelFromAPI(ctx context.Context, p *sendgrid.WebhookSecurityPolicy, clientSecret types.String) (webhookSecurityPolicyResourceModel, diag.Diagnostics) {
	m := webhookSecurityPolicyResourceModel{
		ID:   types.StringValue(p.ID),
		Name: types.StringValue(p.Name),
		Signature: &webhookSecurityPolicySignatureModel{
			Enabled:   types.BoolValue(p.Signature != nil),
			PublicKey: types.StringNull(),
		},
	}
	if p.Signature != nil {
		m.Signature.PublicKey = types.StringValue(p.Signature.PublicKey)
	}

	var diags diag.Diagnostics
	if p.OAuth != nil {
		m.OAuth = &webhookSecurityPolicyOAuthModel{
			ClientID:     types.StringValue(p.OAuth.ClientID),
			ClientSecret: clientSecret,
			TokenURL:     types.StringValue(p.OAuth.TokenURL),
			Scopes:       types.ListNull(types.StringType),
		}
		if len(p.OAuth.Scopes) > 0 {
			m.OAuth.Scopes, diags = types.ListValueFrom(ctx, types.StringType, p.OAuth.Scopes)
		}
	}
	return m, diags
}
