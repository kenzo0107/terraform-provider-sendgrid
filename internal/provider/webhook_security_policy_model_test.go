// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kenzo0107/sendgrid"
)

func TestSecurityPolicyUpdateValue(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		plan, state types.String
		want        *string
	}{
		"never set leaves the field out":        {types.StringNull(), types.StringNull(), nil},
		"set attaches the policy":               {types.StringValue("p1"), types.StringNull(), sendgrid.String("p1")},
		"changed attaches the new policy":       {types.StringValue("p2"), types.StringValue("p1"), sendgrid.String("p2")},
		"removed detaches with an empty string": {types.StringNull(), types.StringValue("p1"), sendgrid.String("")},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := securityPolicyUpdateValue(tc.plan, tc.state)
			switch {
			case tc.want == nil && got != nil:
				t.Fatalf("got %q, want nil", *got)
			case tc.want != nil && (got == nil || *got != *tc.want):
				t.Fatalf("got %v, want %q", got, *tc.want)
			}
		})
	}
}

func TestWebhookSecurityPolicyModelFromAPI(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("oauth and signature, keeping the client secret from the caller", func(t *testing.T) {
		t.Parallel()

		m, diags := webhookSecurityPolicyModelFromAPI(ctx, &sendgrid.WebhookSecurityPolicy{
			ID:   "p1",
			Name: "policy",
			OAuth: &sendgrid.WebhookSecurityPolicyOAuth{
				ClientID: "client",
				TokenURL: "https://example.com/token",
				Scopes:   []string{"a", "b"},
			},
			Signature: &sendgrid.WebhookSecurityPolicySignature{PublicKey: "key"},
		}, types.StringValue("secret"))
		if diags.HasError() {
			t.Fatal(diags)
		}

		var scopes []string
		m.OAuthScopes.ElementsAs(ctx, &scopes, false)
		if m.ID.ValueString() != "p1" || m.Name.ValueString() != "policy" ||
			m.OAuthClientID.ValueString() != "client" || m.OAuthClientSecret.ValueString() != "secret" ||
			m.OAuthTokenURL.ValueString() != "https://example.com/token" || len(scopes) != 2 ||
			!m.SignatureEnabled.ValueBool() || m.PublicKey.ValueString() != "key" {
			t.Fatalf("unexpected model: %+v", m)
		}
	})

	t.Run("no oauth and no signature map to null", func(t *testing.T) {
		t.Parallel()

		m, diags := webhookSecurityPolicyModelFromAPI(ctx, &sendgrid.WebhookSecurityPolicy{ID: "p1", Name: "policy"}, types.StringValue("secret"))
		if diags.HasError() {
			t.Fatal(diags)
		}
		if !m.OAuthClientID.IsNull() || !m.OAuthClientSecret.IsNull() || !m.OAuthTokenURL.IsNull() ||
			!m.OAuthScopes.IsNull() || m.SignatureEnabled.ValueBool() || !m.PublicKey.IsNull() {
			t.Fatalf("unexpected model: %+v", m)
		}
	})
}

func TestWebhookSecurityPolicyOAuthFromModel(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	got, diags := webhookSecurityPolicyOAuthFromModel(ctx, webhookSecurityPolicyResourceModel{
		OAuthClientID:     types.StringNull(),
		OAuthClientSecret: types.StringNull(),
		OAuthTokenURL:     types.StringNull(),
		OAuthScopes:       types.ListNull(types.StringType),
	})
	if diags.HasError() || got != nil {
		t.Fatalf("no oauth: got %+v, %v", got, diags)
	}

	scopes, _ := types.ListValueFrom(ctx, types.StringType, []string{"a"})
	got, diags = webhookSecurityPolicyOAuthFromModel(ctx, webhookSecurityPolicyResourceModel{
		OAuthClientID:     types.StringValue("client"),
		OAuthClientSecret: types.StringValue("secret"),
		OAuthTokenURL:     types.StringValue("https://example.com/token"),
		OAuthScopes:       scopes,
	})
	if diags.HasError() || got == nil || got.ClientID != "client" || got.ClientSecret != "secret" ||
		got.TokenURL != "https://example.com/token" || len(got.Scopes) != 1 || got.Scopes[0] != "a" {
		t.Fatalf("oauth: got %+v, %v", got, diags)
	}
}

func TestRequiresReplaceIfRemovedString(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		state, plan types.String
		want        bool
	}{
		"removed replaces":     {types.StringValue("a"), types.StringNull(), true},
		"changed updates":      {types.StringValue("a"), types.StringValue("b"), false},
		"added updates":        {types.StringNull(), types.StringValue("a"), false},
		"unchanged is no diff": {types.StringValue("a"), types.StringValue("a"), false},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			req := planmodifier.StringRequest{
				Path:       path.Root("oauth_client_id"),
				StateValue: tc.state,
				PlanValue:  tc.plan,
				State:      tfsdk.State{Raw: fakeRawState(t)},
				Plan:       tfsdk.Plan{Raw: fakeRawState(t)},
			}
			resp := &planmodifier.StringResponse{PlanValue: tc.plan}

			requiresReplaceIfRemovedString().PlanModifyString(context.Background(), req, resp)

			if resp.RequiresReplace != tc.want {
				t.Fatalf("RequiresReplace = %v, want %v", resp.RequiresReplace, tc.want)
			}
		})
	}
}

func TestUseStateForUnknownIfSignatureUnchanged(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	schemaResp := &resource.SchemaResponse{}
	newWebhookSecurityPolicyResource().Schema(ctx, resource.SchemaRequest{}, schemaResp)
	s := schemaResp.Schema

	model := func(enabled bool, publicKey types.String) webhookSecurityPolicyResourceModel {
		return webhookSecurityPolicyResourceModel{
			ID:                types.StringValue("p1"),
			Name:              types.StringValue("policy"),
			OAuthClientID:     types.StringNull(),
			OAuthClientSecret: types.StringNull(),
			OAuthTokenURL:     types.StringNull(),
			OAuthScopes:       types.ListNull(types.StringType),
			SignatureEnabled:  types.BoolValue(enabled),
			PublicKey:         publicKey,
		}
	}

	tests := map[string]struct {
		stateEnabled, planEnabled bool
		want                      types.String
	}{
		"unchanged keeps the key":       {true, true, types.StringValue("key")},
		"disabled leaves it unknown":    {true, false, types.StringUnknown()},
		"re-enabled leaves it unknown":  {false, true, types.StringUnknown()},
		"still disabled keeps the null": {false, false, types.StringNull()},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			stateKey := types.StringNull()
			if tc.stateEnabled {
				stateKey = types.StringValue("key")
			}

			state := tfsdk.State{Schema: s}
			if diags := state.Set(ctx, model(tc.stateEnabled, stateKey)); diags.HasError() {
				t.Fatal(diags)
			}
			plan := tfsdk.Plan{Schema: s}
			if diags := plan.Set(ctx, model(tc.planEnabled, types.StringUnknown())); diags.HasError() {
				t.Fatal(diags)
			}

			req := planmodifier.StringRequest{
				Path:       path.Root("public_key"),
				StateValue: stateKey,
				PlanValue:  types.StringUnknown(),
				State:      state,
				Plan:       plan,
			}
			resp := &planmodifier.StringResponse{PlanValue: types.StringUnknown()}

			useStateForUnknownIfSignatureUnchanged().PlanModifyString(ctx, req, resp)

			if resp.Diagnostics.HasError() {
				t.Fatal(resp.Diagnostics)
			}
			if !resp.PlanValue.Equal(tc.want) {
				t.Fatalf("PlanValue = %s, want %s", resp.PlanValue, tc.want)
			}
		})
	}
}

func TestWebhookSecurityPolicyUpToDate(t *testing.T) {
	t.Parallel()

	base := func() *sendgrid.WebhookSecurityPolicy {
		return &sendgrid.WebhookSecurityPolicy{
			Name:      "policy",
			OAuth:     &sendgrid.WebhookSecurityPolicyOAuth{ClientID: "c", TokenURL: "https://example.com/token", Scopes: []string{"a"}},
			Signature: &sendgrid.WebhookSecurityPolicySignature{PublicKey: "key"},
		}
	}

	tests := map[string]struct {
		change func(p *sendgrid.WebhookSecurityPolicy)
		want   bool
	}{
		"same values":          {func(p *sendgrid.WebhookSecurityPolicy) {}, true},
		"public key differs":   {func(p *sendgrid.WebhookSecurityPolicy) { p.Signature.PublicKey = "other" }, true},
		"name differs":         {func(p *sendgrid.WebhookSecurityPolicy) { p.Name = "old" }, false},
		"signature differs":    {func(p *sendgrid.WebhookSecurityPolicy) { p.Signature = nil }, false},
		"oauth missing":        {func(p *sendgrid.WebhookSecurityPolicy) { p.OAuth = nil }, false},
		"oauth token differs":  {func(p *sendgrid.WebhookSecurityPolicy) { p.OAuth.TokenURL = "https://old" }, false},
		"oauth scopes differ":  {func(p *sendgrid.WebhookSecurityPolicy) { p.OAuth.Scopes = []string{"b"} }, false},
		"oauth client differs": {func(p *sendgrid.WebhookSecurityPolicy) { p.OAuth.ClientID = "old" }, false},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := base()
			tc.change(got)
			if r := webhookSecurityPolicyUpToDate(got, base()); r != tc.want {
				t.Fatalf("webhookSecurityPolicyUpToDate = %v, want %v", r, tc.want)
			}
		})
	}
}
