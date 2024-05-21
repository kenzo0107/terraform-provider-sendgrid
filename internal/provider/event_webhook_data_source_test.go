// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccEventWebhookDataSource(t *testing.T) {
	resourceName := "data.sendgrid_event_webhook.test"

	url := fmt.Sprintf("https://test-acc-%s.com", acctest.RandString(16))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testEventWebhookDataSourceConfig(url),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
		},
	})
}

func testEventWebhookDataSourceConfig(url string) string {
	return fmt.Sprintf(`
resource "sendgrid_event_webhook" "test" {
	url = "%s"
	enabled = false
}

data "sendgrid_event_webhook" "test" {
	id = sendgrid_event_webhook.test.id
}
`, url)
}
