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

	"github.com/jackc/pgx/v5/pgconn"

	"metaldocs/internal/modules/documents_v2/approval/domain"
	"metaldocs/internal/modules/documents_v2/approval/repository"
	"metaldocs/internal/modules/iam/authz"
)

type routeAdminRows struct {
	cols   []string
	values []driver.Value
	done   bool
}

func (r *routeAdminRows) Columns() []string { return r.cols }
func (r *routeAdminRows) Close() error      { return nil }
func (r *routeAdminRows) Next(dest []driver.Value) error {
	if r.done {
		return io.EOF
	}
	r.done = true
	for i, v := range r.values {
		dest[i] = v
	}
	return nil
}

type routeAdminEmptyRows struct{ cols []string }

func (r routeAdminEmptyRows) Columns() []string         { return r.cols }
func (r routeAdminEmptyRows) Close() error              { return nil }
func (r routeAdminEmptyRows) Next([]driver.Value) error { return io.EOF }

type routeAdminResult struct{ rowsAffected int64 }

func (r routeAdminResult) LastInsertId() (int64, error) { return 0, nil }
func (r routeAdminResult) RowsAffected() (int64, error) { return r.rowsAffected, nil }

type routeAdminStmt struct {
	conn  *routeAdminConn
	query string
}

func (s *routeAdminStmt) Close() error  { return nil }
func (s *routeAdminStmt) NumInput() int { return -1 }

func (s *routeAdminStmt) Exec(_ []driver.Value) (driver.Result, error) {
	lower := strings.ToLower(s.query)
	if strings.Contains(lower, "update approval_routes") && strings.Contains(lower, "set active = false") {
		if s.conn.deactivateErr != nil {
			return nil, s.conn.deactivateErr
		}
		return routeAdminResult{rowsAffected: 1}, nil
	}
	return routeAdminResult{rowsAffected: 1}, nil
}

func (s *routeAdminStmt) Query(_ []driver.Value) (driver.Rows, error) {
	lower := strings.ToLower(s.query)

	if strings.Contains(lower, "select exists") && strings.Contains(lower, "role_capabilities") {
		return &routeAdminRows{
			cols:   []string{"exists"},
			values: []driver.Value{s.conn.authzGranted},
		}, nil
	}
	if strings.Contains(lower, "current_setting('metaldocs.asserted_caps'") {
		return &routeAdminRows{cols: []string{"v"}, values: []driver.Value{nil}}, nil
	}
	if strings.Contains(lower, "current_setting('metaldocs.actor_id'") {
		return &routeAdminRows{cols: []string{"v"}, values: []driver.Value{s.conn.actorID}}, nil
	}
	if strings.Contains(lower, "set_config") {
		return &routeAdminRows{cols: []string{"v"}, values: []driver.Value{"ok"}}, nil
	}
	if strings.Contains(lower, "insert into approval_routes") && strings.Contains(lower, "returning id") {
		return &routeAdminRows{
			cols:   []string{"id"},
			values: []driver.Value{s.conn.createdRouteID},
		}, nil
	}
	if strings.Contains(lower, "from approval_routes") && strings.Contains(lower, "for update") {
		if !s.conn.routeExists {
			return routeAdminEmptyRows{cols: []string{"id"}}, nil
		}
		return &routeAdminRows{
			cols:   []string{"id"},
			values: []driver.Value{s.conn.lockedRouteID},
		}, nil
	}
	if strings.Contains(lower, "update approval_routes") && strings.Contains(lower, "returning version") {
		if s.conn.updateErr != nil {
			return nil, s.conn.updateErr
		}
		return &routeAdminRows{
			cols:   []string{"version"},
			values: []driver.Value{int64(s.conn.newVersion)},
		}, nil
	}
	return routeAdminEmptyRows{cols: []string{"v"}}, nil
}

type routeAdminConn struct {
	authzGranted   bool
	actorID        string
	createdRouteID string
	lockedRouteID  string
	routeExists    bool
	newVersion     int
	updateErr      error
	deactivateErr  error
}

func (c *routeAdminConn) Prepare(query string) (driver.Stmt, error) {
	return &routeAdminStmt{conn: c, query: query}, nil
}
func (c *routeAdminConn) Close() error              { return nil }
func (c *routeAdminConn) Begin() (driver.Tx, error) { return c, nil }
func (c *routeAdminConn) Commit() error             { return nil }
func (c *routeAdminConn) Rollback() error           { return nil }

type routeAdminDriver struct{ conn *routeAdminConn }

func (d *routeAdminDriver) Open(_ string) (driver.Conn, error) { return d.conn, nil }

func newRouteAdminTestDB(t *testing.T, conn *routeAdminConn) *sql.DB {
	t.Helper()
	if conn.actorID == "" {
		conn.actorID = "user-1"
	}
	if conn.createdRouteID == "" {
		conn.createdRouteID = "route-1"
	}
	if conn.lockedRouteID == "" {
		conn.lockedRouteID = "route-1"
	}
	if conn.newVersion == 0 {
		conn.newVersion = 2
	}
	name := fmt.Sprintf("route_admin_test_%p", conn)
	sql.Register(name, &routeAdminDriver{conn: conn})
	db, err := sql.Open(name, "")
	if err != nil {
		t.Fatalf("open route admin test db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func validRouteStages() []domain.Stage {
	return []domain.Stage{
		{
			Order:              1,
			Name:               "quality",
			RequiredRole:       "qa_reviewer",
			RequiredCapability: "workflow.sign",
			AreaCode:           "tenant",
			Quorum:             domain.QuorumAny1Of,
			OnEligibilityDrift: domain.DriftReduceQuorum,
		},
		{
			Order:              2,
			Name:               "approval",
			RequiredRole:       "qa_manager",
			RequiredCapability: "workflow.sign",
			AreaCode:           "tenant",
			Quorum:             domain.QuorumAllOf,
			OnEligibilityDrift: domain.DriftFailStage,
		},
	}
}

func TestRouteAdminCreate_HappyPath(t *testing.T) {
	conn := &routeAdminConn{authzGranted: true}
	db := newRouteAdminTestDB(t, conn)

	emitter := &MemoryEmitter{}
	svc := &RouteAdminService{
		emitter: emitter,
		clock:   fixedClock{t: time.Date(2026, 4, 22, 12, 0, 0, 0, time.UTC)},
	}

	out, err := svc.Create(context.Background(), db, CreateRouteInput{
		TenantID:    "tenant-1",
		ProfileCode: "po",
		Name:        "PO Route",
		ActorUserID: "user-1",
		Stages:      validRouteStages(),
	})
	if err != nil {
		t.Fatalf("Create: unexpected error: %v", err)
	}
	if out.RouteID != "route-1" {
		t.Errorf("RouteID = %q; want %q", out.RouteID, "route-1")
	}
	if len(emitter.Events) != 1 || emitter.Events[0].EventType != "route.config.created" {
		t.Errorf("expected 1 route.config.created event; got %v", emitter.Events)
	}
}

func TestRouteAdminCreate_CapDenied(t *testing.T) {
	conn := &routeAdminConn{authzGranted: false}
	db := newRouteAdminTestDB(t, conn)

	svc := &RouteAdminService{
		emitter: &MemoryEmitter{},
		clock:   fixedClock{t: time.Now()},
	}

	_, err := svc.Create(context.Background(), db, CreateRouteInput{
		TenantID:    "tenant-1",
		ProfileCode: "po",
		Name:        "PO Route",
		ActorUserID: "user-1",
		Stages:      validRouteStages(),
	})
	var denied authz.ErrCapabilityDenied
	if !errors.As(err, &denied) {
		t.Errorf("expected ErrCapabilityDenied; got %v", err)
	}
}

func TestRouteAdminCreate_InvalidRoute(t *testing.T) {
	conn := &routeAdminConn{authzGranted: true}
	db := newRouteAdminTestDB(t, conn)

	emitter := &MemoryEmitter{}
	svc := &RouteAdminService{
		emitter: emitter,
		clock:   fixedClock{t: time.Now()},
	}

	_, err := svc.Create(context.Background(), db, CreateRouteInput{
		TenantID:    "tenant-1",
		ProfileCode: "po",
		Name:        "PO Route",
		ActorUserID: "user-1",
		Stages:      nil,
	})
	if err == nil || !strings.Contains(err.Error(), "route must have at least one stage") {
		t.Fatalf("expected validation error; got %v", err)
	}
	if len(emitter.Events) != 0 {
		t.Fatalf("expected no events on validation failure; got %d", len(emitter.Events))
	}
}

func TestRouteAdminUpdate_HappyPath(t *testing.T) {
	conn := &routeAdminConn{
		authzGranted: true,
		routeExists:  true,
		newVersion:   4,
	}
	db := newRouteAdminTestDB(t, conn)

	emitter := &MemoryEmitter{}
	svc := &RouteAdminService{
		emitter: emitter,
		clock:   fixedClock{t: time.Date(2026, 4, 22, 12, 0, 0, 0, time.UTC)},
	}

	out, err := svc.Update(context.Background(), db, UpdateRouteInput{
		TenantID:    "tenant-1",
		RouteID:     "route-1",
		Name:        "PO Route v2",
		ActorUserID: "user-1",
		Stages:      validRouteStages(),
	})
	if err != nil {
		t.Fatalf("Update: unexpected error: %v", err)
	}
	if out.RouteID != "route-1" {
		t.Errorf("RouteID = %q; want %q", out.RouteID, "route-1")
	}
	if out.NewVersion != 4 {
		t.Errorf("NewVersion = %d; want %d", out.NewVersion, 4)
	}
	if len(emitter.Events) != 1 || emitter.Events[0].EventType != "route.config.updated" {
		t.Errorf("expected 1 route.config.updated event; got %v", emitter.Events)
	}
}

func TestRouteAdminUpdate_RouteInUse(t *testing.T) {
	conn := &routeAdminConn{
		authzGranted: true,
		routeExists:  true,
		updateErr: &pgconn.PgError{
			Code:    "P0001",
			Message: "ErrRouteInUse: route xyz is referenced by one or more approval instances and cannot be modified",
		},
	}
	db := newRouteAdminTestDB(t, conn)

	svc := &RouteAdminService{
		emitter: &MemoryEmitter{},
		clock:   fixedClock{t: time.Now()},
	}

	_, err := svc.Update(context.Background(), db, UpdateRouteInput{
		TenantID:    "tenant-1",
		RouteID:     "route-1",
		Name:        "PO Route v2",
		ActorUserID: "user-1",
		Stages:      validRouteStages(),
	})
	if !errors.Is(err, repository.ErrRouteInUse) {
		t.Fatalf("expected ErrRouteInUse; got %v", err)
	}
}

func TestRouteAdminDeactivate_HappyPath(t *testing.T) {
	conn := &routeAdminConn{
		authzGranted: true,
		routeExists:  true,
	}
	db := newRouteAdminTestDB(t, conn)

	emitter := &MemoryEmitter{}
	svc := &RouteAdminService{
		emitter: emitter,
		clock:   fixedClock{t: time.Date(2026, 4, 22, 12, 0, 0, 0, time.UTC)},
	}

	out, err := svc.Deactivate(context.Background(), db, DeactivateRouteInput{
		TenantID:    "tenant-1",
		RouteID:     "route-1",
		ActorUserID: "user-1",
	})
	if err != nil {
		t.Fatalf("Deactivate: unexpected error: %v", err)
	}
	if out.RouteID != "route-1" {
		t.Errorf("RouteID = %q; want %q", out.RouteID, "route-1")
	}
	if len(emitter.Events) != 1 || emitter.Events[0].EventType != "route.config.deactivated" {
		t.Errorf("expected 1 route.config.deactivated event; got %v", emitter.Events)
	}
}
