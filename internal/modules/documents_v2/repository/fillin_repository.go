package repository

import (
	"context"
	"database/sql"
	"fmt"

	templatesdomain "metaldocs/internal/modules/templates_v2/domain"
)

// FillInRepository manages document_placeholder_values rows.
type FillInRepository struct {
	db     *sql.DB
	schema string
}

// NewFillInRepository creates a FillInRepository using bare table names.
func NewFillInRepository(db *sql.DB) *FillInRepository {
	return &FillInRepository{db: db}
}

// NewFillInRepositoryWithSchema creates a FillInRepository that qualifies
// table names with the given schema. Used by integration tests.
func NewFillInRepositoryWithSchema(db *sql.DB, schema string) *FillInRepository {
	return &FillInRepository{db: db, schema: schema}
}

func (r *FillInRepository) table(name string) string {
	if r.schema == "" {
		return name
	}
	return fmt.Sprintf("%q.%q", r.schema, name)
}

// SeedDefaults inserts one row per Required placeholder with source='default'.
// Uses ON CONFLICT DO NOTHING so the call is idempotent.
func (r *FillInRepository) SeedDefaults(ctx context.Context, tenantID, revisionID string, phs []templatesdomain.Placeholder) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, p := range phs {
		if !p.Required {
			continue
		}
		if _, err := tx.ExecContext(ctx, fmt.Sprintf(`
			INSERT INTO %s (tenant_id, revision_id, placeholder_id, source, created_at, updated_at)
			VALUES ($1::uuid, $2::uuid, $3, 'default', NOW(), NOW())
			ON CONFLICT DO NOTHING`, r.table("document_placeholder_values")),
			tenantID, revisionID, p.ID,
		); err != nil {
			return fmt.Errorf("seed placeholder %q: %w", p.ID, err)
		}
	}
	return tx.Commit()
}
