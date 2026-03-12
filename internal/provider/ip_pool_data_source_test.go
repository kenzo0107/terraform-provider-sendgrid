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
