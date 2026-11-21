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
	"github.com/hashicorp/terraform-plugin-testing/terraform"
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

var accountTypes = map[string][]string{
	"2fa_required account": twoFactorRequiredAccount,
	"2fa_exempt account":   twoFactorExemptAccount,
}

func existingAPIKey(granted []string) *sendgrid.OutputGetAPIKey {
	return &sendgrid.OutputGetAPIKey{
		ApiKeyId: "existing",
		Name:     "key",
		Scopes:   append([]string{"mail.send"}, granted...),
	}
}

func expectScopes(resourceName string, scopes []string) resource.TestCheckFunc {
	checks := []resource.TestCheckFunc{
		resource.TestCheckResourceAttr(resourceName, "scopes.#", fmt.Sprint(len(scopes))),
	}
	for _, s := range scopes {
		checks = append(checks, resource.TestCheckTypeSetElemAttr(resourceName, "scopes.*", s))
	}
	return resource.ComposeAggregateTestCheckFunc(checks...)
}

func TestAPIKeyResource_AutoScopeLifecycleOnEachAccountType(t *testing.T) {
	resourceName := "sendgrid_api_key.test"

	sequences := map[string][][]string{
		"without 2fa_exempt":            {{"mail.send"}, {"mail.send", "stats.read"}},
		"with 2fa_exempt":               {{"mail.send", "2fa_exempt"}, {"mail.send", "stats.read", "2fa_exempt"}},
		"2fa_exempt added then removed": {{"mail.send"}, {"mail.send", "2fa_exempt"}, {"mail.send"}},
	}

	for accountName, granted := range accountTypes {
		for sequenceName, sequence := range sequences {
			t.Run(accountName+"/"+sequenceName, func(t *testing.T) {
				api := newFakeAPIKeyAPI()
				api.granted = granted

				var steps []resource.TestStep
				var wantSent [][]string
				for i, scopes := range sequence {
					action := plancheck.ResourceActionUpdate
					if i == 0 {
						action = plancheck.ResourceActionCreate
					}
					config := testAPIKeyConfig(scopes...)
					steps = append(steps,
						resource.TestStep{
							Config: config,
							ConfigPlanChecks: resource.ConfigPlanChecks{
								PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(resourceName, action)},
							},
							Check: expectScopes(resourceName, scopes),
						},
						resource.TestStep{
							Config: config,
							ConfigPlanChecks: resource.ConfigPlanChecks{
								PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
							},
						},
					)
					wantSent = append(wantSent, excludeAutoScopes(scopes))
				}

				resource.UnitTest(t, resource.TestCase{
					ProtoV6ProviderFactories: fakeAPIKeyProviderFactories(t, api),
					Steps:                    steps,
				})

				got := api.sentScopes()
				if !slices.EqualFunc(got, wantSent, slices.Equal[[]string]) {
					t.Errorf("sent scopes = %v, want %v (one request per create and update, without auto scopes)", got, wantSent)
				}
			})
		}
	}
}

func TestAPIKeyResource_ImportThenDeclaredAutoScopeIsAddedByOneUpdate(t *testing.T) {
	resourceName := "sendgrid_api_key.test"
	config := testAPIKeyConfig("mail.send", "2fa_exempt")

	for accountName, granted := range accountTypes {
		t.Run(accountName, func(t *testing.T) {
			api := newFakeAPIKeyAPI(existingAPIKey(granted))
			api.granted = granted

			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: fakeAPIKeyProviderFactories(t, api),
				Steps: []resource.TestStep{
					{
						Config:             config,
						ResourceName:       resourceName,
						ImportState:        true,
						ImportStateId:      "existing",
						ImportStatePersist: true,
						ImportStateCheck: func(states []*terraform.InstanceState) error {
							if len(states) != 1 {
								return fmt.Errorf("imported %d states, want 1", len(states))
							}
							attrs := states[0].Attributes
							if attrs["scopes.#"] != "1" || attrs["scopes.0"] != "mail.send" {
								return fmt.Errorf("imported scopes = %v, want [mail.send]", attrs)
							}
							return nil
						},
					},
					{
						Config: config,
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate)},
						},
						Check: expectScopes(resourceName, []string{"mail.send", "2fa_exempt"}),
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
		})
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

func TestAPIKeyResource_NoScopesIsAnErrorWithoutRequest(t *testing.T) {
	cases := map[string]struct {
		config  string
		wantErr *regexp.Regexp
	}{
		"empty scopes": {
			config:  testAPIKeyConfig(),
			wantErr: regexp.MustCompile(`full access API key`),
		},
		"omitted scopes": {
			config: `
resource "sendgrid_api_key" "test" {
  name = "key"
}
`,
			wantErr: regexp.MustCompile(`"scopes" is required`),
		},
		"empty scopes known only at apply": {
			config: `
resource "terraform_data" "scopes" {
  input = []
}

resource "sendgrid_api_key" "test" {
  name   = "key"
  scopes = terraform_data.scopes.output
}
`,
			wantErr: regexp.MustCompile(`full access API key`),
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			api := newFakeAPIKeyAPI()

			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: fakeAPIKeyProviderFactories(t, api),
				Steps: []resource.TestStep{
					{
						Config:      tc.config,
						ExpectError: tc.wantErr,
					},
				},
			})

			if got := api.sentScopes(); len(got) != 0 {
				t.Errorf("sent scopes = %v, want no request", got)
			}
		})
	}
}

func TestAPIKeyResource_UpdateToEmptyScopesIsAnErrorAndKeepsScopes(t *testing.T) {
	resourceName := "sendgrid_api_key.test"
	api := newFakeAPIKeyAPI()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: fakeAPIKeyProviderFactories(t, api),
		Steps: []resource.TestStep{
			{
				Config: testAPIKeyConfig("mail.send"),
				Check:  expectScopes(resourceName, []string{"mail.send"}),
			},
			{
				Config:      testAPIKeyConfig(),
				ExpectError: regexp.MustCompile(`full access API key`),
			},
			{
				Config: testAPIKeyConfig("mail.send"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})

	if got := api.sentScopes(); len(got) != 1 || !slices.Equal(got[0], []string{"mail.send"}) {
		t.Errorf("sent scopes = %v, want only the create with [mail.send]", got)
	}
}

func TestAPIKeyDataSource_ExcludesAutoScopes(t *testing.T) {
	dataSourceName := "data.sendgrid_api_key.test"

	for accountName, granted := range accountTypes {
		t.Run(accountName, func(t *testing.T) {
			api := newFakeAPIKeyAPI(existingAPIKey(granted))

			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: fakeAPIKeyProviderFactories(t, api),
				Steps: []resource.TestStep{
					{
						Config: `
data "sendgrid_api_key" "test" {
  id = "existing"
}
`,
						Check: expectScopes(dataSourceName, []string{"mail.send"}),
					},
				},
			})
		})
	}
}
