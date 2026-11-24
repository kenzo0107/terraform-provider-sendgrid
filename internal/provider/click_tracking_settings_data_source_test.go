// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccClickTrackingSettingsDataSource(t *testing.T) {
	resourceName := "data.sendgrid_click_tracking_settings.test"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccRestoreClickTrackingSettings(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testAccClickTrackingSettingsDataSourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
				),
			},
		},
	})
}

func testAccClickTrackingSettingsDataSourceConfig() string {
	return `
resource "sendgrid_click_tracking_settings" "test" {
  enabled = true
}

data "sendgrid_click_tracking_settings" "test" {
  depends_on = [sendgrid_click_tracking_settings.test]
}
`
}
