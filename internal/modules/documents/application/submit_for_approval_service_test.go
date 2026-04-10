package application

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/google/uuid"
)

type fakeSubmitForApprovalRepo struct {
	calls  int
	lastID uuid.UUID
	retErr error
}

func (f *fakeSubmitForApprovalRepo) TransitionDraftToPendingApproval(ctx context.Context, draftID uuid.UUID) error {
	f.calls++
	f.lastID = draftID
	return f.retErr
}

func TestSubmitForApprovalService_TransitionsDraft(t *testing.T) {
	repo := &fakeSubmitForApprovalRepo{retErr: nil}
	svc := NewSubmitForApprovalService(repo)

	draftID := uuid.New()
	if err := svc.SubmitForApproval(context.Background(), draftID); err != nil {
		t.Fatalf("SubmitForApproval() error = %v", err)
	}
	if repo.calls != 1 {
		t.Fatalf("repo calls = %d, want %d", repo.calls, 1)
	}
	if repo.lastID != draftID {
		t.Fatalf("draftID = %s, want %s", repo.lastID, draftID)
	}
}

func TestSubmitForApprovalService_RejectsNonDraft(t *testing.T) {
	repo := &fakeSubmitForApprovalRepo{retErr: sql.ErrNoRows}
	svc := NewSubmitForApprovalService(repo)

	err := svc.SubmitForApproval(context.Background(), uuid.New())
	if err == nil {
		t.Fatalf("SubmitForApproval() error = nil, want non-nil")
	}
	if !errors.Is(err, ErrSubmitForApprovalDraftNotDraft) {
		t.Fatalf("error = %v, want %v", err, ErrSubmitForApprovalDraftNotDraft)
	}
}

func TestSubmitForApprovalService_PropagatesUnexpectedError(t *testing.T) {
	wantErr := errors.New("db unavailable")
	repo := &fakeSubmitForApprovalRepo{retErr: wantErr}
	svc := NewSubmitForApprovalService(repo)

	err := svc.SubmitForApproval(context.Background(), uuid.New())
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want %v", err, wantErr)
	}
}

func TestSubmitForApprovalService_RejectsNilRepository(t *testing.T) {
	svc := NewSubmitForApprovalService(nil)

	err := svc.SubmitForApproval(context.Background(), uuid.New())
	if err == nil {
		t.Fatalf("SubmitForApproval() error = nil, want non-nil")
	}
}
