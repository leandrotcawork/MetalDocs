package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"metaldocs/internal/modules/documents/domain/mddm"
)

type PostgresByteaStorage struct {
	db *sql.DB
}

func NewPostgresByteaStorage(db *sql.DB) *PostgresByteaStorage {
	return &PostgresByteaStorage{db: db}
}

func (s *PostgresByteaStorage) Put(ctx context.Context, sha256 string, mimeType string, bytes []byte) (uuid.UUID, error) {
	var id uuid.UUID
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO metaldocs.document_images (sha256, mime_type, byte_size, bytes)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (sha256) DO UPDATE SET sha256 = EXCLUDED.sha256
		RETURNING id
	`, sha256, mimeType, len(bytes), bytes).Scan(&id)
	if err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

func (s *PostgresByteaStorage) Get(ctx context.Context, id uuid.UUID) ([]byte, string, error) {
	var bytes []byte
	var mimeType string
	err := s.db.QueryRowContext(ctx, `
		SELECT bytes, mime_type FROM metaldocs.document_images WHERE id = $1
	`, id).Scan(&bytes, &mimeType)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, "", mddm.ErrImageNotFound
	}
	if err != nil {
		return nil, "", err
	}
	return bytes, mimeType, nil
}

func (s *PostgresByteaStorage) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM metaldocs.document_images WHERE id = $1`, id)
	return err
}

func (s *PostgresByteaStorage) Exists(ctx context.Context, sha256 string) (uuid.UUID, bool, error) {
	var id uuid.UUID
	err := s.db.QueryRowContext(ctx, `SELECT id FROM metaldocs.document_images WHERE sha256 = $1`, sha256).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return uuid.Nil, false, nil
	}
	if err != nil {
		return uuid.Nil, false, err
	}
	return id, true, nil
}

var _ mddm.ImageStorage = (*PostgresByteaStorage)(nil)
