// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccTemplateVersionDataSource(t *testing.T) {
	resourceName := "data.sendgrid_template_version.test"

	name := fmt.Sprintf("test-acc-%s", acctest.RandString(16))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testAccTemplateVersionDataSourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "active", "1"),
				),
			},
		},
	})
}

func testAccTemplateVersionDataSourceConfig(name string) string {
	return fmt.Sprintf(`
resource "sendgrid_template" "test" {
	name       = "%[1]s"
  	generation = "dynamic"
}

resource "sendgrid_template_version" "test" {
	template_id = sendgrid_template.test.id
	name        = "%[1]s"
	active      = 1
}

data "sendgrid_template_version" "test" {
	template_id = sendgrid_template_version.test.template_id
	id          = sendgrid_template_version.test.id
}
`, name)
}
