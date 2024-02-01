// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccTemplateVersionVersionResource(t *testing.T) {
	resourceName := "sendgrid_template_version.test"

	name := fmt.Sprintf("test-acc-%s", acctest.RandString(16))
	subject := fmt.Sprintf("test-acc-%s", acctest.RandString(16))
	html_content := fmt.Sprintf("test-acc-%s", acctest.RandString(16))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccTemplateVersionResourceConfig(name, subject, html_content),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "subject", subject),
					resource.TestCheckResourceAttr(resourceName, "active", "1"),
					resource.TestCheckResourceAttr(resourceName, "html_content", html_content),
					// NOTE: If plain_content is empty and html_content is specified, plain_content has the same value as html_content.
					resource.TestCheckResourceAttr(resourceName, "plain_content", html_content),
					resource.TestCheckResourceAttr(resourceName, "generate_plain_content", "true"),
				),
			},
			// ImportState testing
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: importStateIdFunc(resourceName),
			},
			// Update and Read testing
			{
				Config: testAccTemplateVersionResourceConfig(name, subject, html_content),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "subject", subject),
					resource.TestCheckResourceAttr(resourceName, "active", "1"),
					resource.TestCheckResourceAttr(resourceName, "html_content", html_content),
					// NOTE: If plain_content is empty and html_content is specified, plain_content has the same value as html_content.
					resource.TestCheckResourceAttr(resourceName, "plain_content", html_content),
					resource.TestCheckResourceAttr(resourceName, "generate_plain_content", "true"),
				),
			},
		},
	})
}

func testAccTemplateVersionResourceConfig(name, subject, html_content string) string {
	return fmt.Sprintf(`
resource "sendgrid_template" "test" {
	name       = "%[1]s"
	generation = "dynamic"
}

resource "sendgrid_template_version" "test" {
	template_id = sendgrid_template.test.id
	name        = "%[1]s"
	subject     = "%[2]s"
	test_data   = jsonencode({
		"name": "dummy",
	})
	html_content  = "%[3]s"
	active      = 1
}
`, name, subject, html_content)
}

func importStateIdFunc(resourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("Not found: %s", resourceName)
		}
		return fmt.Sprintf("%s/%s", rs.Primary.Attributes["template_id"], rs.Primary.Attributes["id"]), nil
	}
}
