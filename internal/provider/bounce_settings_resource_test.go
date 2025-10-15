// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccBounceSettingsResource(t *testing.T) {
	resourceName := "sendgrid_bounce_settings.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccBounceSettingsResourceConfig(3649, 3649),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "soft_bounces", "3649"),
					resource.TestCheckResourceAttr(resourceName, "hard_bounces", "3649"),
				),
			},
			// ImportState testing
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: importBounceSettingsStateIdFunc(),
			},
			// Update and Read testing
			{
				Config: testAccBounceSettingsResourceConfig(3650, 3650),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "soft_bounces", "3650"),
					resource.TestCheckResourceAttr(resourceName, "hard_bounces", "3650"),
				),
			},
		},
	})
}

func testAccBounceSettingsResourceConfig(soft_bounce_purge_days, hard_bounce_purge_days int64) string {
	return fmt.Sprintf(`
resource "sendgrid_bounce_settings" "test" {
  soft_bounces = %d
  hard_bounces = %d
}`, soft_bounce_purge_days, hard_bounce_purge_days)
}

func importBounceSettingsStateIdFunc() resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		return "", nil
	}
}
