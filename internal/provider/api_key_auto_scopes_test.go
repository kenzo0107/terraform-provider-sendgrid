// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/kenzo0107/sendgrid"
)

const fakeAPIKeyPath = "/api_keys"

var (
	twoFactorRequiredAccount = []string{"2fa_required", "sender_verification_eligible", "sender_verification_exempt"}
	twoFactorExemptAccount   = []string{"2fa_exempt", "sender_verification_eligible", "sender_verification_exempt"}
)

type fakeAPIKeyAPI struct {
	mu      sync.Mutex
	granted []string
	keys    map[string]*sendgrid.OutputGetAPIKey
	ids     int
	sent    [][]string
}

func newFakeAPIKeyAPI(keys ...*sendgrid.OutputGetAPIKey) *fakeAPIKeyAPI {
	f := &fakeAPIKeyAPI{keys: map[string]*sendgrid.OutputGetAPIKey{}, granted: twoFactorRequiredAccount}
	for _, k := range keys {
		f.keys[k.ApiKeyId] = k
	}
	return f
}

func (f *fakeAPIKeyAPI) grant(scopes []string) []string {
	f.sent = append(f.sent, scopes)
	granted := slices.Clone(scopes)
	for _, s := range f.granted {
		if !slices.Contains(granted, s) {
			granted = append(granted, s)
		}
	}
	return granted
}

func (f *fakeAPIKeyAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	id := strings.TrimPrefix(strings.TrimPrefix(r.URL.Path, fakeAPIKeyPath), "/")
	switch {
	case r.Method == http.MethodPost && id == "":
		var in sendgrid.InputCreateAPIKey
		if !decodeFakeRequest(w, r, &in) {
			return
		}
		f.ids++
		k := &sendgrid.OutputGetAPIKey{ApiKeyId: fmt.Sprintf("k%d", f.ids), Name: in.Name, Scopes: f.grant(in.Scopes)}
		f.keys[k.ApiKeyId] = k
		_ = json.NewEncoder(w).Encode(sendgrid.OutputCreateAPIKey{ApiKey: "SG.secret", ApiKeyId: k.ApiKeyId, Name: k.Name, Scopes: k.Scopes})
	case r.Method == http.MethodGet && f.keys[id] != nil:
		_ = json.NewEncoder(w).Encode(f.keys[id])
	case r.Method == http.MethodPut && f.keys[id] != nil:
		var in sendgrid.InputUpdateAPIKeyNameAndScopes
		if !decodeFakeRequest(w, r, &in) {
			return
		}
		k := f.keys[id]
		k.Name = in.Name
		k.Scopes = f.grant(in.Scopes)
		_ = json.NewEncoder(w).Encode(sendgrid.OutputUpdateAPIKeyNameAndScopes{ApiKeyId: id, Name: k.Name, Scopes: k.Scopes})
	case r.Method == http.MethodPatch && f.keys[id] != nil:
		var in sendgrid.InputUpdateAPIKeyName
		if !decodeFakeRequest(w, r, &in) {
			return
		}
		f.keys[id].Name = in.Name
		_ = json.NewEncoder(w).Encode(sendgrid.OutputUpdateAPIKeyName{ApiKeyId: id, Name: in.Name})
	case r.Method == http.MethodDelete && f.keys[id] != nil:
		delete(f.keys, id)
		w.WriteHeader(http.StatusNoContent)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func fakeAPIKeyProviderFactories(t *testing.T, api *fakeAPIKeyAPI) map[string]func() (tfprotov6.ProviderServer, error) {
	t.Helper()
	srv := httptest.NewServer(api)
	t.Cleanup(srv.Close)

	client := sendgrid.New("dummy", sendgrid.OptionBaseURL(srv.URL))
	return map[string]func() (tfprotov6.ProviderServer, error){
		"sendgrid": providerserver.NewProtocol6WithError(&fakeAPIProvider{sendgridProvider: sendgridProvider{version: "test"}, client: client}),
	}
}

func testAPIKeyConfig(scopes ...string) string {
	return fmt.Sprintf(`
resource "sendgrid_api_key" "test" {
  name   = "key"
  scopes = %s
}
`, quoteHCLList(scopes))
}

func quoteHCLList(items []string) string {
	quoted := make([]string, 0, len(items))
	for _, s := range items {
		quoted = append(quoted, fmt.Sprintf("%q", s))
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}

func (f *fakeAPIKeyAPI) sentScopes() [][]string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.sent)
}

func TestAPIKeyResource_AutoScopesGrantedBySendGridCauseNoDiff(t *testing.T) {
	api := newFakeAPIKeyAPI()
	resourceName := "sendgrid_api_key.test"
	config := testAPIKeyConfig("mail.send")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: fakeAPIKeyProviderFactories(t, api),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "scopes.#", "1"),
					resource.TestCheckTypeSetElemAttr(resourceName, "scopes.*", "mail.send"),
				),
			},
			{
				Config: config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}

func TestAPIKeyResource_DeclaredAutoScopeNotGrantedBySendGridCausesNoDiff(t *testing.T) {
	api := newFakeAPIKeyAPI()
	resourceName := "sendgrid_api_key.test"
	created := testAPIKeyConfig("mail.send", "2fa_exempt")
	updated := testAPIKeyConfig("mail.send", "stats.read", "2fa_exempt")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: fakeAPIKeyProviderFactories(t, api),
		Steps: []resource.TestStep{
			{
				Config: created,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "scopes.#", "2"),
					resource.TestCheckTypeSetElemAttr(resourceName, "scopes.*", "2fa_exempt"),
				),
			},
			{
				Config: created,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			{
				Config: updated,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "scopes.#", "3"),
					resource.TestCheckTypeSetElemAttr(resourceName, "scopes.*", "stats.read"),
				),
			},
			{
				Config: updated,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})

	for _, scopes := range api.sentScopes() {
		for _, s := range scopes {
			if slices.Contains(autoScopes, s) {
				t.Errorf("sent scopes %v include auto scope %q", scopes, s)
			}
		}
	}
	if got := len(api.sentScopes()); got != 2 {
		t.Errorf("scope requests = %d, want 2 (create and update)", got)
	}
}

func TestAPIKeyResource_ImportThenDeclaredAutoScopeIsAddedByOneUpdate(t *testing.T) {
	api := newFakeAPIKeyAPI(&sendgrid.OutputGetAPIKey{
		ApiKeyId: "existing",
		Name:     "key",
		Scopes:   []string{"mail.send", "2fa_required", "sender_verification_eligible"},
	})
	resourceName := "sendgrid_api_key.test"
	config := testAPIKeyConfig("mail.send", "2fa_exempt")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: fakeAPIKeyProviderFactories(t, api),
		Steps: []resource.TestStep{
			{
				Config:             config,
				ResourceName:       resourceName,
				ImportState:        true,
				ImportStateId:      "existing",
				ImportStatePersist: true,
			},
			{
				Config: config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate)},
				},
				Check: resource.TestCheckTypeSetElemAttr(resourceName, "scopes.*", "2fa_exempt"),
			},
			{
				Config: config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})

	if got := api.sentScopes(); len(got) != 1 || !slices.Equal(got[0], []string{"mail.send"}) {
		t.Errorf("sent scopes = %v, want one update with [mail.send]", got)
	}
}

func TestAPIKeyResource_OnlyAutoScopesIsAnError(t *testing.T) {
	api := newFakeAPIKeyAPI()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: fakeAPIKeyProviderFactories(t, api),
		Steps: []resource.TestStep{
			{
				Config:      testAPIKeyConfig("2fa_exempt"),
				ExpectError: regexp.MustCompile(`full access API key`),
			},
		},
	})

	if got := api.sentScopes(); len(got) != 0 {
		t.Errorf("sent scopes = %v, want no request", got)
	}
}

func TestAPIKeyDataSource_ExcludesAutoScopes(t *testing.T) {
	api := newFakeAPIKeyAPI(&sendgrid.OutputGetAPIKey{
		ApiKeyId: "existing",
		Name:     "key",
		Scopes:   []string{"mail.send", "2fa_required", "sender_verification_eligible"},
	})
	dataSourceName := "data.sendgrid_api_key.test"

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: fakeAPIKeyProviderFactories(t, api),
		Steps: []resource.TestStep{
			{
				Config: `
data "sendgrid_api_key" "test" {
  id = "existing"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "scopes.#", "1"),
					resource.TestCheckTypeSetElemAttr(dataSourceName, "scopes.*", "mail.send"),
				),
			},
		},
	})
}

func TestAPIKeyResource_AutoScopeLifecycleOnEachAccountType(t *testing.T) {
	without := testAPIKeyConfig("mail.send")
	withExempt := testAPIKeyConfig("mail.send", "2fa_exempt")

	accounts := map[string][]string{
		"2fa_required account": twoFactorRequiredAccount,
		"2fa_exempt account":   twoFactorExemptAccount,
	}
	sequences := map[string][]string{
		"without 2fa_exempt":            {without, testAPIKeyConfig("mail.send", "stats.read")},
		"with 2fa_exempt":               {withExempt, testAPIKeyConfig("mail.send", "stats.read", "2fa_exempt")},
		"2fa_exempt added then removed": {without, withExempt, without},
	}

	for accountName, granted := range accounts {
		for sequenceName, configs := range sequences {
			t.Run(accountName+"/"+sequenceName, func(t *testing.T) {
				api := newFakeAPIKeyAPI()
				api.granted = granted

				var steps []resource.TestStep
				for _, config := range configs {
					steps = append(steps,
						resource.TestStep{Config: config},
						resource.TestStep{
							Config: config,
							ConfigPlanChecks: resource.ConfigPlanChecks{
								PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
							},
						},
					)
				}

				resource.UnitTest(t, resource.TestCase{
					ProtoV6ProviderFactories: fakeAPIKeyProviderFactories(t, api),
					Steps:                    steps,
				})

				for _, scopes := range api.sentScopes() {
					for _, s := range scopes {
						if slices.Contains(autoScopes, s) {
							t.Errorf("sent scopes %v include auto scope %q", scopes, s)
						}
					}
				}
			})
		}
	}
}
