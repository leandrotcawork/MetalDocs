//go:build integration

package domain_test

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"metaldocs/internal/modules/registry/infrastructure"
)

func TestSequenceAllocatorNextAndIncrement_Concurrent(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if dsn == "" {
		dsn = strings.TrimSpace(os.Getenv("METALDOCS_DATABASE_URL"))
	}
	if dsn == "" {
		t.Skip("DATABASE_URL or METALDOCS_DATABASE_URL is not set")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if err := db.PingContext(context.Background()); err != nil {
		t.Skipf("cannot connect to database: %v", err)
	}

	tenantID := "ffffffff-ffff-ffff-ffff-ffffffffffff"
	profileCode := "seqtest" + strings.ToLower(time.Now().UTC().Format("150405"))

	_, err = db.ExecContext(context.Background(), `
		INSERT INTO metaldocs.document_profiles
			(code, tenant_id, family_code, name, description, review_interval_days, editable_by_role)
		VALUES
			($1, $2, 'procedure', 'Seq Test', 'integration sequence test', 30, 'admin')`,
		profileCode, tenantID,
	)
	if err != nil {
		t.Fatalf("insert profile: %v", err)
	}

	allocator := infrastructure.NewPostgresSequenceAllocator(db)
	if err := allocator.EnsureCounter(context.Background(), tenantID, profileCode); err != nil {
		t.Fatalf("ensure counter: %v", err)
	}

	const workers = 50
	results := make([]int, 0, workers)
	var mu sync.Mutex
	var wg sync.WaitGroup
	wg.Add(workers)

	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			next, err := allocator.NextAndIncrement(context.Background(), nil, tenantID, profileCode)
			if err != nil {
				t.Errorf("next and increment: %v", err)
				return
			}
			mu.Lock()
			results = append(results, next)
			mu.Unlock()
		}()
	}
	wg.Wait()

	if len(results) != workers {
		t.Fatalf("expected %d results, got %d", workers, len(results))
	}

	seen := map[int]bool{}
	for _, v := range results {
		if seen[v] {
			t.Fatalf("duplicate sequence value found: %d", v)
		}
		seen[v] = true
	}
	for i := 1; i <= workers; i++ {
		if !seen[i] {
			t.Fatalf("missing sequence value %d in %v", i, results)
		}
	}
}
