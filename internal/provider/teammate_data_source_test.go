// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccTeammateDataSource(t *testing.T) {
	resourceName := "data.sendgrid_teammate.test"

	email := fmt.Sprintf("test-acc-%s@example.com", acctest.RandString(16))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testAccTeammateDataSourceConfig(email),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "email", email),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "is_admin", "false"),
					resource.TestCheckTypeSetElemAttr(resourceName, "scopes.*", "user.profile.read"),
				),
			},
		},
	})
}

func testAccTeammateDataSourceConfig(email string) string {
	return fmt.Sprintf(`
resource "sendgrid_teammate" "test" {
	email = "%[1]s"
}

data "sendgrid_teammate" "test" {
	email = sendgrid_teammate.test.email
}
`, email)
}
