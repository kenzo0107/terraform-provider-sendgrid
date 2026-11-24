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
type fakeGoogleAnalyticsAPI struct {
	mu       sync.Mutex
	settings map[string]any
	patched  []map[string]any
}

func newFakeGoogleAnalyticsAPI(initial map[string]any) *fakeGoogleAnalyticsAPI {
	settings := map[string]any{
		"enabled": false, "utm_source": "", "utm_medium": "", "utm_term": "", "utm_content": "", "utm_campaign": "",
	}
	for k, v := range initial {
		settings[k] = v
	}
	return &fakeGoogleAnalyticsAPI{settings: settings}
}

func (f *fakeGoogleAnalyticsAPI) requests() []map[string]any {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.patched
}

func (f *fakeGoogleAnalyticsAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")

	if r.URL.Path != "/tracking_settings/google_analytics" {
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

func fakeGoogleAnalyticsProviderFactories(t *testing.T, api http.Handler) map[string]func() (tfprotov6.ProviderServer, error) {
	t.Helper()
	srv := httptest.NewServer(api)
	t.Cleanup(srv.Close)

	client := sendgrid.New("dummy", sendgrid.OptionBaseURL(srv.URL))
	return map[string]func() (tfprotov6.ProviderServer, error){
		"sendgrid": providerserver.NewProtocol6WithError(&fakeAPIProvider{sendgridProvider: sendgridProvider{version: "test"}, client: client}),
	}
}

func TestGoogleAnalyticsSettingsResource_Lifecycle(t *testing.T) {
	api := newFakeGoogleAnalyticsAPI(map[string]any{"utm_term": "set outside terraform"})
	resourceName := "sendgrid_google_analytics_settings.test"
	withoutCampaign := `
resource "sendgrid_google_analytics_settings" "test" {
  enabled    = true
  utm_source = "sendgrid"
}
`

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: fakeGoogleAnalyticsProviderFactories(t, api),
		Steps: []resource.TestStep{
			{
				Config: `
resource "sendgrid_google_analytics_settings" "test" {
  enabled      = true
  utm_source   = "sendgrid"
  utm_campaign = "launch"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "utm_source", "sendgrid"),
					resource.TestCheckResourceAttr(resourceName, "utm_campaign", "launch"),
					resource.TestCheckResourceAttr(resourceName, "utm_term", ""),
				),
			},
			{
				Config: withoutCampaign,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate)},
				},
				Check: resource.TestCheckResourceAttr(resourceName, "utm_campaign", ""),
			},
			{
				Config: withoutCampaign,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			{
				Config:        withoutCampaign,
				ResourceName:  resourceName,
				ImportState:   true,
				ImportStateId: "",
				ImportStateCheck: func(states []*terraform.InstanceState) error {
					if len(states) != 1 {
						return fmt.Errorf("imported %d states, want 1", len(states))
					}
					if got := states[0].Attributes["utm_source"]; got != "sendgrid" {
						return fmt.Errorf("utm_source = %q, want %q", got, "sendgrid")
					}
					return nil
				},
			},
		},
	})

	if got := api.settings["utm_term"]; got != "" {
		t.Errorf("utm_term on the account = %q, want it cleared because the configuration omits it", got)
	}

	wantKeys := []string{"enabled", "utm_campaign", "utm_content", "utm_medium", "utm_source", "utm_term"}
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
	if got := requests[1]["utm_campaign"]; got != "" {
		t.Errorf("second PATCH utm_campaign = %#v, want \"\" to clear it", got)
	}
}

func TestGoogleAnalyticsSettingsDataSource_ReadsCurrentSettings(t *testing.T) {
	api := newFakeGoogleAnalyticsAPI(map[string]any{"enabled": true, "utm_medium": "email"})

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: fakeGoogleAnalyticsProviderFactories(t, api),
		Steps: []resource.TestStep{
			{
				Config: `
data "sendgrid_google_analytics_settings" "test" {
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.sendgrid_google_analytics_settings.test", "enabled", "true"),
					resource.TestCheckResourceAttr("data.sendgrid_google_analytics_settings.test", "utm_medium", "email"),
				),
			},
		},
	})

	if got := api.requests(); len(got) != 0 {
		t.Errorf("Google Analytics requests = %+v, want none from a data source read", got)
	}
}
