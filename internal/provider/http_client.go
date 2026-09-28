// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const (
	// httpClientTimeout bounds a single attempt so a stalled connection does not
	// block Terraform indefinitely.
	httpClientTimeout = 60 * time.Second
	// httpMaxIdleConnsPerHost keeps enough idle connections for Terraform's
	// default parallelism (10) so refreshes reuse TLS connections.
	httpMaxIdleConnsPerHost = 20
)

// rateLimitRetryPolicy controls how HTTP 429 responses are retried.
type rateLimitRetryPolicy struct {
	maxRetries int
	minWait    time.Duration
	maxWait    time.Duration
}

var defaultRateLimitRetryPolicy = rateLimitRetryPolicy{
	maxRetries: 5,
	minWait:    1 * time.Second,
	maxWait:    60 * time.Second,
}

// newHTTPClient returns the HTTP client injected into the SendGrid client.
//
// Every request, including the Read paths, is retried on HTTP 429 honouring
// SendGrid's X-RateLimit-Reset header. When the retries are exhausted the last
// 429 response is handed back to the SendGrid client so it still surfaces as
// *sendgrid.RateLimitedError.
func newHTTPClient() *http.Client {
	return newRetryingHTTPClient(defaultRateLimitRetryPolicy)
}

func newRetryingHTTPClient(policy rateLimitRetryPolicy) *http.Client {
	transport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		transport = &http.Transport{}
	}
	transport = transport.Clone()
	transport.MaxIdleConnsPerHost = httpMaxIdleConnsPerHost

	rc := retryablehttp.NewClient()
	rc.HTTPClient = &http.Client{Transport: transport, Timeout: httpClientTimeout}
	// Retries are logged through tflog in rateLimitBackoff instead.
	rc.Logger = nil
	rc.RetryMax = policy.maxRetries
	rc.RetryWaitMin = policy.minWait
	rc.RetryWaitMax = policy.maxWait
	rc.CheckRetry = retryOnTooManyRequests
	rc.Backoff = rateLimitBackoff
	rc.ErrorHandler = retryablehttp.PassthroughErrorHandler

	return rc.StandardClient()
}

// retryOnTooManyRequests retries only HTTP 429 responses.
//
// SendGrid does not process a rate-limited request, so retrying is safe for
// POST/PUT/PATCH as well. Transport errors are not retried because a
// non-idempotent request may already have been applied.
func retryOnTooManyRequests(ctx context.Context, resp *http.Response, err error) (bool, error) {
	if ctx.Err() != nil {
		return false, ctx.Err()
	}
	if err != nil {
		// Propagate the transport error unchanged; retryablehttp returns it as-is.
		return false, err
	}
	if resp == nil {
		return false, nil
	}
	return resp.StatusCode == http.StatusTooManyRequests, nil
}

// rateLimitBackoff waits until the rate-limit window reported by SendGrid resets
// and falls back to exponential backoff when no header is available.
func rateLimitBackoff(minWait, maxWait time.Duration, attemptNum int, resp *http.Response) time.Duration {
	var wait time.Duration
	if reset, ok := rateLimitResetWait(resp); ok {
		// Spread concurrent retries slightly so they do not all fire at the reset instant.
		wait = reset + time.Duration(attemptNum*100)*time.Millisecond
	} else {
		wait = minWait * (1 << uint(attemptNum))
	}
	if wait > maxWait {
		wait = maxWait
	}

	if resp != nil && resp.Request != nil {
		tflog.Info(resp.Request.Context(), "Rate limited, retrying", map[string]interface{}{
			"method":        resp.Request.Method,
			"path":          resp.Request.URL.Path,
			"retry_attempt": attemptNum + 1,
			"wait_seconds":  wait.Seconds(),
		})
	}

	return wait
}

// rateLimitResetWait returns how long to wait according to the response headers.
// SendGrid sends X-RateLimit-Reset (unix seconds); Retry-After (seconds) is
// accepted as well. It reports false when no header points to the future.
func rateLimitResetWait(resp *http.Response) (time.Duration, bool) {
	if resp == nil {
		return 0, false
	}

	if v := resp.Header.Get("X-RateLimit-Reset"); v != "" {
		if unix, err := strconv.ParseInt(v, 10, 64); err == nil {
			// A reset in the past (clock skew, or a window that ends this second)
			// carries no usable information, so the caller backs off exponentially.
			if wait := time.Until(time.Unix(unix, 0)); wait > 0 {
				return wait, true
			}
		}
	}

	if v := resp.Header.Get("Retry-After"); v != "" {
		if secs, err := strconv.Atoi(v); err == nil && secs > 0 {
			return time.Duration(secs) * time.Second, true
		}
	}

	return 0, false
}
