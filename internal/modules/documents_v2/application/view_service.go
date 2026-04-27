package application

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	v2dom "metaldocs/internal/modules/documents_v2/domain"
	documentshttp "metaldocs/internal/modules/documents_v2/http"
	"metaldocs/internal/modules/iam/authz"
)

// ViewPresigner is implemented by objectstore helpers that presign a GET URL.
type ViewPresigner interface {
	PresignObjectGET(ctx context.Context, storageKey string) (string, error)
}

// ViewService serves viewer requests by checking area-scoped RBAC, validating
// the revision's lifecycle state, and returning a presigned PDF URL.
type ViewService struct {
	db        *sql.DB
	presigner ViewPresigner
}

func NewViewService(db *sql.DB, presigner ViewPresigner) *ViewService {
	return &ViewService{db: db, presigner: presigner}
}

var viewableStatuses = map[string]struct{}{
	"approved":  {},
	"scheduled": {},
	"published": {},
}

func (s *ViewService) GetViewURL(ctx context.Context, tenantID, actorID, docID string) (documentshttp.ViewResult, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return documentshttp.ViewResult{}, fmt.Errorf("view: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	ctx = authz.WithCapCache(ctx)
	if err := setAuthzGUC(ctx, tx, tenantID, actorID); err != nil {
		return documentshttp.ViewResult{}, err
	}

	var status, areaCode string
	var pdfKey sql.NullString
	err = tx.QueryRowContext(ctx, `
		SELECT status, coalesce(process_area_code_snapshot,''), final_pdf_s3_key
		  FROM documents
		 WHERE tenant_id=$1::uuid AND id=$2::uuid`,
		tenantID, docID,
	).Scan(&status, &areaCode, &pdfKey)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return documentshttp.ViewResult{}, v2dom.ErrNotFound
		}
		return documentshttp.ViewResult{}, fmt.Errorf("view: load document: %w", err)
	}

	area := areaCode
	if area == "" {
		area = "tenant"
	}
	if err := authz.Require(ctx, tx, "doc.view_published", area); err != nil {
		return documentshttp.ViewResult{}, err
	}

	if _, ok := viewableStatuses[status]; !ok {
		return documentshttp.ViewResult{}, v2dom.ErrNotFound
	}
	if !pdfKey.Valid || pdfKey.String == "" {
		return documentshttp.ViewResult{}, documentshttp.ErrPDFPending
	}

	url, err := s.presigner.PresignObjectGET(ctx, pdfKey.String)
	if err != nil {
		return documentshttp.ViewResult{}, fmt.Errorf("view: presign: %w", err)
	}
	return documentshttp.ViewResult{SignedURL: url}, nil
}
