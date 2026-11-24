// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccSubscriptionTrackingSettingsDataSource(t *testing.T) {
	resourceName := "data.sendgrid_subscription_tracking_settings.test"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccRestoreSubscriptionTrackingSettings(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "sendgrid_subscription_tracking_settings" "test" {
  enabled = true
  replace = "[tf-acc-unsubscribe]"
}

data "sendgrid_subscription_tracking_settings" "test" {
  depends_on = [sendgrid_subscription_tracking_settings.test]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "replace", "[tf-acc-unsubscribe]"),
					resource.TestCheckResourceAttr(resourceName, "url", ""),
				),
			},
		},
	})
}
