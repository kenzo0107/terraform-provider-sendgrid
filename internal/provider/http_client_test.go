// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kenzo0107/sendgrid"
)

// testRetryPolicy keeps the exponential backoff short so tests finish quickly.
var testRetryPolicy = rateLimitRetryPolicy{
	maxRetries: 5,
	minWait:    10 * time.Millisecond,
	maxWait:    100 * time.Millisecond,
}

// rateLimitedServer answers 429 for the first `limited` requests and 200 afterwards.
// X-RateLimit-Reset points to the current second, which is treated as already
// elapsed, so the client falls back to the (short) exponential backoff.
func rateLimitedServer(t *testing.T, limited int32) (*httptest.Server, *int32, *[]string) {
	t.Helper()

	var hits int32
	var bodies []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&hits, 1)
		b, _ := io.ReadAll(r.Body)
		bodies = append(bodies, string(b))

		if n <= limited {
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(time.Now().Unix(), 10))
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":[]}`))
	}))
	t.Cleanup(srv.Close)

	return srv, &hits, &bodies
}

func TestNewHTTPClient_RetriesOn429(t *testing.T) {
	t.Parallel()

	srv, hits, _ := rateLimitedServer(t, 2)

	resp, err := newRetryingHTTPClient(testRetryPolicy).Get(srv.URL + "/teammates")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if got := atomic.LoadInt32(hits); got != 3 {
		t.Errorf("server hit %d times, want 3 (1 + 2 retries)", got)
	}
}

func TestNewHTTPClient_ReplaysRequestBodyOnRetry(t *testing.T) {
	t.Parallel()

	srv, hits, bodies := rateLimitedServer(t, 1)

	resp, err := newRetryingHTTPClient(testRetryPolicy).Post(srv.URL+"/teammates", "application/json", strings.NewReader(`{"email":"a@example.com"}`))
	if err != nil {
		t.Fatalf("Post() error = %v", err)
	}
	defer resp.Body.Close()

	if got := atomic.LoadInt32(hits); got != 2 {
		t.Fatalf("server hit %d times, want 2", got)
	}
	for i, b := range *bodies {
		if b != `{"email":"a@example.com"}` {
			t.Errorf("attempt %d body = %q, want the original body", i+1, b)
		}
	}
}

func TestNewHTTPClient_SurfacesRateLimitedErrorWhenRetriesAreExhausted(t *testing.T) {
	t.Parallel()

	srv, hits, _ := rateLimitedServer(t, 1000)

	client := sendgrid.New("dummy",
		sendgrid.OptionBaseURL(srv.URL),
		sendgrid.OptionHTTPClient(newRetryingHTTPClient(testRetryPolicy)),
	)

	_, err := client.GetPendingTeammates(context.Background())

	var rle *sendgrid.RateLimitedError
	if !errors.As(err, &rle) {
		t.Fatalf("error = %v, want *sendgrid.RateLimitedError", err)
	}
	want := int32(testRetryPolicy.maxRetries + 1)
	if got := atomic.LoadInt32(hits); got != want {
		t.Errorf("server hit %d times, want %d (1 + %d retries)", got, want, testRetryPolicy.maxRetries)
	}
}

func TestNewHTTPClient_DoesNotRetryOtherFailures(t *testing.T) {
	t.Parallel()

	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	resp, err := newRetryingHTTPClient(testRetryPolicy).Get(srv.URL)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusInternalServerError)
	}
	if got := atomic.LoadInt32(&hits); got != 1 {
		t.Errorf("server hit %d times, want 1", got)
	}
}

func TestRetryOnTooManyRequests(t *testing.T) {
	t.Parallel()

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()

	tests := map[string]struct {
		ctx     context.Context
		resp    *http.Response
		err     error
		want    bool
		wantErr bool
	}{
		"429 is retried":              {ctx: context.Background(), resp: &http.Response{StatusCode: 429}, want: true},
		"200 is not retried":          {ctx: context.Background(), resp: &http.Response{StatusCode: 200}, want: false},
		"500 is not retried":          {ctx: context.Background(), resp: &http.Response{StatusCode: 500}, want: false},
		"transport error not retried": {ctx: context.Background(), err: errors.New("boom"), want: false, wantErr: true},
		"cancelled context stops":     {ctx: cancelled, resp: &http.Response{StatusCode: 429}, want: false, wantErr: true},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := retryOnTooManyRequests(tt.ctx, tt.resp, tt.err)
			if got != tt.want {
				t.Errorf("retry = %v, want %v", got, tt.want)
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("err = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRateLimitBackoff(t *testing.T) {
	t.Parallel()

	const (
		minWait = time.Second
		maxWait = 60 * time.Second
	)

	withHeader := func(k, v string) *http.Response {
		h := http.Header{}
		h.Set(k, v)
		return &http.Response{Header: h}
	}

	tests := map[string]struct {
		resp    *http.Response
		attempt int
		wantMin time.Duration
		wantMax time.Duration
	}{
		"X-RateLimit-Reset in the future waits until reset plus jitter": {
			resp:    withHeader("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(10*time.Second).Unix(), 10)),
			attempt: 2,
			wantMin: 9 * time.Second,
			wantMax: 10*time.Second + 200*time.Millisecond,
		},
		"X-RateLimit-Reset in the past falls back to exponential backoff": {
			resp:    withHeader("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(-time.Minute).Unix(), 10)),
			attempt: 3,
			wantMin: 8 * time.Second,
			wantMax: 8 * time.Second,
		},
		"Retry-After of zero falls back to exponential backoff": {
			resp:    withHeader("Retry-After", "0"),
			attempt: 0,
			wantMin: time.Second,
			wantMax: time.Second,
		},
		"Retry-After seconds is honoured": {
			resp:    withHeader("Retry-After", "7"),
			attempt: 0,
			wantMin: 7 * time.Second,
			wantMax: 7 * time.Second,
		},
		"no header falls back to exponential backoff": {
			resp:    &http.Response{Header: http.Header{}},
			attempt: 3,
			wantMin: 8 * time.Second,
			wantMax: 8 * time.Second,
		},
		"nil response falls back to exponential backoff": {
			resp:    nil,
			attempt: 1,
			wantMin: 2 * time.Second,
			wantMax: 2 * time.Second,
		},
		"wait is capped at the maximum": {
			resp:    withHeader("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(time.Hour).Unix(), 10)),
			attempt: 0,
			wantMin: maxWait,
			wantMax: maxWait,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := rateLimitBackoff(minWait, maxWait, tt.attempt, tt.resp)
			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("rateLimitBackoff() = %s, want between %s and %s", got, tt.wantMin, tt.wantMax)
			}
		})
	}
}
