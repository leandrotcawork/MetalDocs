//go:build integration

package application

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestBackfillLegacyDocuments(t *testing.T) {
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

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	if err := BackfillLegacyDocuments(context.Background(), db, logger); err != nil {
		t.Fatalf("BackfillLegacyDocuments returned error: %v", err)
	}
}
