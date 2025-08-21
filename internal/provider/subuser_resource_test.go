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

func TestAccSubuserResource(t *testing.T) {
	resourceName := "sendgrid_subuser.test"

	ipAddressAllowed := os.Getenv("IP_ADDRESS")
	ips := []string{ipAddressAllowed}

	username := fmt.Sprintf("test-acc-%s", acctest.RandString(16))
	email := fmt.Sprintf("test-acc-%s@example.com", acctest.RandString(16))
	password := fmt.Sprintf("test-acc-12345-%s", acctest.RandString(16))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccSubuserResourceConfig(username, email, password, escapesStrings(ips)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "username", username),
					resource.TestCheckResourceAttr(resourceName, "email", email),
					resource.TestCheckResourceAttr(resourceName, "password", password),
					resource.TestCheckTypeSetElemAttr(resourceName, "ips.*", ips[0]),
				),
			},
			// ImportState testing
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"ips", "password"},
				ImportStateId:           username,
			},
			// Update and Read testing
			{
				Config: testAccSubuserResourceConfig(username, email, password, escapesStrings(ips)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "username", username),
					resource.TestCheckResourceAttr(resourceName, "email", email),
					resource.TestCheckResourceAttr(resourceName, "password", password),
					resource.TestCheckTypeSetElemAttr(resourceName, "ips.*", ips[0]),
				),
			},
		},
	})
}

func testAccSubuserResourceConfig(username, email, password string, ips []string) string {
	return fmt.Sprintf(`
resource "sendgrid_subuser" "test" {
	username = "%[1]s"
	email    = "%[2]s"
	password = "%[3]s"
	ips      = %[4]s
}
`, username, email, password, ips)
}

func escapesStrings(x []string) (y []string) {
	for _, v := range x {
		y = append(y, fmt.Sprintf("\"%s\"", v))
	}
	return
}
