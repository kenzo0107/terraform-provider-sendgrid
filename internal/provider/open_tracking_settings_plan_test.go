// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/kenzo0107/sendgrid"
)

const fakeOpenTrackingPath = "/tracking_settings/open"

// fakeOpenTrackingAPI serves the open tracking settings endpoint, which is an
// account-wide singleton: a PATCH replaces the stored setting and a GET returns
// it.
type fakeOpenTrackingAPI struct {
	mu       sync.Mutex
	settings sendgrid.OutputGetOpenTrackingSettings

	// Raw PATCH bodies, so tests can assert exactly what the provider sent.
	patched []map[string]any
}

func (f *fakeOpenTrackingAPI) requests() []map[string]any {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.patched
}

func (f *fakeOpenTrackingAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")

	if r.URL.Path != fakeOpenTrackingPath {
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
		enabled, _ := body["enabled"].(bool)
		f.settings.Enabled = enabled
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	_ = json.NewEncoder(w).Encode(f.settings)
}

func fakeOpenTrackingProviderFactories(t *testing.T, api *fakeOpenTrackingAPI) map[string]func() (tfprotov6.ProviderServer, error) {
	t.Helper()
	srv := httptest.NewServer(api)
	t.Cleanup(srv.Close)

	client := sendgrid.New("dummy", sendgrid.OptionBaseURL(srv.URL))
	return map[string]func() (tfprotov6.ProviderServer, error){
		"sendgrid": providerserver.NewProtocol6WithError(&fakeAPIProvider{sendgridProvider: sendgridProvider{version: "test"}, client: client}),
	}
}

func TestOpenTrackingSettingsResource_Lifecycle(t *testing.T) {
	api := &fakeOpenTrackingAPI{}
	resourceName := "sendgrid_open_tracking_settings.test"
	enabledConfig := `
resource "sendgrid_open_tracking_settings" "test" {
  enabled = true
}
`

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: fakeOpenTrackingProviderFactories(t, api),
		Steps: []resource.TestStep{
			{
				Config: `
resource "sendgrid_open_tracking_settings" "test" {
}
`,
				Check: resource.TestCheckResourceAttr(resourceName, "enabled", "false"),
			},
			{
				Config: enabledConfig,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate)},
				},
				Check: resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
			},
			{
				Config: enabledConfig,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			{
				Config:        enabledConfig,
				ResourceName:  resourceName,
				ImportState:   true,
				ImportStateId: "",
				ImportStateCheck: func(states []*terraform.InstanceState) error {
					if len(states) != 1 {
						return fmt.Errorf("imported %d states, want 1", len(states))
					}
					if got := states[0].Attributes["enabled"]; got != "true" {
						return fmt.Errorf("enabled = %q, want %q", got, "true")
					}
					return nil
				},
			},
		},
	})

	// enabled has no omitempty on the API side, so every write has to carry the
	// planned value rather than only the changed one.
	if got := api.requests(); len(got) != 2 || got[0]["enabled"] != false || got[1]["enabled"] != true {
		t.Errorf("open tracking requests = %+v, want enabled false then true", got)
	}
}

func TestOpenTrackingSettingsDataSource_ReadsCurrentSettings(t *testing.T) {
	api := &fakeOpenTrackingAPI{settings: sendgrid.OutputGetOpenTrackingSettings{Enabled: true}}

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: fakeOpenTrackingProviderFactories(t, api),
		Steps: []resource.TestStep{
			{
				Config: `
data "sendgrid_open_tracking_settings" "test" {
}
`,
				Check: resource.TestCheckResourceAttr("data.sendgrid_open_tracking_settings.test", "enabled", "true"),
			},
		},
	})

	if got := api.requests(); len(got) != 0 {
		t.Errorf("open tracking requests = %+v, want none from a data source read", got)
	}
}
