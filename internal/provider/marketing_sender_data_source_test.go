// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccMarketingSenderDataSource(t *testing.T) {
	resourceName := "data.sendgrid_marketing_sender.test"

	nickname := fmt.Sprintf("test-acc-%s", acctest.RandString(16))
	fromEmail := fmt.Sprintf("test-acc-%s@example.com", acctest.RandString(16))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccMarketingSenderDataSourceConfig(nickname, fromEmail),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(resourceName, "id", "sendgrid_marketing_sender.test", "id"),
					resource.TestCheckResourceAttr(resourceName, "nickname", nickname),
					resource.TestCheckResourceAttr(resourceName, "from_email", fromEmail),
					resource.TestCheckResourceAttr(resourceName, "reply_to", fromEmail),
				),
			},
		},
	})
}

func testAccMarketingSenderDataSourceConfig(nickname, fromEmail string) string {
	return fmt.Sprintf(`
resource "sendgrid_marketing_sender" "test" {
  nickname   = "%[1]s"
  from_email = "%[2]s"
  from_name  = "test-acc"
  reply_to   = "%[2]s"
  address    = "1-1-1 Chiyoda"
  city       = "Tokyo"
  country    = "Japan"
}

data "sendgrid_marketing_sender" "test" {
  id = sendgrid_marketing_sender.test.id
}
`, nickname, fromEmail)
}
