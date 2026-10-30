// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kenzo0107/sendgrid"
)

// The client returns SendGrid's error bodies as plain errors, so the not found check depends on their wording.
func TestIsSubuserBrandedLinkNotFound(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
		want   bool
	}{
		{"no branded link", http.StatusNotFound, `{"errors":[{"message":"resource not found"}]}`, true},
		{"unknown subuser", http.StatusNotFound, `{"errors":[{"message":"subuser does not exist"}]}`, true},
		{"server error", http.StatusInternalServerError, `{"errors":[{"message":"internal error"}]}`, false},
		{"forbidden", http.StatusForbidden, `{"errors":[{"field":null,"message":"access forbidden"}]}`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/whitelabel/links/subuser" || r.URL.Query().Get("username") != "example-subuser" {
					t.Errorf("unexpected request %s %s", r.Method, r.URL)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			t.Cleanup(srv.Close)

			client := sendgrid.New("dummy", sendgrid.OptionBaseURL(srv.URL))
			_, err := client.GetSubuserBrandedLink(context.Background(), "example-subuser")
			if err == nil {
				t.Fatal("expected an error")
			}
			if got := isSubuserBrandedLinkNotFound(err); got != tt.want {
				t.Errorf("isSubuserBrandedLinkNotFound(%q) = %t, want %t", err, got, tt.want)
			}
		})
	}
}
