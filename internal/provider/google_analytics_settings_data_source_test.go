// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccGoogleAnalyticsSettingsDataSource(t *testing.T) {
	resourceName := "data.sendgrid_google_analytics_settings.test"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccRestoreGoogleAnalyticsSettings(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "sendgrid_google_analytics_settings" "test" {
  enabled      = true
  utm_campaign = "tf-acc-campaign"
}

data "sendgrid_google_analytics_settings" "test" {
  depends_on = [sendgrid_google_analytics_settings.test]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "utm_campaign", "tf-acc-campaign"),
					resource.TestCheckResourceAttr(resourceName, "utm_source", ""),
				),
			},
		},
	})
}
