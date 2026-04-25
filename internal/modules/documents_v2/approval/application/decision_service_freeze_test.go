package application

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	docapp "metaldocs/internal/modules/documents_v2/application"
	"metaldocs/internal/modules/documents_v2/approval/repository"
)

type fakeFreezeInvoker struct {
	calls int
	err   error
}

func (f *fakeFreezeInvoker) Freeze(_ context.Context, _ *sql.Tx, _, _ string, _ docapp.ApproverContext) error {
	f.calls++
	return f.err
}

type fakePDFDispatchInvoker struct {
	calls int
	err   error
}

func (f *fakePDFDispatchInvoker) Dispatch(_ context.Context, _, _ string) error {
	f.calls++
	return f.err
}

type freezeDecisionConn struct {
	stageSignoffs []signoffRow
	authzGranted  bool
	authzSet      bool
	areaCode      string
	actorID       string

	documentStatus string
	pendingStatus  *string
	committed      bool
	rolledBack     bool
}

type freezeDecisionStmt struct {
	conn  *freezeDecisionConn
	query string
}

type freezeDecisionNoopResult struct{}
type freezeDecisionEmptyRows struct{}
type freezeDecisionSingleValueRows struct {
	value any
	done  bool
}

func (freezeDecisionNoopResult) LastInsertId() (int64, error) { return 0, nil }
func (freezeDecisionNoopResult) RowsAffected() (int64, error) { return 1, nil }
func (freezeDecisionEmptyRows) Columns() []string             { return nil }
func (freezeDecisionEmptyRows) Close() error                  { return nil }
func (freezeDecisionEmptyRows) Next([]driver.Value) error     { return io.EOF }
func (r *freezeDecisionSingleValueRows) Columns() []string    { return []string{"v"} }
func (r *freezeDecisionSingleValueRows) Close() error         { return nil }
func (r *freezeDecisionSingleValueRows) Next(dest []driver.Value) error {
	if r.done {
		return io.EOF
	}
	r.done = true
	dest[0] = r.value
	return nil
}

func (s *freezeDecisionStmt) Close() error  { return nil }
func (s *freezeDecisionStmt) NumInput() int { return -1 }
func (s *freezeDecisionStmt) Exec(_ []driver.Value) (driver.Result, error) {
	q := strings.ToLower(s.query)
	if strings.Contains(q, "update documents") && strings.Contains(q, "set status") && strings.Contains(q, "'approved'") {
		status := "approved"
		s.conn.pendingStatus = &status
	}
	return freezeDecisionNoopResult{}, nil
}
func (s *freezeDecisionStmt) Query(_ []driver.Value) (driver.Rows, error) {
	q := strings.ToLower(s.query)
	if strings.Contains(q, "from documents") {
		return &freezeDecisionSingleValueRows{value: s.conn.areaCode}, nil
	}
	if strings.Contains(q, "select exists") && strings.Contains(q, "role_capabilities") {
		return &freezeDecisionSingleValueRows{value: s.conn.authzGranted}, nil
	}
	if strings.Contains(q, "current_setting('metaldocs.asserted_caps'") {
		return &freezeDecisionSingleValueRows{value: nil}, nil
	}
	if strings.Contains(q, "current_setting('metaldocs.actor_id'") {
		return &freezeDecisionSingleValueRows{value: s.conn.actorID}, nil
	}
	if strings.Contains(q, "approval_signoffs") && isStageQuery(s.query) {
		return &signoffRows{rows: s.conn.stageSignoffs}, nil
	}
	return freezeDecisionEmptyRows{}, nil
}

func (c *freezeDecisionConn) Prepare(query string) (driver.Stmt, error) {
	return &freezeDecisionStmt{conn: c, query: query}, nil
}
func (c *freezeDecisionConn) Close() error              { return nil }
func (c *freezeDecisionConn) Begin() (driver.Tx, error) { return c, nil }
func (c *freezeDecisionConn) Commit() error {
	c.committed = true
	if c.pendingStatus != nil {
		c.documentStatus = *c.pendingStatus
	}
	c.pendingStatus = nil
	return nil
}
func (c *freezeDecisionConn) Rollback() error {
	c.rolledBack = true
	c.pendingStatus = nil
	return nil
}

type freezeDecisionDriver struct{ conn *freezeDecisionConn }

func (d *freezeDecisionDriver) Open(_ string) (driver.Conn, error) { return d.conn, nil }

var freezeDecisionDBCounter int

func newFreezeDecisionTestDB(t *testing.T, conn *freezeDecisionConn) *sql.DB {
	t.Helper()
	if conn.areaCode == "" {
		conn.areaCode = "QA"
	}
	if conn.actorID == "" {
		conn.actorID = "approver-1"
	}
	if !conn.authzSet {
		conn.authzGranted = true
	}
	if conn.documentStatus == "" {
		conn.documentStatus = "under_review"
	}
	freezeDecisionDBCounter++
	name := fmt.Sprintf("decision_freeze_test_%d", freezeDecisionDBCounter)
	sql.Register(name, &freezeDecisionDriver{conn: conn})
	db, err := sql.Open(name, "")
	if err != nil {
		t.Fatalf("open decision freeze test db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestRecordSignoff_QuorumApproved_CallsFreezeAndApprovesDocument(t *testing.T) {
	const (
		instanceID = "inst-freeze-a"
		stageID    = "stage-freeze-a"
		actorID    = "approver-1"
		authorID   = "author-1"
	)
	signedAt := time.Date(2026, 4, 23, 12, 0, 0, 0, time.UTC)
	repo := &fakeDecisionRepo{
		instance:         buildSingleStageInstance(instanceID, stageID, authorID, []string{actorID}),
		insertSignoffRes: repository.SignoffInsertResult{ID: "sig-a", WasReplay: false},
	}
	freeze := &fakeFreezeInvoker{}
	pdf := &fakePDFDispatchInvoker{}
	conn := &freezeDecisionConn{
		actorID: actorID,
		stageSignoffs: []signoffRow{{
			id:                 "sig-a",
			approvalInstanceID: instanceID,
			stageInstanceID:    stageID,
			actorUserID:        actorID,
			actorTenantID:      "tenant-1",
			decision:           "approve",
			signedAt:           signedAt,
			signatureMethod:    "password",
			signaturePayload:   []byte(`{}`),
			contentHash:        validContentHash,
		}},
	}
	db := newFreezeDecisionTestDB(t, conn)
	svc := &DecisionService{
		repo:          repo,
		emitter:       &MemoryEmitter{},
		clock:         fixedClock{t: signedAt},
		freezeInvoker: freeze,
		pdfDispatcher: pdf,
	}

	result, err := svc.RecordSignoff(context.Background(), db, SignoffRequest{
		TenantID:         "tenant-1",
		InstanceID:       instanceID,
		StageInstanceID:  stageID,
		ActorUserID:      actorID,
		Decision:         "approve",
		SignatureMethod:  "password",
		SignaturePayload: map[string]any{"hash": "abc"},
		ContentFormData:  map[string]any{"title": "Doc"},
	})
	if err != nil {
		t.Fatalf("RecordSignoff() error = %v", err)
	}
	if !result.InstanceApproved || conn.documentStatus != "approved" {
		t.Fatalf("expected approved document, status=%q result=%+v", conn.documentStatus, result)
	}
	if freeze.calls != 1 {
		t.Fatalf("Freeze should be called once, got %d", freeze.calls)
	}
}

func TestRecordSignoff_FreezeError_RollsBackTransaction(t *testing.T) {
	const (
		instanceID = "inst-freeze-b"
		stageID    = "stage-freeze-b"
		actorID    = "approver-1"
		authorID   = "author-1"
	)
	signedAt := time.Date(2026, 4, 23, 12, 10, 0, 0, time.UTC)
	repo := &fakeDecisionRepo{
		instance:         buildSingleStageInstance(instanceID, stageID, authorID, []string{actorID}),
		insertSignoffRes: repository.SignoffInsertResult{ID: "sig-b", WasReplay: false},
	}
	freeze := &fakeFreezeInvoker{err: errors.New("fanout failed")}
	conn := &freezeDecisionConn{
		actorID: actorID,
		stageSignoffs: []signoffRow{{
			id:                 "sig-b",
			approvalInstanceID: instanceID,
			stageInstanceID:    stageID,
			actorUserID:        actorID,
			actorTenantID:      "tenant-1",
			decision:           "approve",
			signedAt:           signedAt,
			signatureMethod:    "password",
			signaturePayload:   []byte(`{}`),
			contentHash:        validContentHash,
		}},
	}
	db := newFreezeDecisionTestDB(t, conn)
	svc := &DecisionService{
		repo:          repo,
		emitter:       &MemoryEmitter{},
		clock:         fixedClock{t: signedAt},
		freezeInvoker: freeze,
	}

	_, err := svc.RecordSignoff(context.Background(), db, SignoffRequest{
		TenantID:         "tenant-1",
		InstanceID:       instanceID,
		StageInstanceID:  stageID,
		ActorUserID:      actorID,
		Decision:         "approve",
		SignatureMethod:  "password",
		SignaturePayload: map[string]any{"hash": "abc"},
		ContentFormData:  map[string]any{"title": "Doc"},
	})
	if err == nil {
		t.Fatal("expected freeze error")
	}
	if conn.committed {
		t.Fatal("transaction should not commit on freeze error")
	}
	if !conn.rolledBack {
		t.Fatal("transaction should roll back on freeze error")
	}
	if conn.documentStatus != "under_review" {
		t.Fatalf("document status should stay under_review, got %q", conn.documentStatus)
	}
}

func TestRecordSignoff_PDFDispatchError_IsBestEffort(t *testing.T) {
	const (
		instanceID = "inst-freeze-c"
		stageID    = "stage-freeze-c"
		actorID    = "approver-1"
		authorID   = "author-1"
	)
	signedAt := time.Date(2026, 4, 23, 12, 20, 0, 0, time.UTC)
	repo := &fakeDecisionRepo{
		instance:         buildSingleStageInstance(instanceID, stageID, authorID, []string{actorID}),
		insertSignoffRes: repository.SignoffInsertResult{ID: "sig-c", WasReplay: false},
	}
	freeze := &fakeFreezeInvoker{}
	pdf := &fakePDFDispatchInvoker{err: errors.New("transient queue error")}
	conn := &freezeDecisionConn{
		actorID: actorID,
		stageSignoffs: []signoffRow{{
			id:                 "sig-c",
			approvalInstanceID: instanceID,
			stageInstanceID:    stageID,
			actorUserID:        actorID,
			actorTenantID:      "tenant-1",
			decision:           "approve",
			signedAt:           signedAt,
			signatureMethod:    "password",
			signaturePayload:   []byte(`{}`),
			contentHash:        validContentHash,
		}},
	}
	db := newFreezeDecisionTestDB(t, conn)
	svc := &DecisionService{
		repo:          repo,
		emitter:       &MemoryEmitter{},
		clock:         fixedClock{t: signedAt},
		freezeInvoker: freeze,
		pdfDispatcher: pdf,
	}

	result, err := svc.RecordSignoff(context.Background(), db, SignoffRequest{
		TenantID:         "tenant-1",
		InstanceID:       instanceID,
		StageInstanceID:  stageID,
		ActorUserID:      actorID,
		Decision:         "approve",
		SignatureMethod:  "password",
		SignaturePayload: map[string]any{"hash": "abc"},
		ContentFormData:  map[string]any{"title": "Doc"},
	})
	if err != nil {
		t.Fatalf("RecordSignoff() error = %v", err)
	}
	if !result.InstanceApproved || conn.documentStatus != "approved" {
		t.Fatalf("expected approved document despite PDF error, status=%q result=%+v", conn.documentStatus, result)
	}
	if pdf.calls != 1 {
		t.Fatalf("PDF dispatch should be attempted once, got %d", pdf.calls)
	}
}

func TestRecordSignoff_WasReplay_DoesNotCallFreeze(t *testing.T) {
	const (
		instanceID = "inst-freeze-d"
		stageID    = "stage-freeze-d"
		actorID    = "approver-1"
		authorID   = "author-1"
	)
	signedAt := time.Date(2026, 4, 23, 12, 30, 0, 0, time.UTC)
	repo := &fakeDecisionRepo{
		instance:         buildSingleStageInstance(instanceID, stageID, authorID, []string{actorID}),
		insertSignoffRes: repository.SignoffInsertResult{ID: "sig-d", WasReplay: true},
	}
	freeze := &fakeFreezeInvoker{}
	conn := &freezeDecisionConn{actorID: actorID}
	db := newFreezeDecisionTestDB(t, conn)
	svc := &DecisionService{
		repo:          repo,
		emitter:       &MemoryEmitter{},
		clock:         fixedClock{t: signedAt},
		freezeInvoker: freeze,
	}

	_, err := svc.RecordSignoff(context.Background(), db, SignoffRequest{
		TenantID:        "tenant-1",
		InstanceID:      instanceID,
		StageInstanceID: stageID,
		ActorUserID:     actorID,
		Decision:        "approve",
		ContentFormData: map[string]any{"title": "Doc"},
	})
	if err != nil {
		t.Fatalf("RecordSignoff() error = %v", err)
	}
	if freeze.calls != 0 {
		t.Fatalf("Freeze must not run on replay, got %d call(s)", freeze.calls)
	}
}
