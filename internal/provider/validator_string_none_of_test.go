// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestValidatorStringNoneOf(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		value       types.String
		wantError   bool
		wantInError string
	}{
		"rejected value reports the reason": {
			value:       types.StringValue("sender_verification_legacy"),
			wantError:   true,
			wantInError: "not valid inside subuser_access",
		},
		"rejected value names itself": {
			value:       types.StringValue("sender_verification_legacy"),
			wantError:   true,
			wantInError: "sender_verification_legacy",
		},
		"allowed value passes": {
			value:     types.StringValue("marketing_campaigns.create"),
			wantError: false,
		},
		"null is left to the schema": {
			value:     types.StringNull(),
			wantError: false,
		},
		"unknown is deferred to apply": {
			value:     types.StringUnknown(),
			wantError: false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			req := validator.StringRequest{
				Path:        path.Root("subuser_access").AtListIndex(0).AtName("scopes"),
				ConfigValue: test.value,
			}
			resp := &validator.StringResponse{}

			stringNoneOf(subuserAccessInvalidScopeReason, "sender_verification_legacy").
				ValidateString(context.Background(), req, resp)

			if got := resp.Diagnostics.HasError(); got != test.wantError {
				t.Fatalf("got error = %v, want %v (%v)", got, test.wantError, resp.Diagnostics)
			}
			if !test.wantError {
				return
			}
			if detail := resp.Diagnostics.Errors()[0].Detail(); !strings.Contains(detail, test.wantInError) {
				t.Errorf("error detail %q does not contain %q", detail, test.wantInError)
			}
		})
	}
}
