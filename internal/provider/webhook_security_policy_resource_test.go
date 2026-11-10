// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccWebhookSecurityPolicyResource(t *testing.T) {
	resourceName := "sendgrid_webhook_security_policy.test"
	name := fmt.Sprintf("test-acc-%s", acctest.RandString(16))
	renamedName := name + "-renamed"
	updatedName := name + "-updated"
	var publicKey string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccWebhookSecurityPolicyResourceConfig(name, "https://example.com/token", "webhooks:read", true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "oauth_client_id", "test-acc-client"),
					resource.TestCheckResourceAttr(resourceName, "oauth_token_url", "https://example.com/token"),
					resource.TestCheckResourceAttr(resourceName, "oauth_scopes.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "signature_enabled", "true"),
					resource.TestCheckResourceAttrWith(resourceName, "public_key", func(v string) error {
						if v == "" {
							return fmt.Errorf("public_key is empty")
						}
						publicKey = v
						return nil
					}),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"oauth_client_secret"},
			},
			{
				Config: testAccWebhookSecurityPolicyResourceConfig(renamedName, "https://example.com/token", "webhooks:read", true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", renamedName),
					resource.TestCheckResourceAttr(resourceName, "signature_enabled", "true"),
					resource.TestCheckResourceAttrWith(resourceName, "public_key", func(v string) error {
						if v != publicKey {
							return fmt.Errorf("public_key changed on rename: %q -> %q", publicKey, v)
						}
						return nil
					}),
				),
			},
			{
				Config: testAccWebhookSecurityPolicyResourceConfig(updatedName, "https://example.com/token", "webhooks:read", false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", updatedName),
					resource.TestCheckResourceAttr(resourceName, "oauth_client_id", "test-acc-client"),
					resource.TestCheckResourceAttr(resourceName, "signature_enabled", "false"),
					resource.TestCheckNoResourceAttr(resourceName, "public_key"),
				),
			},
			{
				Config: testAccWebhookSecurityPolicyResourceConfig(updatedName, "https://example.com/token2", "webhooks:write", false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "oauth_token_url", "https://example.com/token2"),
					resource.TestCheckResourceAttr(resourceName, "oauth_scopes.0", "webhooks:write"),
				),
			},
			{
				Config: testAccWebhookSecurityPolicyResourceConfig(updatedName, "https://example.com/token2", "webhooks:write", true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "signature_enabled", "true"),
					resource.TestCheckResourceAttrSet(resourceName, "public_key"),
				),
			},
		},
	})
}

func testAccWebhookSecurityPolicyResourceConfig(name, tokenURL, scope string, signatureEnabled bool) string {
	return fmt.Sprintf(`
resource "sendgrid_webhook_security_policy" "test" {
  name                = "%s"
  oauth_client_id     = "test-acc-client"
  oauth_client_secret = "test-acc-secret"
  oauth_token_url     = "%s"
  oauth_scopes        = ["%s"]
  signature_enabled   = %t
}
`, name, tokenURL, scope, signatureEnabled)
}
