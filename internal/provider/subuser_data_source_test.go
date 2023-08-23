// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccSubuserDataSource(t *testing.T) {
	resourceName := "data.sendgrid_subuser.test"

	ipAddressAllowed := os.Getenv("IP_ADDRESS")
	ips := []string{ipAddressAllowed}

	username := fmt.Sprintf("test-acc-%s", acctest.RandString(16))
	email := fmt.Sprintf("test-acc-%s@example.com", acctest.RandString(16))
	password := fmt.Sprintf("test-acc-%s", acctest.RandString(16))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testAccSubuserDataSourceConfig(username, email, password, escapesStrings(ips)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "username", username),
					resource.TestCheckResourceAttr(resourceName, "email", email),
				),
			},
		},
	})
}

func testAccSubuserDataSourceConfig(username, email, password string, ips []string) string {
	return fmt.Sprintf(`
resource "sendgrid_subuser" "test" {
	username = "%[1]s"
	email    = "%[2]s"
	password = "%[3]s"
	ips      = %[4]s
}

data "sendgrid_subuser" "test" {
	username = sendgrid_subuser.test.username
}
`, username, email, password, ips)
}
