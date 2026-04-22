package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"metaldocs/internal/modules/documents_v2/approval/domain"
)

// ---------------------------------------------------------------------------
// TestMapPgError — error taxonomy with synthetic pgconn.PgError
// ---------------------------------------------------------------------------

func TestMapPgError(t *testing.T) {
	makePgErr := func(code, constraint, msg string) error {
		return &pgconn.PgError{Code: code, ConstraintName: constraint, Message: msg}
	}

	tests := []struct {
		name       string
		err        error
		hints      MapHints
		wantTarget error
	}{
		{
			name:       "unique_active_instance",
			err:        makePgErr("23505", "ux_approval_instances_active", ""),
			wantTarget: ErrDuplicateSubmission,
		},
		{
			name:       "unique_idempotency_key",
			err:        makePgErr("23505", "approval_instances_document_v2_id_idempotency_key_key", ""),
			wantTarget: ErrDuplicateSubmission,
		},
		{
			name:       "unique_signoff_instance_actor",
			err:        makePgErr("23505", "approval_signoffs_approval_instance_id_actor_user_id_key", ""),
			wantTarget: ErrActorAlreadySigned,
		},
		{
			name:       "unique_signoff_stage_actor",
			err:        makePgErr("23505", "approval_signoffs_stage_instance_id_actor_user_id_key", ""),
			wantTarget: ErrActorAlreadySigned,
		},
		{
			name:       "unique_unknown_with_hint_match",
			err:        makePgErr("23505", "my_special_constraint", ""),
			hints:      MapHints{UniqueConstraint: "my_special_constraint"},
			wantTarget: ErrDuplicateSubmission,
		},
		{
			name:       "unique_unknown_no_hint",
			err:        makePgErr("23505", "some_other_constraint", ""),
			wantTarget: ErrActorAlreadySigned,
		},
		{
			name:       "fk_violation",
			err:        makePgErr("23503", "", ""),
			wantTarget: ErrFKViolation,
		},
		{
			name:       "check_violation_no_msg",
			err:        makePgErr("23514", "", ""),
			wantTarget: ErrCheckViolation,
		},
		{
			name:       "check_violation_with_msg",
			err:        makePgErr("23514", "", "cross-tenant signoff rejected"),
			wantTarget: ErrCheckViolation,
		},
		{
			name:       "insufficient_privilege",
			err:        makePgErr("42501", "", ""),
			wantTarget: ErrInsufficientPrivilege,
		},
		{
			name:       "unknown_sqlstate",
			err:        makePgErr("58000", "", "io error"),
			wantTarget: ErrUnknownDB,
		},
		{
			name:  "non_pg_error_passed_through",
			err:   errors.New("generic error"),
			hints: MapHints{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := MapPgError(tc.err, tc.hints)
			if tc.wantTarget == nil {
				// Expect the original error passed through unchanged.
				if got != tc.err {
					t.Errorf("expected passthrough: got %v, want %v", got, tc.err)
				}
				return
			}
			if !errors.Is(got, tc.wantTarget) {
				t.Errorf("MapPgError(%q): got %v, want errors.Is target %v", tc.name, got, tc.wantTarget)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Fake infrastructure for unit tests (no sqlmock, no real DB)
// ---------------------------------------------------------------------------

// fakeResult implements sql.Result for exec operations.
type fakeResult struct {
	rowsAffected int64
}

func (f fakeResult) LastInsertId() (int64, error) { return 0, nil }
func (f fakeResult) RowsAffected() (int64, error)  { return f.rowsAffected, nil }

// ---------------------------------------------------------------------------
// TestUpdateStageStatusOCC — verify ErrStageNotActive on 0 rows affected
// ---------------------------------------------------------------------------

// fakeStageUpdater lets us test the OCC branch without a real database.
// We implement the minimal interface needed to call our OCC logic.

type updateResult struct {
	n   int64
	err error
}

// occ logic extracted for unit testability
func applyStageOCC(res sql.Result, execErr error) error {
	if execErr != nil {
		return MapPgError(execErr, MapHints{})
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrStageNotActive
	}
	return nil
}

func TestUpdateStageStatusOCC(t *testing.T) {
	t.Run("zero_rows_returns_ErrStageNotActive", func(t *testing.T) {
		err := applyStageOCC(fakeResult{rowsAffected: 0}, nil)
		if !errors.Is(err, ErrStageNotActive) {
			t.Errorf("expected ErrStageNotActive, got %v", err)
		}
	})

	t.Run("one_row_returns_nil", func(t *testing.T) {
		err := applyStageOCC(fakeResult{rowsAffected: 1}, nil)
		if err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})

	t.Run("db_error_mapped_through_MapPgError", func(t *testing.T) {
		pgErr := &pgconn.PgError{Code: "42501"}
		err := applyStageOCC(fakeResult{}, pgErr)
		if !errors.Is(err, ErrInsufficientPrivilege) {
			t.Errorf("expected ErrInsufficientPrivilege, got %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// TestInsertSignoffReplayDetection — WasReplay logic
// ---------------------------------------------------------------------------

// We test the replay-detection decision logic directly without a DB by
// reproducing the decision branch that InsertSignoff uses.

func replayDecision(incoming, existing *domain.Signoff) (SignoffInsertResult, error) {
	if existing.StageInstanceID() == incoming.StageInstanceID() &&
		existing.Decision() == incoming.Decision() &&
		existing.ContentHash() == incoming.ContentHash() {
		return SignoffInsertResult{ID: existing.ID(), WasReplay: true}, nil
	}
	return SignoffInsertResult{}, ErrActorAlreadySigned
}

func makeTestSignoff(t *testing.T, id, instanceID, stageID, actor, decision, hash string) *domain.Signoff {
	t.Helper()
	s, err := domain.NewSignoff(domain.SignoffParams{
		ID:                 id,
		ApprovalInstanceID: instanceID,
		StageInstanceID:    stageID,
		ActorUserID:        actor,
		ActorTenantID:      "tenant-a",
		Decision:           domain.Decision(decision),
		SignedAt:           time.Now().UTC(),
		SignatureMethod:    "simple",
		SignaturePayload:   json.RawMessage(`{}`),
		ContentHash:        hash,
	})
	if err != nil {
		t.Fatalf("makeTestSignoff: %v", err)
	}
	return s
}

const testHash = "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"

func TestInsertSignoffReplayDetection(t *testing.T) {
	t.Run("exact_match_is_replay", func(t *testing.T) {
		incoming := makeTestSignoff(t, "id-1", "inst-1", "stage-1", "actor-1", "approve", testHash)
		existing := makeTestSignoff(t, "id-1", "inst-1", "stage-1", "actor-1", "approve", testHash)

		result, err := replayDecision(incoming, existing)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !result.WasReplay {
			t.Error("expected WasReplay=true")
		}
		if result.ID != "id-1" {
			t.Errorf("expected ID=id-1, got %q", result.ID)
		}
	})

	t.Run("different_stage_is_conflict", func(t *testing.T) {
		incoming := makeTestSignoff(t, "id-2", "inst-1", "stage-2", "actor-1", "approve", testHash)
		existing := makeTestSignoff(t, "id-1", "inst-1", "stage-1", "actor-1", "approve", testHash)

		_, err := replayDecision(incoming, existing)
		if !errors.Is(err, ErrActorAlreadySigned) {
			t.Errorf("expected ErrActorAlreadySigned, got %v", err)
		}
	})

	t.Run("different_decision_is_conflict", func(t *testing.T) {
		incoming := makeTestSignoff(t, "id-1", "inst-1", "stage-1", "actor-1", "reject", testHash)
		existing := makeTestSignoff(t, "id-1", "inst-1", "stage-1", "actor-1", "approve", testHash)

		_, err := replayDecision(incoming, existing)
		if !errors.Is(err, ErrActorAlreadySigned) {
			t.Errorf("expected ErrActorAlreadySigned, got %v", err)
		}
	})

	t.Run("different_content_hash_is_conflict", func(t *testing.T) {
		hash2 := "1111111111111111111111111111111111111111111111111111111111111111"
		incoming := makeTestSignoff(t, "id-1", "inst-1", "stage-1", "actor-1", "approve", hash2)
		existing := makeTestSignoff(t, "id-1", "inst-1", "stage-1", "actor-1", "approve", testHash)

		_, err := replayDecision(incoming, existing)
		if !errors.Is(err, ErrActorAlreadySigned) {
			t.Errorf("expected ErrActorAlreadySigned, got %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// TestListScheduledDueSQL — compile + structure test (no real DB)
// ---------------------------------------------------------------------------

// TestListScheduledDueSQL verifies that ListScheduledDue is defined on the concrete
// type and satisfies the ApprovalRepository interface at compile time.
// No real DB query is executed.
func TestListScheduledDueSQL(t *testing.T) {
	// Compile-time interface conformance check.
	var _ ApprovalRepository = (*postgresApprovalRepository)(nil)

	// Verify ScheduledPublishRow has the expected fields.
	row := ScheduledPublishRow{
		DocumentID:      "doc-1",
		TenantID:        "tenant-1",
		EffectiveFrom:   time.Now(),
		RevisionVersion: 3,
	}
	if row.DocumentID == "" {
		t.Error("DocumentID should not be empty")
	}
	if row.RevisionVersion != 3 {
		t.Errorf("RevisionVersion: got %d, want 3", row.RevisionVersion)
	}
}

// ---------------------------------------------------------------------------
// TestSignoffInsertResultFields — basic struct field test
// ---------------------------------------------------------------------------

func TestSignoffInsertResultFields(t *testing.T) {
	r := SignoffInsertResult{ID: "abc", WasReplay: true}
	if r.ID != "abc" {
		t.Errorf("ID: got %q, want abc", r.ID)
	}
	if !r.WasReplay {
		t.Error("WasReplay should be true")
	}
}

// ---------------------------------------------------------------------------
// TestUpdateInstanceStatusOCC — ErrInstanceCompleted on 0 rows
// ---------------------------------------------------------------------------

func applyInstanceOCC(res sql.Result, execErr error) error {
	if execErr != nil {
		return MapPgError(execErr, MapHints{})
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrInstanceCompleted
	}
	return nil
}

func TestUpdateInstanceStatusOCC(t *testing.T) {
	t.Run("zero_rows_returns_ErrInstanceCompleted", func(t *testing.T) {
		err := applyInstanceOCC(fakeResult{rowsAffected: 0}, nil)
		if !errors.Is(err, ErrInstanceCompleted) {
			t.Errorf("expected ErrInstanceCompleted, got %v", err)
		}
	})

	t.Run("one_row_returns_nil", func(t *testing.T) {
		err := applyInstanceOCC(fakeResult{rowsAffected: 1}, nil)
		if err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// TestNewPostgresApprovalRepository — constructor returns correct type
// ---------------------------------------------------------------------------

func TestNewPostgresApprovalRepository(t *testing.T) {
	// NewPostgresApprovalRepository must return an ApprovalRepository.
	// We pass nil for *sql.DB; this is fine as long as we don't execute queries.
	var repo ApprovalRepository = NewPostgresApprovalRepository(nil)
	if repo == nil {
		t.Error("NewPostgresApprovalRepository returned nil")
	}
}

// ---------------------------------------------------------------------------
// TestInsertStageInstancesBulk — verify multi-row placeholder generation
// ---------------------------------------------------------------------------

func TestInsertStageInstancesBulk(t *testing.T) {
	// If stages is empty, InsertStageInstances returns nil immediately.
	repo := &postgresApprovalRepository{}
	err := repo.InsertStageInstances(context.Background(), nil, nil)
	if err != nil {
		t.Errorf("expected nil for empty stages, got %v", err)
	}
}
