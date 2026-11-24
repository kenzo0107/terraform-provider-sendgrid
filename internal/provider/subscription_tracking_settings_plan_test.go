// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/kenzo0107/sendgrid"
)

// PATCH merges instead of replacing the whole setting because SendGrid keeps
// fields missing from the request; replacing would hide a provider that forgets
// to send a cleared value.
type fakeSubscriptionTrackingAPI struct {
	mu       sync.Mutex
	settings map[string]any
	patched  []map[string]any
}

func newFakeSubscriptionTrackingAPI(initial map[string]any) *fakeSubscriptionTrackingAPI {
	settings := map[string]any{
		"enabled": false, "html_content": "", "plain_content": "", "landing": "", "replace": "", "url": "",
	}
	for k, v := range initial {
		settings[k] = v
	}
	return &fakeSubscriptionTrackingAPI{settings: settings}
}

func (f *fakeSubscriptionTrackingAPI) requests() []map[string]any {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.patched
}

func (f *fakeSubscriptionTrackingAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")

	if r.URL.Path != "/tracking_settings/subscription" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	switch r.Method {
	case http.MethodGet:
	case http.MethodPatch:
		var body map[string]any
		if !decodeFakeRequest(w, r, &body) {
			return
		}
		f.patched = append(f.patched, body)
		for k, v := range body {
			f.settings[k] = v
		}
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	_ = json.NewEncoder(w).Encode(f.settings)
}

func fakeSubscriptionTrackingProviderFactories(t *testing.T, api http.Handler) map[string]func() (tfprotov6.ProviderServer, error) {
	t.Helper()
	srv := httptest.NewServer(api)
	t.Cleanup(srv.Close)

	client := sendgrid.New("dummy", sendgrid.OptionBaseURL(srv.URL))
	return map[string]func() (tfprotov6.ProviderServer, error){
		"sendgrid": providerserver.NewProtocol6WithError(&fakeAPIProvider{sendgridProvider: sendgridProvider{version: "test"}, client: client}),
	}
}

func TestSubscriptionTrackingSettingsResource_Lifecycle(t *testing.T) {
	api := newFakeSubscriptionTrackingAPI(map[string]any{"landing": "<p>set outside terraform</p>"})
	resourceName := "sendgrid_subscription_tracking_settings.test"
	withoutURL := `
resource "sendgrid_subscription_tracking_settings" "test" {
  enabled      = true
  html_content = "<p><% Unsubscribe %></p>"
}
`

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: fakeSubscriptionTrackingProviderFactories(t, api),
		Steps: []resource.TestStep{
			{
				Config: `
resource "sendgrid_subscription_tracking_settings" "test" {
  enabled      = true
  html_content = "<p><% Unsubscribe %></p>"
  url          = "https://example.com/unsubscribe"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "html_content", "<p><% Unsubscribe %></p>"),
					resource.TestCheckResourceAttr(resourceName, "url", "https://example.com/unsubscribe"),
					resource.TestCheckResourceAttr(resourceName, "landing", ""),
				),
			},
			{
				Config: withoutURL,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate)},
				},
				Check: resource.TestCheckResourceAttr(resourceName, "url", ""),
			},
			{
				Config: withoutURL,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			{
				Config:        withoutURL,
				ResourceName:  resourceName,
				ImportState:   true,
				ImportStateId: "",
				ImportStateCheck: func(states []*terraform.InstanceState) error {
					if len(states) != 1 {
						return fmt.Errorf("imported %d states, want 1", len(states))
					}
					if got := states[0].Attributes["html_content"]; got != "<p><% Unsubscribe %></p>" {
						return fmt.Errorf("html_content = %q, want %q", got, "<p><% Unsubscribe %></p>")
					}
					return nil
				},
			},
		},
	})

	if got := api.settings["landing"]; got != "" {
		t.Errorf("landing on the account = %q, want it cleared because the configuration omits it", got)
	}

	wantKeys := []string{"enabled", "html_content", "landing", "plain_content", "replace", "url"}
	requests := api.requests()
	if len(requests) != 2 {
		t.Fatalf("got %d PATCH requests, want 2: %+v", len(requests), requests)
	}
	for i, req := range requests {
		keys := make([]string, 0, len(req))
		for k := range req {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		if !reflect.DeepEqual(keys, wantKeys) {
			t.Errorf("PATCH %d sent %v, want every attribute %v", i, keys, wantKeys)
		}
	}
	if got := requests[1]["url"]; got != "" {
		t.Errorf("second PATCH url = %#v, want \"\" to clear it", got)
	}
}

func TestSubscriptionTrackingSettingsDataSource_ReadsCurrentSettings(t *testing.T) {
	api := newFakeSubscriptionTrackingAPI(map[string]any{"enabled": true, "replace": "[unsubscribe]"})

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: fakeSubscriptionTrackingProviderFactories(t, api),
		Steps: []resource.TestStep{
			{
				Config: `
data "sendgrid_subscription_tracking_settings" "test" {
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.sendgrid_subscription_tracking_settings.test", "enabled", "true"),
					resource.TestCheckResourceAttr("data.sendgrid_subscription_tracking_settings.test", "replace", "[unsubscribe]"),
				),
			},
		},
	})

	if got := api.requests(); len(got) != 0 {
		t.Errorf("subscription tracking requests = %+v, want none from a data source read", got)
	}
}
