package postgres

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
)

type ImageReconciler struct {
	db *sql.DB
}

func NewImageReconciler(db *sql.DB) *ImageReconciler {
	return &ImageReconciler{db: db}
}

// Reconcile replaces the document_version_images entries for this version
// with exactly the given imageIDs, in a single transaction.
// Implementation: delete all existing entries for the version, then insert the new ones.
func (r *ImageReconciler) Reconcile(ctx context.Context, versionID uuid.UUID, imageIDs []uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete all existing refs for this version
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM metaldocs.document_version_images
		WHERE document_version_id = $1
	`, versionID); err != nil {
		return err
	}

	// Insert the new set
	for _, id := range imageIDs {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO metaldocs.document_version_images (document_version_id, image_id)
			VALUES ($1, $2)
			ON CONFLICT DO NOTHING
		`, versionID, id); err != nil {
			return err
		}
	}

	return tx.Commit()
}
