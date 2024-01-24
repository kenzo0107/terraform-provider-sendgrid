// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccTemplateDataSource(t *testing.T) {
	resourceName := "data.sendgrid_template.test"

	name := fmt.Sprintf("test-acc-%s", acctest.RandString(16))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testAccTemplateDataSourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "generation", "dynamic"),
				),
			},
		},
	})
}

func testAccTemplateDataSourceConfig(name string) string {
	return fmt.Sprintf(`
resource "sendgrid_template" "test" {
	name       = "%s"
  	generation = "dynamic"
}

data "sendgrid_template" "test" {
	id = sendgrid_template.test.id
}
`, name)
}
