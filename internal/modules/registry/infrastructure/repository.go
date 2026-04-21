package infrastructure

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	registrydomain "metaldocs/internal/modules/registry/domain"
	taxonomydomain "metaldocs/internal/modules/taxonomy/domain"
)

type PostgresControlledDocumentRepository struct {
	db *sql.DB
}

func NewPostgresControlledDocumentRepository(db *sql.DB) *PostgresControlledDocumentRepository {
	return &PostgresControlledDocumentRepository{db: db}
}

func (r *PostgresControlledDocumentRepository) GetByID(ctx context.Context, tenantID, id string) (*registrydomain.ControlledDocument, error) {
	const q = `
SELECT id::text, tenant_id::text, profile_code, process_area_code, department_code,
       code, sequence_num, title, owner_user_id, coalesce(override_template_version_id::text, ''),
       status, created_at, updated_at
FROM controlled_documents
WHERE tenant_id = $1 AND id = $2`
	return scanControlledDocument(r.db.QueryRowContext(ctx, q, tenantID, id))
}

func (r *PostgresControlledDocumentRepository) GetByCode(ctx context.Context, tenantID, profileCode, code string) (*registrydomain.ControlledDocument, error) {
	const q = `
SELECT id::text, tenant_id::text, profile_code, process_area_code, department_code,
       code, sequence_num, title, owner_user_id, coalesce(override_template_version_id::text, ''),
       status, created_at, updated_at
FROM controlled_documents
WHERE tenant_id = $1 AND profile_code = $2 AND code = $3`
	return scanControlledDocument(r.db.QueryRowContext(ctx, q, tenantID, profileCode, code))
}

func (r *PostgresControlledDocumentRepository) CodeExists(ctx context.Context, tenantID, profileCode, code string) (bool, error) {
	const q = `SELECT EXISTS(
		SELECT 1 FROM controlled_documents WHERE tenant_id = $1 AND profile_code = $2 AND code = $3
	)`
	var exists bool
	if err := r.db.QueryRowContext(ctx, q, tenantID, profileCode, code).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}

func (r *PostgresControlledDocumentRepository) List(ctx context.Context, tenantID string, filter registrydomain.CDFilter) ([]registrydomain.ControlledDocument, error) {
	q := `
SELECT id::text, tenant_id::text, profile_code, process_area_code, department_code,
       code, sequence_num, title, owner_user_id, coalesce(override_template_version_id::text, ''),
       status, created_at, updated_at
FROM controlled_documents
WHERE tenant_id = $1`
	args := []any{tenantID}
	idx := 2

	if filter.ProfileCode != nil {
		q += fmt.Sprintf(" AND profile_code = $%d", idx)
		args = append(args, *filter.ProfileCode)
		idx++
	}
	if filter.ProcessAreaCode != nil {
		q += fmt.Sprintf(" AND process_area_code = $%d", idx)
		args = append(args, *filter.ProcessAreaCode)
		idx++
	}
	if len(filter.UserAreaCodes) > 0 {
		q += fmt.Sprintf(" AND process_area_code = ANY($%d)", idx)
		args = append(args, pgtype.FlatArray[string](filter.UserAreaCodes))
		idx++
	}
	if filter.DepartmentCode != nil {
		q += fmt.Sprintf(" AND department_code = $%d", idx)
		args = append(args, *filter.DepartmentCode)
		idx++
	}
	if filter.OwnerUserID != nil {
		q += fmt.Sprintf(" AND owner_user_id = $%d", idx)
		args = append(args, *filter.OwnerUserID)
		idx++
	}
	if filter.Status != nil {
		q += fmt.Sprintf(" AND status = $%d", idx)
		args = append(args, *filter.Status)
		idx++
	}
	if filter.Query != nil && strings.TrimSpace(*filter.Query) != "" {
		q += fmt.Sprintf(" AND (code ILIKE $%d OR title ILIKE $%d)", idx, idx)
		args = append(args, "%"+strings.TrimSpace(*filter.Query)+"%")
		idx++
	}
	q += " ORDER BY created_at DESC"
	if filter.Limit > 0 {
		q += fmt.Sprintf(" LIMIT $%d", idx)
		args = append(args, filter.Limit)
		idx++
	}
	if filter.Offset > 0 {
		q += fmt.Sprintf(" OFFSET $%d", idx)
		args = append(args, filter.Offset)
	}

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]registrydomain.ControlledDocument, 0)
	for rows.Next() {
		doc, err := scanControlledDocument(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *doc)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *PostgresControlledDocumentRepository) Create(ctx context.Context, doc *registrydomain.ControlledDocument) error {
	return r.createWithQueryer(ctx, r.db, doc)
}

func (r *PostgresControlledDocumentRepository) CreateTx(ctx context.Context, tx *sql.Tx, doc *registrydomain.ControlledDocument) error {
	if tx == nil {
		return errors.New("nil transaction")
	}
	return r.createWithQueryer(ctx, tx, doc)
}

func (r *PostgresControlledDocumentRepository) createWithQueryer(ctx context.Context, qr queryRower, doc *registrydomain.ControlledDocument) error {
	const insertQ = `
INSERT INTO controlled_documents
	(tenant_id, profile_code, process_area_code, department_code, code, sequence_num, title, owner_user_id,
	 override_template_version_id, status, created_at, updated_at)
VALUES
	($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
RETURNING id::text`
	var id string
	err := qr.QueryRowContext(
		ctx,
		insertQ,
		doc.TenantID,
		doc.ProfileCode,
		doc.ProcessAreaCode,
		stringPtrToNull(doc.DepartmentCode),
		doc.Code,
		intPtrToNull(doc.SequenceNum),
		doc.Title,
		doc.OwnerUserID,
		stringPtrToNull(doc.OverrideTemplateVersionID),
		doc.Status,
		doc.CreatedAt,
		doc.UpdatedAt,
	).Scan(&id)
	if err == nil {
		doc.ID = id
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		existing, getErr := r.GetByCode(ctx, doc.TenantID, doc.ProfileCode, doc.Code)
		if getErr == nil && existing.Status != registrydomain.CDStatusActive {
			return registrydomain.ErrCDArchivedCodeReuse
		}
		return registrydomain.ErrCDCodeTaken
	}
	return err
}

func (r *PostgresControlledDocumentRepository) UpdateStatus(ctx context.Context, tenantID, id string, status registrydomain.CDStatus, updatedAt time.Time) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE controlled_documents SET status = $1, updated_at = $2 WHERE tenant_id = $3 AND id = $4`,
		status, updatedAt, tenantID, id,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return registrydomain.ErrCDNotFound
	}
	return nil
}

type PostgresSequenceAllocator struct {
	db *sql.DB
}

func NewPostgresSequenceAllocator(db *sql.DB) *PostgresSequenceAllocator {
	return &PostgresSequenceAllocator{db: db}
}

func (a *PostgresSequenceAllocator) EnsureCounter(ctx context.Context, tenantID, profileCode string) error {
	return a.ensureCounter(ctx, a.db, tenantID, profileCode)
}

func (a *PostgresSequenceAllocator) ensureCounter(ctx context.Context, execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}, tenantID, profileCode string) error {
	_, err := execer.ExecContext(ctx, `
		INSERT INTO profile_sequence_counters (tenant_id, profile_code, next_seq)
		VALUES ($1, $2, 1)
		ON CONFLICT (tenant_id, profile_code) DO NOTHING`,
		tenantID, profileCode,
	)
	return err
}

func (a *PostgresSequenceAllocator) NextAndIncrement(ctx context.Context, tx interface{}, tenantID, profileCode string) (int, error) {
	exec := sequenceQueryExecutor(a.db)
	if provided, ok := tx.(*sql.Tx); ok && provided != nil {
		exec = provided
	}

	if err := a.ensureCounter(ctx, exec, tenantID, profileCode); err != nil {
		return 0, err
	}

	var next int
	if err := exec.QueryRowContext(ctx, `
		UPDATE profile_sequence_counters
		SET next_seq = next_seq + 1
		WHERE tenant_id = $1 AND profile_code = $2
		RETURNING next_seq - 1`,
		tenantID, profileCode,
	).Scan(&next); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, registrydomain.ErrSequenceCounterNotFound
		}
		return 0, err
	}
	return next, nil
}

type PostgresTemplateVersionChecker struct {
	db *sql.DB
}

func NewPostgresTemplateVersionChecker(db *sql.DB) *PostgresTemplateVersionChecker {
	return &PostgresTemplateVersionChecker{db: db}
}

func (c *PostgresTemplateVersionChecker) GetTemplateVersionState(ctx context.Context, templateVersionID string) (*string, string, error) {
	var status sql.NullString
	var profileCode sql.NullString
	err := c.db.QueryRowContext(ctx, `
		SELECT v.status, t.profile_code
		FROM templates_v2_template_version v
		JOIN templates_v2_template t ON t.id = v.template_id
		WHERE v.id = $1`, templateVersionID,
	).Scan(&status, &profileCode)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, "", nil
	}
	if err != nil {
		return nil, "", err
	}
	if !status.Valid {
		return nil, profileCode.String, nil
	}
	state := status.String
	return &state, profileCode.String, nil
}

type TaxonomyProfileReader struct {
	db *sql.DB
}

func NewTaxonomyProfileReader(db *sql.DB) *TaxonomyProfileReader {
	return &TaxonomyProfileReader{db: db}
}

func (r *TaxonomyProfileReader) GetByCode(ctx context.Context, tenantID, code string) (*taxonomydomain.DocumentProfile, error) {
	const q = `
SELECT code, tenant_id, family_code, name, description, review_interval_days,
       default_template_version_id, owner_user_id, editable_by_role, archived_at, created_at
FROM metaldocs.document_profiles
WHERE tenant_id = $1 AND code = $2`

	var profile taxonomydomain.DocumentProfile
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
		return nil, taxonomydomain.ErrProfileNotFound
	}
	if err != nil {
		return nil, err
	}
	profile.DefaultTemplateVersionID = nullStringPtr(defaultTemplateVersionID)
	profile.OwnerUserID = nullStringPtr(ownerUserID)
	return &profile, nil
}

type TaxonomyAreaReader struct {
	db *sql.DB
}

func NewTaxonomyAreaReader(db *sql.DB) *TaxonomyAreaReader { return &TaxonomyAreaReader{db: db} }

func (r *TaxonomyAreaReader) GetByCode(ctx context.Context, tenantID, code string) (*taxonomydomain.ProcessArea, error) {
	const q = `
SELECT code, tenant_id, name, description, parent_code, owner_user_id, default_approver_role, archived_at, created_at
FROM metaldocs.document_process_areas
WHERE tenant_id = $1 AND code = $2`

	var area taxonomydomain.ProcessArea
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
		return nil, taxonomydomain.ErrAreaNotFound
	}
	if err != nil {
		return nil, err
	}
	area.ParentCode = nullStringPtr(parentCode)
	area.OwnerUserID = nullStringPtr(ownerUserID)
	area.DefaultApproverRole = nullStringPtr(defaultApproverRole)
	return &area, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

type queryRower interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type sequenceQueryExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func scanControlledDocument(row rowScanner) (*registrydomain.ControlledDocument, error) {
	var (
		doc                     registrydomain.ControlledDocument
		departmentCode          sql.NullString
		sequenceNum             sql.NullInt64
		overrideTemplateVersion string
	)
	if err := row.Scan(
		&doc.ID,
		&doc.TenantID,
		&doc.ProfileCode,
		&doc.ProcessAreaCode,
		&departmentCode,
		&doc.Code,
		&sequenceNum,
		&doc.Title,
		&doc.OwnerUserID,
		&overrideTemplateVersion,
		&doc.Status,
		&doc.CreatedAt,
		&doc.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, registrydomain.ErrCDNotFound
		}
		return nil, err
	}
	doc.DepartmentCode = nullStringPtr(departmentCode)
	if sequenceNum.Valid {
		v := int(sequenceNum.Int64)
		doc.SequenceNum = &v
	}
	if strings.TrimSpace(overrideTemplateVersion) != "" {
		doc.OverrideTemplateVersionID = &overrideTemplateVersion
	}
	return &doc, nil
}

func stringPtrToNull(v *string) sql.NullString {
	if v == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *v, Valid: true}
}

func intPtrToNull(v *int) sql.NullInt64 {
	if v == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(*v), Valid: true}
}

func nullStringPtr(v sql.NullString) *string {
	if !v.Valid {
		return nil
	}
	value := v.String
	return &value
}
