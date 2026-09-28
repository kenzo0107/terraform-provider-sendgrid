// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/kenzo0107/sendgrid"
	"golang.org/x/sync/singleflight"
)

// teammateCacheTTL bounds how long the teammate lists are reused within one
// provider instance. Terraform starts a fresh provider process per command, so
// the cache never outlives a single plan/apply run.
const teammateCacheTTL = 30 * time.Second

// teammateCache memoises the responses of GET /v3/teammates and
// GET /v3/teammates/pending for one provider instance.
//
// Every sendgrid_teammate whose username is not yet in state, and every
// data.sendgrid_teammate, has to scan both lists to find the teammate by email.
// Terraform refreshes resources in parallel, so without the cache each of them
// fetches the same lists again. With the cache the lists are fetched once per
// TTL window and the per-resource cost drops to the detail request.
//
// The cache lives in process memory only; it is never written to disk or to
// the Terraform state. It must be invalidated after any write that changes the
// membership of either list (invite, accept, delete).
type teammateCache struct {
	client *sendgrid.Client
	ttl    time.Duration
	// now is injectable for tests.
	now func() time.Time

	pending   cachedList[sendgrid.PendingTeammate]
	teammates cachedList[sendgrid.Teammate]
}

func newTeammateCache(client *sendgrid.Client) *teammateCache {
	return &teammateCache{
		client: client,
		ttl:    teammateCacheTTL,
		now:    time.Now,
	}
}

// pendingTeammates returns the pending invitations of the account.
func (c *teammateCache) pendingTeammates(ctx context.Context) ([]sendgrid.PendingTeammate, error) {
	return c.pending.get(ctx, c.ttl, c.now, func(ctx context.Context) ([]sendgrid.PendingTeammate, error) {
		r, err := c.client.GetPendingTeammates(ctx)
		if err != nil {
			return nil, err
		}
		return r.PendingTeammates, nil
	})
}

// allTeammates returns every teammate that has accepted the invitation.
func (c *teammateCache) allTeammates(ctx context.Context) ([]sendgrid.Teammate, error) {
	return c.teammates.get(ctx, c.ttl, c.now, func(ctx context.Context) ([]sendgrid.Teammate, error) {
		return listAllTeammates(ctx, c.client)
	})
}

// invalidate drops both lists so the next lookup fetches fresh data. Call it
// after a successful write to a teammate or SSO teammate.
func (c *teammateCache) invalidate() {
	c.pending.invalidate()
	c.teammates.invalidate()
}

// listAllTeammates walks the paginated GET /v3/teammates endpoint.
func listAllTeammates(ctx context.Context, client *sendgrid.Client) ([]sendgrid.Teammate, error) {
	var teammates []sendgrid.Teammate

	for offset := 0; ; offset += teammatesPageLimit {
		r, err := client.GetTeammates(ctx, &sendgrid.InputGetTeammates{
			Limit:  teammatesPageLimit,
			Offset: offset,
		})
		if err != nil {
			return nil, err
		}

		teammates = append(teammates, r.Teammates...)

		// A short (or empty) page is the last one.
		if len(r.Teammates) < teammatesPageLimit {
			return teammates, nil
		}
	}
}

// cachedList holds one memoised list.
//
// Concurrent callers that miss the cache join the in-flight fetch through
// singleflight and all receive its result, error included, so an outage fails
// them together instead of one after another. A failed fetch is not stored and
// the next caller retries.
type cachedList[T any] struct {
	mu        sync.Mutex
	items     []T
	fetchedAt time.Time

	group singleflight.Group
}

func (l *cachedList[T]) get(ctx context.Context, ttl time.Duration, now func() time.Time, fetch func(context.Context) ([]T, error)) ([]T, error) {
	for {
		if items, ok := l.fresh(ttl, now()); ok {
			return items, nil
		}

		// The first caller runs fetch with its own ctx; the others wait for it.
		ch := l.group.DoChan("list", func() (interface{}, error) {
			items, err := fetch(ctx)
			if err != nil {
				return nil, err
			}
			l.store(items, now())
			return items, nil
		})

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case res := <-ch:
			if res.Err == nil {
				items, ok := res.Val.([]T)
				if !ok {
					return nil, fmt.Errorf("teammate cache: unexpected value %T", res.Val)
				}
				return items, nil
			}
			// The leader's ctx was cancelled but ours is alive: run our own fetch.
			if res.Shared && isContextError(res.Err) && ctx.Err() == nil {
				continue
			}
			return nil, res.Err
		}
	}
}

func (l *cachedList[T]) fresh(ttl time.Duration, now time.Time) ([]T, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.fetchedAt.IsZero() || now.Sub(l.fetchedAt) >= ttl {
		return nil, false
	}
	return l.items, true
}

func (l *cachedList[T]) store(items []T, now time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.items = items
	l.fetchedAt = now
}

func (l *cachedList[T]) invalidate() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.items = nil
	l.fetchedAt = time.Time{}
}

func isContextError(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}
