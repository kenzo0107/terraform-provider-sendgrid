// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/kenzo0107/sendgrid"
)

func testAccRestoreGoogleAnalyticsSettings(t *testing.T) {
	t.Helper()

	ctx := context.Background()
	c := testAccTrackingSettingsClient(t)

	o, err := c.GetGoogleAnalyticsSettings(ctx)
	if err != nil {
		t.Fatalf("reading Google Analytics settings before the test: %s", err)
	}

	t.Cleanup(func() {
		if _, err := c.UpdateGoogleAnalyticsSettings(ctx, &sendgrid.InputUpdateGoogleAnalyticsSettings{
			Enabled:     o.Enabled,
			UTMSource:   sendgrid.String(o.UTMSource),
			UTMMedium:   sendgrid.String(o.UTMMedium),
			UTMTerm:     sendgrid.String(o.UTMTerm),
			UTMContent:  sendgrid.String(o.UTMContent),
			UTMCampaign: sendgrid.String(o.UTMCampaign),
		}); err != nil {
			t.Errorf("restoring Google Analytics settings: %s", err)
		}
	})
}

func TestAccGoogleAnalyticsSettingsResource(t *testing.T) {
	resourceName := "sendgrid_google_analytics_settings.test"

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
  utm_source   = "sendgrid"
  utm_medium   = "email"
  utm_term     = "tf-acc-term"
  utm_content  = "tf-acc-content"
  utm_campaign = "tf-acc-campaign"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "utm_source", "sendgrid"),
					resource.TestCheckResourceAttr(resourceName, "utm_medium", "email"),
					resource.TestCheckResourceAttr(resourceName, "utm_term", "tf-acc-term"),
					resource.TestCheckResourceAttr(resourceName, "utm_content", "tf-acc-content"),
					resource.TestCheckResourceAttr(resourceName, "utm_campaign", "tf-acc-campaign"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: func(*terraform.State) (string, error) { return "", nil },
			},
			{
				Config: `
resource "sendgrid_google_analytics_settings" "test" {
  enabled    = false
  utm_source = "sendgrid"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "enabled", "false"),
					resource.TestCheckResourceAttr(resourceName, "utm_source", "sendgrid"),
					resource.TestCheckResourceAttr(resourceName, "utm_medium", ""),
					resource.TestCheckResourceAttr(resourceName, "utm_term", ""),
					resource.TestCheckResourceAttr(resourceName, "utm_content", ""),
					resource.TestCheckResourceAttr(resourceName, "utm_campaign", ""),
				),
			},
		},
	})
}
