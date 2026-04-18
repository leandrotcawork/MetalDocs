package application

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"

	"metaldocs/internal/modules/documents_v2/domain"
	"metaldocs/internal/platform/servicebus"
)

type ExportRepo interface {
	GetDocument(ctx context.Context, tenantID, id string) (*domain.Document, error)
	GetRevision(ctx context.Context, docID, revID string) (*domain.Revision, error)
	InsertExport(ctx context.Context, e *domain.Export) (*domain.Export, error)
	GetExportByHash(ctx context.Context, documentID string, compositeHash []byte) (*domain.Export, error)
}

type ExportPresigner interface {
	PresignObjectGET(ctx context.Context, storageKey string) (url string, err error)
	HeadObject(ctx context.Context, key string) (bool, error)
	SizeObject(ctx context.Context, key string) (int64, error)
}

type DocgenPDFClient interface {
	ConvertPDF(ctx context.Context, req servicebus.ConvertPDFRequest) (servicebus.ConvertPDFResult, error)
}

type ExportService struct {
	repo       ExportRepo
	presigner  ExportPresigner
	docgen     DocgenPDFClient
	audit      Audit
	docgenVer  string
	grammarVer string
}

func NewExportService(repo ExportRepo, presigner ExportPresigner, docgen DocgenPDFClient, audit Audit, docgenVer, grammarVer string) *ExportService {
	return &ExportService{
		repo:       repo,
		presigner:  presigner,
		docgen:     docgen,
		audit:      audit,
		docgenVer:  docgenVer,
		grammarVer: grammarVer,
	}
}

func (s *ExportService) ExportPDF(ctx context.Context, tenantID, userID, documentID string, opts domain.RenderOptions) (*domain.ExportResult, error) {
	doc, err := s.repo.GetDocument(ctx, tenantID, documentID)
	if err != nil {
		return nil, err
	}
	if doc.CurrentRevisionID == "" {
		return nil, domain.ErrExportDocxMissing
	}

	rev, err := s.repo.GetRevision(ctx, documentID, doc.CurrentRevisionID)
	if err != nil {
		return nil, err
	}

	contentHashBytes, err := hex.DecodeString(rev.ContentHash)
	if err != nil {
		return nil, fmt.Errorf("decode revision content hash: %w", err)
	}

	compositeHash, err := domain.ComputeCompositeHash(contentHashBytes, doc.TemplateVersionID, s.grammarVer, s.docgenVer, opts)
	if err != nil {
		return nil, err
	}

	storageKey := fmt.Sprintf("tenants/%s/documents/%s/exports/%s.pdf", tenantID, documentID, hex.EncodeToString(compositeHash))

	existing, err := s.repo.GetExportByHash(ctx, documentID, compositeHash)
	if err == nil {
		s.audit.Write(ctx, tenantID, userID, "export.pdf_generated", documentID, map[string]any{"cached": true, "storage_key": existing.StorageKey})
		return &domain.ExportResult{Export: existing, Cached: true}, nil
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}

	headFound, err := s.presigner.HeadObject(ctx, storageKey)
	if err != nil {
		return nil, err
	}
	if !headFound {
		_, err = s.docgen.ConvertPDF(ctx, servicebus.ConvertPDFRequest{
			DocxKey:   rev.StorageKey,
			OutputKey: storageKey,
			RenderOpts: &servicebus.PDFRenderOpts{
				PaperSize: opts.PaperSize,
				Landscape: opts.LandscapeP,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("%w: %v", domain.ErrExportGotenbergFailed, err)
		}
	}

	sizeBytes, err := s.presigner.SizeObject(ctx, storageKey)
	if err != nil {
		return nil, err
	}

	exp, err := s.repo.InsertExport(ctx, &domain.Export{
		DocumentID:    documentID,
		RevisionID:    rev.ID,
		CompositeHash: compositeHash,
		StorageKey:    storageKey,
		SizeBytes:     sizeBytes,
		PaperSize:     opts.PaperSize,
		Landscape:     opts.LandscapeP,
		DocgenV2Ver:   s.docgenVer,
	})
	if err != nil {
		return nil, err
	}

	s.audit.Write(ctx, tenantID, userID, "export.pdf_generated", documentID, map[string]any{"cached": false, "storage_key": exp.StorageKey})
	return &domain.ExportResult{Export: exp, Cached: false}, nil
}

func (s *ExportService) SignExportURL(ctx context.Context, storageKey string) (string, error) {
	return s.presigner.PresignObjectGET(ctx, storageKey)
}

func (s *ExportService) GetDocumentSummary(ctx context.Context, tenantID, documentID string) (*domain.Document, error) {
	return s.repo.GetDocument(ctx, tenantID, documentID)
}

func (s *ExportService) SignedDocxURL(ctx context.Context, tenantID, userID, documentID string) (string, error) {
	doc, err := s.repo.GetDocument(ctx, tenantID, documentID)
	if err != nil {
		return "", err
	}
	if doc.CurrentRevisionID == "" {
		return "", domain.ErrExportDocxMissing
	}

	rev, err := s.repo.GetRevision(ctx, documentID, doc.CurrentRevisionID)
	if err != nil {
		return "", err
	}

	url, err := s.presigner.PresignObjectGET(ctx, rev.StorageKey)
	if err != nil {
		return "", err
	}

	s.audit.Write(ctx, tenantID, userID, "export.docx_downloaded", documentID, map[string]any{"revision_id": rev.ID, "storage_key": rev.StorageKey})
	return url, nil
}
