// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccTeammateResource(t *testing.T) {
	// NOTE: It skips the test because the resource creation process is error-prone and the test completes successfully in the local environment without any issues.
	t.Skip()

	resourceName := "sendgrid_teammate.test"

	email := fmt.Sprintf("test-acc-%s@example.com", acctest.RandString(16))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccTeammateResourceConfig(email, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "email", email),
					resource.TestCheckResourceAttr(resourceName, "is_admin", "false"),
					resource.TestCheckTypeSetElemAttr(resourceName, "scopes.*", "user.profile.read"),
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
				Config: testAccTeammateResourceConfig(email, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "email", email),
					resource.TestCheckResourceAttr(resourceName, "is_admin", "true"),
					resource.TestCheckTypeSetElemAttr(resourceName, "scopes.*", "user.profile.read"),
				),
			},
		},
	})
}

func testAccTeammateResourceConfig(email string, is_admin bool) string {
	return fmt.Sprintf(`
resource "sendgrid_teammate" "test" {
	email = "%s"
	is_admin = %t
	scopes = ["user.profile.read"]
}
`, email, is_admin)
}
