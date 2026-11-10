// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccWebhookSecurityPolicyDataSource(t *testing.T) {
	dataSourceName := "data.sendgrid_webhook_security_policy.test"
	resourceName := "sendgrid_webhook_security_policy.test"
	name := fmt.Sprintf("test-acc-%s", acctest.RandString(16))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "sendgrid_webhook_security_policy" "test" {
  name              = "%s"
  signature = {
    enabled = true
  }
}

data "sendgrid_webhook_security_policy" "test" {
  id = sendgrid_webhook_security_policy.test.id
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(dataSourceName, "id", resourceName, "id"),
					resource.TestCheckResourceAttr(dataSourceName, "name", name),
					resource.TestCheckResourceAttr(dataSourceName, "signature.enabled", "true"),
					resource.TestCheckResourceAttrPair(dataSourceName, "signature.public_key", resourceName, "signature.public_key"),
				),
			},
		},
	})
}
