// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
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

const fakePolicyPath = "/user/webhooks/security/policies"

type fakePolicyAPI struct {
	mu       sync.Mutex
	policies map[string]*sendgrid.WebhookSecurityPolicy
	keys     int
	created  []sendgrid.InputCreateWebhookSecurityPolicy
	updated  []sendgrid.InputUpdateWebhookSecurityPolicy
}

func newFakePolicyAPI(policies ...*sendgrid.WebhookSecurityPolicy) *fakePolicyAPI {
	f := &fakePolicyAPI{policies: map[string]*sendgrid.WebhookSecurityPolicy{}}
	for _, p := range policies {
		f.policies[p.ID] = p
	}
	return f
}

func (f *fakePolicyAPI) signature(in *sendgrid.WebhookSecurityPolicySignature) *sendgrid.WebhookSecurityPolicySignature {
	if in == nil || !in.Enabled {
		return nil
	}
	f.keys++
	return &sendgrid.WebhookSecurityPolicySignature{PublicKey: fmt.Sprintf("key-%d", f.keys)}
}

func withoutSecret(o *sendgrid.WebhookSecurityPolicyOAuth) *sendgrid.WebhookSecurityPolicyOAuth {
	if o == nil {
		return nil
	}
	c := *o
	c.ClientSecret = ""
	return &c
}

func (f *fakePolicyAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	id := strings.TrimPrefix(strings.TrimPrefix(r.URL.Path, fakePolicyPath), "/")
	switch {
	case r.Method == http.MethodPost && id == "":
		var in sendgrid.InputCreateWebhookSecurityPolicy
		_ = json.NewDecoder(r.Body).Decode(&in)
		f.created = append(f.created, in)
		p := &sendgrid.WebhookSecurityPolicy{
			ID:        fmt.Sprintf("p%d", len(f.policies)+1),
			Name:      in.Name,
			OAuth:     withoutSecret(in.OAuth),
			Signature: f.signature(in.Signature),
		}
		f.policies[p.ID] = p
		_ = json.NewEncoder(w).Encode(sendgrid.OutputCreateWebhookSecurityPolicy{Policy: p})
	case r.Method == http.MethodGet && f.policies[id] != nil:
		_ = json.NewEncoder(w).Encode(sendgrid.OutputGetWebhookSecurityPolicy{Policy: f.policies[id]})
	case r.Method == http.MethodPatch && f.policies[id] != nil:
		var in sendgrid.InputUpdateWebhookSecurityPolicy
		_ = json.NewDecoder(r.Body).Decode(&in)
		f.updated = append(f.updated, in)
		p := f.policies[id]
		p.Name = in.Name
		if in.OAuth != nil {
			p.OAuth = withoutSecret(in.OAuth)
		}
		if in.Signature != nil {
			p.Signature = f.signature(in.Signature)
		}
		_ = json.NewEncoder(w).Encode(sendgrid.OutputUpdateWebhookSecurityPolicy{Policy: p})
	case r.Method == http.MethodDelete && f.policies[id] != nil:
		delete(f.policies, id)
		w.WriteHeader(http.StatusNoContent)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func fakePolicyProviderFactories(t *testing.T, api *fakePolicyAPI) map[string]func() (tfprotov6.ProviderServer, error) {
	t.Helper()
	srv := httptest.NewServer(api)
	t.Cleanup(srv.Close)

	client := sendgrid.New("dummy", sendgrid.OptionBaseURL(srv.URL))
	return map[string]func() (tfprotov6.ProviderServer, error){
		"sendgrid": providerserver.NewProtocol6WithError(&fakeAPIProvider{sendgridProvider: sendgridProvider{version: "test"}, client: client}),
	}
}

func TestWebhookSecurityPolicyResource_OAuthOnlyHasNoDiffAfterCreate(t *testing.T) {
	api := newFakePolicyAPI()
	resourceName := "sendgrid_webhook_security_policy.test"
	config := `
resource "sendgrid_webhook_security_policy" "test" {
  name = "policy"

  oauth = {
    client_id     = "client"
    client_secret = "secret"
    token_url     = "https://example.com/token"
  }
}
`

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: fakePolicyProviderFactories(t, api),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "signature.enabled", "false"),
					resource.TestCheckNoResourceAttr(resourceName, "signature.public_key"),
					resource.TestCheckResourceAttr(resourceName, "oauth.client_secret", "secret"),
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

	api.mu.Lock()
	defer api.mu.Unlock()
	if len(api.created) != 1 || api.created[0].Signature != nil {
		t.Errorf("create requests = %+v, want one without signature", api.created)
	}
}

func TestWebhookSecurityPolicyResource_UnknownValuesPassValidation(t *testing.T) {
	api := newFakePolicyAPI()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: fakePolicyProviderFactories(t, api),
		Steps: []resource.TestStep{
			{
				Config: `
resource "terraform_data" "enabled" {
  input = true
}

resource "terraform_data" "oauth" {
  input = {
    client_id     = "client"
    client_secret = "secret"
    token_url     = "https://example.com/token"
  }
}

resource "sendgrid_webhook_security_policy" "signature" {
  name = "signature"

  signature = {
    enabled = terraform_data.enabled.output
  }
}

resource "sendgrid_webhook_security_policy" "oauth" {
  name  = "oauth"
  oauth = terraform_data.oauth.output
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("sendgrid_webhook_security_policy.signature", "signature.enabled", "true"),
					resource.TestCheckResourceAttrSet("sendgrid_webhook_security_policy.signature", "signature.public_key"),
					resource.TestCheckResourceAttr("sendgrid_webhook_security_policy.oauth", "oauth.client_id", "client"),
				),
			},
		},
	})
}

func TestWebhookSecurityPolicyResource_ImportUpdatesClientSecretInPlace(t *testing.T) {
	api := newFakePolicyAPI(&sendgrid.WebhookSecurityPolicy{
		ID:        "existing",
		Name:      "policy",
		OAuth:     &sendgrid.WebhookSecurityPolicyOAuth{ClientID: "client", TokenURL: "https://example.com/token"},
		Signature: &sendgrid.WebhookSecurityPolicySignature{PublicKey: "existing-key"},
	})
	resourceName := "sendgrid_webhook_security_policy.test"
	config := `
resource "sendgrid_webhook_security_policy" "test" {
  name = "policy"

  signature = {
    enabled = true
  }

  oauth = {
    client_id     = "client"
    client_secret = "secret"
    token_url     = "https://example.com/token"
  }
}
`

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: fakePolicyProviderFactories(t, api),
		Steps: []resource.TestStep{
			{
				Config:             config,
				ResourceName:       resourceName,
				ImportState:        true,
				ImportStateId:      "existing",
				ImportStatePersist: true,
				ImportStateCheck: func(states []*terraform.InstanceState) error {
					if got, ok := states[0].Attributes["oauth.client_secret"]; ok && got != "" {
						return fmt.Errorf("oauth.client_secret = %q, want unset after import", got)
					}
					return nil
				},
			},
			{
				Config: config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate)},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "id", "existing"),
					resource.TestCheckResourceAttr(resourceName, "oauth.client_secret", "secret"),
					resource.TestCheckResourceAttr(resourceName, "signature.public_key", "existing-key"),
				),
			},
		},
	})

	api.mu.Lock()
	defer api.mu.Unlock()
	if len(api.updated) != 1 || api.updated[0].OAuth == nil || api.updated[0].OAuth.ClientSecret != "secret" || api.updated[0].Signature != nil {
		t.Errorf("update requests = %+v, want one sending the client secret and no signature", api.updated)
	}
}
