// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/kenzo0107/sendgrid"
)

func TestWebhookSecurityPolicyResource_RequiresOAuthOrSignature(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected API call: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	client := sendgrid.New("dummy", sendgrid.OptionBaseURL(srv.URL))
	factories := map[string]func() (tfprotov6.ProviderServer, error){
		"sendgrid": providerserver.NewProtocol6WithError(&fakeAPIProvider{sendgridProvider: sendgridProvider{version: "test"}, client: client}),
	}
	wantErr := regexp.MustCompile(`at least one of oauth or signature`)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "sendgrid_webhook_security_policy" "test" {
  name = "policy"
}
`,
				PlanOnly:    true,
				ExpectError: wantErr,
			},
			{
				Config: `
resource "sendgrid_webhook_security_policy" "test" {
  name = "policy"

  signature = {
    enabled = false
  }
}
`,
				PlanOnly:    true,
				ExpectError: wantErr,
			},
			{
				Config: `
resource "sendgrid_webhook_security_policy" "test" {
  name = "policy"

  signature = {
    enabled = true
  }
}
`,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			{
				Config: `
resource "sendgrid_webhook_security_policy" "test" {
  name = "policy"

  oauth = {
    client_id     = "client"
    client_secret = "secret"
    token_url     = "https://example.com/token"
  }
}
`,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
