// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kenzo0107/sendgrid"
)

// TestTeammateCache_ListsAreFetchedOncePerRefresh mirrors a refresh of N
// teammates whose username is not in state yet: the lists are fetched once
// and each teammate costs only its detail request (1 + 1 + N in total).
func TestTeammateCache_ListsAreFetchedOncePerRefresh(t *testing.T) {
	t.Parallel()

	const n = 7

	api, client := newFakeTeammateAPI(t)
	api.teammates = makeTeammates(n)
	for _, tm := range api.teammates {
		api.details[tm.Username] = sendgrid.OutputGetTeammate{Username: tm.Username, Email: tm.Email}
	}
	cache := newTeammateCache(client)

	for _, tm := range api.teammates {
		got, err := lookupTeammate(context.Background(), client, cache, teammateResourceModel{Email: types.StringValue(tm.Email)})
		if err != nil {
			t.Fatalf("lookupTeammate(%s) error = %v", tm.Email, err)
		}
		if got == nil || got.Username.ValueString() != tm.Username {
			t.Fatalf("lookupTeammate(%s) = %+v, want username %q", tm.Email, got, tm.Username)
		}
	}

	want := map[string]int{callPending: 1, callList: 1, callDetail: n}
	for call, w := range want {
		if calls := api.got(call); calls != w {
			t.Errorf("%s called %d times, want %d", call, calls, w)
		}
	}
}

func TestTeammateCache_DataSourceLookupsShareTheLists(t *testing.T) {
	t.Parallel()

	api, client := newFakeTeammateAPI(t)
	api.teammates = makeTeammates(3)
	api.pending = []sendgrid.PendingTeammate{{Email: "invited@example.com"}}
	cache := newTeammateCache(client)

	for i := 0; i < 3; i++ {
		email := fmt.Sprintf("user%04d@example.com", i)
		if p, err := pendingTeammateByEmail(context.Background(), cache, email); err != nil || p != nil {
			t.Fatalf("pendingTeammateByEmail(%s) = %v, %v; want nil, nil", email, p, err)
		}
		if tm, err := getTeammateByEmail(context.Background(), cache, email); err != nil || tm == nil {
			t.Fatalf("getTeammateByEmail(%s) = %v, %v; want teammate", email, tm, err)
		}
	}
	if p, err := pendingTeammateByEmail(context.Background(), cache, "invited@example.com"); err != nil || p == nil {
		t.Fatalf("pendingTeammateByEmail(invited) = %v, %v; want pending teammate", p, err)
	}

	if calls := api.got(callPending); calls != 1 {
		t.Errorf("%s called %d times, want 1", callPending, calls)
	}
	if calls := api.got(callList); calls != 1 {
		t.Errorf("%s called %d times, want 1", callList, calls)
	}
}

func TestTeammateCache_InvalidateRefetches(t *testing.T) {
	t.Parallel()

	api, client := newFakeTeammateAPI(t)
	api.teammates = makeTeammates(1)
	cache := newTeammateCache(client)

	for i := 0; i < 2; i++ {
		if _, err := getTeammateByEmail(context.Background(), cache, "user0000@example.com"); err != nil {
			t.Fatal(err)
		}
		if _, err := pendingTeammateByEmail(context.Background(), cache, "user0000@example.com"); err != nil {
			t.Fatal(err)
		}
	}
	if calls := api.got(callList); calls != 1 {
		t.Fatalf("%s called %d times before invalidate, want 1", callList, calls)
	}

	// A write changed the account: the next lookup must see the new membership.
	api.teammates = makeTeammates(2)
	cache.invalidate()

	got, err := getTeammateByEmail(context.Background(), cache, "user0001@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("teammate added after invalidate() was not found")
	}
	if _, err := pendingTeammateByEmail(context.Background(), cache, "user0000@example.com"); err != nil {
		t.Fatal(err)
	}
	if calls := api.got(callList); calls != 2 {
		t.Errorf("%s called %d times after invalidate, want 2", callList, calls)
	}
	if calls := api.got(callPending); calls != 2 {
		t.Errorf("%s called %d times after invalidate, want 2", callPending, calls)
	}
}

func TestTeammateCache_ExpiresAfterTTL(t *testing.T) {
	t.Parallel()

	api, client := newFakeTeammateAPI(t)
	api.teammates = makeTeammates(1)

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	var mu sync.Mutex
	cache := newTeammateCache(client)
	cache.now = func() time.Time {
		mu.Lock()
		defer mu.Unlock()
		return now
	}
	advance := func(d time.Duration) {
		mu.Lock()
		defer mu.Unlock()
		now = now.Add(d)
	}

	lookup := func() {
		if _, err := getTeammateByEmail(context.Background(), cache, "user0000@example.com"); err != nil {
			t.Fatal(err)
		}
	}

	lookup()
	advance(teammateCacheTTL - time.Second)
	lookup()
	if calls := api.got(callList); calls != 1 {
		t.Fatalf("%s called %d times within the TTL, want 1", callList, calls)
	}

	advance(2 * time.Second)
	lookup()
	if calls := api.got(callList); calls != 2 {
		t.Errorf("%s called %d times after the TTL, want 2", callList, calls)
	}
}

// TestTeammateCache_ConcurrentMissesAreCoalesced mirrors Terraform's default
// parallelism of 10: simultaneous cache misses must result in one request.
func TestTeammateCache_ConcurrentMissesAreCoalesced(t *testing.T) {
	t.Parallel()

	const parallelism = 10

	api, client := newFakeTeammateAPI(t)
	api.teammates = makeTeammates(parallelism)
	api.listDelay = 50 * time.Millisecond
	cache := newTeammateCache(client)

	var wg sync.WaitGroup
	errs := make(chan error, parallelism)
	for i := 0; i < parallelism; i++ {
		wg.Add(1)
		go func(email string) {
			defer wg.Done()
			if _, err := pendingTeammateByEmail(context.Background(), cache, email); err != nil {
				errs <- err
				return
			}
			got, err := getTeammateByEmail(context.Background(), cache, email)
			if err != nil {
				errs <- err
				return
			}
			if got == nil {
				errs <- fmt.Errorf("%s not found", email)
			}
		}(fmt.Sprintf("user%04d@example.com", i))
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}

	if calls := api.got(callPending); calls != 1 {
		t.Errorf("%s called %d times, want 1", callPending, calls)
	}
	if calls := api.got(callList); calls != 1 {
		t.Errorf("%s called %d times, want 1", callList, calls)
	}
}

func TestTeammateCache_FailedFetchIsNotCached(t *testing.T) {
	t.Parallel()

	api, client := newFakeTeammateAPI(t)
	api.pending = []sendgrid.PendingTeammate{{Email: "invited@example.com"}}
	api.failPending = 1
	cache := newTeammateCache(client)

	if _, err := pendingTeammateByEmail(context.Background(), cache, "invited@example.com"); err == nil {
		t.Fatal("first lookup should surface the API error")
	}

	got, err := pendingTeammateByEmail(context.Background(), cache, "invited@example.com")
	if err != nil {
		t.Fatalf("second lookup error = %v", err)
	}
	if got == nil {
		t.Fatal("second lookup should fetch again and find the pending teammate")
	}
	if calls := api.got(callPending); calls != 2 {
		t.Errorf("%s called %d times, want 2", callPending, calls)
	}
}

// TestTeammateCache_ConcurrentFailureIsSharedWithWaiters ensures an outage
// fails all concurrent callers with the single in-flight request instead of
// making each of them wait for its own attempt.
func TestTeammateCache_ConcurrentFailureIsSharedWithWaiters(t *testing.T) {
	t.Parallel()

	const parallelism = 10

	api, client := newFakeTeammateAPI(t)
	api.listDelay = 50 * time.Millisecond
	api.failPending = parallelism
	cache := newTeammateCache(client)

	var wg sync.WaitGroup
	errs := make(chan error, parallelism)
	for i := 0; i < parallelism; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := pendingTeammateByEmail(context.Background(), cache, "invited@example.com")
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)

	failed := 0
	for err := range errs {
		if err != nil {
			failed++
		}
	}
	if failed != parallelism {
		t.Errorf("%d callers failed, want all %d", failed, parallelism)
	}
	if calls := api.got(callPending); calls != 1 {
		t.Errorf("%s called %d times, want 1", callPending, calls)
	}
}

func TestTeammateCache_WaiterStopsOnItsOwnContext(t *testing.T) {
	t.Parallel()

	api, client := newFakeTeammateAPI(t)
	api.listDelay = 200 * time.Millisecond
	cache := newTeammateCache(client)

	leaderDone := make(chan error, 1)
	go func() {
		_, err := pendingTeammateByEmail(context.Background(), cache, "x@example.com")
		leaderDone <- err
	}()
	time.Sleep(20 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := pendingTeammateByEmail(ctx, cache, "x@example.com")
	if err == nil || time.Since(start) > 150*time.Millisecond {
		t.Errorf("waiter returned err=%v after %s, want a context error before the fetch finished", err, time.Since(start))
	}
	if err := <-leaderDone; err != nil {
		t.Errorf("leader error = %v, want nil", err)
	}
}

func TestTeammateCache_PaginatesTheWholeList(t *testing.T) {
	t.Parallel()

	api, client := newFakeTeammateAPI(t)
	api.teammates = makeTeammates(2*teammatesPageLimit + 1)
	cache := newTeammateCache(client)

	got, err := cache.allTeammates(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(api.teammates) {
		t.Errorf("allTeammates() returned %d teammates, want %d", len(got), len(api.teammates))
	}
	if calls := api.got(callList); calls != 3 {
		t.Errorf("%s called %d times, want 3", callList, calls)
	}
}
