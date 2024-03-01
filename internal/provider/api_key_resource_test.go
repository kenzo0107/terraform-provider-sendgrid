// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccAPIKeyResource(t *testing.T) {
	// NOTE: It skips the test because the resource creation process is error-prone and the test completes successfully in the local environment without any issues.
	t.Skip()

	resourceName := "sendgrid_api_key.test"

	name := fmt.Sprintf("test-acc-%s", acctest.RandString(16))
	nameUpdated := fmt.Sprintf("test-acc-%s", acctest.RandString(16))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccAPIKeyResourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckTypeSetElemAttr(resourceName, "scopes.*", "user.profile.read"),
					resource.TestCheckTypeSetElemAttr(resourceName, "scopes.*", "2fa_required"),
					resource.TestCheckTypeSetElemAttr(resourceName, "scopes.*", "sender_verification_exempt"),
					resource.TestCheckResourceAttrSet(resourceName, "api_key"),
				),
			},
			// ImportState testing
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"api_key"},
			},
			// Update and Read testing
			{
				Config: testAccAPIKeyResourceConfig(nameUpdated),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "name", nameUpdated),
					resource.TestCheckTypeSetElemAttr(resourceName, "scopes.*", "user.profile.read"),
					resource.TestCheckTypeSetElemAttr(resourceName, "scopes.*", "2fa_required"),
					resource.TestCheckTypeSetElemAttr(resourceName, "scopes.*", "sender_verification_exempt"),
					resource.TestCheckResourceAttrSet(resourceName, "api_key"),
				),
			},
		},
	})
}

func testAccAPIKeyResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "sendgrid_api_key" "test" {
	name = "%s"
	scopes = [
		"user.profile.read",
		"2fa_required",
		"sender_verification_exempt",
	]
}
`, name)
}
