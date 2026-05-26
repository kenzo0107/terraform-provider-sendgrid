// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccSubuserResource(t *testing.T) {
	resourceName := "sendgrid_subuser.test"

	ipAddressAllowed := os.Getenv("IP_ADDRESS")
	ips := []string{ipAddressAllowed}

	username := fmt.Sprintf("test-acc-%s", acctest.RandString(16))
	email := fmt.Sprintf("test-acc-%s@example.com", acctest.RandString(16))
	password := fmt.Sprintf("test-acc-%s", acctest.RandString(16))

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
			// Re-apply after import: state values for password and ips are null
			// because they cannot be retrieved from the SendGrid API. The next plan
			// must absorb the config values via an in-place update (NOT a replace),
			// which validates the fix for #196.
			{
				Config: testAccSubuserResourceConfig(username, email, password, escapesStrings(ips)),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "username", username),
					resource.TestCheckResourceAttr(resourceName, "email", email),
					resource.TestCheckResourceAttr(resourceName, "password", password),
					resource.TestCheckTypeSetElemAttr(resourceName, "ips.*", ips[0]),
				),
			},
			// Verify the plan is stable after the post-import sync.
			{
				Config: testAccSubuserResourceConfig(username, email, password, escapesStrings(ips)),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionNoop),
					},
				},
			},
		},
	})
}

// TestAccSubuserResource_PasswordWO covers the user-reported scenario in #196,
// where the resource is managed via password_wo / password_wo_version. Importing
// then applying the same config must not force a replace.
func TestAccSubuserResource_PasswordWO(t *testing.T) {
	resourceName := "sendgrid_subuser.test"

	ipAddressAllowed := os.Getenv("IP_ADDRESS")
	ips := []string{ipAddressAllowed}

	username := fmt.Sprintf("test-acc-%s", acctest.RandString(16))
	email := fmt.Sprintf("test-acc-%s@example.com", acctest.RandString(16))
	password := fmt.Sprintf("test-acc-%s", acctest.RandString(16))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSubuserResourceConfigPasswordWO(username, email, password, 1, escapesStrings(ips)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "username", username),
					resource.TestCheckResourceAttr(resourceName, "email", email),
					resource.TestCheckResourceAttr(resourceName, "password_wo_version", "1"),
					resource.TestCheckTypeSetElemAttr(resourceName, "ips.*", ips[0]),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"ips", "password_wo", "password_wo_version"},
				ImportStateId:           username,
			},
			{
				Config: testAccSubuserResourceConfigPasswordWO(username, email, password, 1, escapesStrings(ips)),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
					},
				},
			},
			{
				Config: testAccSubuserResourceConfigPasswordWO(username, email, password, 1, escapesStrings(ips)),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionNoop),
					},
				},
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

func testAccSubuserResourceConfigPasswordWO(username, email, password string, passwordWOVersion int, ips []string) string {
	return fmt.Sprintf(`
resource "sendgrid_subuser" "test" {
	username            = "%[1]s"
	email               = "%[2]s"
	password_wo         = "%[3]s"
	password_wo_version = %[4]d
	ips                 = %[5]s
}
`, username, email, password, passwordWOVersion, ips)
}

func escapesStrings(x []string) (y []string) {
	for _, v := range x {
		y = append(y, fmt.Sprintf("\"%s\"", v))
	}
	return
}
