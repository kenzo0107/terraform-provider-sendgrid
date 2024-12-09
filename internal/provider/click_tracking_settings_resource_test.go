// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccClickTrackingSettingsResource(t *testing.T) {
	resourceName := "sendgrid_click_tracking_settings.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccClickTrackingSettingsResource(false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "enabled", "false"),
				),
			},
			// ImportState testing
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: importClickTrackingSettingsStateIdFunc(),
			},
		},
	})
}

func testAccClickTrackingSettingsResource(enabled bool) string {
	return fmt.Sprintf(`
resource "sendgrid_click_tracking_settings" "test" {
  enabled = %t
}
`, enabled)
}

func importClickTrackingSettingsStateIdFunc() resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		return "", nil
	}
}
