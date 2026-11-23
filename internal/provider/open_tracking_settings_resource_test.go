// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/kenzo0107/sendgrid"
)

// testAccTrackingSettingsClient builds a client the same way the provider does
// during acceptance tests, so the snapshot/restore helper below talks to the
// same account (or subuser) the test changes.
func testAccTrackingSettingsClient(t *testing.T) *sendgrid.Client {
	t.Helper()

	var opts []sendgrid.Option
	if subuser := os.Getenv("SENDGRID_SUBUSER"); subuser != "" {
		opts = append(opts, sendgrid.OptionSubuser(subuser))
	}
	if os.Getenv("SENDGRID_REGION") == "eu" {
		opts = append(opts, sendgrid.OptionBaseURL(sendgrid.BaseURLEU))
	}

	return sendgrid.New(os.Getenv("SENDGRID_API_KEY"), opts...)
}

// testAccRestoreOpenTrackingSettings snapshots the account-wide open tracking
// setting and puts it back once the test finishes, since acceptance tests here
// change real account settings.
func testAccRestoreOpenTrackingSettings(t *testing.T) {
	t.Helper()

	ctx := context.Background()
	c := testAccTrackingSettingsClient(t)

	o, err := c.GetOpenTrackingSettings(ctx)
	if err != nil {
		t.Fatalf("reading open tracking settings before the test: %s", err)
	}

	t.Cleanup(func() {
		if _, err := c.UpdateOpenTrackingSettings(ctx, &sendgrid.InputUpdateOpenTrackingSettings{
			Enabled: o.Enabled,
		}); err != nil {
			t.Errorf("restoring open tracking settings: %s", err)
		}
	})
}

func TestAccOpenTrackingSettingsResource(t *testing.T) {
	resourceName := "sendgrid_open_tracking_settings.test"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccRestoreOpenTrackingSettings(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccOpenTrackingSettingsResource(false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "enabled", "false"),
				),
			},
			// ImportState testing
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: importOpenTrackingSettingsStateIdFunc(),
			},
			// Update and Read testing
			{
				Config: testAccOpenTrackingSettingsResource(true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
				),
			},
		},
	})
}

func testAccOpenTrackingSettingsResource(enabled bool) string {
	return fmt.Sprintf(`
resource "sendgrid_open_tracking_settings" "test" {
  enabled = %t
}
`, enabled)
}

func importOpenTrackingSettingsStateIdFunc() resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		return "", nil
	}
}
