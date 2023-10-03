// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccSenderAuthenticationDataSource(t *testing.T) {
	resourceName := "data.sendgrid_sender_authentication.test"

	domain := fmt.Sprintf("test-acc-%s.com", acctest.RandString(16))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testAccSenderAuthenticationDataSourceConfig(domain),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
		},
	})
}

func testAccSenderAuthenticationDataSourceConfig(domain string) string {
	return fmt.Sprintf(`
resource "sendgrid_sender_authentication" "test" {
	domain = "%[1]s"
}

data "sendgrid_sender_authentication" "test" {
	id = sendgrid_sender_authentication.test.id
}
`, domain)
}
