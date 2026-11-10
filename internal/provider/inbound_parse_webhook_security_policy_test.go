// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/kenzo0107/sendgrid"
)

type fakeParseSettingsAPI struct {
	mu            sync.Mutex
	setting       map[string]any
	patchedPolicy []any
}

func (f *fakeParseSettingsAPI) attachOutsideTerraform(policyID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.setting["security_policy"] = policyID
}

func (f *fakeParseSettingsAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	switch {
	case r.Method == http.MethodPost && r.URL.Path == "/user/webhooks/parse/settings":
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		f.setting = body
	case r.Method == http.MethodPatch && r.URL.Path == "/user/webhooks/parse/settings/parse.example.com":
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		for k, v := range body {
			if k == "security_policy" {
				continue
			}
			f.setting[k] = v
		}
		if v, ok := body["security_policy"]; ok {
			f.patchedPolicy = append(f.patchedPolicy, v)
			if v == nil {
				delete(f.setting, "security_policy")
			} else {
				f.setting["security_policy"] = v
			}
		}
	case r.Method == http.MethodGet && r.URL.Path == "/user/webhooks/parse/settings/parse.example.com":
	case r.Method == http.MethodDelete && r.URL.Path == "/user/webhooks/parse/settings/parse.example.com":
		f.setting = nil
		w.WriteHeader(http.StatusNoContent)
		return
	default:
		w.WriteHeader(http.StatusNotFound)
		return
	}
	_ = json.NewEncoder(w).Encode(f.setting)
}

type fakeAPIProvider struct {
	sendgridProvider
	client *sendgrid.Client
}

func (p *fakeAPIProvider) Configure(_ context.Context, _ provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	pd := &providerData{client: p.client}
	resp.DataSourceData = pd
	resp.ResourceData = pd
}

func TestInboundParseWebhookResource_SecurityPolicyLifecycle(t *testing.T) {
	api := &fakeParseSettingsAPI{}
	srv := httptest.NewServer(api)
	t.Cleanup(srv.Close)

	client := sendgrid.New("dummy", sendgrid.OptionBaseURL(srv.URL))
	factories := map[string]func() (tfprotov6.ProviderServer, error){
		"sendgrid": providerserver.NewProtocol6WithError(&fakeAPIProvider{sendgridProvider: sendgridProvider{version: "test"}, client: client}),
	}

	config := func(securityPolicy string) string {
		c := `
resource "sendgrid_inbound_parse_webhook" "test" {
  hostname = "parse.example.com"
  url      = "https://example.com/inbound"
`
		if securityPolicy != "" {
			c += `  security_policy = "` + securityPolicy + `"
`
		}
		return c + "}\n"
	}
	resourceName := "sendgrid_inbound_parse_webhook.test"

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: config(""),
				Check:  resource.TestCheckNoResourceAttr(resourceName, "security_policy"),
			},
			{
				PreConfig: func() { api.attachOutsideTerraform("external-policy") },
				Config:    config(""),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.TestCheckNoResourceAttr(resourceName, "security_policy"),
			},
			{
				Config: config("p1"),
				Check:  resource.TestCheckResourceAttr(resourceName, "security_policy", "p1"),
			},
			{
				Config: config(""),
				Check:  resource.TestCheckNoResourceAttr(resourceName, "security_policy"),
			},
			{
				Config: config(""),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})

	api.mu.Lock()
	defer api.mu.Unlock()
	if len(api.patchedPolicy) != 2 || api.patchedPolicy[0] != "p1" || api.patchedPolicy[1] != nil {
		t.Errorf("security_policy sent in PATCH = %v, want [p1 <nil>]", api.patchedPolicy)
	}
}

func TestInboundParseWebhookResource_ImportManagesAttachedSecurityPolicy(t *testing.T) {
	api := &fakeParseSettingsAPI{setting: map[string]any{
		"hostname":        "parse.example.com",
		"url":             "https://example.com/inbound",
		"security_policy": "external-policy",
	}}
	srv := httptest.NewServer(api)
	t.Cleanup(srv.Close)

	client := sendgrid.New("dummy", sendgrid.OptionBaseURL(srv.URL))
	factories := map[string]func() (tfprotov6.ProviderServer, error){
		"sendgrid": providerserver.NewProtocol6WithError(&fakeAPIProvider{sendgridProvider: sendgridProvider{version: "test"}, client: client}),
	}
	config := `
resource "sendgrid_inbound_parse_webhook" "test" {
  hostname = "parse.example.com"
  url      = "https://example.com/inbound"
}
`
	resourceName := "sendgrid_inbound_parse_webhook.test"

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config:             config,
				ResourceName:       resourceName,
				ImportState:        true,
				ImportStateId:      "parse.example.com",
				ImportStatePersist: true,
				ImportStateCheck: func(states []*terraform.InstanceState) error {
					if got := states[0].Attributes["security_policy"]; got != "external-policy" {
						return fmt.Errorf("security_policy = %q, want %q", got, "external-policy")
					}
					return nil
				},
			},
			{
				Config: config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate)},
				},
				Check: resource.TestCheckNoResourceAttr(resourceName, "security_policy"),
			},
		},
	})

	api.mu.Lock()
	defer api.mu.Unlock()
	if len(api.patchedPolicy) != 1 || api.patchedPolicy[0] != nil {
		t.Errorf("security_policy sent in PATCH = %v, want [<nil>]", api.patchedPolicy)
	}
}

func TestSecurityPolicyStateValue(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		managed  bool
		apiValue string
		want     types.String
	}{
		"unmanaged hides a policy attached outside Terraform": {false, "external", types.StringNull()},
		"managed shows the attached policy":                   {true, "p1", types.StringValue("p1")},
		"managed shows a detached policy as null":             {true, "", types.StringNull()},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := securityPolicyStateValue(tc.managed, tc.apiValue); !got.Equal(tc.want) {
				t.Fatalf("got %s, want %s", got, tc.want)
			}
		})
	}
}
