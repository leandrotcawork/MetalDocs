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

// IAMUser is a minimal user record from the auth system.
// Exported so that adapters in cmd/ or infrastructure packages can implement IAMUserLister.
type IAMUser struct {
	UserID      string
	DisplayName string
}

// IAMUserLister is the narrow auth-system port consumed by IAMUserOptionsAdapter.
// Production wiring: wrap auth.Service with an adapter that maps authdomain.ManagedUser → IAMUser.
type IAMUserLister interface {
	ListUsers(ctx context.Context) ([]IAMUser, error)
}

// IAMUserOptionsAdapter adapts an IAMUserLister to IAMUserOptionsReader.
// tenantID is accepted for API compatibility but currently ignored (auth.Service is global-scoped).
type IAMUserOptionsAdapter struct {
	lister IAMUserLister
}

func NewIAMUserOptionsAdapter(lister IAMUserLister) *IAMUserOptionsAdapter {
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
