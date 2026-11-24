// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/kenzo0107/sendgrid"
)

func testAccRestoreClickTrackingSettings(t *testing.T) {
	t.Helper()

	ctx := context.Background()
	c := testAccTrackingSettingsClient(t)

	o, err := c.GetClickTrackingSettings(ctx)
	if err != nil {
		t.Fatalf("reading click tracking settings before the test: %s", err)
	}

	t.Cleanup(func() {
		if _, err := c.UpdateClickTrackingSettings(ctx, &sendgrid.InputUpdateClickTrackingSettings{
			Enabled: o.Enabled,
		}); err != nil {
			t.Errorf("restoring click tracking settings: %s", err)
		}
	})
}

func TestAccClickTrackingSettingsResource(t *testing.T) {
	resourceName := "sendgrid_click_tracking_settings.test"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccRestoreClickTrackingSettings(t)
		},
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
