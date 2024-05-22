// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccInboundParseWebhookResource(t *testing.T) {
	resourceName := "sendgrid_inbound_parse_webhook.test"

	hostname := os.Getenv("INBOUND_PARSE_WEBHOOK_HOSTNAME")
	url := fmt.Sprintf("https://test-acc-%s.com", acctest.RandString(16))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccInboundParseWebhookResourceConfig(hostname, url, false, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "hostname", hostname),
					resource.TestCheckResourceAttr(resourceName, "url", url),
					resource.TestCheckResourceAttr(resourceName, "spam_check", "false"),
					resource.TestCheckResourceAttr(resourceName, "send_raw", "false"),
				),
			},
			// ImportState testing
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateId:     hostname,
			},
			// Update and Read testing
			{
				Config: testAccInboundParseWebhookResourceConfig(hostname, url, false, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "hostname", hostname),
					resource.TestCheckResourceAttr(resourceName, "url", url),
					resource.TestCheckResourceAttr(resourceName, "spam_check", "false"),
					resource.TestCheckResourceAttr(resourceName, "send_raw", "true"),
				),
			},
		},
	})
}

func testAccInboundParseWebhookResourceConfig(hostname, url string, spamCheck, sendRaw bool) string {
	return fmt.Sprintf(`
resource "sendgrid_inbound_parse_webhook" "test" {
  hostname = "%s"
  url = "%s"
  spam_check = %t
  send_raw = %t
}
`, hostname, url, spamCheck, sendRaw)
}
