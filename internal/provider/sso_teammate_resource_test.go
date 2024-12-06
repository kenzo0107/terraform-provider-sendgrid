// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccSSOTeammateResource(t *testing.T) {
	resourceName := "sendgrid_sso_teammate.test"

	email := fmt.Sprintf("test-acc-%s@example.com", acctest.RandString(16))
	firstName := fmt.Sprintf("test-acc-%s", acctest.RandString(16))
	lastName := fmt.Sprintf("test-acc-%s", acctest.RandString(16))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccSSOTeammateResourceConfig(email, firstName, lastName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "email", email),
					resource.TestCheckResourceAttr(resourceName, "first_name", firstName),
					resource.TestCheckResourceAttr(resourceName, "last_name", lastName),
					resource.TestCheckResourceAttr(resourceName, "is_admin", "true"),
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
				Config: testAccSSOTeammateResourceConfig(email, "dummy", "dummy"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "email", email),
					resource.TestCheckResourceAttr(resourceName, "first_name", "dummy"),
					resource.TestCheckResourceAttr(resourceName, "last_name", "dummy"),
					resource.TestCheckResourceAttr(resourceName, "is_admin", "true"),
				),
			},
		},
	})
}

func testAccSSOTeammateResourceConfig(email, firstName, lastName string) string {
	return fmt.Sprintf(`
resource "sendgrid_sso_teammate" "test" {
	email      = "%s"
	first_name = "%s"
	last_name  = "%s"
	is_admin   = true
	scopes     = []
}
`, email, firstName, lastName)
}
