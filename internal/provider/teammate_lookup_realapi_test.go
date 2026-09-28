// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kenzo0107/sendgrid"
)

// recordingTransport records every request that goes through the provider's HTTP client.
type recordingTransport struct {
	next http.RoundTripper

	mu       sync.Mutex
	requests []string
}

func (r *recordingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	r.mu.Lock()
	r.requests = append(r.requests, req.Method+" "+req.URL.Path)
	r.mu.Unlock()
	return r.next.RoundTrip(req)
}

func (r *recordingTransport) reset() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	got := r.requests
	r.requests = nil
	return got
}

// TestLookupTeammate_AcceptedTeammateRealAPI verifies the direct lookup against
// the real SendGrid API. It needs a teammate that has already accepted the
// invitation, so it runs only when both variables are set:
//
//	SENDGRID_API_KEY=... SENDGRID_TEST_ACCEPTED_TEAMMATE_EMAIL=alice@example.com \
//	  go test -v -run TestLookupTeammate_AcceptedTeammateRealAPI ./internal/provider/
func TestLookupTeammate_AcceptedTeammateRealAPI(t *testing.T) {
	apiKey := os.Getenv("SENDGRID_API_KEY")
	email := os.Getenv("SENDGRID_TEST_ACCEPTED_TEAMMATE_EMAIL")
	if apiKey == "" || email == "" {
		t.Skip("SENDGRID_API_KEY and SENDGRID_TEST_ACCEPTED_TEAMMATE_EMAIL must be set")
	}

	rec := &recordingTransport{next: newHTTPClient().Transport}
	client := sendgrid.New(apiKey, sendgrid.OptionHTTPClient(&http.Client{Transport: rec}))
	ctx := context.Background()

	// First read after import / acceptance: the username is unknown, so the
	// email-based scan (pending list, teammate list, detail) must discover it.
	discovered, err := lookupTeammate(ctx, client, teammateResourceModel{Email: types.StringValue(email)})
	if err != nil {
		t.Fatalf("lookupTeammate() without username: %v", err)
	}
	if discovered == nil {
		t.Fatalf("teammate %s not found; it must exist and have accepted the invitation", email)
	}
	if discovered.Username.ValueString() == "" {
		t.Fatalf("teammate %s is still pending; the direct lookup needs an accepted teammate", email)
	}
	scan := rec.reset()
	t.Logf("scan without username: %d requests %v", len(scan), scan)

	// Steady state: the username is in state, so exactly one request is issued.
	direct, err := lookupTeammate(ctx, client, *discovered)
	if err != nil {
		t.Fatalf("lookupTeammate() with username: %v", err)
	}
	got := rec.reset()
	t.Logf("direct lookup with username: %d requests %v", len(got), got)

	want := fmt.Sprintf("GET /v3/teammates/%s", discovered.Username.ValueString())
	if len(got) != 1 || got[0] != want {
		t.Errorf("direct lookup issued %v, want exactly [%s]", got, want)
	}
	if direct == nil || fmt.Sprintf("%+v", *direct) != fmt.Sprintf("%+v", *discovered) {
		t.Errorf("direct lookup = %+v, want the same model as the scan %+v", direct, discovered)
	}
}
