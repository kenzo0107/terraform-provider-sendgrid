// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccSubuserWhitelabelDomainResource(t *testing.T) {
	resourceName := "sendgrid_subuser_whitelabel_domain.test"

	ipAddressAllowed := os.Getenv("IP_ADDRESS")
	ips := []string{ipAddressAllowed}

	domain := fmt.Sprintf("test-acc-%s.com", acctest.RandString(16))
	username := fmt.Sprintf("test-acc-%s", acctest.RandString(16))
	email := fmt.Sprintf("test-acc-%s@example.com", acctest.RandString(16))
	password := fmt.Sprintf("test-acc-%s", acctest.RandString(16))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccSubuserWhitelabelDomainResourceConfig(domain, username, email, password, escapesStrings(ips)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "domain_id"),
					resource.TestCheckResourceAttr(resourceName, "subuser", username),
				),
			},
			// ImportState testing with subuser only
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateId:     username,
			},
			// ImportState testing with domain_id:subuser format
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs := s.RootModule().Resources[resourceName]
					domainID := rs.Primary.Attributes["domain_id"]
					return fmt.Sprintf("%s:%s", domainID, username), nil
				},
			},
		},
	})
}

func testAccSubuserWhitelabelDomainResourceConfig(domain, username, email, password string, ips []string) string {
	return fmt.Sprintf(`
resource "sendgrid_sender_authentication" "test" {
  domain = "%[1]s"
}

resource "sendgrid_subuser" "test" {
	username = "%[2]s"
	email    = "%[3]s"
	password = "%[4]s"
	ips      = %[5]s
}

resource "sendgrid_subuser_whitelabel_domain" "test" {
	domain_id = sendgrid_sender_authentication.test.id
	subuser   = sendgrid_subuser.test.username
}
`, domain, username, email, password, ips)
}
