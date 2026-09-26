// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccIPPoolDataSource(t *testing.T) {
	resourceName := "data.sendgrid_ip_pool.test"

	name := fmt.Sprintf("test-acc-%s", acctest.RandString(16))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create resource first. Reading the pool by name in the same apply
			// as its creation intermittently fails with "Unable to locate
			// specified IPs Pool" because the SendGrid API is eventually
			// consistent, so the data source lookup runs in a separate step.
			{
				Config: testAccIPPoolResourceConfig(name),
			},
			// Read testing
			{
				Config: testAccIPPoolDataSourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
				),
			},
		},
	})
}

func testAccIPPoolDataSourceConfig(name string) string {
	return fmt.Sprintf(`
resource "sendgrid_ip_pool" "test" {
	name = "%s"
	ips  = []
}

data "sendgrid_ip_pool" "test" {
	name = sendgrid_ip_pool.test.name
}
`, name)
}
