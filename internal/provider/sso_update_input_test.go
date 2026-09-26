// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kenzo0107/sendgrid"
)

func TestNewUpdateSSOIntegrationInput(t *testing.T) {
	t.Parallel()

	state := ssoIntegrationResourceModel{
		ID:                   types.StringValue("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"),
		Name:                 types.StringValue("Okta"),
		Enabled:              types.BoolValue(true),
		SigninURL:            types.StringValue("https://idp.example.com/sso/saml"),
		SignoutURL:           types.StringValue(""),
		EntityID:             types.StringValue("http://www.okta.com/aaaaaaaaaaaaaaaaaaaa"),
		CompletedIntegration: types.BoolValue(true),
	}

	tests := map[string]struct {
		plan ssoIntegrationResourceModel
		want *sendgrid.InputUpdateSSOIntegration
	}{
		// Regression for #211: adding signout_url must not flip the
		// untouched enabled / completed_integration flags to false.
		"unchanged bools are preserved when only signout_url changes": {
			plan: func() ssoIntegrationResourceModel {
				p := state
				p.SignoutURL = types.StringValue("https://idp.example.com/slo/saml")
				return p
			}(),
			want: &sendgrid.InputUpdateSSOIntegration{
				Name:                 "Okta",
				Enabled:              true,
				SigninURL:            "https://idp.example.com/sso/saml",
				SignoutURL:           "https://idp.example.com/slo/saml",
				EntityID:             "http://www.okta.com/aaaaaaaaaaaaaaaaaaaa",
				CompletedIntegration: true,
			},
		},
		"changed values are taken from the plan": {
			plan: func() ssoIntegrationResourceModel {
				p := state
				p.Name = types.StringValue("Okta (renamed)")
				p.Enabled = types.BoolValue(false)
				return p
			}(),
			want: &sendgrid.InputUpdateSSOIntegration{
				Name:                 "Okta (renamed)",
				Enabled:              false,
				SigninURL:            "https://idp.example.com/sso/saml",
				SignoutURL:           "",
				EntityID:             "http://www.okta.com/aaaaaaaaaaaaaaaaaaaa",
				CompletedIntegration: true,
			},
		},
		"unknown completed_integration falls back to state": {
			plan: func() ssoIntegrationResourceModel {
				p := state
				p.CompletedIntegration = types.BoolUnknown()
				return p
			}(),
			want: &sendgrid.InputUpdateSSOIntegration{
				Name:                 "Okta",
				Enabled:              true,
				SigninURL:            "https://idp.example.com/sso/saml",
				SignoutURL:           "",
				EntityID:             "http://www.okta.com/aaaaaaaaaaaaaaaaaaaa",
				CompletedIntegration: true,
			},
		},
		"null completed_integration falls back to state": {
			plan: func() ssoIntegrationResourceModel {
				p := state
				p.CompletedIntegration = types.BoolNull()
				return p
			}(),
			want: &sendgrid.InputUpdateSSOIntegration{
				Name:                 "Okta",
				Enabled:              true,
				SigninURL:            "https://idp.example.com/sso/saml",
				SignoutURL:           "",
				EntityID:             "http://www.okta.com/aaaaaaaaaaaaaaaaaaaa",
				CompletedIntegration: true,
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := newUpdateSSOIntegrationInput(tt.plan, state)
			if *got != *tt.want {
				t.Errorf("newUpdateSSOIntegrationInput() = %+v, want %+v", *got, *tt.want)
			}
		})
	}
}

func TestNewUpdateSSOCertificateInput(t *testing.T) {
	t.Parallel()

	plan := ssoCertificateResourceModel{
		ID:                types.StringValue("9999"),
		PublicCertificate: types.StringValue("-----BEGIN CERTIFICATE-----\nMIIB...\n-----END CERTIFICATE-----"),
		IntegrationID:     types.StringValue("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"),
	}

	got := newUpdateSSOCertificateInput(plan)
	want := &sendgrid.InputUpdateSSOCertificate{
		PublicCertificate: "-----BEGIN CERTIFICATE-----\nMIIB...\n-----END CERTIFICATE-----",
		IntegrationID:     "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
		Enabled:           true,
	}
	if *got != *want {
		t.Errorf("newUpdateSSOCertificateInput() = %+v, want %+v", *got, *want)
	}
}
