// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kenzo0107/sendgrid"
)

// fakeTeammateAPI serves the subset of the SendGrid teammates API used by the
// provider and counts requests per endpoint.
type fakeTeammateAPI struct {
	t *testing.T

	teammates []sendgrid.Teammate
	pending   []sendgrid.PendingTeammate
	// details holds GET /teammates/{username} responses keyed by username.
	details map[string]sendgrid.OutputGetTeammate
	// rateLimitDetails makes GET /teammates/{username} answer 429 when set.
	rateLimitDetails bool

	mu    sync.Mutex
	calls map[string]int
}

const (
	callList    = "GET /teammates"
	callPending = "GET /teammates/pending"
	callDetail  = "GET /teammates/{username}"
)

func newFakeTeammateAPI(t *testing.T) (*fakeTeammateAPI, *sendgrid.Client) {
	t.Helper()

	api := &fakeTeammateAPI{
		t:       t,
		details: map[string]sendgrid.OutputGetTeammate{},
		calls:   map[string]int{},
	}
	srv := httptest.NewServer(api)
	t.Cleanup(srv.Close)

	return api, sendgrid.New("dummy", sendgrid.OptionBaseURL(srv.URL))
}

func (a *fakeTeammateAPI) count(name string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.calls[name]++
}

func (a *fakeTeammateAPI) got(name string) int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.calls[name]
}

func (a *fakeTeammateAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		a.t.Errorf("unexpected method %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	switch {
	case r.URL.Path == "/teammates/pending":
		a.count(callPending)
		writeJSON(w, http.StatusOK, sendgrid.OutputGetPendingTeammates{PendingTeammates: a.pending})

	case r.URL.Path == "/teammates":
		a.count(callList)
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		if limit != teammatesPageLimit {
			a.t.Errorf("GET /teammates: limit = %d, want %d", limit, teammatesPageLimit)
		}
		page := []sendgrid.Teammate{}
		if offset < len(a.teammates) {
			end := offset + limit
			if end > len(a.teammates) {
				end = len(a.teammates)
			}
			page = a.teammates[offset:end]
		}
		writeJSON(w, http.StatusOK, sendgrid.OutputGetTeammates{Teammates: page})

	case strings.HasPrefix(r.URL.Path, "/teammates/"):
		a.count(callDetail)
		if a.rateLimitDetails {
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(time.Minute).Unix(), 10))
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		username := strings.TrimPrefix(r.URL.Path, "/teammates/")
		o, ok := a.details[username]
		if !ok {
			// Real response observed from the API for an unknown username.
			writeJSON(w, http.StatusNotFound, map[string]interface{}{
				"errors": []map[string]string{{"message": "unable to get teammate data at this time"}},
			})
			return
		}
		writeJSON(w, http.StatusOK, o)

	default:
		a.t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func makeTeammates(n int) []sendgrid.Teammate {
	teammates := make([]sendgrid.Teammate, 0, n)
	for i := 0; i < n; i++ {
		teammates = append(teammates, sendgrid.Teammate{
			Username: fmt.Sprintf("user%04d", i),
			Email:    fmt.Sprintf("user%04d@example.com", i),
		})
	}
	return teammates
}

func TestGetTeammateByEmail(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		teammates int
		email     string
		wantFound bool
		wantCalls int
	}{
		"found on the first page": {
			teammates: 10,
			email:     "user0003@example.com",
			wantFound: true,
			wantCalls: 1,
		},
		"found on the second page": {
			teammates: teammatesPageLimit + 2,
			email:     fmt.Sprintf("user%04d@example.com", teammatesPageLimit+1),
			wantFound: true,
			wantCalls: 2,
		},
		"not found stops at the short page": {
			teammates: teammatesPageLimit + 2,
			email:     "missing@example.com",
			wantFound: false,
			wantCalls: 2,
		},
		"not found stops at the empty page": {
			teammates: teammatesPageLimit,
			email:     "missing@example.com",
			wantFound: false,
			wantCalls: 2,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			api, client := newFakeTeammateAPI(t)
			api.teammates = makeTeammates(tt.teammates)

			got, err := getTeammateByEmail(context.Background(), client, tt.email)
			if err != nil {
				t.Fatalf("getTeammateByEmail() error = %v", err)
			}
			if (got != nil) != tt.wantFound {
				t.Fatalf("getTeammateByEmail() found = %v, want %v", got != nil, tt.wantFound)
			}
			if got != nil && got.Email != tt.email {
				t.Errorf("getTeammateByEmail().Email = %q, want %q", got.Email, tt.email)
			}
			if calls := api.got(callList); calls != tt.wantCalls {
				t.Errorf("GET /teammates called %d times, want %d", calls, tt.wantCalls)
			}
		})
	}
}

func TestLookupTeammate(t *testing.T) {
	t.Parallel()

	const (
		email    = "alice@example.com"
		username = "alice"
	)
	accepted := sendgrid.OutputGetTeammate{
		Username: username,
		Email:    email,
		Scopes:   []string{"mail.send", "2fa_required"},
	}

	tests := map[string]struct {
		setup     func(api *fakeTeammateAPI)
		state     teammateResourceModel
		wantModel *teammateResourceModel
		wantCalls map[string]int
	}{
		"username in state is looked up directly without scanning the lists": {
			setup: func(api *fakeTeammateAPI) {
				api.details[username] = accepted
				api.teammates = []sendgrid.Teammate{{Username: username, Email: email}}
			},
			state: teammateResourceModel{
				Email:    types.StringValue(email),
				Username: types.StringValue(username),
			},
			wantModel: &teammateResourceModel{
				ID:       types.StringValue(email),
				Email:    types.StringValue(email),
				IsAdmin:  types.BoolValue(false),
				Username: types.StringValue(username),
				Scopes:   []types.String{types.StringValue("mail.send")},
			},
			wantCalls: map[string]int{callDetail: 1, callPending: 0, callList: 0},
		},
		"unknown username falls back to the email lookup": {
			setup: func(api *fakeTeammateAPI) {
				api.details["alice2"] = sendgrid.OutputGetTeammate{Username: "alice2", Email: email}
				api.teammates = []sendgrid.Teammate{{Username: "alice2", Email: email}}
			},
			state: teammateResourceModel{
				Email:    types.StringValue(email),
				Username: types.StringValue("alice-old"),
			},
			wantModel: &teammateResourceModel{
				ID:       types.StringValue(email),
				Email:    types.StringValue(email),
				IsAdmin:  types.BoolValue(false),
				Username: types.StringValue("alice2"),
				Scopes:   []types.String{},
			},
			wantCalls: map[string]int{callDetail: 2, callPending: 1, callList: 1},
		},
		"username now belonging to another email falls back to the email lookup": {
			setup: func(api *fakeTeammateAPI) {
				api.details[username] = sendgrid.OutputGetTeammate{Username: username, Email: "someone-else@example.com"}
			},
			state: teammateResourceModel{
				Email:    types.StringValue(email),
				Username: types.StringValue(username),
			},
			wantModel: nil,
			wantCalls: map[string]int{callDetail: 1, callPending: 1, callList: 1},
		},
		"pending teammate is read from the pending list": {
			setup: func(api *fakeTeammateAPI) {
				api.pending = []sendgrid.PendingTeammate{{Email: email, Scopes: []string{"mail.send", "2fa_exempt"}}}
			},
			state: teammateResourceModel{
				Email:   types.StringValue(email),
				IsAdmin: types.BoolValue(false),
			},
			wantModel: &teammateResourceModel{
				ID:      types.StringValue(email),
				Email:   types.StringValue(email),
				IsAdmin: types.BoolValue(false),
				Scopes:  []types.String{types.StringValue("mail.send")},
			},
			wantCalls: map[string]int{callDetail: 0, callPending: 1, callList: 0},
		},
		"accepted teammate without username in state is found by email": {
			setup: func(api *fakeTeammateAPI) {
				api.details[username] = accepted
				api.teammates = []sendgrid.Teammate{{Username: username, Email: email}}
			},
			state: teammateResourceModel{
				Email: types.StringValue(email),
			},
			wantModel: &teammateResourceModel{
				ID:       types.StringValue(email),
				Email:    types.StringValue(email),
				IsAdmin:  types.BoolValue(false),
				Username: types.StringValue(username),
				Scopes:   []types.String{types.StringValue("mail.send")},
			},
			wantCalls: map[string]int{callDetail: 1, callPending: 1, callList: 1},
		},
		"teammate removed outside terraform returns nil": {
			setup: func(api *fakeTeammateAPI) {},
			state: teammateResourceModel{
				Email: types.StringValue(email),
			},
			wantModel: nil,
			wantCalls: map[string]int{callDetail: 0, callPending: 1, callList: 1},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			api, client := newFakeTeammateAPI(t)
			tt.setup(api)

			got, err := lookupTeammate(context.Background(), client, tt.state)
			if err != nil {
				t.Fatalf("lookupTeammate() error = %v", err)
			}

			if (got == nil) != (tt.wantModel == nil) {
				t.Fatalf("lookupTeammate() = %+v, want %+v", got, tt.wantModel)
			}
			if got != nil && fmt.Sprintf("%+v", *got) != fmt.Sprintf("%+v", *tt.wantModel) {
				t.Errorf("lookupTeammate() = %+v, want %+v", *got, *tt.wantModel)
			}

			for call, want := range tt.wantCalls {
				if calls := api.got(call); calls != want {
					t.Errorf("%s called %d times, want %d", call, calls, want)
				}
			}
		})
	}
}

func TestLookupTeammate_RateLimitIsNotSwallowedByFallback(t *testing.T) {
	t.Parallel()

	api, client := newFakeTeammateAPI(t)
	api.rateLimitDetails = true

	_, err := lookupTeammate(context.Background(), client, teammateResourceModel{
		Email:    types.StringValue("alice@example.com"),
		Username: types.StringValue("alice"),
	})

	var rle *sendgrid.RateLimitedError
	if !errors.As(err, &rle) {
		t.Fatalf("lookupTeammate() error = %v, want *sendgrid.RateLimitedError", err)
	}
	if calls := api.got(callPending) + api.got(callList); calls != 0 {
		t.Errorf("fallback issued %d list requests after a rate limit, want 0", calls)
	}
}
