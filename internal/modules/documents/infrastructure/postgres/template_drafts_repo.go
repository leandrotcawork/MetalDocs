package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"metaldocs/internal/modules/documents/domain"
)

// GetTemplateDraft fetches the draft for the given template key, or returns
// ErrTemplateDraftNotFound if no draft exists.
func (r *Repository) GetTemplateDraft(ctx context.Context, templateKey string) (*domain.TemplateDraft, error) {
	const q = `
SELECT template_key, profile_code, base_version, name,
       theme_json, meta_json, blocks_json,
       draft_status, published_html,
       lock_version, has_stripped_fields, stripped_fields_json,
       created_by, created_at, updated_at
FROM metaldocs.template_drafts
WHERE template_key = $1
`
	var d domain.TemplateDraft
	var themeJSON, metaJSON, blocksJSON []byte
	var strippedJSON []byte
	var draftStatus string
	var publishedHTML sql.NullString
	var strippedNull sql.NullString

	row := r.db.QueryRowContext(ctx, q, strings.TrimSpace(templateKey))
	if err := row.Scan(
		&d.TemplateKey,
		&d.ProfileCode,
		&d.BaseVersion,
		&d.Name,
		&themeJSON,
		&metaJSON,
		&blocksJSON,
		&draftStatus,
		&publishedHTML,
		&d.LockVersion,
		&d.HasStrippedFields,
		&strippedNull,
		&d.CreatedBy,
		&d.CreatedAt,
		&d.UpdatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrTemplateDraftNotFound
		}
		return nil, fmt.Errorf("get template draft: %w", err)
	}

	d.ThemeJSON = themeJSON
	d.MetaJSON = metaJSON
	d.BlocksJSON = blocksJSON
	d.DraftStatus = domain.TemplateStatus(draftStatus)
	if publishedHTML.Valid {
		s := publishedHTML.String
		d.PublishedHTML = &s
	}
	if strippedNull.Valid {
		strippedJSON = []byte(strippedNull.String)
	}
	d.StrippedFieldsJSON = strippedJSON

	return &d, nil
}

// UpsertTemplateDraftCAS saves draft using compare-and-swap on lock_version.
//
//   - expectedLockVersion == 0: INSERT new draft (lock_version starts at 1).
//   - expectedLockVersion > 0: UPDATE existing draft WHERE lock_version = expectedLockVersion.
//     Returns ErrTemplateLockConflict if affected rows == 0.
func (r *Repository) UpsertTemplateDraftCAS(ctx context.Context, draft *domain.TemplateDraft, expectedLockVersion int) (*domain.TemplateDraft, error) {
	key := strings.TrimSpace(draft.TemplateKey)

	if expectedLockVersion == 0 {
		// INSERT — first save for this key.
		const insertQ = `
INSERT INTO metaldocs.template_drafts (
    template_key, profile_code, base_version, name,
    theme_json, meta_json, blocks_json,
    lock_version, has_stripped_fields, stripped_fields_json,
    created_by, created_at, updated_at
) VALUES ($1, $2, $3, $4, $5::jsonb, $6::jsonb, $7::jsonb, 1, $8, $9::jsonb, $10, now(), now())
RETURNING lock_version, created_at, updated_at
`
		strippedParam := nullableJSON(draft.StrippedFieldsJSON)

		var lockVer int
		if err := r.db.QueryRowContext(ctx, insertQ,
			key,
			strings.TrimSpace(draft.ProfileCode),
			draft.BaseVersion,
			draft.Name,
			nullableJSONOrDefault(draft.ThemeJSON, "{}"),
			nullableJSONOrDefault(draft.MetaJSON, "{}"),
			jsonOrEmpty(draft.BlocksJSON),
			draft.HasStrippedFields,
			strippedParam,
			strings.TrimSpace(draft.CreatedBy),
		).Scan(&lockVer, &draft.CreatedAt, &draft.UpdatedAt); err != nil {
			if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23505" {
				return nil, domain.ErrTemplateLockConflict
			}
			return nil, fmt.Errorf("insert template draft: %w", err)
		}
		out := *draft
		out.TemplateKey = key
		out.LockVersion = lockVer
		return &out, nil
	}

	// UPDATE with CAS.
	const updateQ = `
UPDATE metaldocs.template_drafts
SET blocks_json          = $1::jsonb,
    theme_json           = $2::jsonb,
    meta_json            = $3::jsonb,
    name                 = $4,
    lock_version         = lock_version + 1,
    has_stripped_fields  = $5,
    stripped_fields_json = $6::jsonb,
    updated_at           = now()
WHERE template_key = $7 AND lock_version = $8
RETURNING lock_version, created_at, updated_at
`
	strippedParam := nullableJSON(draft.StrippedFieldsJSON)

	var newLockVer int
	err := r.db.QueryRowContext(ctx, updateQ,
		jsonOrEmpty(draft.BlocksJSON),
		nullableJSONOrDefault(draft.ThemeJSON, "{}"),
		nullableJSONOrDefault(draft.MetaJSON, "{}"),
		draft.Name,
		draft.HasStrippedFields,
		strippedParam,
		key,
		expectedLockVersion,
	).Scan(&newLockVer, &draft.CreatedAt, &draft.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			// Either key doesn't exist or lock_version mismatch.
			// Check which one so we can return the right sentinel.
			exists, checkErr := r.templateDraftExists(ctx, key)
			if checkErr != nil {
				return nil, fmt.Errorf("check template draft existence: %w", checkErr)
			}
			if !exists {
				return nil, domain.ErrTemplateDraftNotFound
			}
			return nil, domain.ErrTemplateLockConflict
		}
		return nil, fmt.Errorf("update template draft cas: %w", err)
	}

	out := *draft
	out.TemplateKey = key
	out.LockVersion = newLockVer
	return &out, nil
}

func (r *Repository) templateDraftExists(ctx context.Context, templateKey string) (bool, error) {
	const q = `SELECT 1 FROM metaldocs.template_drafts WHERE template_key = $1`
	var x int
	err := r.db.QueryRowContext(ctx, q, templateKey).Scan(&x)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// DeleteTemplateDraft removes the draft for the given template key.
func (r *Repository) DeleteTemplateDraft(ctx context.Context, templateKey string) error {
	const q = `DELETE FROM metaldocs.template_drafts WHERE template_key = $1`
	result, err := r.db.ExecContext(ctx, q, strings.TrimSpace(templateKey))
	if err != nil {
		return fmt.Errorf("delete template draft: %w", err)
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return domain.ErrTemplateDraftNotFound
	}
	return nil
}

func (r *Repository) UpdateTemplateDraftStatus(ctx context.Context, templateKey string, newStatus domain.TemplateStatus) error {
	const q = `UPDATE metaldocs.template_drafts SET draft_status = $1 WHERE template_key = $2`
	res, err := r.db.ExecContext(ctx, q, string(newStatus), templateKey)
	if err != nil {
		return fmt.Errorf("update template draft status: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrTemplateDraftNotFound
	}
	return nil
}

func (r *Repository) SetTemplateDraftPublished(ctx context.Context, templateKey string, publishedHTML string) error {
	const q = `UPDATE metaldocs.template_drafts SET draft_status = 'published', published_html = $1 WHERE template_key = $2`
	res, err := r.db.ExecContext(ctx, q, publishedHTML, templateKey)
	if err != nil {
		return fmt.Errorf("set template draft published: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrTemplateDraftNotFound
	}
	return nil
}

// UpdateTemplateVersionStatus sets the status column on the given template version.
func (r *Repository) UpdateTemplateVersionStatus(ctx context.Context, templateKey string, version int, status domain.TemplateStatus) error {
	const q = `
UPDATE metaldocs.document_template_versions
SET status = $1
WHERE template_key = $2 AND version = $3
`
	result, err := r.db.ExecContext(ctx, q, string(status), strings.TrimSpace(templateKey), version)
	if err != nil {
		return fmt.Errorf("update template version status: %w", err)
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return domain.ErrTemplateNotFound
	}
	return nil
}

// WriteTemplateAuditEvent appends an audit event to template_audit_log.
func (r *Repository) WriteTemplateAuditEvent(ctx context.Context, event domain.TemplateAuditEvent) error {
	const q = `
INSERT INTO metaldocs.template_audit_log (
    template_key, version, action, actor_id, diff_summary, trace_id
) VALUES ($1, $2, $3, $4, $5, $6)
`
	_, err := r.db.ExecContext(ctx, q,
		strings.TrimSpace(event.TemplateKey),
		event.Version,
		strings.TrimSpace(event.Action),
		strings.TrimSpace(event.ActorID),
		nullIfEmpty(event.DiffSummary),
		nullIfEmpty(event.TraceID),
	)
	if err != nil {
		return fmt.Errorf("write template audit event: %w", err)
	}
	return nil
}

// ListTemplateAuditEvents returns all audit events for a template key, ordered by creation time.
func (r *Repository) ListTemplateAuditEvents(ctx context.Context, templateKey string) ([]domain.TemplateAuditEvent, error) {
	const q = `
SELECT template_key, version, action, actor_id, diff_summary, trace_id
FROM metaldocs.template_audit_log
WHERE template_key = $1
ORDER BY created_at ASC
`
	rows, err := r.db.QueryContext(ctx, q, strings.TrimSpace(templateKey))
	if err != nil {
		return nil, fmt.Errorf("list template audit events: %w", err)
	}
	defer rows.Close()

	var events []domain.TemplateAuditEvent
	for rows.Next() {
		var e domain.TemplateAuditEvent
		var version sql.NullInt64
		var diffSummary, traceID sql.NullString
		if err := rows.Scan(
			&e.TemplateKey,
			&version,
			&e.Action,
			&e.ActorID,
			&diffSummary,
			&traceID,
		); err != nil {
			return nil, fmt.Errorf("scan template audit event: %w", err)
		}
		if version.Valid {
			v := int(version.Int64)
			e.Version = &v
		}
		if diffSummary.Valid {
			e.DiffSummary = diffSummary.String
		}
		if traceID.Valid {
			e.TraceID = traceID.String
		}
		events = append(events, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list template audit events rows: %w", err)
	}
	return events, nil
}

// InsertTemplateVersion inserts a new version row into document_template_versions.
func (r *Repository) InsertTemplateVersion(ctx context.Context, version domain.DocumentTemplateVersion) error {
	const q = `
		INSERT INTO metaldocs.document_template_versions
			(template_key, version, profile_code, schema_version, name, editor, content_format, body, definition, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::jsonb, $10, NOW())`
	def, err := json.Marshal(version.Definition)
	if err != nil {
		return fmt.Errorf("marshal definition: %w", err)
	}
	_, err = r.db.ExecContext(ctx, q,
		version.TemplateKey, version.Version, version.ProfileCode,
		version.SchemaVersion, version.Name, version.Editor,
		version.ContentFormat, version.Body, def, version.Status,
	)
	if err != nil {
		return fmt.Errorf("insert template version: %w", err)
	}
	return nil
}

// PublishTemplateAtomic inserts a published template version and deletes its draft in one transaction.
func (r *Repository) PublishTemplateAtomic(ctx context.Context, version *domain.DocumentTemplateVersion, draftKey domain.TemplateDraftKey) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx publish template: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	const insertVersionQ = `
		INSERT INTO metaldocs.document_template_versions
			(template_key, version, profile_code, schema_version, name, editor, content_format, body, definition, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::jsonb, $10, NOW())`
	def, err := json.Marshal(version.Definition)
	if err != nil {
		return fmt.Errorf("marshal definition: %w", err)
	}
	if _, err := tx.ExecContext(ctx, insertVersionQ,
		version.TemplateKey, version.Version, version.ProfileCode,
		version.SchemaVersion, version.Name, version.Editor,
		version.ContentFormat, version.Body, def, version.Status,
	); err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23505" {
			return fmt.Errorf("insert template version: %w", err)
		}
		return fmt.Errorf("insert template version: %w", err)
	}

	const deleteDraftQ = `DELETE FROM metaldocs.template_drafts WHERE template_key = $1`
	result, err := tx.ExecContext(ctx, deleteDraftQ, strings.TrimSpace(string(draftKey)))
	if err != nil {
		return fmt.Errorf("delete template draft: %w", err)
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return domain.ErrTemplateDraftNotFound
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx publish template: %w", err)
	}
	return nil
}

// nullableJSON returns nil if the slice is empty, or the raw JSON string otherwise.
func nullableJSON(b []byte) any {
	if len(b) == 0 {
		return nil
	}
	return string(b)
}

// nullableJSONOrDefault returns def if b is empty, else the raw JSON string.
func nullableJSONOrDefault(b []byte, def string) string {
	if len(b) == 0 {
		return def
	}
	return string(b)
}

// jsonOrEmpty returns "{}" if b is empty, else the raw JSON string.
func jsonOrEmpty(b []byte) string {
	if len(b) == 0 {
		return "{}"
	}
	return string(b)
}
