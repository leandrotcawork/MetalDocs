//go:build integration

package application

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestCodeImmutability(t *testing.T) {
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

	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	defer tx.Rollback()

	oldCode := "immut_old_" + time.Now().UTC().Format("150405")
	newCode := "immut_new_" + time.Now().UTC().Format("150405")
	tenantID := "ffffffff-ffff-ffff-ffff-ffffffffffff"

	_, err = tx.ExecContext(
		context.Background(),
		`INSERT INTO metaldocs.document_profiles
		    (code, tenant_id, family_code, name, description, review_interval_days, editable_by_role)
		 VALUES
		    ($1, $2, $3, $4, $5, $6, $7)`,
		oldCode,
		tenantID,
		"procedure",
		"Immutability Test",
		"test",
		30,
		"admin",
	)
	if err != nil {
		t.Fatalf("insert profile: %v", err)
	}

	_, err = tx.ExecContext(
		context.Background(),
		`UPDATE metaldocs.document_profiles
		 SET code = $1
		 WHERE tenant_id = $2 AND code = $3`,
		newCode,
		tenantID,
		oldCode,
	)
	if err == nil {
		t.Fatal("expected immutability error when updating code, got nil")
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23514" {
		return
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "check_violation") || strings.Contains(msg, "code column is immutable") {
		return
	}
	t.Fatalf("expected check_violation or immutable error, got: %v", err)
}
