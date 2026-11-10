// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
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

		if m.ID.ValueString() != "p1" || m.Name.ValueString() != "policy" {
			t.Fatalf("unexpected model: %+v", m)
		}
		if !m.Signature.Enabled.ValueBool() || m.Signature.PublicKey.ValueString() != "key" {
			t.Fatalf("unexpected signature: %+v", m.Signature)
		}
		var scopes []string
		m.OAuth.Scopes.ElementsAs(ctx, &scopes, false)
		if m.OAuth.ClientID.ValueString() != "client" || m.OAuth.ClientSecret.ValueString() != "secret" ||
			m.OAuth.TokenURL.ValueString() != "https://example.com/token" || len(scopes) != 2 {
			t.Fatalf("unexpected oauth: %+v", m.OAuth)
		}
	})

	t.Run("no oauth and no signature map to null oauth and a disabled signature", func(t *testing.T) {
		t.Parallel()

		m, diags := webhookSecurityPolicyModelFromAPI(ctx, &sendgrid.WebhookSecurityPolicy{ID: "p1", Name: "policy"}, types.StringValue("secret"))
		if diags.HasError() {
			t.Fatal(diags)
		}
		if m.OAuth != nil || m.Signature.Enabled.ValueBool() || !m.Signature.PublicKey.IsNull() {
			t.Fatalf("unexpected model: oauth=%+v signature=%+v", m.OAuth, m.Signature)
		}
	})
}

func TestWebhookSecurityPolicyOAuthFromModel(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	got, diags := webhookSecurityPolicyOAuthFromModel(ctx, nil)
	if diags.HasError() || got != nil {
		t.Fatalf("no oauth: got %+v, %v", got, diags)
	}

	scopes, _ := types.ListValueFrom(ctx, types.StringType, []string{"a"})
	got, diags = webhookSecurityPolicyOAuthFromModel(ctx, &webhookSecurityPolicyOAuthModel{
		ClientID:     types.StringValue("client"),
		ClientSecret: types.StringValue("secret"),
		TokenURL:     types.StringValue("https://example.com/token"),
		Scopes:       scopes,
	})
	if diags.HasError() || got == nil || got.ClientID != "client" || got.ClientSecret != "secret" ||
		got.TokenURL != "https://example.com/token" || len(got.Scopes) != 1 || got.Scopes[0] != "a" {
		t.Fatalf("oauth: got %+v, %v", got, diags)
	}
}

func TestRequiresReplaceIfRemovedObject(t *testing.T) {
	t.Parallel()

	attrTypes := map[string]attr.Type{"client_id": types.StringType}
	object := func(clientID string) types.Object {
		return types.ObjectValueMust(attrTypes, map[string]attr.Value{"client_id": types.StringValue(clientID)})
	}
	null := types.ObjectNull(attrTypes)

	tests := map[string]struct {
		state, plan types.Object
		want        bool
	}{
		"removed replaces":     {object("a"), null, true},
		"changed updates":      {object("a"), object("b"), false},
		"added updates":        {null, object("a"), false},
		"unchanged is no diff": {object("a"), object("a"), false},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			req := planmodifier.ObjectRequest{
				Path:       path.Root("oauth"),
				StateValue: tc.state,
				PlanValue:  tc.plan,
				State:      tfsdk.State{Raw: fakeRawState(t)},
				Plan:       tfsdk.Plan{Raw: fakeRawState(t)},
			}
			resp := &planmodifier.ObjectResponse{PlanValue: tc.plan}

			requiresReplaceIfRemovedObject().PlanModifyObject(context.Background(), req, resp)

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
			ID:        types.StringValue("p1"),
			Name:      types.StringValue("policy"),
			Signature: &webhookSecurityPolicySignatureModel{Enabled: types.BoolValue(enabled), PublicKey: publicKey},
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
				Path:       path.Root("signature").AtName("public_key"),
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
