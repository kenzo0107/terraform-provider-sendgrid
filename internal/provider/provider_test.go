// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// testAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"sendgrid": providerserver.NewProtocol6WithError(New("test")()),
}

func testAccPreCheck(t *testing.T) {
	t.Helper()

	if err := os.Getenv("SENDGRID_API_KEY"); err == "" {
		t.Fatal("SENDGRID_API_KEY must be set for acceptance tests")
	}
	if err := os.Getenv("IP_ADDRESS"); err == "" {
		t.Fatal("IP_ADDRESS must be set for acceptance tests")
	}
}
