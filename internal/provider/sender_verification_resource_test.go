// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccSenderVerificationResource(t *testing.T) {
	resourceName := "sendgrid_sender_verification.test"

	reply_to := fmt.Sprintf("test-acc-%s@example.com", acctest.RandString(16))
	from_name := fmt.Sprintf("test-acc-%s", acctest.RandString(16))
	from_email := fmt.Sprintf("test-acc-%s@example.com", acctest.RandString(16))
	address := fmt.Sprintf("test-acc-%s", acctest.RandString(16))
	address2 := fmt.Sprintf("test-acc-%s", acctest.RandString(16))
	city := fmt.Sprintf("test-acc-%s", acctest.RandString(16))
	country := fmt.Sprintf("test-acc-%s", acctest.RandString(16))
	nickname := fmt.Sprintf("test-acc-%s", acctest.RandString(16))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccSenderVerificationResourceConfig(nickname, reply_to, from_name, from_email, address, address2, city, country),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "reply_to", reply_to),
					resource.TestCheckResourceAttr(resourceName, "from_name", from_name),
					resource.TestCheckResourceAttr(resourceName, "from_email", from_email),
					resource.TestCheckResourceAttr(resourceName, "address", address),
					resource.TestCheckResourceAttr(resourceName, "address2", address2),
					resource.TestCheckResourceAttr(resourceName, "city", city),
					resource.TestCheckResourceAttr(resourceName, "country", country),
					resource.TestCheckResourceAttr(resourceName, "nickname", nickname),
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
				Config: testAccSenderVerificationResourceConfig(nickname, reply_to, from_name, from_email, address, address2, city, "JPN"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "nickname", nickname),
					resource.TestCheckResourceAttr(resourceName, "reply_to", reply_to),
					resource.TestCheckResourceAttr(resourceName, "from_name", from_name),
					resource.TestCheckResourceAttr(resourceName, "from_email", from_email),
					resource.TestCheckResourceAttr(resourceName, "address", address),
					resource.TestCheckResourceAttr(resourceName, "address2", address2),
					resource.TestCheckResourceAttr(resourceName, "city", city),
					resource.TestCheckResourceAttr(resourceName, "country", "JPN"),
				),
			},
		},
	})
}

func testAccSenderVerificationResourceConfig(nickname, reply_to, from_name, from_email, address, address2, city, country string) string {
	return fmt.Sprintf(`
resource "sendgrid_sender_verification" "test" {
  nickname   = "%[1]s"
  reply_to   = "%[2]s"
  from_name  = "%[3]s"
  from_email = "%[4]s"
  address    = "%[5]s"
  address2   = "%[6]s"
  city       = "%[7]s"
  country    = "%[8]s"
}
`, nickname, reply_to, from_name, from_email, address, address2, city, country)
}
