package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"metaldocs/internal/modules/iam/domain"
)

type UserAreaRepository struct {
	db *sql.DB
}

func NewUserAreaRepository(db *sql.DB) *UserAreaRepository {
	return &UserAreaRepository{db: db}
}

func (r *UserAreaRepository) ListActive(ctx context.Context, userID, tenantID string, now time.Time) ([]domain.UserProcessArea, error) {
	const q = `
SELECT user_id, tenant_id::text, area_code, role, effective_from, effective_to, granted_by
FROM user_process_areas
WHERE user_id = $1
  AND tenant_id::text = $2
  AND effective_from <= $3
  AND (effective_to IS NULL OR effective_to > $3)
ORDER BY area_code ASC, effective_from DESC
`
	rows, err := r.db.QueryContext(ctx, q, userID, tenantID, now)
	if err != nil {
		return nil, fmt.Errorf("query active user process areas: %w", err)
	}
	defer rows.Close()

	result := make([]domain.UserProcessArea, 0, 8)
	for rows.Next() {
		item, err := scanUserProcessArea(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate active user process areas: %w", err)
	}
	return result, nil
}

func (r *UserAreaRepository) Insert(ctx context.Context, membership domain.UserProcessArea) error {
	const q = `
INSERT INTO user_process_areas
  (user_id, tenant_id, area_code, role, effective_from, effective_to, granted_by)
VALUES
  ($1, $2::uuid, $3, $4, $5, $6, $7)
`
	_, err := r.db.ExecContext(
		ctx,
		q,
		membership.UserID,
		membership.TenantID,
		membership.AreaCode,
		string(membership.Role),
		membership.EffectiveFrom,
		membership.EffectiveTo,
		membership.GrantedBy,
	)
	if err != nil {
		return fmt.Errorf("insert user process area: %w", err)
	}
	return nil
}

func (r *UserAreaRepository) CloseActive(ctx context.Context, userID, tenantID, areaCode string, effectiveTo time.Time) error {
	const q = `
UPDATE user_process_areas
SET effective_to = $4
WHERE user_id = $1
  AND tenant_id::text = $2
  AND area_code = $3
  AND effective_to IS NULL
`
	if _, err := r.db.ExecContext(ctx, q, userID, tenantID, areaCode, effectiveTo); err != nil {
		return fmt.Errorf("close active user process area: %w", err)
	}
	return nil
}

func (r *UserAreaRepository) GrantAtomic(ctx context.Context, oldMembership, newMembership domain.UserProcessArea) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin grant transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	const closeQ = `
UPDATE user_process_areas
SET effective_to = $5
WHERE user_id = $1
  AND tenant_id::text = $2
  AND area_code = $3
  AND effective_from = $4
  AND effective_to IS NULL
`
	closeResult, err := tx.ExecContext(
		ctx,
		closeQ,
		oldMembership.UserID,
		oldMembership.TenantID,
		oldMembership.AreaCode,
		oldMembership.EffectiveFrom,
		newMembership.EffectiveFrom,
	)
	if err != nil {
		return fmt.Errorf("close active membership in grant transaction: %w", err)
	}
	rowsAffected, err := closeResult.RowsAffected()
	if err != nil {
		return fmt.Errorf("read affected rows for close active membership: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("close active membership in grant transaction: no rows updated")
	}

	const insertQ = `
INSERT INTO user_process_areas
  (user_id, tenant_id, area_code, role, effective_from, effective_to, granted_by)
VALUES
  ($1, $2::uuid, $3, $4, $5, $6, $7)
`
	if _, err := tx.ExecContext(
		ctx,
		insertQ,
		newMembership.UserID,
		newMembership.TenantID,
		newMembership.AreaCode,
		string(newMembership.Role),
		newMembership.EffectiveFrom,
		newMembership.EffectiveTo,
		newMembership.GrantedBy,
	); err != nil {
		return fmt.Errorf("insert membership in grant transaction: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit grant transaction: %w", err)
	}
	return nil
}

func (r *UserAreaRepository) GetActiveByUserAndArea(ctx context.Context, userID, tenantID, areaCode string, now time.Time) (*domain.UserProcessArea, error) {
	const q = `
SELECT user_id, tenant_id::text, area_code, role, effective_from, effective_to, granted_by
FROM user_process_areas
WHERE user_id = $1
  AND tenant_id::text = $2
  AND area_code = $3
  AND effective_from <= $4
  AND (effective_to IS NULL OR effective_to > $4)
ORDER BY effective_from DESC
LIMIT 1
`
	row := r.db.QueryRowContext(ctx, q, userID, tenantID, areaCode, now)
	item, err := scanUserProcessArea(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanUserProcessArea(s scanner) (domain.UserProcessArea, error) {
	var item domain.UserProcessArea
	var effectiveTo sql.NullTime
	var grantedBy sql.NullString
	if err := s.Scan(
		&item.UserID,
		&item.TenantID,
		&item.AreaCode,
		&item.Role,
		&item.EffectiveFrom,
		&effectiveTo,
		&grantedBy,
	); err != nil {
		return domain.UserProcessArea{}, fmt.Errorf("scan user process area: %w", err)
	}
	if effectiveTo.Valid {
		value := effectiveTo.Time
		item.EffectiveTo = &value
	}
	if grantedBy.Valid {
		value := grantedBy.String
		item.GrantedBy = &value
	}
	return item, nil
}
