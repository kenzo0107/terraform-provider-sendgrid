// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccAlertResource(t *testing.T) {
	resourceName := "sendgrid_alert.test"

	emailTo := fmt.Sprintf("test-acc-%s@example.com", acctest.RandString(16))
	percentage := int64(90)

	emailToUpdated := fmt.Sprintf("test-acc-%s", acctest.RandString(16))
	percentageUpdated := int64(80)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccAlertResourceConfig(emailTo, percentage),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "email_to", emailTo),
					resource.TestCheckResourceAttr(resourceName, "percentage", strconv.FormatInt(percentage, 10)),
				),
			},
			// ImportState testing
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: testAccAlertResourceConfig(emailToUpdated, percentageUpdated),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "email_to", emailToUpdated),
					resource.TestCheckResourceAttr(resourceName, "percentage", strconv.FormatInt(percentageUpdated, 10)),
				),
			},
		},
	})
}

func testAccAlertResourceConfig(email_to string, percentage int64) string {
	return fmt.Sprintf(`
resource "sendgrid_alert" "test" {
	type       = "usage_limit"
	email_to   = "%[1]s"
	percentage = %[2]d
}
`, email_to, percentage)
}
