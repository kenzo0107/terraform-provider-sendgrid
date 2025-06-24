// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccTeammateResource(t *testing.T) {
	resourceName := "sendgrid_teammate.test"

	email := fmt.Sprintf("test-acc-%s@example.com", acctest.RandString(16))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccTeammateResourceConfig(email, []string{"user.profile.read"}),
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
		},
	})
}

func testAccTeammateResourceConfig(email string, scopes []string) string {
	for i, s := range scopes {
		scopes[i] = `"` + s + `"`
	}
	return fmt.Sprintf(`
resource "sendgrid_teammate" "test" {
	email = "%s"
	scopes = [%s]
}
`, email, strings.Join(scopes, ", "))
}
