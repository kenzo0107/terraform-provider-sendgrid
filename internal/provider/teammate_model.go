package provider

import (
	"context"

	"github.com/kenzo0107/sendgrid"
)

func pendingTeammateByEmail(ctx context.Context, client *sendgrid.Client, email string) (*sendgrid.PendingTeammate, error) {
	// Invited users are treated as pending users until they set up their profiles.
	r, err := client.GetPendingTeammates(ctx)
	if err != nil {
		return nil, err
	}

	// retrieve specific pending user
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
	// NOTE: When retrieving a list of teammates that exceeds the default fetch limit,
	//       we need to implement pagination using `limit` and `offset`.
	//       In that case, you should redesign the logic with performance in mind.
	r, err := client.GetTeammates(ctx, nil)
	if err != nil {
		return nil, err
	}

	var teammate *sendgrid.Teammate
	for _, t := range r.Teammates {
		t := &t
		if email != t.Email {
			continue
		}
		teammate = t
		break
	}
	return teammate, nil
}
