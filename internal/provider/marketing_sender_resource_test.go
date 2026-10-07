// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccMarketingSenderResource(t *testing.T) {
	resourceName := "sendgrid_marketing_sender.test"

	nickname := fmt.Sprintf("test-acc-%s", acctest.RandString(16))
	fromEmail := fmt.Sprintf("test-acc-%s@example.com", acctest.RandString(16))
	replyTo := fmt.Sprintf("test-acc-%s@example.com", acctest.RandString(16))
	nicknameUpdated := fmt.Sprintf("test-acc-%s", acctest.RandString(16))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccMarketingSenderResourceConfig(nickname, fromEmail, replyTo, "Tokyo"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "nickname", nickname),
					resource.TestCheckResourceAttr(resourceName, "from_email", fromEmail),
					resource.TestCheckResourceAttr(resourceName, "from_name", "test-acc"),
					resource.TestCheckResourceAttr(resourceName, "reply_to", replyTo),
					resource.TestCheckResourceAttr(resourceName, "reply_to_name", "test-acc-reply"),
					resource.TestCheckResourceAttr(resourceName, "address", "1-1-1 Chiyoda"),
					resource.TestCheckResourceAttr(resourceName, "city", "Tokyo"),
					resource.TestCheckResourceAttr(resourceName, "country", "Japan"),
					resource.TestCheckResourceAttr(resourceName, "verified", "false"),
					resource.TestCheckResourceAttr(resourceName, "locked", "false"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccMarketingSenderResourceConfig(nicknameUpdated, fromEmail, replyTo, "Osaka"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "nickname", nicknameUpdated),
					resource.TestCheckResourceAttr(resourceName, "city", "Osaka"),
				),
			},
			{
				Config: testAccMarketingSenderResourceConfigWithOptionalAttrs(nicknameUpdated, fromEmail, replyTo),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "address2", "Building 1"),
					resource.TestCheckResourceAttr(resourceName, "state", "TK"),
					resource.TestCheckResourceAttr(resourceName, "zip", "100-0001"),
				),
			},
			{
				Config: testAccMarketingSenderResourceConfig(nicknameUpdated, fromEmail, replyTo, "Osaka"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "address2", ""),
					resource.TestCheckResourceAttr(resourceName, "state", ""),
					resource.TestCheckResourceAttr(resourceName, "zip", ""),
				),
			},
		},
	})
}

func testAccMarketingSenderResourceConfig(nickname, fromEmail, replyTo, city string) string {
	return fmt.Sprintf(`
resource "sendgrid_marketing_sender" "test" {
  nickname      = "%[1]s"
  from_email    = "%[2]s"
  from_name     = "test-acc"
  reply_to      = "%[3]s"
  reply_to_name = "test-acc-reply"
  address       = "1-1-1 Chiyoda"
  city          = "%[4]s"
  country       = "Japan"
}
`, nickname, fromEmail, replyTo, city)
}

func testAccMarketingSenderResourceConfigWithOptionalAttrs(nickname, fromEmail, replyTo string) string {
	return fmt.Sprintf(`
resource "sendgrid_marketing_sender" "test" {
  nickname      = "%[1]s"
  from_email    = "%[2]s"
  from_name     = "test-acc"
  reply_to      = "%[3]s"
  reply_to_name = "test-acc-reply"
  address       = "1-1-1 Chiyoda"
  address2      = "Building 1"
  state         = "TK"
  zip           = "100-0001"
  city          = "Osaka"
  country       = "Japan"
}
`, nickname, fromEmail, replyTo)
}
