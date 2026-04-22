//go:build integration
// +build integration

package scenarios_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestIdempotency_SameKeyReplay(t *testing.T) {
	ctx := context.Background()
	db := openDirectDB(t)

	tenantID := "11111111-1111-1111-1111-111111111181"
	actorUserID := "idem-user-1"
	routeTemplate := "POST /api/v2/documents/{id}/submit"
	key := "idem-key-1"

	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), `
			DELETE FROM metaldocs.idempotency_keys
			 WHERE tenant_id = $1::uuid
			   AND actor_user_id = $2
			   AND route_template = $3
			   AND key = $4`,
			tenantID, actorUserID, routeTemplate, key,
		)
	})

	if _, err := db.ExecContext(ctx, `
		INSERT INTO metaldocs.idempotency_keys
			(tenant_id, actor_user_id, route_template, key, payload_hash, response_status, response_body, status, expires_at)
		VALUES
			($1::uuid, $2, $3, $4, 'abc', 201, '{}'::jsonb, 'completed', now() + interval '1 day')`,
		tenantID, actorUserID, routeTemplate, key,
	); err != nil {
		t.Fatalf("seed insert: %v", err)
	}

	var payloadHash, status string
	if err := db.QueryRowContext(ctx, `
		INSERT INTO metaldocs.idempotency_keys
			(tenant_id, actor_user_id, route_template, key, payload_hash, response_status, response_body, status, expires_at)
		VALUES
			($1::uuid, $2, $3, $4, 'abc', 201, '{}'::jsonb, 'completed', now() + interval '1 day')
		ON CONFLICT (tenant_id, actor_user_id, route_template, key)
		DO UPDATE SET
			payload_hash = metaldocs.idempotency_keys.payload_hash
		RETURNING payload_hash, status`,
		tenantID, actorUserID, routeTemplate, key,
	).Scan(&payloadHash, &status); err != nil {
		t.Fatalf("replay insert returning existing row: %v", err)
	}

	if payloadHash != "abc" || status != "completed" {
		t.Fatalf("unexpected replay row: payload_hash=%q status=%q", payloadHash, status)
	}

	var count int
	if err := db.QueryRowContext(ctx, `
		SELECT count(*)
		  FROM metaldocs.idempotency_keys
		 WHERE tenant_id = $1::uuid
		   AND actor_user_id = $2
		   AND route_template = $3
		   AND key = $4`,
		tenantID, actorUserID, routeTemplate, key,
	).Scan(&count); err != nil {
		t.Fatalf("count rows: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 row for replay key, got %d", count)
	}
}

func TestIdempotency_SameKeyDifferentPayload(t *testing.T) {
	ctx := context.Background()
	db := openDirectDB(t)

	tenantID := "11111111-1111-1111-1111-111111111182"
	actorUserID := "idem-user-2"
	routeTemplate := "POST /api/v2/documents/{id}/submit"
	key := "idem-key-2"

	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), `
			DELETE FROM metaldocs.idempotency_keys
			 WHERE tenant_id = $1::uuid
			   AND actor_user_id = $2
			   AND route_template = $3
			   AND key = $4`,
			tenantID, actorUserID, routeTemplate, key,
		)
	})

	if _, err := db.ExecContext(ctx, `
		INSERT INTO metaldocs.idempotency_keys
			(tenant_id, actor_user_id, route_template, key, payload_hash, response_status, response_body, status, expires_at)
		VALUES
			($1::uuid, $2, $3, $4, 'hash-A', 200, '{}'::jsonb, 'completed', now() + interval '1 day')`,
		tenantID, actorUserID, routeTemplate, key,
	); err != nil {
		t.Fatalf("seed insert: %v", err)
	}

	if _, err := db.ExecContext(ctx, `
		INSERT INTO metaldocs.idempotency_keys
			(tenant_id, actor_user_id, route_template, key, payload_hash, response_status, response_body, status, expires_at)
		VALUES
			($1::uuid, $2, $3, $4, 'hash-B', 200, '{}'::jsonb, 'completed', now() + interval '1 day')
		ON CONFLICT (tenant_id, actor_user_id, route_template, key) DO NOTHING`,
		tenantID, actorUserID, routeTemplate, key,
	); err != nil {
		t.Fatalf("conflict insert with different payload hash: %v", err)
	}

	var payloadHash string
	if err := db.QueryRowContext(ctx, `
		SELECT payload_hash
		  FROM metaldocs.idempotency_keys
		 WHERE tenant_id = $1::uuid
		   AND actor_user_id = $2
		   AND route_template = $3
		   AND key = $4`,
		tenantID, actorUserID, routeTemplate, key,
	).Scan(&payloadHash); err != nil {
		t.Fatalf("load key row: %v", err)
	}
	if payloadHash != "hash-A" {
		t.Fatalf("expected original payload hash to win; got %q", payloadHash)
	}
}

func TestIdempotency_Expired_NewEntry(t *testing.T) {
	ctx := context.Background()
	db := openDirectDB(t)

	tenantID := "11111111-1111-1111-1111-111111111183"
	actorUserID := "idem-user-3"
	routeTemplate := "POST /api/v2/documents/{id}/submit"
	key := "idem-key-3"

	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), `
			DELETE FROM metaldocs.idempotency_keys
			 WHERE tenant_id = $1::uuid
			   AND actor_user_id = $2
			   AND route_template = $3
			   AND key = $4`,
			tenantID, actorUserID, routeTemplate, key,
		)
	})

	if _, err := db.ExecContext(ctx, `
		INSERT INTO metaldocs.idempotency_keys
			(tenant_id, actor_user_id, route_template, key, payload_hash, response_status, response_body, status, expires_at)
		VALUES
			($1::uuid, $2, $3, $4, 'old-hash', 200, '{}'::jsonb, 'completed', now() - interval '1 hour')`,
		tenantID, actorUserID, routeTemplate, key,
	); err != nil {
		t.Fatalf("seed expired row: %v", err)
	}

	if _, err := db.ExecContext(ctx, `
		DELETE FROM metaldocs.idempotency_keys
		 WHERE expires_at < now()
		   AND status = 'completed'
		   AND tenant_id = $1::uuid
		   AND actor_user_id = $2
		   AND route_template = $3
		   AND key = $4`,
		tenantID, actorUserID, routeTemplate, key,
	); err != nil {
		t.Fatalf("janitor delete: %v", err)
	}

	newExpiry := time.Now().UTC().Add(2 * time.Hour)
	if _, err := db.ExecContext(ctx, `
		INSERT INTO metaldocs.idempotency_keys
			(tenant_id, actor_user_id, route_template, key, payload_hash, response_status, response_body, status, expires_at)
		VALUES
			($1::uuid, $2, $3, $4, 'new-hash', 201, '{}'::jsonb, 'completed', $5)`,
		tenantID, actorUserID, routeTemplate, key, newExpiry,
	); err != nil {
		t.Fatalf("insert new row after janitor cleanup: %v", err)
	}

	var payloadHash string
	var expiresAt time.Time
	if err := db.QueryRowContext(ctx, `
		SELECT payload_hash, expires_at
		  FROM metaldocs.idempotency_keys
		 WHERE tenant_id = $1::uuid
		   AND actor_user_id = $2
		   AND route_template = $3
		   AND key = $4`,
		tenantID, actorUserID, routeTemplate, key,
	).Scan(&payloadHash, &expiresAt); err != nil {
		t.Fatalf("load new row: %v", err)
	}

	if payloadHash != "new-hash" {
		t.Fatalf("expected new payload hash, got %q", payloadHash)
	}
	if !expiresAt.After(time.Now().UTC()) {
		t.Fatalf("expected new expires_at in the future, got %s", expiresAt.Format(time.RFC3339))
	}
}

func TestIdempotency_Concurrent_OnlyOneWins(t *testing.T) {
	ctx := context.Background()
	db := openDirectDB(t)

	tenantID := "11111111-1111-1111-1111-111111111184"
	actorUserID := "idem-user-4"
	routeTemplate := "POST /api/v2/documents/{id}/submit"
	key := "idem-key-4"

	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), `
			DELETE FROM metaldocs.idempotency_keys
			 WHERE tenant_id = $1::uuid
			   AND actor_user_id = $2
			   AND route_template = $3
			   AND key = $4`,
			tenantID, actorUserID, routeTemplate, key,
		)
	})

	start := make(chan struct{})
	var wg sync.WaitGroup

	errs := make([]error, 5)
	wg.Add(5)
	for i := 0; i < 5; i++ {
		i := i
		go func() {
			defer wg.Done()
			<-start
			_, err := db.ExecContext(ctx, `
				INSERT INTO metaldocs.idempotency_keys
					(tenant_id, actor_user_id, route_template, key, payload_hash, response_status, response_body, status, expires_at)
				VALUES
					($1::uuid, $2, $3, $4, $5, 200, '{}'::jsonb, 'completed', now() + interval '1 day')`,
				tenantID, actorUserID, routeTemplate, key, fmt.Sprintf("concurrent-hash-%d", i),
			)
			errs[i] = err
		}()
	}

	close(start)
	wg.Wait()

	successes := 0
	violations := 0
	for _, err := range errs {
		if err == nil {
			successes++
			continue
		}
		if hasSQLState(err, "23505") {
			violations++
			continue
		}
		t.Fatalf("unexpected concurrent insert error: %v", err)
	}
	if successes != 1 || violations != 4 {
		t.Fatalf("expected 1 success and 4 unique violations, got successes=%d violations=%d errs=%v", successes, violations, errs)
	}

	var count int
	if err := db.QueryRowContext(ctx, `
		SELECT count(*)
		  FROM metaldocs.idempotency_keys
		 WHERE tenant_id = $1::uuid
		   AND actor_user_id = $2
		   AND route_template = $3
		   AND key = $4`,
		tenantID, actorUserID, routeTemplate, key,
	).Scan(&count); err != nil {
		t.Fatalf("count final rows: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected exactly one row after concurrent inserts, got %d", count)
	}
}
