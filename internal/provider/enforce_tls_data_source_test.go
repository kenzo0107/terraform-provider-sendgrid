// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccEnforceTLSDataSource(t *testing.T) {
	resourceName := "data.sendgrid_enforce_tls.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testAccEnforceTLSDataSourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "require_tls", "false"),
					resource.TestCheckResourceAttr(resourceName, "require_valid_cert", "false"),
				),
			},
		},
	})
}

func testAccEnforceTLSDataSourceConfig() string {
	return `
data "sendgrid_enforce_tls" "test" {
}
`
}
