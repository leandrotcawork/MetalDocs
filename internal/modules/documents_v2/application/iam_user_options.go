package application

import (
	"context"
	"sort"
)

type UserOption struct {
	UserID      string `json:"user_id"`
	DisplayName string `json:"display_name"`
}

// IAMUserOptionsReader is the consumer-defined port for listing user options.
type IAMUserOptionsReader interface {
	ListUserOptions(ctx context.Context, tenantID string) ([]UserOption, error)
}

// iamUser is a minimal user record from the auth system.
type iamUser struct {
	UserID      string
	DisplayName string
}

// iamUserLister is the narrow auth-system port consumed by IAMUserOptionsAdapter.
type iamUserLister interface {
	ListUsers(ctx context.Context) ([]iamUser, error)
}

// IAMUserOptionsAdapter adapts an iamUserLister to IAMUserOptionsReader.
// tenantID is accepted for API compatibility but currently ignored (auth.Service is global-scoped).
type IAMUserOptionsAdapter struct {
	lister iamUserLister
}

func NewIAMUserOptionsAdapter(lister iamUserLister) *IAMUserOptionsAdapter {
	return &IAMUserOptionsAdapter{lister: lister}
}

func (a *IAMUserOptionsAdapter) ListUserOptions(ctx context.Context, _ string) ([]UserOption, error) {
	users, err := a.lister.ListUsers(ctx)
	if err != nil {
		return nil, err
	}
	opts := make([]UserOption, len(users))
	for i, u := range users {
		opts[i] = UserOption{UserID: u.UserID, DisplayName: u.DisplayName}
	}
	sort.Slice(opts, func(i, j int) bool {
		return opts[i].DisplayName < opts[j].DisplayName
	})
	return opts, nil
}
