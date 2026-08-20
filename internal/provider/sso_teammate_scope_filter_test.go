// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func subuserAccessEntry(id int64, permissionType string, scopes ...string) ssoSubuserAccessResourceModel {
	model := ssoSubuserAccessResourceModel{
		ID:             types.Int64Value(id),
		PermissionType: types.StringValue(permissionType),
	}
	for _, scope := range scopes {
		model.Scopes = append(model.Scopes, types.StringValue(scope))
	}
	return model
}

func TestFilterUndeclaredSubuserAccessScopes(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		state   []ssoSubuserAccessResourceModel
		fetched []ssoSubuserAccessResourceModel
		want    []ssoSubuserAccessResourceModel
	}{
		// The scenario from issue #205: SendGrid added baseline read scopes the
		// configuration never declared. They must not end up in state.
		"baseline scopes added by SendGrid are dropped": {
			state: []ssoSubuserAccessResourceModel{
				subuserAccessEntry(57132954, "restricted", "mail.send", "stats.read"),
			},
			fetched: []ssoSubuserAccessResourceModel{
				subuserAccessEntry(57132954, "restricted",
					"mail.send", "mail_settings.read", "partner_settings.read", "stats.read", "tracking_settings.read"),
			},
			want: []ssoSubuserAccessResourceModel{
				subuserAccessEntry(57132954, "restricted", "mail.send", "stats.read"),
			},
		},
		// A declared baseline scope (stats.read here) is kept: filtering is driven by the
		// state, not by a fixed list, so declared scopes never disappear.
		"declared scope removed outside terraform surfaces as drift": {
			state: []ssoSubuserAccessResourceModel{
				subuserAccessEntry(57132954, "restricted", "mail.send", "templates.read"),
			},
			fetched: []ssoSubuserAccessResourceModel{
				subuserAccessEntry(57132954, "restricted", "mail.send", "mail_settings.read"),
			},
			want: []ssoSubuserAccessResourceModel{
				subuserAccessEntry(57132954, "restricted", "mail.send"),
			},
		},
		"entry unknown to the state is kept as returned": {
			state: []ssoSubuserAccessResourceModel{
				subuserAccessEntry(57132954, "restricted", "mail.send"),
			},
			fetched: []ssoSubuserAccessResourceModel{
				subuserAccessEntry(57132954, "restricted", "mail.send"),
				subuserAccessEntry(99999999, "restricted", "mail.send", "mail_settings.read"),
			},
			want: []ssoSubuserAccessResourceModel{
				subuserAccessEntry(57132954, "restricted", "mail.send"),
				subuserAccessEntry(99999999, "restricted", "mail.send", "mail_settings.read"),
			},
		},
		"empty state keeps the fetched value": {
			state: nil,
			fetched: []ssoSubuserAccessResourceModel{
				subuserAccessEntry(57132954, "restricted", "mail.send", "mail_settings.read"),
			},
			want: []ssoSubuserAccessResourceModel{
				subuserAccessEntry(57132954, "restricted", "mail.send", "mail_settings.read"),
			},
		},
		"nothing fetched stays nil": {
			state: []ssoSubuserAccessResourceModel{
				subuserAccessEntry(57132954, "restricted", "mail.send"),
			},
			fetched: nil,
			want:    nil,
		},
		// An admin entry declares no scopes; whatever the API reports for it is unmanaged.
		"entry declared without scopes keeps none": {
			state: []ssoSubuserAccessResourceModel{
				subuserAccessEntry(57132954, "admin"),
			},
			fetched: []ssoSubuserAccessResourceModel{
				subuserAccessEntry(57132954, "admin", "mail_settings.read"),
			},
			want: []ssoSubuserAccessResourceModel{
				subuserAccessEntry(57132954, "admin"),
			},
		},
		"entries are matched by subuser id, not by position": {
			state: []ssoSubuserAccessResourceModel{
				subuserAccessEntry(2, "restricted", "templates.read"),
				subuserAccessEntry(1, "restricted", "mail.send"),
			},
			fetched: []ssoSubuserAccessResourceModel{
				subuserAccessEntry(1, "restricted", "mail.send", "mail_settings.read"),
				subuserAccessEntry(2, "restricted", "templates.read", "stats.read"),
			},
			want: []ssoSubuserAccessResourceModel{
				subuserAccessEntry(1, "restricted", "mail.send"),
				subuserAccessEntry(2, "restricted", "templates.read"),
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := filterUndeclaredSubuserAccessScopes(test.state, test.fetched)
			if (got == nil) != (test.want == nil) {
				t.Fatalf("got nil = %v, want nil = %v", got == nil, test.want == nil)
			}
			if len(got) != len(test.want) {
				t.Fatalf("got %d entries, want %d", len(got), len(test.want))
			}
			for i := range test.want {
				if got[i].ID != test.want[i].ID {
					t.Errorf("entry %d: got id %v, want %v", i, got[i].ID, test.want[i].ID)
				}
				if got[i].PermissionType != test.want[i].PermissionType {
					t.Errorf("entry %d: got permission_type %v, want %v", i, got[i].PermissionType, test.want[i].PermissionType)
				}
				if len(got[i].Scopes) != len(test.want[i].Scopes) {
					t.Fatalf("entry %d: got %d scopes %v, want %d", i, len(got[i].Scopes), got[i].Scopes, len(test.want[i].Scopes))
				}
				for j := range test.want[i].Scopes {
					if got[i].Scopes[j] != test.want[i].Scopes[j] {
						t.Errorf("entry %d scope %d: got %v, want %v", i, j, got[i].Scopes[j], test.want[i].Scopes[j])
					}
				}
			}
		})
	}
}
