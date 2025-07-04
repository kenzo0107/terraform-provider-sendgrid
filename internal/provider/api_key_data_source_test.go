// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccAPIKeyDataSource(t *testing.T) {
	resourceName := "data.sendgrid_api_key.test"

	name := acctest.RandString(16)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testAccAPIKeyDataSourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckTypeSetElemAttr(resourceName, "scopes.*", "user.profile.read"),
				),
			},
		},
	})
}

func testAccAPIKeyDataSourceConfig(name string) string {
	return fmt.Sprintf(`
resource "sendgrid_api_key" "test" {
	name = "%s"
	scopes = [
		"user.profile.read",
	]
}

data "sendgrid_api_key" "test" {
	id = sendgrid_api_key.test.id
}
`, name)
}
