package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"metaldocs/internal/modules/documents_v2/domain"
)

type Repository struct {
	db *sql.DB
}

func New(db *sql.DB) *Repository { return &Repository{db: db} }

// CreateDocument inserts document + initial session + initial revision in one
// deferrable-FK transaction. The initial revision's storage_key is empty - the
// caller uploads the .docx to the final content-addressed key via
// Presigner.AdoptTempObject, then calls SetRevisionStorageKey to finalize.
func (r *Repository) CreateDocument(ctx context.Context, d *domain.Document, initialContentHash string) (docID, revID, sessionID string, err error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return "", "", "", err
	}
	defer tx.Rollback()

	// Deferrable FKs allow inserting doc -> session -> revision in any order in tx.
	if err := tx.QueryRowContext(ctx,
		`INSERT INTO documents (tenant_id, template_version_id, name, status, form_data_json, created_by)
		 VALUES ($1, $2, $3, 'draft', $4, $5) RETURNING id`,
		d.TenantID, d.TemplateVersionID, d.Name, d.FormDataJSON, d.CreatedBy,
	).Scan(&docID); err != nil {
		return "", "", "", fmt.Errorf("insert document: %w", err)
	}

	if err := tx.QueryRowContext(ctx,
		`INSERT INTO editor_sessions (document_id, user_id, expires_at, last_acknowledged_revision_id, status)
		 VALUES ($1, $2, now() + interval '5 minutes', '00000000-0000-0000-0000-000000000000', 'active') RETURNING id`,
		docID, d.CreatedBy,
	).Scan(&sessionID); err != nil {
		return "", "", "", fmt.Errorf("insert session: %w", err)
	}

	if err := tx.QueryRowContext(ctx,
		`INSERT INTO document_revisions (document_id, parent_revision_id, session_id, storage_key, content_hash, form_data_snapshot)
		 VALUES ($1, NULL, $2, '', $3, $4) RETURNING id`,
		docID, sessionID, initialContentHash, d.FormDataJSON,
	).Scan(&revID); err != nil {
		return "", "", "", fmt.Errorf("insert revision: %w", err)
	}

	if _, err := tx.ExecContext(ctx,
		`UPDATE editor_sessions SET last_acknowledged_revision_id = $1 WHERE id = $2`,
		revID, sessionID,
	); err != nil {
		return "", "", "", fmt.Errorf("update session ack: %w", err)
	}

	if _, err := tx.ExecContext(ctx,
		`UPDATE documents SET current_revision_id = $1, active_session_id = $2, updated_at = now() WHERE id = $3`,
		revID, sessionID, docID,
	); err != nil {
		return "", "", "", fmt.Errorf("update document pointers: %w", err)
	}

	return docID, revID, sessionID, tx.Commit()
}

// SetRevisionStorageKey finalizes the initial revision's storage_key after the
// .docx has been copied to its content-addressed final key. Idempotent:
// succeeds only while storage_key is still empty.
func (r *Repository) SetRevisionStorageKey(ctx context.Context, revID, storageKey string) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE document_revisions SET storage_key = $1 WHERE id = $2 AND storage_key = ''`,
		storageKey, revID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("revision %s already has storage_key set", revID)
	}
	return nil
}

func (r *Repository) GetDocument(ctx context.Context, tenantID, id string) (*domain.Document, error) {
	var d domain.Document
	err := r.db.QueryRowContext(ctx,
		`SELECT id, tenant_id, template_version_id, name, status, form_data_json,
		        coalesce(current_revision_id::text, ''), coalesce(active_session_id::text, ''),
		        finalized_at, archived_at, created_at, updated_at, created_by
		 FROM documents WHERE id=$1 AND tenant_id=$2`, id, tenantID,
	).Scan(&d.ID, &d.TenantID, &d.TemplateVersionID, &d.Name, &d.Status, &d.FormDataJSON,
		&d.CurrentRevisionID, &d.ActiveSessionID, &d.FinalizedAt, &d.ArchivedAt,
		&d.CreatedAt, &d.UpdatedAt, &d.CreatedBy)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *Repository) ListDocuments(ctx context.Context, tenantID string) ([]domain.Document, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, tenant_id, template_version_id, name, status, form_data_json,
		        coalesce(current_revision_id::text, ''), coalesce(active_session_id::text, ''),
		        finalized_at, archived_at, created_at, updated_at, created_by
		 FROM documents WHERE tenant_id=$1 ORDER BY updated_at DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Document{}
	for rows.Next() {
		var d domain.Document
		if err := rows.Scan(&d.ID, &d.TenantID, &d.TemplateVersionID, &d.Name, &d.Status, &d.FormDataJSON,
			&d.CurrentRevisionID, &d.ActiveSessionID, &d.FinalizedAt, &d.ArchivedAt,
			&d.CreatedAt, &d.UpdatedAt, &d.CreatedBy); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// ListDocumentsForUser restricts metadata leakage for document_filler role -
// returns only docs the actor created. Admins / template_* roles use the
// unrestricted ListDocuments path instead.
func (r *Repository) ListDocumentsForUser(ctx context.Context, tenantID, userID string) ([]domain.Document, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, tenant_id, template_version_id, name, status, form_data_json,
		        coalesce(current_revision_id::text, ''), coalesce(active_session_id::text, ''),
		        finalized_at, archived_at, created_at, updated_at, created_by
		 FROM documents WHERE tenant_id=$1 AND created_by=$2 ORDER BY updated_at DESC`, tenantID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Document{}
	for rows.Next() {
		var d domain.Document
		if err := rows.Scan(&d.ID, &d.TenantID, &d.TemplateVersionID, &d.Name, &d.Status, &d.FormDataJSON,
			&d.CurrentRevisionID, &d.ActiveSessionID, &d.FinalizedAt, &d.ArchivedAt,
			&d.CreatedAt, &d.UpdatedAt, &d.CreatedBy); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (r *Repository) UpdateDocumentStatus(ctx context.Context, tenantID, id string, cur, next domain.DocumentStatus, stampTime bool) error {
	col := ""
	if stampTime {
		if next == domain.DocStatusFinalized {
			col = "finalized_at = now(),"
		}
		if next == domain.DocStatusArchived {
			col = "archived_at  = now(),"
		}
	}
	res, err := r.db.ExecContext(ctx,
		fmt.Sprintf(`UPDATE documents SET status=$1, %s updated_at=now() WHERE id=$2 AND tenant_id=$3 AND status=$4`, col),
		next, id, tenantID, cur)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrInvalidStateTransition
	}
	return nil
}

// AcquireSession attempts to claim the single active-session slot for a doc.
// Relies on partial unique index idx_one_active_session_per_doc.
// Returns (newSession, nil) on success. Returns existing active session
// with ErrSessionTaken if another user holds it.
func (r *Repository) AcquireSession(ctx context.Context, tenantID, docID, userID string) (*domain.Session, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var existingID, existingUser, existingStatus string
	err = tx.QueryRowContext(ctx,
		`SELECT id::text, user_id::text, status FROM editor_sessions
		 WHERE document_id=$1 AND status='active' FOR UPDATE`, docID,
	).Scan(&existingID, &existingUser, &existingStatus)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if err == nil {
		// Caller already holds it - refresh.
		if existingUser == userID {
			if _, err := tx.ExecContext(ctx, `UPDATE editor_sessions SET expires_at = now() + interval '5 minutes' WHERE id=$1`, existingID); err != nil {
				return nil, err
			}
			s := &domain.Session{ID: existingID, DocumentID: docID, UserID: userID, Status: domain.SessionActive}
			return s, tx.Commit()
		}
		// Someone else owns it.
		s := &domain.Session{ID: existingID, DocumentID: docID, UserID: existingUser, Status: domain.SessionActive}
		return s, domain.ErrSessionTaken
	}

	// No active session. Grab current_revision_id from documents.
	var curRev string
	if err := tx.QueryRowContext(ctx,
		`SELECT coalesce(current_revision_id::text,'') FROM documents WHERE id=$1 AND tenant_id=$2`, docID, tenantID,
	).Scan(&curRev); err != nil {
		return nil, err
	}
	if curRev == "" {
		return nil, fmt.Errorf("document has no current revision")
	}

	var newID string
	if err := tx.QueryRowContext(ctx,
		`INSERT INTO editor_sessions (document_id, user_id, expires_at, last_acknowledged_revision_id, status)
		 VALUES ($1, $2, now() + interval '5 minutes', $3, 'active') RETURNING id`,
		docID, userID, curRev,
	).Scan(&newID); err != nil {
		return nil, err
	}

	if _, err := tx.ExecContext(ctx, `UPDATE documents SET active_session_id=$1, updated_at=now() WHERE id=$2`, newID, docID); err != nil {
		return nil, err
	}

	return &domain.Session{ID: newID, DocumentID: docID, UserID: userID, LastAcknowledgedRevisionID: curRev, Status: domain.SessionActive}, tx.Commit()
}

func (r *Repository) HeartbeatSession(ctx context.Context, sessionID, userID string) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE editor_sessions SET expires_at = now() + interval '5 minutes'
		 WHERE id=$1 AND user_id=$2 AND status='active'`, sessionID, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrSessionInactive
	}
	return nil
}

func (r *Repository) ReleaseSession(ctx context.Context, sessionID, userID string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx,
		`UPDATE editor_sessions SET status='released', released_at=now()
		 WHERE id=$1 AND user_id=$2 AND status='active'`, sessionID, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrSessionInactive
	}
	if _, err := tx.ExecContext(ctx, `UPDATE documents SET active_session_id=NULL, updated_at=now() WHERE active_session_id=$1`, sessionID); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *Repository) ForceReleaseSession(ctx context.Context, sessionID string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx,
		`UPDATE editor_sessions SET status='force_released', released_at=now()
		 WHERE id=$1 AND status='active'`, sessionID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrSessionInactive
	}
	if _, err := tx.ExecContext(ctx, `UPDATE documents SET active_session_id=NULL, updated_at=now() WHERE active_session_id=$1`, sessionID); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *Repository) ExpireStaleSessions(ctx context.Context, now time.Time) (int, error) {
	// Single atomic CTE: expire sessions and clear document pointers in one tx.
	var n int
	err := r.db.QueryRowContext(ctx, `
		WITH expired AS (
			UPDATE editor_sessions SET status='expired'
			WHERE status='active' AND expires_at < $1
			RETURNING id
		)
		UPDATE documents SET active_session_id=NULL, updated_at=now()
		WHERE active_session_id IN (SELECT id FROM expired)
		RETURNING (SELECT count(*) FROM expired)`, now,
	).Scan(&n)
	if err != nil {
		return 0, err
	}
	return n, nil
}

func (r *Repository) PresignReserve(ctx context.Context, sessionID, userID, docID, baseRevisionID, contentHash, storageKey string, expiresAt time.Time) (pendingID string, err error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	// Verify session active, holder matches, and session points at base.
	var sessUser, sessDoc, sessAck, sessStatus string
	err = tx.QueryRowContext(ctx,
		`SELECT user_id::text, document_id::text, last_acknowledged_revision_id::text, status
		 FROM editor_sessions WHERE id=$1 FOR UPDATE`, sessionID,
	).Scan(&sessUser, &sessDoc, &sessAck, &sessStatus)
	if err != nil {
		return "", err
	}
	if sessStatus != string(domain.SessionActive) {
		return "", domain.ErrSessionInactive
	}
	if sessUser != userID || sessDoc != docID {
		return "", domain.ErrSessionNotHolder
	}
	if sessAck != baseRevisionID {
		return "", domain.ErrStaleBase
	}

	// Idempotent on (session, base, hash): ON CONFLICT returns existing row's id.
	err = tx.QueryRowContext(ctx,
		`INSERT INTO autosave_pending_uploads
		   (session_id, document_id, base_revision_id, content_hash, storage_key, expires_at)
		 VALUES ($1,$2,$3,$4,$5,$6)
		 ON CONFLICT (session_id, base_revision_id, content_hash)
		 DO UPDATE SET presigned_at = autosave_pending_uploads.presigned_at
		 RETURNING id`,
		sessionID, docID, baseRevisionID, contentHash, storageKey, expiresAt,
	).Scan(&pendingID)
	if err != nil {
		return "", err
	}
	return pendingID, tx.Commit()
}

// CommitResult + PendingCommitMeta + RestoreResult are mirrored in application
// (same-shape type aliases) so handlers depend only on application types.
type CommitResult struct {
	RevisionID      string
	RevisionNum     int64
	AlreadyConsumed bool
}

type PendingCommitMeta struct {
	SessionID           string
	DocumentID          string
	BaseRevisionID      string
	ExpectedContentHash string
	StorageKey          string
	ExpiresAt           time.Time
	ConsumedAt          *time.Time
}

type RestoreResult struct {
	NewRevisionID   string
	NewRevisionNum  int64
	CheckpointRevID string
	// Idempotent is true when ON CONFLICT (document_id, content_hash) DO
	// UPDATE SET id = id fired - the current head already matched the
	// checkpoint hash, so no new row was inserted. Used by the handler to
	// surface `idempotent: true` on the restore response.
	Idempotent bool
}

// GetPendingForCommit returns the minimal metadata the service needs before
// performing server-authoritative hash verification. Short, unlocked read;
// CommitUpload re-locks and re-checks under FOR UPDATE.
func (r *Repository) GetPendingForCommit(ctx context.Context, pendingID string) (*PendingCommitMeta, error) {
	var m PendingCommitMeta
	err := r.db.QueryRowContext(ctx,
		`SELECT session_id::text, document_id::text, base_revision_id::text,
		        content_hash, storage_key, expires_at, consumed_at
		 FROM autosave_pending_uploads WHERE id=$1`, pendingID,
	).Scan(&m.SessionID, &m.DocumentID, &m.BaseRevisionID,
		&m.ExpectedContentHash, &m.StorageKey, &m.ExpiresAt, &m.ConsumedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrPendingNotFound
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// CommitUpload encodes every DB-level rejection branch below as explicit errors.
// Callers translate each to the matching HTTP status (404/409/410/422 per spec).
// Server-authoritative content_hash verification happens in Service BEFORE this
// method is called; `serverComputedHash` is the hash the service streamed from
// S3 and therefore trusted. CommitUpload still compares it to pending.content_hash
// under FOR UPDATE to catch TOCTOU races.
func (r *Repository) CommitUpload(ctx context.Context, sessionID, userID, docID, pendingID, serverComputedHash string, formDataSnapshot []byte) (*CommitResult, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Lock pending row.
	var p domain.PendingUpload
	err = tx.QueryRowContext(ctx,
		`SELECT id::text, session_id::text, document_id::text, base_revision_id::text, content_hash,
		        storage_key, expires_at, consumed_at
		 FROM autosave_pending_uploads WHERE id=$1 FOR UPDATE`, pendingID,
	).Scan(&p.ID, &p.SessionID, &p.DocumentID, &p.BaseRevisionID, &p.ContentHash, &p.StorageKey, &p.ExpiresAt, &p.ConsumedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrPendingNotFound
	}
	if err != nil {
		return nil, err
	}

	if p.SessionID != sessionID || p.DocumentID != docID {
		return nil, domain.ErrMisbound
	}
	if p.ConsumedAt != nil {
		// Idempotent replay - look up the revision previously created for this pending.
		var rid string
		var rnum int64
		if err := tx.QueryRowContext(ctx,
			`SELECT id::text, revision_num FROM document_revisions
			 WHERE document_id=$1 AND content_hash=$2`, docID, p.ContentHash,
		).Scan(&rid, &rnum); err != nil {
			return nil, fmt.Errorf("replay lookup: %w", err)
		}
		return &CommitResult{RevisionID: rid, RevisionNum: rnum, AlreadyConsumed: true}, tx.Commit()
	}
	if time.Now().After(p.ExpiresAt) {
		return nil, domain.ErrExpiredUpload
	}

	// Re-verify session still active + holder + ack still matches base.
	var sessUser, sessAck, sessStatus string
	err = tx.QueryRowContext(ctx,
		`SELECT user_id::text, last_acknowledged_revision_id::text, status
		 FROM editor_sessions WHERE id=$1 FOR UPDATE`, sessionID,
	).Scan(&sessUser, &sessAck, &sessStatus)
	if err != nil {
		return nil, err
	}
	if sessStatus != string(domain.SessionActive) {
		return nil, domain.ErrSessionInactive
	}
	if sessUser != userID {
		return nil, domain.ErrSessionNotHolder
	}
	if sessAck != p.BaseRevisionID {
		return nil, domain.ErrStaleBase
	}

	// TOCTOU guard: service verified S3 hash matches pending.content_hash moments
	// before this call, but a concurrent tx could have rewritten the pending row.
	// Re-check under lock.
	if serverComputedHash != p.ContentHash {
		return nil, domain.ErrContentHashMismatch
	}

	var revID string
	var revNum int64
	if err := tx.QueryRowContext(ctx,
		`INSERT INTO document_revisions
		   (document_id, parent_revision_id, session_id, storage_key, content_hash, form_data_snapshot)
		 VALUES ($1,$2,$3,$4,$5,$6) RETURNING id::text, revision_num`,
		docID, p.BaseRevisionID, sessionID, p.StorageKey, p.ContentHash, formDataSnapshot,
	).Scan(&revID, &revNum); err != nil {
		return nil, fmt.Errorf("insert revision: %w", err)
	}

	if _, err := tx.ExecContext(ctx,
		`UPDATE documents SET current_revision_id=$1, form_data_json=$2, updated_at=now() WHERE id=$3`,
		revID, formDataSnapshot, docID,
	); err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE editor_sessions SET last_acknowledged_revision_id=$1 WHERE id=$2`, revID, sessionID,
	); err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE autosave_pending_uploads SET consumed_at=now() WHERE id=$1`, pendingID,
	); err != nil {
		return nil, err
	}

	return &CommitResult{RevisionID: revID, RevisionNum: revNum}, tx.Commit()
}

func (r *Repository) DeleteExpiredPending(ctx context.Context, olderThan time.Time) (int, error) {
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM autosave_pending_uploads WHERE expires_at < $1 AND consumed_at IS NULL`,
		olderThan)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

func (r *Repository) CreateCheckpoint(ctx context.Context, docID, actorUserID, label string) (*domain.Checkpoint, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var revID string
	if err := tx.QueryRowContext(ctx,
		`SELECT current_revision_id::text FROM documents WHERE id=$1 FOR UPDATE`, docID,
	).Scan(&revID); err != nil {
		return nil, err
	}
	if revID == "" {
		return nil, fmt.Errorf("document has no current revision")
	}

	var nextVer int
	if err := tx.QueryRowContext(ctx,
		`SELECT coalesce(max(version_num),0)+1 FROM document_checkpoints WHERE document_id=$1`, docID,
	).Scan(&nextVer); err != nil {
		return nil, err
	}

	cp := &domain.Checkpoint{DocumentID: docID, RevisionID: revID, VersionNum: nextVer, Label: label, CreatedBy: actorUserID}
	if err := tx.QueryRowContext(ctx,
		`INSERT INTO document_checkpoints (document_id, revision_id, version_num, label, created_by)
		 VALUES ($1,$2,$3,$4,$5) RETURNING id::text, created_at`,
		docID, revID, nextVer, label, actorUserID,
	).Scan(&cp.ID, &cp.CreatedAt); err != nil {
		return nil, err
	}

	return cp, tx.Commit()
}

func (r *Repository) ListCheckpoints(ctx context.Context, docID string) ([]domain.Checkpoint, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id::text, document_id::text, revision_id::text, version_num, coalesce(label,''), created_at, created_by::text
		 FROM document_checkpoints WHERE document_id=$1 ORDER BY version_num DESC`, docID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Checkpoint{}
	for rows.Next() {
		var c domain.Checkpoint
		if err := rows.Scan(&c.ID, &c.DocumentID, &c.RevisionID, &c.VersionNum, &c.Label, &c.CreatedAt, &c.CreatedBy); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *Repository) GetRevision(ctx context.Context, docID, revID string) (*domain.Revision, error) {
	var rv domain.Revision
	err := r.db.QueryRowContext(ctx,
		`SELECT id::text, document_id::text, revision_num, coalesce(parent_revision_id::text,''), session_id::text, storage_key, content_hash, form_data_snapshot, created_at
		 FROM document_revisions WHERE id=$1 AND document_id=$2`, revID, docID,
	).Scan(&rv.ID, &rv.DocumentID, &rv.RevisionNum, &rv.ParentRevisionID, &rv.SessionID, &rv.StorageKey, &rv.ContentHash, &rv.FormDataSnapshot, &rv.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &rv, nil
}

// RestoreCheckpoint is forward-only: it resolves the checkpoint's revision,
// copies that revision's storage_key + content_hash + form_data_snapshot into
// a NEW revision appended to head (parent = current_revision_id). It never
// rewrites history or rewinds revision_num. Session holder/active checks apply
// - restore is a session-authoritative action equivalent to a large autosave.
func (r *Repository) RestoreCheckpoint(ctx context.Context, docID, actorUserID string, versionNum int) (*RestoreResult, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Lock document + require active session held by actor.
	var sessID, sessUser, sessStatus string
	var curRev string
	if err := tx.QueryRowContext(ctx,
		`SELECT coalesce(active_session_id::text,''), coalesce(current_revision_id::text,'')
		 FROM documents WHERE id=$1 FOR UPDATE`, docID,
	).Scan(&sessID, &curRev); err != nil {
		return nil, err
	}
	if sessID == "" {
		return nil, domain.ErrSessionInactive
	}
	if err := tx.QueryRowContext(ctx,
		`SELECT user_id::text, status FROM editor_sessions WHERE id=$1 FOR UPDATE`, sessID,
	).Scan(&sessUser, &sessStatus); err != nil {
		return nil, err
	}
	if sessStatus != string(domain.SessionActive) {
		return nil, domain.ErrSessionInactive
	}
	if sessUser != actorUserID {
		return nil, domain.ErrSessionNotHolder
	}

	// Resolve checkpoint.
	var cpRevID, cpStorageKey, cpContentHash string
	var cpFormData []byte
	err = tx.QueryRowContext(ctx,
		`SELECT cp.revision_id::text, r.storage_key, r.content_hash, r.form_data_snapshot
		 FROM document_checkpoints cp
		 JOIN document_revisions r ON r.id = cp.revision_id
		 WHERE cp.document_id=$1 AND cp.version_num=$2`, docID, versionNum,
	).Scan(&cpRevID, &cpStorageKey, &cpContentHash, &cpFormData)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrCheckpointNotFound
	}
	if err != nil {
		return nil, err
	}

	// Append a new head revision pointing at the checkpoint's storage_key. The
	// content_hash is reused (content-addressed storage means no new upload).
	// NOTE: document_revisions.UNIQUE (document_id, content_hash) means restoring
	// to a hash that already exists as head is a no-op - ON CONFLICT returns the
	// existing row (xmax != 0 in pg). We detect that via `xmax::text::bigint <> 0`
	// on the RETURNING row to set RestoreResult.Idempotent = true.
	var newRevID string
	var newRevNum int64
	var idempotent bool
	err = tx.QueryRowContext(ctx,
		`INSERT INTO document_revisions
		   (document_id, parent_revision_id, session_id, storage_key, content_hash, form_data_snapshot)
		 VALUES ($1,$2,$3,$4,$5,$6)
		 ON CONFLICT (document_id, content_hash)
		 DO UPDATE SET id = document_revisions.id
		 RETURNING id::text, revision_num, (xmax::text::bigint <> 0)`,
		docID, curRev, sessID, cpStorageKey, cpContentHash, cpFormData,
	).Scan(&newRevID, &newRevNum, &idempotent)
	if err != nil {
		return nil, fmt.Errorf("restore insert: %w", err)
	}

	// On idempotent (head already equals checkpoint hash): do NOT rewrite
	// documents.current_revision_id or the session ack - they already match.
	// On fresh insert: advance head + session ack.
	if !idempotent {
		if _, err := tx.ExecContext(ctx,
			`UPDATE documents SET current_revision_id=$1, form_data_json=$2, updated_at=now() WHERE id=$3`,
			newRevID, cpFormData, docID,
		); err != nil {
			return nil, err
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE editor_sessions SET last_acknowledged_revision_id=$1 WHERE id=$2`, newRevID, sessID,
		); err != nil {
			return nil, err
		}
	}

	return &RestoreResult{
		NewRevisionID:   newRevID,
		NewRevisionNum:  newRevNum,
		CheckpointRevID: cpRevID,
		Idempotent:      idempotent,
	}, tx.Commit()
}

// IsDocumentOwner returns true iff the document was created by userID under
// tenantID. Used by handler-level defense-in-depth for document_filler routes.
func (r *Repository) IsDocumentOwner(ctx context.Context, tenantID, docID, userID string) (bool, error) {
	var ok bool
	err := r.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM documents WHERE id=$1 AND tenant_id=$2 AND created_by=$3)`,
		docID, tenantID, userID,
	).Scan(&ok)
	return ok, err
}
