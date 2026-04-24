package application

import (
	"context"
	"errors"
	"testing"
)

type fakeIAMUserLister struct {
	users []iamUser
	err   error
}

func (f *fakeIAMUserLister) ListUsers(_ context.Context) ([]iamUser, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.users, nil
}

func TestIAMUserOptionsAdapter_SortsByDisplayName(t *testing.T) {
	lister := &fakeIAMUserLister{users: []iamUser{
		{UserID: "u3", DisplayName: "Charlie"},
		{UserID: "u1", DisplayName: "Alice"},
		{UserID: "u2", DisplayName: "Bob"},
	}}
	adapter := NewIAMUserOptionsAdapter(lister)

	opts, err := adapter.ListUserOptions(context.Background(), "tenant-1")
	if err != nil {
		t.Fatalf("ListUserOptions: %v", err)
	}
	if len(opts) != 3 {
		t.Fatalf("len=%d, want 3", len(opts))
	}
	want := []string{"Alice", "Bob", "Charlie"}
	for i, w := range want {
		if opts[i].DisplayName != w {
			t.Errorf("opts[%d].DisplayName=%q, want %q", i, opts[i].DisplayName, w)
		}
	}
	if opts[1].UserID != "u2" {
		t.Errorf("opts[1].UserID=%q, want u2", opts[1].UserID)
	}
}

func TestIAMUserOptionsAdapter_PropagatesError(t *testing.T) {
	lister := &fakeIAMUserLister{err: errors.New("iam down")}
	adapter := NewIAMUserOptionsAdapter(lister)

	_, err := adapter.ListUserOptions(context.Background(), "tenant-1")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestIAMUserOptionsAdapter_EmptyTenantReturnsEmpty(t *testing.T) {
	adapter := NewIAMUserOptionsAdapter(&fakeIAMUserLister{users: nil})
	opts, err := adapter.ListUserOptions(context.Background(), "t")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(opts) != 0 {
		t.Errorf("len=%d, want 0", len(opts))
	}
}
