// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccOpenTrackingSettingsDataSource(t *testing.T) {
	resourceName := "data.sendgrid_open_tracking_settings.test"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccRestoreOpenTrackingSettings(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testAccOpenTrackingSettingsDataSourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
				),
			},
		},
	})
}

func testAccOpenTrackingSettingsDataSourceConfig() string {
	return `
resource "sendgrid_open_tracking_settings" "test" {
  enabled = true
}

data "sendgrid_open_tracking_settings" "test" {
  depends_on = [sendgrid_open_tracking_settings.test]
}
`
}
