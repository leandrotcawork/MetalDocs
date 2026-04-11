package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/google/uuid"

	"metaldocs/internal/modules/documents/application"
)

type ReleaseRepo struct {
	db *sql.DB

	mu  sync.Mutex
	txs map[context.Context]*sql.Tx
}

func NewReleaseRepo(db *sql.DB) *ReleaseRepo {
	return &ReleaseRepo{
		db:  db,
		txs: make(map[context.Context]*sql.Tx),
	}
}

func (r *ReleaseRepo) GetDraft(ctx context.Context, id uuid.UUID) (*application.DraftSnapshot, error) {
	var snapshot application.DraftSnapshot
	var templateRef []byte
	err := r.db.QueryRowContext(ctx, `
		SELECT id, document_id, version_number, content_blocks, template_ref
		FROM metaldocs.document_versions_mddm
		WHERE id = $1 AND status IN ('draft', 'pending_approval')
	`, id).Scan(&snapshot.ID, &snapshot.DocumentID, &snapshot.VersionNumber, &snapshot.ContentBlocks, &templateRef)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("get draft %s: %w", id, sql.ErrNoRows)
	}
	if err != nil {
		return nil, fmt.Errorf("get draft %s: %w", id, err)
	}
	// MDDM versions always originate from the browser editor.
	snapshot.ContentSource = "browser_editor"
	// Parse template_ref JSONB to extract template key and version for pin capture.
	if len(templateRef) > 0 {
		var ref struct {
			TemplateKey     string `json:"template_key"`
			TemplateID      string `json:"template_id"` // legacy fallback
			TemplateVersion int    `json:"template_version"`
		}
		if err := json.Unmarshal(templateRef, &ref); err == nil {
			if ref.TemplateKey != "" {
				snapshot.TemplateKey = ref.TemplateKey
			} else {
				snapshot.TemplateKey = ref.TemplateID
			}
			snapshot.TemplateVersion = ref.TemplateVersion
		}
	}
	return &snapshot, nil
}

func (r *ReleaseRepo) ArchivePreviousReleased(ctx context.Context, documentID string) (uuid.UUID, []byte, []byte, error) {
	tx, err := r.beginOrReuseTx(ctx)
	if err != nil {
		return uuid.Nil, nil, nil, err
	}

	var prevID uuid.UUID
	var prevContentBlocks []byte
	var prevDocx []byte
	err = tx.QueryRowContext(ctx, `
		SELECT id, content_blocks, docx_bytes
		FROM metaldocs.document_versions_mddm
		WHERE document_id = $1 AND status = 'released'
	`, documentID).Scan(&prevID, &prevContentBlocks, &prevDocx)
	if errors.Is(err, sql.ErrNoRows) {
		return uuid.Nil, nil, nil, nil
	}
	if err != nil {
		return uuid.Nil, nil, nil, r.rollbackWithError(ctx, fmt.Errorf("archive previous released lookup for %s: %w", documentID, err))
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE metaldocs.document_versions_mddm
		SET status = 'archived', content_blocks = NULL
		WHERE id = $1
	`, prevID); err != nil {
		return uuid.Nil, nil, nil, r.rollbackWithError(ctx, fmt.Errorf("archive previous released %s: %w", prevID, err))
	}

	return prevID, prevContentBlocks, prevDocx, nil
}

func (r *ReleaseRepo) PromoteDraftToReleased(ctx context.Context, draftID uuid.UUID, docxBytes []byte, approvedBy string) error {
	tx, err := r.beginOrReuseTx(ctx)
	if err != nil {
		return err
	}

	res, err := tx.ExecContext(ctx, `
		UPDATE metaldocs.document_versions_mddm
		SET status = 'released', docx_bytes = $1, approved_at = now(), approved_by = $2
		WHERE id = $3 AND status IN ('draft', 'pending_approval')
	`, docxBytes, approvedBy, draftID)
	if err != nil {
		return r.rollbackWithError(ctx, fmt.Errorf("promote draft %s: %w", draftID, err))
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return r.rollbackWithError(ctx, fmt.Errorf("promote draft %s rows affected: %w", draftID, err))
	}
	if rows == 0 {
		return r.rollbackWithError(ctx, fmt.Errorf("promote draft %s: %w", draftID, sql.ErrNoRows))
	}

	return nil
}

func (r *ReleaseRepo) StoreRevisionDiff(ctx context.Context, versionID uuid.UUID, diff json.RawMessage) error {
	tx, err := r.beginOrReuseTx(ctx)
	if err != nil {
		return err
	}

	res, err := tx.ExecContext(ctx, `
		UPDATE metaldocs.document_versions_mddm
		SET revision_diff = $1
		WHERE id = $2
	`, diff, versionID)
	if err != nil {
		return r.rollbackWithError(ctx, fmt.Errorf("store revision diff for %s: %w", versionID, err))
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return r.rollbackWithError(ctx, fmt.Errorf("store revision diff %s rows affected: %w", versionID, err))
	}
	if rows == 0 {
		return r.rollbackWithError(ctx, fmt.Errorf("store revision diff %s: %w", versionID, sql.ErrNoRows))
	}

	return nil
}

func (r *ReleaseRepo) DeleteImageRefs(ctx context.Context, versionID uuid.UUID) error {
	tx, err := r.beginOrReuseTx(ctx)
	if err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `
		DELETE FROM metaldocs.document_version_images
		WHERE document_version_id = $1
	`, versionID); err != nil {
		return r.rollbackWithError(ctx, fmt.Errorf("delete image refs for %s: %w", versionID, err))
	}

	return nil
}

func (r *ReleaseRepo) CleanupOrphanImages(ctx context.Context) error {
	tx, err := r.beginOrReuseTx(ctx)
	if err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `
		DELETE FROM metaldocs.document_images
		WHERE NOT EXISTS (
			SELECT 1
			FROM metaldocs.document_version_images
			WHERE image_id = document_images.id
		)
	`); err != nil {
		return r.rollbackWithError(ctx, fmt.Errorf("cleanup orphan images: %w", err))
	}

	if err := r.commitActiveTx(ctx); err != nil {
		return fmt.Errorf("commit release transaction: %w", err)
	}

	return nil
}

func (r *ReleaseRepo) beginOrReuseTx(ctx context.Context) (*sql.Tx, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if tx, ok := r.txs[ctx]; ok {
		return tx, nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin release transaction: %w", err)
	}

	r.txs[ctx] = tx
	return tx, nil
}

func (r *ReleaseRepo) rollbackWithError(ctx context.Context, err error) error {
	r.rollbackActiveTx(ctx)
	return err
}

func (r *ReleaseRepo) rollbackActiveTx(ctx context.Context) {
	r.mu.Lock()
	tx := r.txs[ctx]
	delete(r.txs, ctx)
	r.mu.Unlock()

	if tx != nil {
		_ = tx.Rollback()
	}
}

func (r *ReleaseRepo) commitActiveTx(ctx context.Context) error {
	r.mu.Lock()
	tx := r.txs[ctx]
	delete(r.txs, ctx)
	r.mu.Unlock()

	if tx == nil {
		return nil
	}

	return tx.Commit()
}

var _ application.ReleaseRepo = (*ReleaseRepo)(nil)
