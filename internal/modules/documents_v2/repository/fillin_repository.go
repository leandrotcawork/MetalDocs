package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	templatesdomain "metaldocs/internal/modules/templates_v2/domain"
)

// FillInRepository manages document_placeholder_values rows.
type FillInRepository struct {
	db     *sql.DB
	schema string
}

type PlaceholderValue struct {
	TenantID        string
	RevisionID      string
	PlaceholderID   string
	ValueText       *string
	ValueTyped      map[string]any
	Source          string
	ComputedFrom    *string
	ResolverVersion *int
	InputsHash      []byte
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

func (r *FillInRepository) UpsertValue(ctx context.Context, v PlaceholderValue) error {
	var valueTyped any
	if v.ValueTyped != nil {
		b, err := json.Marshal(v.ValueTyped)
		if err != nil {
			return err
		}
		valueTyped = b
	}

	_, err := r.db.ExecContext(ctx, fmt.Sprintf(`
		INSERT INTO %s
		    (tenant_id, revision_id, placeholder_id, value_text, value_typed,
		     source, computed_from, resolver_version, inputs_hash, validated_at, created_at, updated_at)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW(), NOW())
		ON CONFLICT (tenant_id, revision_id, placeholder_id) DO UPDATE SET
			value_text       = EXCLUDED.value_text,
			value_typed      = EXCLUDED.value_typed,
			source           = EXCLUDED.source,
			computed_from    = EXCLUDED.computed_from,
			resolver_version = EXCLUDED.resolver_version,
			inputs_hash      = EXCLUDED.inputs_hash,
			validated_at     = NOW(),
			updated_at       = NOW()`, r.table("document_placeholder_values")),
		v.TenantID, v.RevisionID, v.PlaceholderID, v.ValueText, valueTyped,
		v.Source, v.ComputedFrom, v.ResolverVersion, v.InputsHash,
	)
	return err
}

func (r *FillInRepository) ListValues(ctx context.Context, tenantID, revisionID string) ([]PlaceholderValue, error) {
	rows, err := r.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT placeholder_id, value_text, value_typed, source, computed_from, resolver_version, inputs_hash
		  FROM %s
		 WHERE tenant_id=$1::uuid AND revision_id=$2::uuid`, r.table("document_placeholder_values")),
		tenantID, revisionID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []PlaceholderValue
	for rows.Next() {
		var v PlaceholderValue
		var valueTyped []byte
		if err := rows.Scan(
			&v.PlaceholderID,
			&v.ValueText,
			&valueTyped,
			&v.Source,
			&v.ComputedFrom,
			&v.ResolverVersion,
			&v.InputsHash,
		); err != nil {
			return nil, err
		}
		if len(valueTyped) > 0 {
			v.ValueTyped = map[string]any{}
			if err := json.Unmarshal(valueTyped, &v.ValueTyped); err != nil {
				return nil, err
			}
		}
		v.TenantID = tenantID
		v.RevisionID = revisionID
		out = append(out, v)
	}

	return out, rows.Err()
}
