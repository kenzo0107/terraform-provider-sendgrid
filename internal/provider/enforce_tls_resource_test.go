// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccEnforceTLSResource(t *testing.T) {
	resourceName := "sendgrid_enforce_tls.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccEnforceTLSResourceConfig(1.2, false, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "version", "1.2"),
					resource.TestCheckResourceAttr(resourceName, "require_tls", "false"),
					resource.TestCheckResourceAttr(resourceName, "require_valid_cert", "false"),
				),
			},
			// ImportState testing
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: importEnforceTLSStateIdFunc(),
			},
		},
	})
}

func testAccEnforceTLSResourceConfig(version float64, require_tls, require_valid_cert bool) string {
	return fmt.Sprintf(`
resource "sendgrid_enforce_tls" "test" {
  version = %f
  require_tls = %t
  require_valid_cert = %t
}
`, version, require_tls, require_valid_cert)
}

func importEnforceTLSStateIdFunc() resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		return "", nil
	}
}
