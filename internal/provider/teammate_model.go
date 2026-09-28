// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/kenzo0107/sendgrid"
)

// teammatesPageLimit is the maximum page size accepted by GET /v3/teammates.
// https://www.twilio.com/docs/sendgrid/api-reference/teammates/retrieve-all-teammates
const teammatesPageLimit = 500

func pendingTeammateByEmail(ctx context.Context, client *sendgrid.Client, email string) (*sendgrid.PendingTeammate, error) {
	r, err := client.GetPendingTeammates(ctx)
	if err != nil {
		return nil, err
	}

	var pendingTeammate *sendgrid.PendingTeammate
	for _, t := range r.PendingTeammates {
		t := &t
		if email != t.Email {
			continue
		}
		pendingTeammate = t
		break
	}
	return pendingTeammate, nil
}

func getTeammateByEmail(ctx context.Context, client *sendgrid.Client, email string) (*sendgrid.Teammate, error) {
	offset := 0

	for {
		input := &sendgrid.InputGetTeammates{
			Limit:  teammatesPageLimit,
			Offset: offset,
		}

		r, err := client.GetTeammates(ctx, input)
		if err != nil {
			return nil, err
		}

		for _, t := range r.Teammates {
			t := &t
			if email == t.Email {
				return t, nil
			}
		}

		// A short (or empty) page is the last one.
		if len(r.Teammates) < teammatesPageLimit {
			break
		}

		offset += teammatesPageLimit
	}

	return nil, nil
}

// lookupTeammate returns the current state of the teammate recorded in state,
// or nil when SendGrid no longer knows the teammate.
//
// When the username is already in state (the teammate accepted the invitation
// on an earlier read) the teammate is fetched directly with
// GET /v3/teammates/{username}, which costs one request instead of scanning
// the pending list and the paginated teammate list. The email-based scan is
// used only when the username is unknown or the direct lookup fails.
func lookupTeammate(ctx context.Context, client *sendgrid.Client, state teammateResourceModel) (*teammateResourceModel, error) {
	email := state.Email.ValueString()

	if username := state.Username.ValueString(); username != "" {
		o, err := client.GetTeammate(ctx, username)
		switch {
		case err == nil && o.Email == email:
			m := teammateModelFromOutput(o)
			return &m, nil
		case err != nil && !shouldFallbackToEmailLookup(err):
			return nil, fmt.Errorf("unable to read teammate (username: %s): %w", username, err)
		}

		// SendGrid reports an unknown username as a 404 with a generic error
		// message, so the status code is not available here. Any other failure
		// (or an email mismatch) falls back to the email-based lookup, which is
		// authoritative and removes the resource when the teammate is gone.
		tflog.Debug(ctx, "Direct teammate lookup failed, falling back to email lookup", map[string]interface{}{
			"username": username,
			"email":    email,
			"error":    fmt.Sprint(err),
		})
	}

	pendingTeammate, err := pendingTeammateByEmail(ctx, client, email)
	if err != nil {
		return nil, fmt.Errorf("unable to get pending teammates: %w", err)
	}
	if pendingTeammate != nil {
		// NOTE: As per the SendGrid API specifications,
		//       pending teammates cannot update the administrator flag.
		//       In such cases, discrepancies arise between the Terraform code and the tfstate,
		//       leading to errors during the execution of terraform apply.
		//       For pending teammates, it update the is_admin value in the tfstate to prevent any discrepancies.
		//       While there might be differences from the actual code,
		//       not accommodating the above would hinder team member management, making it unavoidable.
		m := pendingTeammateModel(pendingTeammate, state.IsAdmin)
		return &m, nil
	}

	teammateByEmail, err := getTeammateByEmail(ctx, client, email)
	if err != nil {
		return nil, fmt.Errorf("unable to read teammate (%s): %w", email, err)
	}
	if teammateByEmail == nil {
		return nil, nil
	}

	o, err := client.GetTeammate(ctx, teammateByEmail.Username)
	if err != nil {
		return nil, fmt.Errorf("unable to read teammate (username: %s): %w", teammateByEmail.Username, err)
	}

	m := teammateModelFromOutput(o)
	return &m, nil
}

// shouldFallbackToEmailLookup reports whether a failed direct lookup should be
// retried through the email-based scan. Rate limits and cancelled contexts are
// returned as-is because the scan would only add more requests.
func shouldFallbackToEmailLookup(err error) bool {
	var rle *sendgrid.RateLimitedError
	if errors.As(err, &rle) {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	return true
}

// teammateModelFromOutput builds the resource model of a teammate that has
// accepted the invitation.
func teammateModelFromOutput(o *sendgrid.OutputGetTeammate) teammateResourceModel {
	scopes := []types.String{}
	// admin users have all scopes, so we don't need to set them.
	if !o.IsAdmin {
		for _, s := range o.Scopes {
			// Automatically assigned scopes in SendGrid are not managed.
			if slices.Contains(autoScopes, s) {
				continue
			}
			scopes = append(scopes, types.StringValue(s))
		}
	}

	return teammateResourceModel{
		ID:       types.StringValue(o.Email),
		Email:    types.StringValue(o.Email),
		IsAdmin:  types.BoolValue(o.IsAdmin),
		Username: types.StringValue(o.Username),
		Scopes:   scopes,
	}
}

// pendingTeammateModel builds the resource model of a teammate whose
// invitation has not been accepted yet. Pending teammates have no username.
func pendingTeammateModel(p *sendgrid.PendingTeammate, isAdmin types.Bool) teammateResourceModel {
	scopes := []types.String{}
	// administrators have all scopes, so we don't need to set them.
	if !isAdmin.ValueBool() {
		for _, s := range p.Scopes {
			if slices.Contains(autoScopes, s) {
				continue
			}
			scopes = append(scopes, types.StringValue(s))
		}
	}

	return teammateResourceModel{
		ID:      types.StringValue(p.Email),
		Email:   types.StringValue(p.Email),
		IsAdmin: isAdmin,
		Scopes:  scopes,
	}
}
