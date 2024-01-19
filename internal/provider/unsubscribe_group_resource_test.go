// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccUnsubscribeGroupResource(t *testing.T) {
	resourceName := "sendgrid_unsubscribe_group.test"

	name := fmt.Sprintf("test-acc-%s", acctest.RandString(16))
	description := fmt.Sprintf("test-acc-%s", acctest.RandString(16))

	nameUpdated := fmt.Sprintf("test-acc-%s", acctest.RandString(16))
	descriptionUpdated := fmt.Sprintf("test-acc-%s", acctest.RandString(16))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccUnsubscribeGroupResourceConfig(name, description, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "description", description),
					resource.TestCheckResourceAttr(resourceName, "is_default", "false"),
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
				Config: testAccUnsubscribeGroupResourceConfig(nameUpdated, descriptionUpdated, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "name", nameUpdated),
					resource.TestCheckResourceAttr(resourceName, "description", descriptionUpdated),
					resource.TestCheckResourceAttr(resourceName, "is_default", "false"),
				),
			},
		},
	})
}

func testAccUnsubscribeGroupResourceConfig(name, description string, is_default bool) string {
	return fmt.Sprintf(`
resource "sendgrid_unsubscribe_group" "test" {
	name = "%s"
	description = "%s"
	is_default = %t
}
`, name, description, is_default)
}
