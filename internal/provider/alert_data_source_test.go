// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccAlertDataSource(t *testing.T) {
	resourceName := "data.sendgrid_alert.test"

	emailTo := fmt.Sprintf("test-acc-%s@example.com", acctest.RandString(16))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testAccAlertDataSourceConfig(emailTo),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "email_to", emailTo),
					resource.TestCheckResourceAttr(resourceName, "type", "stats_notification"),
					resource.TestCheckResourceAttr(resourceName, "frequency", "daily"),
				),
			},
		},
	})
}

func testAccAlertDataSourceConfig(emailTo string) string {
	return fmt.Sprintf(`
resource "sendgrid_alert" "test" {
	email_to    = "%s"
	type        = "stats_notification"
	frequency   = "daily"
}

data "sendgrid_alert" "test" {
	id = sendgrid_alert.test.id
}
`, emailTo)
}
