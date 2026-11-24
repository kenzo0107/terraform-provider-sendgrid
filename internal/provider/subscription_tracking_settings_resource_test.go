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

func testAccRestoreSubscriptionTrackingSettings(t *testing.T) {
	t.Helper()

	ctx := context.Background()
	c := testAccTrackingSettingsClient(t)

	o, err := c.GetSubscriptionTrackingSettings(ctx)
	if err != nil {
		t.Fatalf("reading subscription tracking settings before the test: %s", err)
	}

	t.Cleanup(func() {
		if _, err := c.UpdateSubscriptionTrackingSettings(ctx, &sendgrid.InputUpdateSubscriptionTrackingSettings{
			Enabled:      o.Enabled,
			HTMLContent:  sendgrid.String(o.HTMLContent),
			PlainContent: sendgrid.String(o.PlainContent),
			Landing:      sendgrid.String(o.Landing),
			Replace:      sendgrid.String(o.Replace),
			URL:          sendgrid.String(o.URL),
		}); err != nil {
			t.Errorf("restoring subscription tracking settings: %s", err)
		}
	})
}

func TestAccSubscriptionTrackingSettingsResource(t *testing.T) {
	resourceName := "sendgrid_subscription_tracking_settings.test"

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
  enabled       = true
  html_content  = "<p><% Unsubscribe %></p>"
  plain_content = "Unsubscribe: <% %>"
  landing       = "<p>You have been unsubscribed.</p>"
  replace       = "[tf-acc-unsubscribe]"
  url           = "https://example.com/unsubscribe"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "html_content", "<p><% Unsubscribe %></p>"),
					resource.TestCheckResourceAttr(resourceName, "plain_content", "Unsubscribe: <% %>"),
					resource.TestCheckResourceAttr(resourceName, "landing", "<p>You have been unsubscribed.</p>"),
					resource.TestCheckResourceAttr(resourceName, "replace", "[tf-acc-unsubscribe]"),
					resource.TestCheckResourceAttr(resourceName, "url", "https://example.com/unsubscribe"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: func(*terraform.State) (string, error) { return "", nil },
			},
			{
				Config: `
resource "sendgrid_subscription_tracking_settings" "test" {
  enabled      = false
  html_content = "<p><% Unsubscribe %></p>"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "enabled", "false"),
					resource.TestCheckResourceAttr(resourceName, "html_content", "<p><% Unsubscribe %></p>"),
					resource.TestCheckResourceAttr(resourceName, "plain_content", ""),
					resource.TestCheckResourceAttr(resourceName, "landing", ""),
					resource.TestCheckResourceAttr(resourceName, "replace", ""),
					resource.TestCheckResourceAttr(resourceName, "url", ""),
				),
			},
		},
	})
}
