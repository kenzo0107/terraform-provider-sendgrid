// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccBounceSettingsResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccBounceSettingsResourceConfig("30"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("sendgrid_bounce_settings.test", "soft_bounce_purge_days", "30"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "sendgrid_bounce_settings.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: testAccBounceSettingsResourceConfig("60"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("sendgrid_bounce_settings.test", "soft_bounce_purge_days", "60"),
				),
			},
		},
	})
}

func testAccBounceSettingsResourceConfig(days string) string {
	return `
resource "sendgrid_bounce_settings" "test" {
  soft_bounce_purge_days = ` + days + `
}
`
}