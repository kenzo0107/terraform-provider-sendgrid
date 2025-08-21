package provider

import (
	"context"

	"github.com/kenzo0107/sendgrid"
)

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
	limit := 50
	
	for {
		input := &sendgrid.InputGetTeammates{
			Limit:  limit,
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
		
		if len(r.Teammates) < limit {
			break
		}
		
		offset += limit
	}
	
	return nil, nil
}
