// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"slices"
	"testing"
)

func TestExcludeAutoScopes(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		scopes []string
		want   []string
	}{
		"every auto scope is dropped": {
			scopes: []string{
				"2fa_exempt",
				"2fa_required",
				"sender_verification_exempt",
				"sender_verification_eligible",
			},
			want: []string{},
		},
		"2fa_exempt is dropped alongside user scopes": {
			scopes: []string{"mail.send", "2fa_exempt", "stats.read"},
			want:   []string{"mail.send", "stats.read"},
		},
		"user scopes are kept in order": {
			scopes: []string{"mail.send", "alerts.read", "stats.read"},
			want:   []string{"mail.send", "alerts.read", "stats.read"},
		},
		"empty input": {
			scopes: []string{},
			want:   []string{},
		},
		"nil input": {
			scopes: nil,
			want:   []string{},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := excludeAutoScopes(tt.scopes)
			if got == nil {
				t.Fatal("excludeAutoScopes returned nil, want non-nil slice")
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("excludeAutoScopes(%v) = %v, want %v", tt.scopes, got, tt.want)
			}
		})
	}
}

func TestWithDeclaredAutoScopes(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		scopes   []string
		declared []string
		want     []string
	}{
		"granted and declared auto scope is kept": {
			scopes:   []string{"mail.send", "2fa_exempt"},
			declared: []string{"mail.send", "2fa_exempt"},
			want:     []string{"mail.send", "2fa_exempt"},
		},
		"declared auto scope is kept even if SendGrid granted its counterpart": {
			scopes:   []string{"mail.send", "2fa_required"},
			declared: []string{"mail.send", "2fa_exempt"},
			want:     []string{"mail.send", "2fa_exempt"},
		},
		"undeclared auto scopes are dropped": {
			scopes:   []string{"mail.send", "2fa_required"},
			declared: []string{"mail.send"},
			want:     []string{"mail.send"},
		},
		"undeclared user scopes are kept to surface drift": {
			scopes:   []string{"mail.send", "stats.read"},
			declared: []string{"mail.send"},
			want:     []string{"mail.send", "stats.read"},
		},
		"nil declared behaves like excludeAutoScopes": {
			scopes:   []string{"mail.send", "2fa_exempt"},
			declared: nil,
			want:     []string{"mail.send"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := withDeclaredAutoScopes(tt.scopes, tt.declared)
			if !slices.Equal(got, tt.want) {
				t.Errorf("withDeclaredAutoScopes(%v, %v) = %v, want %v", tt.scopes, tt.declared, got, tt.want)
			}
		})
	}
}
