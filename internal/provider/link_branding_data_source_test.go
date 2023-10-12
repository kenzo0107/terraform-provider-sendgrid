// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccLinkBrandingDataSource(t *testing.T) {
	resourceName := "data.sendgrid_link_branding.test"

	domain := fmt.Sprintf("test-acc-%s.com", acctest.RandString(16))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testLinkBrandingDataSourceConfig(domain),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
		},
	})
}

func testLinkBrandingDataSourceConfig(domain string) string {
	return fmt.Sprintf(`
resource "sendgrid_link_branding" "test" {
	domain = "%[1]s"
}

data "sendgrid_link_branding" "test" {
	id = sendgrid_link_branding.test.id
}
`, domain)
}
