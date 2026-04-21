package infrastructure

import (
	"context"
	"database/sql"
	"errors"

	"metaldocs/internal/modules/taxonomy/domain"
)

type ProfileRepository struct {
	db *sql.DB
}

func NewProfileRepository(db *sql.DB) *ProfileRepository {
	return &ProfileRepository{db: db}
}

func (r *ProfileRepository) GetByCode(ctx context.Context, tenantID, code string) (*domain.DocumentProfile, error) {
	const q = `
SELECT code, tenant_id, family_code, name, description, review_interval_days,
       default_template_version_id, owner_user_id, editable_by_role, archived_at, created_at
FROM metaldocs.document_profiles
WHERE tenant_id = $1 AND code = $2`

	var profile domain.DocumentProfile
	var defaultTemplateVersionID sql.NullString
	var ownerUserID sql.NullString
	err := r.db.QueryRowContext(ctx, q, tenantID, code).Scan(
		&profile.Code,
		&profile.TenantID,
		&profile.FamilyCode,
		&profile.Name,
		&profile.Description,
		&profile.ReviewIntervalDays,
		&defaultTemplateVersionID,
		&ownerUserID,
		&profile.EditableByRole,
		&profile.ArchivedAt,
		&profile.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrProfileNotFound
	}
	if err != nil {
		return nil, err
	}
	profile.DefaultTemplateVersionID = nullStringPtr(defaultTemplateVersionID)
	profile.OwnerUserID = nullStringPtr(ownerUserID)
	return &profile, nil
}

func (r *ProfileRepository) List(ctx context.Context, tenantID string, includeArchived bool) ([]domain.DocumentProfile, error) {
	q := `
SELECT code, tenant_id, family_code, name, description, review_interval_days,
       default_template_version_id, owner_user_id, editable_by_role, archived_at, created_at
FROM metaldocs.document_profiles
WHERE tenant_id = $1`
	if !includeArchived {
		q += " AND archived_at IS NULL"
	}
	q += " ORDER BY code ASC"

	rows, err := r.db.QueryContext(ctx, q, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.DocumentProfile, 0)
	for rows.Next() {
		var profile domain.DocumentProfile
		var defaultTemplateVersionID sql.NullString
		var ownerUserID sql.NullString
		if err := rows.Scan(
			&profile.Code,
			&profile.TenantID,
			&profile.FamilyCode,
			&profile.Name,
			&profile.Description,
			&profile.ReviewIntervalDays,
			&defaultTemplateVersionID,
			&ownerUserID,
			&profile.EditableByRole,
			&profile.ArchivedAt,
			&profile.CreatedAt,
		); err != nil {
			return nil, err
		}
		profile.DefaultTemplateVersionID = nullStringPtr(defaultTemplateVersionID)
		profile.OwnerUserID = nullStringPtr(ownerUserID)
		out = append(out, profile)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *ProfileRepository) Create(ctx context.Context, p *domain.DocumentProfile) error {
	const q = `
INSERT INTO metaldocs.document_profiles
    (code, tenant_id, family_code, name, description, review_interval_days, default_template_version_id, owner_user_id, editable_by_role, archived_at)
VALUES
    ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	_, err := r.db.ExecContext(
		ctx,
		q,
		p.Code,
		p.TenantID,
		p.FamilyCode,
		p.Name,
		p.Description,
		p.ReviewIntervalDays,
		stringPtrToNull(p.DefaultTemplateVersionID),
		stringPtrToNull(p.OwnerUserID),
		p.EditableByRole,
		p.ArchivedAt,
	)
	return err
}

func (r *ProfileRepository) Update(ctx context.Context, p *domain.DocumentProfile) error {
	const q = `
UPDATE metaldocs.document_profiles
SET family_code = $1,
    name = $2,
    description = $3,
    review_interval_days = $4,
    default_template_version_id = $5,
    owner_user_id = $6,
    editable_by_role = $7,
    archived_at = $8
WHERE tenant_id = $9 AND code = $10`

	result, err := r.db.ExecContext(
		ctx,
		q,
		p.FamilyCode,
		p.Name,
		p.Description,
		p.ReviewIntervalDays,
		stringPtrToNull(p.DefaultTemplateVersionID),
		stringPtrToNull(p.OwnerUserID),
		p.EditableByRole,
		p.ArchivedAt,
		p.TenantID,
		p.Code,
	)
	if err != nil {
		return err
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return domain.ErrProfileNotFound
	}
	return nil
}

type AreaRepository struct {
	db *sql.DB
}

func NewAreaRepository(db *sql.DB) *AreaRepository {
	return &AreaRepository{db: db}
}

func (r *AreaRepository) GetByCode(ctx context.Context, tenantID, code string) (*domain.ProcessArea, error) {
	const q = `
SELECT code, tenant_id, name, description, parent_code, owner_user_id, default_approver_role, archived_at, created_at
FROM metaldocs.document_process_areas
WHERE tenant_id = $1 AND code = $2`

	var area domain.ProcessArea
	var parentCode sql.NullString
	var ownerUserID sql.NullString
	var defaultApproverRole sql.NullString
	err := r.db.QueryRowContext(ctx, q, tenantID, code).Scan(
		&area.Code,
		&area.TenantID,
		&area.Name,
		&area.Description,
		&parentCode,
		&ownerUserID,
		&defaultApproverRole,
		&area.ArchivedAt,
		&area.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrAreaNotFound
	}
	if err != nil {
		return nil, err
	}
	area.ParentCode = nullStringPtr(parentCode)
	area.OwnerUserID = nullStringPtr(ownerUserID)
	area.DefaultApproverRole = nullStringPtr(defaultApproverRole)
	return &area, nil
}

func (r *AreaRepository) List(ctx context.Context, tenantID string, includeArchived bool) ([]domain.ProcessArea, error) {
	q := `
SELECT code, tenant_id, name, description, parent_code, owner_user_id, default_approver_role, archived_at, created_at
FROM metaldocs.document_process_areas
WHERE tenant_id = $1`
	if !includeArchived {
		q += " AND archived_at IS NULL"
	}
	q += " ORDER BY code ASC"

	rows, err := r.db.QueryContext(ctx, q, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.ProcessArea, 0)
	for rows.Next() {
		var area domain.ProcessArea
		var parentCode sql.NullString
		var ownerUserID sql.NullString
		var defaultApproverRole sql.NullString
		if err := rows.Scan(
			&area.Code,
			&area.TenantID,
			&area.Name,
			&area.Description,
			&parentCode,
			&ownerUserID,
			&defaultApproverRole,
			&area.ArchivedAt,
			&area.CreatedAt,
		); err != nil {
			return nil, err
		}
		area.ParentCode = nullStringPtr(parentCode)
		area.OwnerUserID = nullStringPtr(ownerUserID)
		area.DefaultApproverRole = nullStringPtr(defaultApproverRole)
		out = append(out, area)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *AreaRepository) Create(ctx context.Context, a *domain.ProcessArea) error {
	const q = `
INSERT INTO metaldocs.document_process_areas
    (code, tenant_id, name, description, parent_code, owner_user_id, default_approver_role, archived_at)
VALUES
    ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := r.db.ExecContext(
		ctx,
		q,
		a.Code,
		a.TenantID,
		a.Name,
		a.Description,
		stringPtrToNull(a.ParentCode),
		stringPtrToNull(a.OwnerUserID),
		stringPtrToNull(a.DefaultApproverRole),
		a.ArchivedAt,
	)
	return err
}

func (r *AreaRepository) Update(ctx context.Context, a *domain.ProcessArea) error {
	const q = `
UPDATE metaldocs.document_process_areas
SET name = $1,
    description = $2,
    parent_code = $3,
    owner_user_id = $4,
    default_approver_role = $5,
    archived_at = $6
WHERE tenant_id = $7 AND code = $8`

	result, err := r.db.ExecContext(
		ctx,
		q,
		a.Name,
		a.Description,
		stringPtrToNull(a.ParentCode),
		stringPtrToNull(a.OwnerUserID),
		stringPtrToNull(a.DefaultApproverRole),
		a.ArchivedAt,
		a.TenantID,
		a.Code,
	)
	if err != nil {
		return err
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return domain.ErrAreaNotFound
	}
	return nil
}

func (r *AreaRepository) ListAncestors(ctx context.Context, tenantID, code string) ([]string, error) {
	const q = `
WITH RECURSIVE ancestors AS (
    SELECT p.code, p.parent_code
    FROM metaldocs.document_process_areas p
    WHERE p.tenant_id = $1
      AND p.code = (
          SELECT parent_code
          FROM metaldocs.document_process_areas
          WHERE tenant_id = $1 AND code = $2
      )
    UNION
    SELECT p.code, p.parent_code
    FROM metaldocs.document_process_areas p
    INNER JOIN ancestors a ON p.tenant_id = $1 AND p.code = a.parent_code
)
SELECT code FROM ancestors`

	rows, err := r.db.QueryContext(ctx, q, tenantID, code)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ancestors := make([]string, 0)
	for rows.Next() {
		var ancestorCode string
		if err := rows.Scan(&ancestorCode); err != nil {
			return nil, err
		}
		ancestors = append(ancestors, ancestorCode)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return ancestors, nil
}

func stringPtrToNull(v *string) sql.NullString {
	if v == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *v, Valid: true}
}

func nullStringPtr(v sql.NullString) *string {
	if !v.Valid {
		return nil
	}
	value := v.String
	return &value
}
