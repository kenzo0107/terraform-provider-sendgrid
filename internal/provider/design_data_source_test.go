// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDesignDataSource(t *testing.T) {
	resourceName := "data.sendgrid_design.test"

	name := fmt.Sprintf("test-acc-%s", acctest.RandString(16))
	subject := fmt.Sprintf("test-acc-subject-%s", acctest.RandString(16))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testAccDesignDataSourceConfig(name, subject),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "subject", subject),
					resource.TestCheckResourceAttr(resourceName, "editor", "code"),
					resource.TestCheckResourceAttr(resourceName, "generate_plain_content", "true"),
					resource.TestCheckResourceAttrSet(resourceName, "created_at"),
					resource.TestCheckResourceAttrSet(resourceName, "updated_at"),
				),
			},
		},
	})
}

func testAccDesignDataSourceConfig(name, subject string) string {
	return fmt.Sprintf(`
resource "sendgrid_design" "test" {
  name                   = "%s"
  subject                = "%s"
  editor                 = "code"
  html_content           = "<html><body><p>Hello</p></body></html>"
  generate_plain_content = true
}

data "sendgrid_design" "test" {
  id = sendgrid_design.test.id
}
`, name, subject)
}
