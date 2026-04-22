package application

// phase5_integration_test.go — cross-service scenario tests for Spec 2 Phase 5.
//
// These are NOT database integration tests: no real Postgres is required.
// They wire Submit, Decision, Publish, and Scheduler services together using
// the same fake-driver pattern as the per-service tests in this package, then
// chain real service method calls to prove end-to-end service wiring works.
//
// Three scenarios:
//   1. FullApprovalAndPublish   — Submit → approve signoff → PublishApproved
//   2. RejectThenResubmit       — Submit → reject signoff → Submit again (new instance)
//   3. ScheduleAndRunScheduler  — SchedulePublish → RunDuePublishes (clock advanced)

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"metaldocs/internal/modules/documents_v2/approval/domain"
	"metaldocs/internal/modules/documents_v2/approval/repository"
)

// ---------------------------------------------------------------------------
// Phase 5 combined fake repo
//
// Satisfies repository.ApprovalRepository for the three scenarios.
// Each service only calls the methods it needs; all others forward to the
// embedded no-op and would panic if unexpectedly called.
// ---------------------------------------------------------------------------

type phase5Repo struct {
	repository.ApprovalRepository // no-op embed

	// Submit methods
	insertInstanceErr       error
	insertStageInstancesErr error

	// Decision methods
	instance          *domain.Instance
	loadInstanceErr   error
	insertSignoffRes  repository.SignoffInsertResult
	insertSignoffErr  error
	updateStageErr    error
	updateInstanceErr error

	// Scheduler method
	scheduledRows []repository.ScheduledPublishRow
	listDueErr    error
}

func (r *phase5Repo) InsertInstance(_ context.Context, _ *sql.Tx, _ domain.Instance) error {
	return r.insertInstanceErr
}

func (r *phase5Repo) InsertStageInstances(_ context.Context, _ *sql.Tx, _ []domain.StageInstance) error {
	return r.insertStageInstancesErr
}

func (r *phase5Repo) LoadInstance(_ context.Context, _ *sql.Tx, _, _ string) (*domain.Instance, error) {
	return r.instance, r.loadInstanceErr
}

func (r *phase5Repo) InsertSignoff(_ context.Context, _ *sql.Tx, _ domain.Signoff) (repository.SignoffInsertResult, error) {
	return r.insertSignoffRes, r.insertSignoffErr
}

func (r *phase5Repo) UpdateStageStatus(_ context.Context, _ *sql.Tx, _, _ string, _, _ domain.StageStatus) error {
	return r.updateStageErr
}

func (r *phase5Repo) UpdateInstanceStatus(_ context.Context, _ *sql.Tx, _, _ string, _ domain.InstanceStatus, _ domain.InstanceStatus, _ *time.Time) error {
	return r.updateInstanceErr
}

func (r *phase5Repo) ListScheduledDue(_ context.Context, _ *sql.Tx, _ time.Time, _ int) ([]repository.ScheduledPublishRow, error) {
	return r.scheduledRows, r.listDueErr
}

// ---------------------------------------------------------------------------
// Phase 5 combined fake SQL driver
//
// Handles all four service shapes in a single conn:
//   • Submit:    route + stage queries (SELECT)
//   • Decision:  signoff queries (SELECT with/without "!=")
//   • Publish:   UPDATE documents
//   • Scheduler: UPDATE documents (with configurable rowsAffected)
//
// updateResults is consumed left-to-right; one value per UPDATE Exec call.
// stageSignoffs feeds the decision service's loadStageSignoffs query.
// ---------------------------------------------------------------------------

type phase5Conn struct {
	stageSignoffs []signoffRow // reuse decisionTestConn's signoffRow type
	updateResults []int64
	updateIdx     int32 // atomic
}

func (c *phase5Conn) nextRowsAffected() int64 {
	idx := atomic.AddInt32(&c.updateIdx, 1) - 1
	if int(idx) >= len(c.updateResults) {
		return 1 // default: success
	}
	return c.updateResults[idx]
}

type phase5NoopResult struct{ ra int64 }

func (r phase5NoopResult) LastInsertId() (int64, error) { return 0, nil }
func (r phase5NoopResult) RowsAffected() (int64, error) { return r.ra, nil }

type phase5EmptyRows struct{}

func (phase5EmptyRows) Columns() []string         { return nil }
func (phase5EmptyRows) Close() error              { return nil }
func (phase5EmptyRows) Next([]driver.Value) error { return io.EOF }

type phase5Stmt struct {
	conn  *phase5Conn
	query string
}

func (s *phase5Stmt) Close() error  { return nil }
func (s *phase5Stmt) NumInput() int { return -1 }

func (s *phase5Stmt) Exec(_ []driver.Value) (driver.Result, error) {
	q := strings.ToLower(s.query)
	if strings.Contains(q, "update") {
		return phase5NoopResult{ra: s.conn.nextRowsAffected()}, nil
	}
	return phase5NoopResult{ra: 1}, nil
}

func (s *phase5Stmt) Query(_ []driver.Value) (driver.Rows, error) {
	q := strings.ToLower(s.query)

	// Submit: approval_routes SELECT
	if strings.Contains(q, "approval_routes") && strings.Contains(q, "where") {
		return &routeRows{}, nil // reuse submit_service_test.go's routeRows
	}
	// Submit: approval_route_stages SELECT
	if strings.Contains(q, "approval_route_stages") {
		return &stageRows{}, nil // reuse submit_service_test.go's stageRows
	}
	// Decision: approval_signoffs SELECT
	if strings.Contains(q, "approval_signoffs") {
		if isStageQuery(s.query) {
			return &signoffRows{rows: s.conn.stageSignoffs}, nil
		}
		return phase5EmptyRows{}, nil // prior-signoffs query (empty)
	}

	return phase5EmptyRows{}, nil
}

func (c *phase5Conn) Prepare(query string) (driver.Stmt, error) {
	return &phase5Stmt{conn: c, query: query}, nil
}

func (c *phase5Conn) Close() error              { return nil }
func (c *phase5Conn) Begin() (driver.Tx, error) { return c, nil }

// BeginTx honours non-default isolation levels (used by SchedulerService fetch tx).
func (c *phase5Conn) BeginTx(_ context.Context, _ driver.TxOptions) (driver.Tx, error) {
	return c, nil
}

func (c *phase5Conn) Commit() error   { return nil }
func (c *phase5Conn) Rollback() error { return nil }

type phase5Driver struct{ conn *phase5Conn }

func (d *phase5Driver) Open(_ string) (driver.Conn, error) { return d.conn, nil }

// newPhase5DB registers a unique fake driver and opens a *sql.DB against it.
func newPhase5DB(t *testing.T, conn *phase5Conn) *sql.DB {
	t.Helper()
	name := fmt.Sprintf("phase5_test_%p", conn)
	sql.Register(name, &phase5Driver{conn: conn})
	db, err := sql.Open(name, "")
	if err != nil {
		t.Fatalf("open phase5 test db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// ---------------------------------------------------------------------------
// Scenario 1: FullApprovalAndPublish
//
// Flow: Submit → RecordSignoff (approve, quorum met) → PublishApproved
//
// After RecordSignoff we expect InstanceApproved=true.
// After PublishApproved we expect NewStatus="published" and a
// "document_published" governance event in the emitter.
//
// Each service call gets its own repo/DB pair because:
//   • RecordSignoff requires instance.Status == InProgress
//   • PublishApproved requires instance.Status == Approved
// The shared emitter accumulates events across all three phases.
// ---------------------------------------------------------------------------

func TestPhase5_FullApprovalAndPublish(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 4, 22, 10, 0, 0, 0, time.UTC)

	const (
		tenantID   = "tenant-p5-1"
		documentID = "doc-p5-1"
		routeID    = "route-uuid-1" // matches routeRows fixture
		actorID    = "approver-p5-1"
		authorID   = "author-p5-1"
		instanceID = "inst-p5-1"
		stageID    = "stage-p5-1"
	)

	emitter := &MemoryEmitter{}
	clock := fixedClock{t: now}

	// --- Step 1: Submit ---
	// Submit only needs InsertInstance + InsertStageInstances from the repo.
	submitRepo := &phase5Repo{}
	submitConn := &phase5Conn{}
	submitDB := newPhase5DB(t, submitConn)

	submitSvc := &SubmitService{repo: submitRepo, emitter: emitter, clock: clock}

	submitReq := SubmitRequest{
		TenantID:        tenantID,
		DocumentID:      documentID,
		RouteID:         routeID,
		SubmittedBy:     authorID,
		ContentFormData: map[string]any{"title": "P5 Doc"},
		RevisionVersion: 1,
	}
	submitResult, err := submitSvc.SubmitRevisionForReview(ctx, submitDB, submitReq)
	if err != nil {
		t.Fatalf("Submit: unexpected error: %v", err)
	}
	if submitResult.InstanceID == "" {
		t.Fatal("Submit: InstanceID must not be empty")
	}
	if len(emitter.Events) != 1 {
		t.Fatalf("after Submit: want 1 event; got %d", len(emitter.Events))
	}
	if emitter.Events[0].EventType != "approval_submitted" {
		t.Errorf("event[0].EventType = %q; want approval_submitted", emitter.Events[0].EventType)
	}

	// --- Step 2: RecordSignoff (approve, quorum met) ---
	// Instance must be InProgress for RecordSignoff to accept it.
	inProgressInstance := &domain.Instance{
		ID:              instanceID,
		TenantID:        tenantID,
		DocumentID:      documentID,
		RouteID:         routeID,
		Status:          domain.InstanceInProgress,
		SubmittedBy:     authorID,
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
				EligibleActorIDs:           []string{actorID},
				Status:                     domain.StageActive,
				OpenedAt:                   &now,
			},
		},
	}

	// signoff row visible in loadStageSignoffs — quorum met immediately (any_1_of).
	decisionStageSignoffs := []signoffRow{
		{
			id:                 "signoff-p5-1",
			approvalInstanceID: instanceID,
			stageInstanceID:    stageID,
			actorUserID:        actorID,
			actorTenantID:      tenantID,
			decision:           "approve",
			comment:            "LGTM",
			signedAt:           now,
			signatureMethod:    "password",
			signaturePayload:   []byte(`{}`),
			contentHash:        validContentHash,
		},
	}

	decisionRepo := &phase5Repo{
		instance:         inProgressInstance,
		insertSignoffRes: repository.SignoffInsertResult{ID: "signoff-p5-1", WasReplay: false},
	}
	decisionConn := &phase5Conn{stageSignoffs: decisionStageSignoffs}
	decisionDB := newPhase5DB(t, decisionConn)
	decisionSvc := &DecisionService{repo: decisionRepo, emitter: emitter, clock: clock}

	signoffReq := SignoffRequest{
		TenantID:        tenantID,
		InstanceID:      instanceID,
		StageInstanceID: stageID,
		ActorUserID:     actorID,
		Decision:        "approve",
		Comment:         "LGTM",
		SignatureMethod:  "password",
		SignaturePayload: map[string]any{"hash": "abc"},
		ContentFormData: map[string]any{"title": "P5 Doc"},
	}
	signoffResult, err := decisionSvc.RecordSignoff(ctx, decisionDB, signoffReq)
	if err != nil {
		t.Fatalf("RecordSignoff: unexpected error: %v", err)
	}
	if !signoffResult.InstanceApproved {
		t.Error("after signoff: want InstanceApproved=true")
	}
	if signoffResult.InstanceRejected {
		t.Error("after signoff: want InstanceRejected=false")
	}
	if len(emitter.Events) != 2 {
		t.Fatalf("after RecordSignoff: want 2 events; got %d", len(emitter.Events))
	}
	if emitter.Events[1].EventType != "signoff_recorded" {
		t.Errorf("event[1].EventType = %q; want signoff_recorded", emitter.Events[1].EventType)
	}

	// --- Step 3: PublishApproved ---
	// LoadInstance must now return an Approved instance (post-signoff DB state).
	approvedInstance := &domain.Instance{
		ID:              instanceID,
		TenantID:        tenantID,
		DocumentID:      documentID,
		Status:          domain.InstanceApproved,
		RevisionVersion: 1,
	}
	publishRepo := &phase5Repo{instance: approvedInstance}
	// UPDATE documents returns rowsAffected=1.
	publishConn := &phase5Conn{updateResults: []int64{1}}
	publishDB := newPhase5DB(t, publishConn)
	publishSvc := &PublishService{repo: publishRepo, emitter: emitter, clock: clock}

	publishReq := PublishRequest{
		TenantID:    tenantID,
		InstanceID:  instanceID,
		PublishedBy: actorID,
	}
	publishResult, err := publishSvc.PublishApproved(ctx, publishDB, publishReq)
	if err != nil {
		t.Fatalf("PublishApproved: unexpected error: %v", err)
	}
	if publishResult.NewStatus != "published" {
		t.Errorf("PublishApproved.NewStatus = %q; want published", publishResult.NewStatus)
	}
	if publishResult.DocumentID != documentID {
		t.Errorf("PublishApproved.DocumentID = %q; want %q", publishResult.DocumentID, documentID)
	}
	// Total events: submit + signoff + publish = 3.
	if len(emitter.Events) != 3 {
		t.Fatalf("after PublishApproved: want 3 events; got %d", len(emitter.Events))
	}
	if emitter.Events[2].EventType != "document_published" {
		t.Errorf("event[2].EventType = %q; want document_published", emitter.Events[2].EventType)
	}
}

// ---------------------------------------------------------------------------
// Scenario 2: RejectThenResubmit
//
// Flow: Submit → RecordSignoff (reject) → verify InstanceRejected=true
//       → Submit again → verify new InstanceID returned
//
// Two separate Submit calls each get their own DB (distinct fake instances)
// to isolate idempotency keys between submissions.
// ---------------------------------------------------------------------------

func TestPhase5_RejectThenResubmit(t *testing.T) {
	ctx := context.Background()

	// Two distinct clock values so the two submissions produce different idempotency keys.
	clockAtSubmit1 := fixedClock{t: time.Date(2026, 4, 22, 11, 0, 0, 0, time.UTC)}
	clockAtSignoff := clockAtSubmit1
	clockAtSubmit2 := fixedClock{t: time.Date(2026, 4, 22, 12, 0, 0, 0, time.UTC)}

	const (
		tenantID   = "tenant-p5-rej"
		documentID = "doc-p5-rej"
		routeID    = "route-uuid-1"
		actorID    = "approver-p5-rej"
		authorID   = "author-p5-rej"
		instanceID = "inst-p5-rej"
		stageID    = "stage-p5-rej"
	)

	// Instance in in-progress state for the signoff.
	inProgressInstance := &domain.Instance{
		ID:              instanceID,
		TenantID:        tenantID,
		DocumentID:      documentID,
		RouteID:         routeID,
		Status:          domain.InstanceInProgress,
		SubmittedBy:     authorID,
		SubmittedAt:     clockAtSubmit1.t,
		RevisionVersion: 1,
		Stages: []domain.StageInstance{
			{
				ID:                         stageID,
				ApprovalInstanceID:         instanceID,
				StageOrder:                 1,
				NameSnapshot:               "QA Review",
				QuorumSnapshot:             domain.QuorumAny1Of,
				OnEligibilityDriftSnapshot: domain.DriftKeepSnapshot,
				EligibleActorIDs:           []string{actorID},
				Status:                     domain.StageActive,
				OpenedAt:                   &clockAtSubmit1.t,
			},
		},
	}

	// Reject signoff row returned by loadStageSignoffs.
	rejectSignoffRows := []signoffRow{
		{
			id:                 "signoff-rej-1",
			approvalInstanceID: instanceID,
			stageInstanceID:    stageID,
			actorUserID:        actorID,
			actorTenantID:      tenantID,
			decision:           "reject",
			comment:            "Not ready",
			signedAt:           clockAtSignoff.t,
			signatureMethod:    "password",
			signaturePayload:   []byte(`{}`),
			contentHash:        validContentHash,
		},
	}

	// --- Phase A: Submit #1 + reject signoff ---
	repo := &phase5Repo{
		instance:         inProgressInstance,
		insertSignoffRes: repository.SignoffInsertResult{ID: "signoff-rej-1", WasReplay: false},
	}
	emitter := &MemoryEmitter{}

	// Submit #1 DB (route + stage queries only).
	conn1 := &phase5Conn{}
	db1 := newPhase5DB(t, conn1)

	submitSvc1 := &SubmitService{repo: repo, emitter: emitter, clock: clockAtSubmit1}

	submitReq1 := SubmitRequest{
		TenantID:        tenantID,
		DocumentID:      documentID,
		RouteID:         routeID,
		SubmittedBy:     authorID,
		ContentFormData: map[string]any{"title": "Reject Doc v1"},
		RevisionVersion: 1,
	}
	submitResult1, err := submitSvc1.SubmitRevisionForReview(ctx, db1, submitReq1)
	if err != nil {
		t.Fatalf("Submit #1: unexpected error: %v", err)
	}
	if submitResult1.InstanceID == "" {
		t.Fatal("Submit #1: InstanceID must not be empty")
	}

	// Decision DB — needs approval_signoffs rows.
	connDecision := &phase5Conn{stageSignoffs: rejectSignoffRows}
	dbDecision := newPhase5DB(t, connDecision)

	decisionSvc := &DecisionService{repo: repo, emitter: emitter, clock: clockAtSignoff}

	signoffReq := SignoffRequest{
		TenantID:        tenantID,
		InstanceID:      instanceID,
		StageInstanceID: stageID,
		ActorUserID:     actorID,
		Decision:        "reject",
		Comment:         "Not ready",
		SignatureMethod:  "password",
		SignaturePayload: map[string]any{"hash": "rej"},
		ContentFormData: map[string]any{"title": "Reject Doc v1"},
	}
	signoffResult, err := decisionSvc.RecordSignoff(ctx, dbDecision, signoffReq)
	if err != nil {
		t.Fatalf("RecordSignoff (reject): unexpected error: %v", err)
	}
	if !signoffResult.InstanceRejected {
		t.Error("after reject signoff: want InstanceRejected=true")
	}
	if signoffResult.InstanceApproved {
		t.Error("after reject signoff: want InstanceApproved=false")
	}

	// Verify governance events so far: 1 submit + 1 signoff.
	if len(emitter.Events) != 2 {
		t.Fatalf("after reject signoff: want 2 events; got %d", len(emitter.Events))
	}
	if emitter.Events[1].EventType != "signoff_recorded" {
		t.Errorf("event[1].EventType = %q; want signoff_recorded", emitter.Events[1].EventType)
	}

	// --- Phase B: Submit #2 (resubmission after rejection) ---
	conn2 := &phase5Conn{}
	db2 := newPhase5DB(t, conn2)

	// Fresh submit service with a later clock to guarantee a distinct idempotency key.
	submitSvc2 := &SubmitService{repo: repo, emitter: emitter, clock: clockAtSubmit2}

	submitReq2 := SubmitRequest{
		TenantID:        tenantID,
		DocumentID:      documentID,
		RouteID:         routeID,
		SubmittedBy:     authorID,
		ContentFormData: map[string]any{"title": "Reject Doc v2"},
		RevisionVersion: 2, // bumped revision
	}
	submitResult2, err := submitSvc2.SubmitRevisionForReview(ctx, db2, submitReq2)
	if err != nil {
		t.Fatalf("Submit #2: unexpected error: %v", err)
	}
	if submitResult2.InstanceID == "" {
		t.Fatal("Submit #2: InstanceID must not be empty")
	}

	// The two submissions must produce distinct instance IDs (both are UUIDs generated
	// inside the service; verifying they are non-empty and different is the meaningful
	// assertion — the exact UUIDs are non-deterministic).
	if submitResult1.InstanceID == submitResult2.InstanceID {
		t.Errorf("Submit #1 and #2 must produce distinct InstanceIDs; both returned %q", submitResult1.InstanceID)
	}

	// Total events: 1 (submit1) + 1 (signoff) + 1 (submit2) = 3.
	if len(emitter.Events) != 3 {
		t.Fatalf("after resubmit: want 3 events; got %d", len(emitter.Events))
	}
	if emitter.Events[2].EventType != "approval_submitted" {
		t.Errorf("event[2].EventType = %q; want approval_submitted", emitter.Events[2].EventType)
	}
}

// ---------------------------------------------------------------------------
// Scenario 3: ScheduleAndRunScheduler
//
// Flow: SchedulePublish (future effective_date) → RunDuePublishes (clock advanced past it)
//
// SchedulePublish needs:
//   • clock.Now() strictly before effective_date (guard check)
//   • LoadInstance returns an approved instance
//   • UPDATE documents returns rowsAffected=1
//
// RunDuePublishes needs:
//   • ListScheduledDue returns 1 row (the scheduled doc)
//   • UPDATE documents returns rowsAffected=1
//   • Result.Processed == 1
// ---------------------------------------------------------------------------

func TestPhase5_ScheduleAndRunScheduler(t *testing.T) {
	ctx := context.Background()

	// Clock is set to "before" for SchedulePublish, "after" for RunDuePublishes.
	pastClock := fixedClock{t: time.Date(2026, 4, 22, 9, 0, 0, 0, time.UTC)}
	effectiveDate := time.Date(2026, 4, 23, 0, 0, 0, 0, time.UTC) // 1 day in the future
	futureClock := fixedClock{t: time.Date(2026, 4, 24, 0, 0, 0, 0, time.UTC)}

	const (
		tenantID   = "tenant-p5-sched"
		documentID = "doc-p5-sched"
		instanceID = "inst-p5-sched"
	)

	approvedInstance := &domain.Instance{
		ID:              instanceID,
		TenantID:        tenantID,
		DocumentID:      documentID,
		Status:          domain.InstanceApproved,
		RevisionVersion: 2,
	}

	scheduledRow := repository.ScheduledPublishRow{
		DocumentID:      documentID,
		TenantID:        tenantID,
		EffectiveFrom:   effectiveDate,
		RevisionVersion: 2,
	}

	repo := &phase5Repo{
		instance:      approvedInstance,
		scheduledRows: []repository.ScheduledPublishRow{scheduledRow},
	}
	emitter := &MemoryEmitter{}

	// --- Step 1: SchedulePublish ---
	// DB needs: UPDATE documents (rowsAffected=1).
	schedConn := &phase5Conn{updateResults: []int64{1}}
	schedDB := newPhase5DB(t, schedConn)

	publishSvc := &PublishService{repo: repo, emitter: emitter, clock: pastClock}

	schedReq := SchedulePublishRequest{
		TenantID:      tenantID,
		InstanceID:    instanceID,
		EffectiveDate: effectiveDate, // strictly after pastClock.Now()
		ScheduledBy:   "scheduler-user",
	}
	schedResult, err := publishSvc.SchedulePublish(ctx, schedDB, schedReq)
	if err != nil {
		t.Fatalf("SchedulePublish: unexpected error: %v", err)
	}
	if schedResult.DocumentID != documentID {
		t.Errorf("SchedulePublish.DocumentID = %q; want %q", schedResult.DocumentID, documentID)
	}
	if !schedResult.EffectiveDate.Equal(effectiveDate) {
		t.Errorf("SchedulePublish.EffectiveDate = %v; want %v", schedResult.EffectiveDate, effectiveDate)
	}
	if len(emitter.Events) != 1 {
		t.Fatalf("after SchedulePublish: want 1 event; got %d", len(emitter.Events))
	}
	if emitter.Events[0].EventType != "publish_scheduled" {
		t.Errorf("event[0].EventType = %q; want publish_scheduled", emitter.Events[0].EventType)
	}

	// --- Step 2: RunDuePublishes (clock advanced past effective_date) ---
	// The fetch tx uses LevelReadCommitted — schedulerTestConn handles that.
	// We reuse the scheduler driver pattern: BeginTx must be satisfied.
	runConn := &schedulerTestConn{updateResults: []int64{1}}
	runConnName := fmt.Sprintf("phase5_run_test_%p", runConn)
	sql.Register(runConnName, &schedulerTestDriver{conn: runConn})
	runDB, err := sql.Open(runConnName, "")
	if err != nil {
		t.Fatalf("open scheduler run test db: %v", err)
	}
	t.Cleanup(func() { _ = runDB.Close() })

	schedulerSvc := &SchedulerService{repo: repo, emitter: emitter, clock: futureClock}

	runResult, err := schedulerSvc.RunDuePublishes(ctx, runDB)
	if err != nil {
		t.Fatalf("RunDuePublishes: unexpected top-level error: %v", err)
	}
	if runResult.Processed != 1 {
		t.Errorf("RunDuePublishes.Processed = %d; want 1", runResult.Processed)
	}
	if len(runResult.Errors) != 0 {
		t.Errorf("RunDuePublishes.Errors = %v; want empty", runResult.Errors)
	}

	// Total events: 1 (publish_scheduled) + 1 (document_published from scheduler).
	if len(emitter.Events) != 2 {
		t.Fatalf("after RunDuePublishes: want 2 events; got %d", len(emitter.Events))
	}
	if emitter.Events[1].EventType != "document_published" {
		t.Errorf("event[1].EventType = %q; want document_published", emitter.Events[1].EventType)
	}
	if emitter.Events[1].ResourceID != documentID {
		t.Errorf("event[1].ResourceID = %q; want %q", emitter.Events[1].ResourceID, documentID)
	}
}
