package application

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"testing"
	"time"

	"metaldocs/internal/modules/documents_v2/approval/domain"
	"metaldocs/internal/modules/documents_v2/approval/repository"
)

// ---------------------------------------------------------------------------
// Fake repo — only the methods called by RecordSignoff.
// ---------------------------------------------------------------------------

type fakeDecisionRepo struct {
	// Embed no-op to satisfy interface; listed methods are real overrides.
	repository.ApprovalRepository

	instance          *domain.Instance
	loadInstanceErr   error
	insertSignoffRes  repository.SignoffInsertResult
	insertSignoffErr  error
	updateStageErr    error
	updateInstanceErr error
}

func (r *fakeDecisionRepo) LoadInstance(_ context.Context, _ *sql.Tx, _, _ string) (*domain.Instance, error) {
	return r.instance, r.loadInstanceErr
}

func (r *fakeDecisionRepo) InsertSignoff(_ context.Context, _ *sql.Tx, _ domain.Signoff) (repository.SignoffInsertResult, error) {
	return r.insertSignoffRes, r.insertSignoffErr
}

func (r *fakeDecisionRepo) UpdateStageStatus(_ context.Context, _ *sql.Tx, _, _ string, _, _ domain.StageStatus) error {
	return r.updateStageErr
}

func (r *fakeDecisionRepo) UpdateInstanceStatus(_ context.Context, _ *sql.Tx, _, _ string, _ domain.InstanceStatus, _ domain.InstanceStatus, _ *time.Time) error {
	return r.updateInstanceErr
}

// ---------------------------------------------------------------------------
// Minimal in-memory SQL driver for DecisionService tests.
//
// RecordSignoff calls db.BeginTx and then tx.QueryContext for:
//   1. loadPriorSignoffs   → approval_signoffs WHERE stage_instance_id != active
//   2. loadStageSignoffs   → approval_signoffs WHERE stage_instance_id = active
//
// decisionTestConn is configurable: stageSignoffs contains the rows that
// loadStageSignoffs should return (simulating the just-inserted signoff being
// visible within the same transaction).
// ---------------------------------------------------------------------------

// signoffRow holds the raw column values for one approval_signoffs row.
type signoffRow struct {
	id                 string
	approvalInstanceID string
	stageInstanceID    string
	actorUserID        string
	actorTenantID      string
	decision           string
	comment            string
	signedAt           time.Time
	signatureMethod    string
	signaturePayload   []byte
	contentHash        string
}

type decisionTestConn struct {
	stageSignoffs []signoffRow // rows returned by loadStageSignoffs
}

type decisionNoopResult struct{}
type decisionEmptyRows struct{}

func (decisionNoopResult) LastInsertId() (int64, error) { return 0, nil }
func (decisionNoopResult) RowsAffected() (int64, error) { return 1, nil }
func (decisionEmptyRows) Columns() []string              { return nil }
func (decisionEmptyRows) Close() error                   { return nil }
func (decisionEmptyRows) Next([]driver.Value) error      { return io.EOF }

// signoffRows returns configured signoff rows.
type signoffRows struct {
	rows []signoffRow
	idx  int
}

func (r *signoffRows) Columns() []string {
	return []string{
		"id", "approval_instance_id", "stage_instance_id",
		"actor_user_id", "actor_tenant_id", "decision",
		"comment", "signed_at", "signature_method", "signature_payload", "content_hash",
	}
}
func (r *signoffRows) Close() error { return nil }
func (r *signoffRows) Next(dest []driver.Value) error {
	if r.idx >= len(r.rows) {
		return io.EOF
	}
	row := r.rows[r.idx]
	r.idx++
	dest[0] = row.id
	dest[1] = row.approvalInstanceID
	dest[2] = row.stageInstanceID
	dest[3] = row.actorUserID
	dest[4] = row.actorTenantID
	dest[5] = row.decision
	dest[6] = row.comment
	dest[7] = row.signedAt
	dest[8] = row.signatureMethod
	dest[9] = row.signaturePayload
	dest[10] = row.contentHash
	return nil
}

type decisionTestStmt struct {
	conn  *decisionTestConn
	query string
}

func (s *decisionTestStmt) Close() error  { return nil }
func (s *decisionTestStmt) NumInput() int { return -1 }
func (s *decisionTestStmt) Exec(_ []driver.Value) (driver.Result, error) {
	return decisionNoopResult{}, nil
}
func (s *decisionTestStmt) Query(_ []driver.Value) (driver.Rows, error) {
	// loadStageSignoffs queries WHERE stage_instance_id = $1 (no "!=")
	// loadPriorSignoffs queries WHERE stage_instance_id != $2
	// Both hit "approval_signoffs" table.
	// We return configured stageSignoffs for the stage query and empty for prior.
	if isStageQuery(s.query) {
		return &signoffRows{rows: s.conn.stageSignoffs}, nil
	}
	return decisionEmptyRows{}, nil
}

// isStageQuery returns true when the query is for a single stage (no "!=" exclusion).
func isStageQuery(q string) bool {
	// loadStageSignoffs: "WHERE stage_instance_id = $1" — no "!="
	// loadPriorSignoffs: "WHERE ... AND stage_instance_id != $2"
	for i := 0; i < len(q)-1; i++ {
		if q[i] == '!' && q[i+1] == '=' {
			return false
		}
	}
	return true
}

func (c *decisionTestConn) Prepare(query string) (driver.Stmt, error) {
	return &decisionTestStmt{conn: c, query: query}, nil
}
func (c *decisionTestConn) Close() error              { return nil }
func (c *decisionTestConn) Begin() (driver.Tx, error) { return c, nil }
func (c *decisionTestConn) Commit() error             { return nil }
func (c *decisionTestConn) Rollback() error           { return nil }

type decisionTestDriver struct{ conn *decisionTestConn }

func (d *decisionTestDriver) Open(_ string) (driver.Conn, error) { return d.conn, nil }

var decisionDBCounter int

func newDecisionTestDB(t *testing.T, conn *decisionTestConn) *sql.DB {
	t.Helper()
	decisionDBCounter++
	name := fmt.Sprintf("decision_test_%d", decisionDBCounter)
	sql.Register(name, &decisionTestDriver{conn: conn})
	db, err := sql.Open(name, "")
	if err != nil {
		t.Fatalf("open decision test db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// buildSingleStageInstance returns an Instance with one active stage using
// any_1_of quorum and the given eligible actors.
func buildSingleStageInstance(instanceID, stageID, authorUserID string, eligible []string) *domain.Instance {
	now := time.Now().UTC()
	return &domain.Instance{
		ID:              instanceID,
		TenantID:        "tenant-1",
		DocumentID:      "doc-1",
		RouteID:         "route-1",
		Status:          domain.InstanceInProgress,
		SubmittedBy:     authorUserID,
		SubmittedAt:     now,
		RevisionVersion: 1,
		Stages: []domain.StageInstance{
			{
				ID:                         stageID,
				ApprovalInstanceID:         instanceID,
				StageOrder:                 1,
				NameSnapshot:               "QA Review",
				QuorumSnapshot:             domain.QuorumAny1Of,
				OnEligibilityDriftSnapshot: domain.DriftKeepSnapshot,
				EligibleActorIDs:           eligible,
				Status:                     domain.StageActive,
				OpenedAt:                   &now,
			},
		},
	}
}

// buildTwoApproverInstance returns an Instance with one active stage using
// all_of quorum and two eligible actors.
func buildTwoApproverInstance(instanceID, stageID, authorUserID string, eligible []string) *domain.Instance {
	inst := buildSingleStageInstance(instanceID, stageID, authorUserID, eligible)
	inst.Stages[0].QuorumSnapshot = domain.QuorumAllOf
	return inst
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

// validContentHash is a 64-char lowercase hex string used in test signoff rows.
const validContentHash = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

// TestRecordSignoff_ApprovePath_QuorumMet: single approver quorum (any_1_of).
// Expect StageCompleted=true, InstanceApproved=true (only 1 stage).
func TestRecordSignoff_ApprovePath_QuorumMet(t *testing.T) {
	const (
		instanceID = "inst-1"
		stageID    = "stage-1"
		actorID    = "approver-1"
		authorID   = "author-1"
	)

	inst := buildSingleStageInstance(instanceID, stageID, authorID, []string{actorID})

	// Simulate the just-inserted signoff being visible in loadStageSignoffs.
	signedAt := time.Date(2026, 4, 22, 12, 0, 0, 0, time.UTC)
	stageSignoffs := []signoffRow{
		{
			id:                 "signoff-1",
			approvalInstanceID: instanceID,
			stageInstanceID:    stageID,
			actorUserID:        actorID,
			actorTenantID:      "tenant-1",
			decision:           "approve",
			comment:            "LGTM",
			signedAt:           signedAt,
			signatureMethod:    "password",
			signaturePayload:   []byte(`{}`),
			contentHash:        validContentHash,
		},
	}

	conn := &decisionTestConn{stageSignoffs: stageSignoffs}
	repo := &fakeDecisionRepo{
		instance:         inst,
		insertSignoffRes: repository.SignoffInsertResult{ID: "signoff-1", WasReplay: false},
	}
	emitter := &MemoryEmitter{}
	clock := fixedClock{t: signedAt}
	svc := &DecisionService{repo: repo, emitter: emitter, clock: clock}
	db := newDecisionTestDB(t, conn)

	req := SignoffRequest{
		TenantID:        "tenant-1",
		InstanceID:      instanceID,
		StageInstanceID: stageID,
		ActorUserID:     actorID,
		Decision:        "approve",
		Comment:         "LGTM",
		SignatureMethod:  "password",
		SignaturePayload: map[string]any{"hash": "abc"},
		ContentFormData: map[string]any{"title": "Doc"},
	}

	result, err := svc.RecordSignoff(context.Background(), db, req)
	if err != nil {
		t.Fatalf("RecordSignoff: unexpected error: %v", err)
	}
	if !result.StageCompleted {
		t.Error("expected StageCompleted=true")
	}
	if !result.InstanceApproved {
		t.Error("expected InstanceApproved=true (single-stage instance)")
	}
	if result.InstanceRejected {
		t.Error("expected InstanceRejected=false")
	}
	if len(emitter.Events) != 1 {
		t.Errorf("expected 1 governance event; got %d", len(emitter.Events))
	}
	if emitter.Events[0].EventType != "signoff_recorded" {
		t.Errorf("event type = %q; want %q", emitter.Events[0].EventType, "signoff_recorded")
	}
}

// TestRecordSignoff_ApprovePath_QuorumNotYetMet: all_of quorum with two eligible actors.
// First signoff (only one of two) should leave StageCompleted=false.
func TestRecordSignoff_ApprovePath_QuorumNotYetMet(t *testing.T) {
	const (
		instanceID = "inst-2"
		stageID    = "stage-2"
		actorID    = "approver-1"
		authorID   = "author-2"
	)

	eligible := []string{"approver-1", "approver-2"}
	inst := buildTwoApproverInstance(instanceID, stageID, authorID, eligible)

	signedAt := time.Date(2026, 4, 22, 12, 0, 0, 0, time.UTC)
	// Only one signoff visible — quorum all_of needs 2.
	stageSignoffs := []signoffRow{
		{
			id:                 "signoff-2",
			approvalInstanceID: instanceID,
			stageInstanceID:    stageID,
			actorUserID:        actorID,
			actorTenantID:      "tenant-1",
			decision:           "approve",
			signedAt:           signedAt,
			signatureMethod:    "password",
			signaturePayload:   []byte(`{}`),
			contentHash:        validContentHash,
		},
	}

	conn := &decisionTestConn{stageSignoffs: stageSignoffs}
	repo := &fakeDecisionRepo{
		instance:         inst,
		insertSignoffRes: repository.SignoffInsertResult{ID: "signoff-2", WasReplay: false},
	}
	emitter := &MemoryEmitter{}
	clock := fixedClock{t: signedAt}
	svc := &DecisionService{repo: repo, emitter: emitter, clock: clock}
	db := newDecisionTestDB(t, conn)

	req := SignoffRequest{
		TenantID:        "tenant-1",
		InstanceID:      instanceID,
		StageInstanceID: stageID,
		ActorUserID:     actorID,
		Decision:        "approve",
		ContentFormData: map[string]any{"title": "Doc"},
	}

	result, err := svc.RecordSignoff(context.Background(), db, req)
	if err != nil {
		t.Fatalf("RecordSignoff: unexpected error: %v", err)
	}
	if result.StageCompleted {
		t.Error("expected StageCompleted=false — quorum not yet met (1 of 2 approved)")
	}
	if result.InstanceApproved {
		t.Error("expected InstanceApproved=false")
	}
}

// TestRecordSignoff_SoDViolation: actor == document author → ErrAuthorCannotSign.
func TestRecordSignoff_SoDViolation(t *testing.T) {
	const (
		instanceID = "inst-3"
		stageID    = "stage-3"
		authorID   = "author-and-actor" // same person!
	)

	inst := buildSingleStageInstance(instanceID, stageID, authorID, []string{authorID})

	conn := &decisionTestConn{} // empty — SoD check fires before any SQL read
	repo := &fakeDecisionRepo{instance: inst}
	emitter := &MemoryEmitter{}
	clock := fixedClock{t: time.Date(2026, 4, 22, 12, 0, 0, 0, time.UTC)}
	svc := &DecisionService{repo: repo, emitter: emitter, clock: clock}
	db := newDecisionTestDB(t, conn)

	req := SignoffRequest{
		TenantID:        "tenant-1",
		InstanceID:      instanceID,
		StageInstanceID: stageID,
		ActorUserID:     authorID, // same as SubmittedBy
		Decision:        "approve",
		ContentFormData: map[string]any{"title": "Doc"},
	}

	_, err := svc.RecordSignoff(context.Background(), db, req)
	if err == nil {
		t.Fatal("expected error for SoD violation; got nil")
	}
	if !errors.Is(err, domain.ErrAuthorCannotSign) {
		t.Errorf("expected domain.ErrAuthorCannotSign; got %v", err)
	}
	if len(emitter.Events) != 0 {
		t.Errorf("no governance event should be emitted on SoD violation; got %d", len(emitter.Events))
	}
}
