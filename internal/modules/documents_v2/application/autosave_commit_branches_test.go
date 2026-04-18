package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"metaldocs/internal/modules/documents_v2/application"
	"metaldocs/internal/modules/documents_v2/domain"
)

func TestCommitAutosave_RejectionBranches(t *testing.T) {
	baseMeta := &application.PendingCommitMeta{
		SessionID:           "sess_1",
		DocumentID:          "doc_1",
		BaseRevisionID:      "rev_base",
		ExpectedContentHash: "h_expected",
		StorageKey:          "documents/doc_1/pending/p_1.docx",
		ExpiresAt:           time.Now().Add(10 * time.Minute),
	}

	tests := []struct {
		name                string
		pendingErr          error
		hashErr             error
		hashReturn          string
		commitResult        *application.CommitResult
		commitErr           error
		wantErr             error
		assertOrphanDeleted bool
	}{
		{name: "pending_not_found", pendingErr: domain.ErrPendingNotFound, wantErr: domain.ErrPendingNotFound},
		{name: "upload_missing", hashErr: domain.ErrUploadMissing, wantErr: domain.ErrUploadMissing},
		{name: "content_hash_mismatch", hashReturn: "wronghash", wantErr: domain.ErrContentHashMismatch, assertOrphanDeleted: true},
		{name: "misbound_session", commitErr: domain.ErrMisbound, wantErr: domain.ErrMisbound},
		{name: "already_consumed_replay", commitResult: &application.CommitResult{AlreadyConsumed: true}, wantErr: nil},
		{name: "expired_upload", commitErr: domain.ErrExpiredUpload, wantErr: domain.ErrExpiredUpload},
		{name: "session_inactive", commitErr: domain.ErrSessionInactive, wantErr: domain.ErrSessionInactive},
		{name: "session_not_holder", commitErr: domain.ErrSessionNotHolder, wantErr: domain.ErrSessionNotHolder},
		{name: "stale_base", commitErr: domain.ErrStaleBase, wantErr: domain.ErrStaleBase},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeRepo{
				pendingMeta: &application.PendingCommitMeta{
					SessionID:           baseMeta.SessionID,
					DocumentID:          baseMeta.DocumentID,
					BaseRevisionID:      baseMeta.BaseRevisionID,
					ExpectedContentHash: baseMeta.ExpectedContentHash,
					StorageKey:          baseMeta.StorageKey,
					ExpiresAt:           baseMeta.ExpiresAt,
				},
				pendingErr:   tc.pendingErr,
				commitResult: tc.commitResult,
				commitErr:    tc.commitErr,
			}
			presigner := &fakePresigner{
				hashReturn: "h_expected",
				hashErr:    tc.hashErr,
			}
			if tc.hashReturn != "" {
				presigner.hashReturn = tc.hashReturn
			}
			svc := application.New(repo, nil, presigner, nil, nil, &noopAudit{})

			result, err := svc.CommitAutosave(context.Background(), application.CommitAutosaveCmd{
				TenantID:         "tenant_1",
				ActorUserID:      "user_1",
				DocumentID:       "doc_1",
				SessionID:        "sess_1",
				PendingUploadID:  "pending_1",
				FormDataSnapshot: []byte(`{"a":1}`),
			})

			if tc.wantErr == nil {
				if err != nil {
					t.Fatalf("expected nil error, got %v", err)
				}
			} else if !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected error %v, got %v", tc.wantErr, err)
			}

			if tc.assertOrphanDeleted {
				if presigner.deleteCalls != 1 {
					t.Fatalf("expected orphan delete call, got %d", presigner.deleteCalls)
				}
			} else if presigner.deleteCalls != 0 {
				t.Fatalf("unexpected orphan delete calls: %d", presigner.deleteCalls)
			}

			if tc.name == "already_consumed_replay" {
				if result == nil || !result.AlreadyConsumed {
					t.Fatalf("expected AlreadyConsumed=true result, got %+v", result)
				}
			}
		})
	}
}
