//go:build integration
// +build integration

package scenarios_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"

	"metaldocs/tests/integration/fixtures"
	"metaldocs/tests/integration/testdb"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
)

const concurrencyTestSeed = 0xDEADBEEF
const occRaceWorkers = 2

func TestConcurrencyScenarios(t *testing.T) {
	t.Logf("testSeed=0x%X", concurrencyTestSeed)

	t.Run("OCC_StaleRevision", func(t *testing.T) {
		testOCCStaleRevision(t)
	})
	t.Run("SkipLocked_NoDuplicateProcessing", func(t *testing.T) {
		testSkipLockedNoDuplicateProcessing(t)
	})
	t.Run("LeaseEpoch_Fencing", func(t *testing.T) {
		testLeaseEpochFencing(t)
	})
	t.Run("SignoffUnique_DuplicateBlocked", func(t *testing.T) {
		testSignoffUniqueDuplicateBlocked(t)
	})
	t.Run("OCC_Race_N50", func(t *testing.T) {
		db := openDirectDB(t)
		ctx := context.Background()
		workers := occRaceWorkers
		for i := 0; i < 50; i++ {
			t.Run(fmt.Sprintf("iter_%02d", i+1), func(t *testing.T) {
				winners, losers := runSingleOCCRace(t, ctx, db, fmt.Sprintf("n50-%02d", i+1), workers)
				if winners != 1 || losers != workers-1 {
					t.Fatalf("expected exactly one winner and %d stale loser(s); got winners=%d losers=%d", workers-1, winners, losers)
				}
			})
		}
	})
}

func testOCCStaleRevision(t *testing.T) {
	db := openDirectDB(t)
	ctx := context.Background()
	winners, losers := runSingleOCCRace(t, ctx, db, "single", occRaceWorkers)
	if winners != 1 || losers != occRaceWorkers-1 {
		t.Fatalf("expected exactly one winner and %d stale loser(s); got winners=%d losers=%d", occRaceWorkers-1, winners, losers)
	}
}

func runSingleOCCRace(t *testing.T, ctx context.Context, db *sql.DB, suffix string, workers int) (int, int) {
	t.Helper()
	if workers < 2 {
		t.Fatalf("workers must be >=2, got %d", workers)
	}
	tenantID := testdb.DeterministicID(t, "tenant-"+suffix)
	docID := testdb.DeterministicID(t, "doc-"+suffix)
	userID := testdb.DeterministicID(t, "user-"+suffix)

	fixtures.SeedUser(t, ctx, db, "metaldocs", userID, "Race User")
	fixtures.SeedDocument(t, ctx, db, "metaldocs", docID, tenantID, userID)
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), `DELETE FROM metaldocs.documents WHERE id = $1::uuid`, docID)
		_, _ = db.ExecContext(context.Background(), `DELETE FROM metaldocs.iam_users WHERE tenant_id = $1::uuid`, tenantID)
	})

	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(workers)

	results := make([]int64, workers)
	errs := make([]error, workers)

	for i := 0; i < workers; i++ {
		i := i
		go func() {
			defer wg.Done()
			<-start
			res, err := db.ExecContext(ctx, `
				UPDATE metaldocs.documents
				   SET revision_version = revision_version + 1
				 WHERE id = $1::uuid
				   AND tenant_id = $2::uuid
				   AND revision_version = $3`,
				docID, tenantID, 1,
			)
			if err != nil {
				errs[i] = err
				return
			}
			ra, err := res.RowsAffected()
			if err != nil {
				errs[i] = err
				return
			}
			results[i] = ra
		}()
	}

	close(start)
	wg.Wait()

	for _, err := range errs {
		if err != nil {
			t.Fatalf("race update failed: %v", err)
		}
	}

	winners := 0
	losers := 0
	for _, ra := range results {
		if ra == 1 {
			winners++
		}
		if ra == 0 {
			losers++
		}
	}
	return winners, losers
}

func testSkipLockedNoDuplicateProcessing(t *testing.T) {
	db := openDirectDB(t)
	ctx := context.Background()
	tenantID := testdb.DeterministicID(t, "tenant-skiplocked")
	actorID := testdb.DeterministicID(t, "actor-skiplocked")
	routeTemplate := "POST /integration/skiplocked"

	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), `
			DELETE FROM metaldocs.idempotency_keys
			 WHERE tenant_id = $1::uuid
			   AND actor_user_id = $2
			   AND route_template = $3`,
			tenantID, actorID, routeTemplate,
		)
	})

	for i := 0; i < 5; i++ {
		key := fmt.Sprintf("k-%d", i)
		_, err := db.ExecContext(ctx, `
			INSERT INTO metaldocs.idempotency_keys
				(tenant_id, actor_user_id, route_template, key, payload_hash, response_status, response_body, status, expires_at)
			VALUES
				($1::uuid, $2, $3, $4, $5, 200, '{}'::jsonb, 'completed', now() - interval '1 hour')`,
			tenantID, actorID, routeTemplate, key, fmt.Sprintf("h-%d", i),
		)
		if err != nil {
			t.Fatalf("seed idempotency key %d: %v", i, err)
		}
	}

	type rowRef struct {
		tenantID      string
		actorUserID   string
		routeTemplate string
		key           string
	}

	var mu sync.Mutex
	processed := map[string]int{}

	start := make(chan struct{})
	selected := sync.WaitGroup{}
	selected.Add(3)
	var wg sync.WaitGroup
	errs := make([]error, 3)
	wg.Add(3)
	for w := 0; w < 3; w++ {
		w := w
		go func() {
			defer wg.Done()
			<-start
			defer selected.Done()

			tx, err := db.BeginTx(ctx, nil)
			if err != nil {
				errs[w] = fmt.Errorf("worker %d begin tx: %w", w, err)
				return
			}
			defer tx.Rollback()

			rows, err := tx.QueryContext(ctx, `
				SELECT tenant_id::text, actor_user_id, route_template, key
				  FROM metaldocs.idempotency_keys
				 WHERE tenant_id = $1::uuid
				   AND actor_user_id = $2
				   AND route_template = $3
				   AND status = 'completed'
				 FOR UPDATE SKIP LOCKED
				 LIMIT 2`,
				tenantID, actorID, routeTemplate,
			)
			if err != nil {
				errs[w] = fmt.Errorf("worker %d select for update skip locked: %w", w, err)
				return
			}

			picked := make([]rowRef, 0, 2)
			for rows.Next() {
				var r rowRef
				if err := rows.Scan(&r.tenantID, &r.actorUserID, &r.routeTemplate, &r.key); err != nil {
					_ = rows.Close()
					errs[w] = fmt.Errorf("worker %d scan: %w", w, err)
					return
				}
				picked = append(picked, r)
			}
			if err := rows.Err(); err != nil {
				_ = rows.Close()
				errs[w] = fmt.Errorf("worker %d rows err: %w", w, err)
				return
			}
			_ = rows.Close()

			selected.Wait()

			for _, r := range picked {
				res, err := tx.ExecContext(ctx, `
					UPDATE metaldocs.idempotency_keys
					   SET status = 'failed'
					 WHERE tenant_id = $1::uuid
					   AND actor_user_id = $2
					   AND route_template = $3
					   AND key = $4
					   AND status = 'completed'`,
					r.tenantID, r.actorUserID, r.routeTemplate, r.key,
				)
				if err != nil {
					errs[w] = fmt.Errorf("worker %d update key %s: %w", w, r.key, err)
					return
				}
				ra, err := res.RowsAffected()
				if err != nil {
					errs[w] = fmt.Errorf("worker %d rows affected key %s: %w", w, r.key, err)
					return
				}
				if ra != 1 {
					errs[w] = fmt.Errorf("worker %d expected rows affected=1 for key %s, got %d", w, r.key, ra)
					return
				}
				mu.Lock()
				processed[r.key]++
				mu.Unlock()
			}

			if err := tx.Commit(); err != nil {
				errs[w] = fmt.Errorf("worker %d commit: %w", w, err)
				return
			}
		}()
	}

	close(start)
	wg.Wait()
	for _, err := range errs {
		if err != nil {
			t.Fatalf("skip-locked worker failed: %v", err)
		}
	}

	if len(processed) != 5 {
		t.Fatalf("expected 5 processed keys, got %d (%v)", len(processed), processed)
	}
	for k, cnt := range processed {
		if cnt != 1 {
			t.Fatalf("key %s processed %d times (expected 1)", k, cnt)
		}
	}
}

func testLeaseEpochFencing(t *testing.T) {
	db := openDirectDB(t)
	ctx := context.Background()
	jobName := "test-fencing-job"

	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), `DELETE FROM metaldocs.job_leases WHERE job_name = $1`, jobName)
	})

	_, _ = db.ExecContext(ctx, `DELETE FROM metaldocs.job_leases WHERE job_name = $1`, jobName)
	if _, err := db.ExecContext(ctx, `
		INSERT INTO metaldocs.job_leases (job_name, leader_id, lease_epoch, acquired_at, heartbeat_at, expires_at)
		VALUES ($1, 'A', 5, now(), now(), now() + interval '5 minutes')`,
		jobName,
	); err != nil {
		t.Fatalf("seed lease row: %v", err)
	}

	_, err := db.ExecContext(ctx, `SELECT metaldocs.assert_lease_epoch($1, $2)`, jobName, 6)
	if err == nil {
		t.Fatal("expected stale lease epoch error for epoch=6")
	}
	if !hasSQLState(err, "P0001") || !strings.Contains(err.Error(), "ErrLeaseEpochStale") {
		t.Fatalf("expected P0001 ErrLeaseEpochStale, got: %v", err)
	}

	if _, err := db.ExecContext(ctx, `SELECT metaldocs.assert_lease_epoch($1, $2)`, jobName, 5); err != nil {
		t.Fatalf("assert_lease_epoch with current epoch should succeed: %v", err)
	}
}

func testSignoffUniqueDuplicateBlocked(t *testing.T) {
	db := openDirectDB(t)
	ctx := context.Background()

	tenantID := testdb.DeterministicID(t, "tenant-signoff")
	authorID := testdb.DeterministicID(t, "author-signoff")
	actorID := testdb.DeterministicID(t, "actor-signoff")
	docID := testdb.DeterministicID(t, "doc-signoff")
	routeID := testdb.DeterministicID(t, "route-signoff")
	instanceID := testdb.DeterministicID(t, "instance-signoff")
	stageID := testdb.DeterministicID(t, "stage-signoff")

	fixtures.SeedUser(t, ctx, db, "metaldocs", authorID, "Doc Author")
	fixtures.SeedUser(t, ctx, db, "metaldocs", actorID, "Signer")
	fixtures.SeedDocument(t, ctx, db, "metaldocs", docID, tenantID, authorID)
	fixtures.SeedRouteConfig(t, ctx, db, "metaldocs", routeID, tenantID, "INT_SIGNOFF")

	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), `DELETE FROM metaldocs.approval_signoffs WHERE approval_instance_id = $1::uuid`, instanceID)
		_, _ = db.ExecContext(context.Background(), `DELETE FROM metaldocs.approval_stage_instances WHERE id = $1::uuid`, stageID)
		_, _ = db.ExecContext(context.Background(), `DELETE FROM metaldocs.approval_instances WHERE id = $1::uuid`, instanceID)
		_, _ = db.ExecContext(context.Background(), `DELETE FROM metaldocs.approval_routes WHERE id = $1::uuid`, routeID)
		_, _ = db.ExecContext(context.Background(), `DELETE FROM metaldocs.documents WHERE id = $1::uuid`, docID)
		_, _ = db.ExecContext(context.Background(), `DELETE FROM metaldocs.iam_users WHERE tenant_id = $1::uuid`, tenantID)
	})

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin setup tx: %v", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `SELECT set_config('metaldocs.bypass_authz', 'scheduler', true)`); err != nil {
		t.Fatalf("set bypass_authz for setup: %v", err)
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO metaldocs.approval_instances
			(id, tenant_id, document_v2_id, route_id, route_version_snapshot, status, submitted_by, submitted_at, content_hash_at_submit, idempotency_key)
		VALUES
			($1::uuid, $2::uuid, $3::uuid, $4::uuid, 1, 'in_progress', $5, now(), 'hash-signoff', 'idem-signoff')`,
		instanceID, tenantID, docID, routeID, authorID,
	); err != nil {
		t.Fatalf("seed approval_instance: %v", err)
	}

	eligible, _ := json.Marshal([]string{actorID})
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO metaldocs.approval_stage_instances
			(id, approval_instance_id, stage_order, name_snapshot, required_role_snapshot, required_capability_snapshot, area_code_snapshot, quorum_snapshot, quorum_m_snapshot, on_eligibility_drift_snapshot, eligible_actor_ids, effective_denominator, status, opened_at)
		VALUES
			($1::uuid, $2::uuid, 1, 'Stage 1', 'reviewer', 'doc.signoff', 'AREA_INT', 'any_1_of', NULL, 'keep_snapshot', $3::jsonb, 1, 'active', now())`,
		stageID, instanceID, string(eligible),
	); err != nil {
		t.Fatalf("seed stage_instance: %v", err)
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("commit setup tx: %v", err)
	}

	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)

	errs := make([]error, 2)
	for i := 0; i < 2; i++ {
		i := i
		go func() {
			defer wg.Done()
			<-start

			tx, err := db.BeginTx(ctx, nil)
			if err != nil {
				errs[i] = err
				return
			}
			defer tx.Rollback()

			if _, err := tx.ExecContext(ctx, `SELECT set_config('metaldocs.bypass_authz', 'scheduler', true)`); err != nil {
				errs[i] = err
				return
			}

			_, err = tx.ExecContext(ctx, `
				INSERT INTO metaldocs.approval_signoffs
					(id, approval_instance_id, stage_instance_id, actor_user_id, actor_tenant_id, decision, comment, signed_at, signature_method, signature_payload, content_hash)
				VALUES
					($1::uuid, $2::uuid, $3::uuid, $4, $5::uuid, 'approve', 'integration race', now(), 'password', '{}'::jsonb, 'content-hash')`,
				testdb.DeterministicID(t, fmt.Sprintf("signoff-%d", i)), instanceID, stageID, actorID, tenantID,
			)
			if err != nil {
				errs[i] = err
				return
			}

			if err := tx.Commit(); err != nil {
				errs[i] = err
				return
			}
		}()
	}

	close(start)
	wg.Wait()

	uniqueViolations := 0
	successes := 0
	for _, err := range errs {
		if err == nil {
			successes++
			continue
		}
		if hasSQLState(err, "23505") {
			uniqueViolations++
			continue
		}
		t.Fatalf("unexpected signoff insert error: %v", err)
	}
	if successes != 1 || uniqueViolations != 1 {
		t.Fatalf("expected one success and one 23505 violation, got successes=%d uniqueViolations=%d errs=%v", successes, uniqueViolations, errs)
	}

	var count int
	if err := db.QueryRowContext(ctx, `
		SELECT count(*)
		  FROM metaldocs.approval_signoffs
		 WHERE approval_instance_id = $1::uuid
		   AND stage_instance_id = $2::uuid
		   AND actor_user_id = $3`,
		instanceID, stageID, actorID,
	).Scan(&count); err != nil {
		t.Fatalf("count signoffs: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 signoff row, got %d", count)
	}
}

func openDirectDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("METALDOCS_DATABASE_URL"))
	if dsn == "" {
		dsn = strings.TrimSpace(os.Getenv("DATABASE_URL"))
	}
	if dsn == "" {
		t.Skip("DATABASE_URL/METALDOCS_DATABASE_URL not set")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := db.PingContext(context.Background()); err != nil {
		t.Skipf("integration DB unreachable: %v", err)
	}
	return db
}

func hasSQLState(err error, state string) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == state
	}
	return strings.Contains(err.Error(), state)
}
