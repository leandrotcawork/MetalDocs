//go:build integration

package repository_test

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"metaldocs/internal/modules/templates/domain"
	"metaldocs/internal/modules/templates/repository"
)

func openDB(t *testing.T) *sql.DB {
	dsn := os.Getenv("PGCONN")
	if dsn == "" {
		t.Skip("PGCONN not set; integration test skipped")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Ping(); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestTemplateRepo_CreateAndGet(t *testing.T) {
	db := openDB(t)
	repo := repository.New(db)
	ctx := context.Background()

	tpl := &domain.Template{
		TenantID:  "00000000-0000-0000-0000-000000000001",
		Key:       "test-" + time.Now().Format("150405"),
		Name:      "Test Template",
		CreatedBy: "00000000-0000-0000-0000-000000000002",
	}
	id, err := repo.CreateTemplate(ctx, tpl)
	if err != nil {
		t.Fatal(err)
	}
	got, err := repo.GetTemplate(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if got.Key != tpl.Key {
		t.Fatalf("key mismatch")
	}
}

func TestTemplateRepo_CreateDraftVersion_OneDraftRule(t *testing.T) {
	db := openDB(t)
	repo := repository.New(db)
	ctx := context.Background()

	tplID, _ := repo.CreateTemplate(ctx, &domain.Template{
		TenantID:  "00000000-0000-0000-0000-000000000001",
		Key:       "d-" + time.Now().Format("150405"),
		Name:      "N",
		CreatedBy: "00000000-0000-0000-0000-000000000002",
	})
	v1 := domain.NewTemplateVersion(tplID, 1)
	v1.CreatedBy = "00000000-0000-0000-0000-000000000002"
	v1.DocxStorageKey = "k1"
	v1.SchemaStorageKey = "s1"
	v1.DocxContentHash = "h"
	v1.SchemaContentHash = "h"
	if _, err := repo.CreateVersion(ctx, v1); err != nil {
		t.Fatal(err)
	}

	v2 := domain.NewTemplateVersion(tplID, 2)
	v2.CreatedBy = "00000000-0000-0000-0000-000000000002"
	v2.DocxStorageKey = "k2"
	v2.SchemaStorageKey = "s2"
	v2.DocxContentHash = "h"
	v2.SchemaContentHash = "h"
	if _, err := repo.CreateVersion(ctx, v2); err == nil {
		t.Fatal("expected duplicate-draft error")
	}
}
