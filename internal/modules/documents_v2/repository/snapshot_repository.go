package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"metaldocs/internal/modules/documents_v2/domain"
)

// SnapshotRepository reads and writes the template snapshot columns on documents.
type SnapshotRepository struct {
	db     *sql.DB
	schema string // optional schema prefix; empty = bare table name
}

// NewSnapshotRepository creates a SnapshotRepository using bare table names.
// In tests, use NewSnapshotRepositoryWithSchema to point at the isolated test schema.
func NewSnapshotRepository(db *sql.DB) *SnapshotRepository {
	return &SnapshotRepository{db: db}
}

// NewSnapshotRepositoryWithSchema creates a SnapshotRepository that qualifies
// table names with the given schema. Used by integration tests.
func NewSnapshotRepositoryWithSchema(db *sql.DB, schema string) *SnapshotRepository {
	return &SnapshotRepository{db: db, schema: schema}
}

func (r *SnapshotRepository) table(name string) string {
	if r.schema == "" {
		return name
	}
	return fmt.Sprintf("%q.%q", r.schema, name)
}

// WriteSnapshot updates the snapshot columns for a document.
func (r *SnapshotRepository) WriteSnapshot(ctx context.Context, tenantID, docID string, s domain.TemplateSnapshot) error {
	h := s.Hashes()
	_, err := r.db.ExecContext(ctx, fmt.Sprintf(`
		UPDATE %s
		   SET placeholder_schema_snapshot    = $1,
		       placeholder_schema_hash        = $2,
		       composition_config_snapshot    = $3,
		       composition_config_hash        = $4,
		       editable_zones_schema_snapshot = $5,
		       body_docx_snapshot_s3_key      = $6,
		       body_docx_hash                 = $7
		 WHERE tenant_id = $8::uuid AND id = $9::uuid`, r.table("documents")),
		s.PlaceholderSchemaJSON, h.PlaceholderSchemaHash,
		s.CompositionJSON, h.CompositionHash,
		s.ZonesSchemaJSON,
		s.BodyDocxS3Key, h.BodyDocxHash,
		tenantID, docID,
	)
	return err
}

// ReadSnapshot reads the snapshot columns for a document.
func (r *SnapshotRepository) ReadSnapshot(ctx context.Context, tenantID, docID string) (domain.TemplateSnapshot, error) {
	var s domain.TemplateSnapshot
	err := r.db.QueryRowContext(ctx, fmt.Sprintf(`
		SELECT placeholder_schema_snapshot,
		       composition_config_snapshot,
		       editable_zones_schema_snapshot,
		       coalesce(body_docx_snapshot_s3_key, '')
		  FROM %s
		 WHERE tenant_id = $1::uuid AND id = $2::uuid`, r.table("documents")),
		tenantID, docID,
	).Scan(
		&s.PlaceholderSchemaJSON,
		&s.CompositionJSON,
		&s.ZonesSchemaJSON,
		&s.BodyDocxS3Key,
	)
	return s, err
}

func (r *SnapshotRepository) WriteFreeze(ctx context.Context, tenant, docID string, valuesHash []byte, frozenAt time.Time) error {
	_, err := r.db.ExecContext(ctx, fmt.Sprintf(`
        UPDATE %s
           SET values_hash=$1, values_frozen_at=$2
         WHERE tenant_id=$3 AND id=$4`, r.table("documents")),
		valuesHash, frozenAt, tenant, docID)
	return err
}

// WriteFinalDocx persists the fanout output pointer and content hash onto a document.
func (r *SnapshotRepository) WriteFinalDocx(ctx context.Context, tenant, docID, s3Key string, contentHash []byte) error {
	_, err := r.db.ExecContext(ctx, fmt.Sprintf(`
        UPDATE %s
           SET final_docx_s3_key=$1, content_hash=$2
         WHERE tenant_id=$3::uuid AND id=$4::uuid`, r.table("documents")),
		s3Key, contentHash, tenant, docID)
	return err
}
