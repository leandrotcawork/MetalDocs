//go:build ignore

// Seed a published templates_v2 template version for the 'po' profile
// and set it as the profile's default_template_version_id.
// Usage: go run ./cmd/seed-spec1-template/main.go
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
)

const (
	dsn      = "host=localhost port=5433 user=metaldocs_app password=Lepa12<>! dbname=metaldocs sslmode=disable"
	tenantID = "ffffffff-ffff-ffff-ffff-ffffffffffff"
)

func main() {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("ping db: %v", err)
	}

	// 5. Drop old FK on documents.template_version_id (references legacy table)
	_, err = db.Exec(`ALTER TABLE documents DROP CONSTRAINT IF EXISTS documents_template_version_id_fkey`)
	if err != nil {
		log.Fatalf("drop FK: %v", err)
	}

	// 1-4: upsert template + version (idempotent)
	templateID := uuid.NewString()
	versionID := uuid.NewString()
	now := time.Now().UTC()

	// Check if already seeded
	var existingVersionID string
	err = db.QueryRowContext(context.Background(), `
		SELECT tv.id FROM templates_v2_template_version tv
		JOIN templates_v2_template t ON t.id = tv.template_id
		WHERE t.tenant_id = $1 AND t.key = 'po-seed'`, tenantID).Scan(&existingVersionID)
	if err == nil {
		fmt.Printf("OK (already seeded)\n  version_id:   %s\n  FK dropped\n", existingVersionID)
		return
	}

	// 1. Insert template
	_, err = db.Exec(`
		INSERT INTO templates_v2_template
			(id, tenant_id, doc_type_code, key, name, description, areas, visibility,
			 specific_areas, latest_version, published_version_id, created_by, created_at)
		VALUES ($1,$2,'po','po-seed','PO Seed Template','Seed template for spec1 validation',
			'{}','public','{}',1,NULL,'seed-script',$3)`,
		templateID, tenantID, now)
	if err != nil {
		log.Fatalf("insert template: %v", err)
	}

	// 2. Insert published version
	_, err = db.Exec(`
		INSERT INTO templates_v2_template_version
			(id, template_id, version_number, status, docx_storage_key, content_hash,
			 metadata_schema, placeholder_schema, editable_zones,
			 author_id, pending_approver_role,
			 approved_at, published_at, created_at)
		VALUES ($1,$2,1,'published','','',
			'{"doc_code_pattern":"","retention_days":0,"distribution_default":null,"required_metadata":null}','[]','[]',
			'seed-script','admin',
			$3,$3,$3)`,
		versionID, templateID, now)
	if err != nil {
		log.Fatalf("insert version: %v", err)
	}

	// 3. Set published_version_id on template
	_, err = db.Exec(`UPDATE templates_v2_template SET published_version_id = $1 WHERE id = $2`,
		versionID, templateID)
	if err != nil {
		log.Fatalf("update template published_version_id: %v", err)
	}

	// 4. Set default_template_version_id on po profile
	res, err := db.Exec(`
		UPDATE metaldocs.document_profiles
		SET default_template_version_id = $1
		WHERE tenant_id = $2 AND code = 'po'`,
		versionID, tenantID)
	if err != nil {
		log.Fatalf("update profile: %v", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		log.Fatalf("profile 'po' not found for tenant %s", tenantID)
	}

	fmt.Printf("OK\n  template_id:  %s\n  version_id:   %s\n  profile 'po' default set\n  FK dropped\n", templateID, versionID)
}
