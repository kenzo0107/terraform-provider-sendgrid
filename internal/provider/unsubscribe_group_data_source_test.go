// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccUnsubscribeGroupDataSource(t *testing.T) {
	resourceName := "data.sendgrid_unsubscribe_group.test"

	name := fmt.Sprintf("test-acc-%s", acctest.RandString(16))
	description := fmt.Sprintf("test-acc-%s", acctest.RandString(16))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testAccUnsubscribeGroupDataSourceConfig(name, description, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "description", description),
					resource.TestCheckResourceAttr(resourceName, "is_default", "false"),
				),
			},
		},
	})
}

func testAccUnsubscribeGroupDataSourceConfig(name, description string, is_default bool) string {
	return fmt.Sprintf(`
resource "sendgrid_unsubscribe_group" "test" {
	name = "%s"
	description = "%s"
	is_default = %t
}

data "sendgrid_unsubscribe_group" "test" {
	id = sendgrid_unsubscribe_group.test.id
}
`, name, description, is_default)
}
