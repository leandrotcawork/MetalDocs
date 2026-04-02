package application

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"metaldocs/internal/modules/documents/domain"
	"metaldocs/internal/platform/messaging"
	"metaldocs/internal/platform/render/docgen"
)

func (s *Service) GetNativeContentAuthorized(ctx context.Context, documentID string) (domain.Version, error) {
	if strings.TrimSpace(documentID) == "" {
		return domain.Version{}, domain.ErrInvalidCommand
	}
	doc, err := s.repo.GetDocument(ctx, strings.TrimSpace(documentID))
	if err != nil {
		return domain.Version{}, err
	}
	allowed, err := s.isAllowed(ctx, doc, domain.CapabilityDocumentView)
	if err != nil {
		return domain.Version{}, err
	}
	if !allowed {
		return domain.Version{}, domain.ErrDocumentNotFound
	}
	return s.latestVersion(ctx, doc.ID)
}

func (s *Service) SaveNativeContentAuthorized(ctx context.Context, cmd domain.SaveNativeContentCommand) (domain.Version, error) {
	if s.attachmentStore == nil {
		return domain.Version{}, domain.ErrAttachmentStoreUnavailable
	}
	if strings.TrimSpace(cmd.DocumentID) == "" {
		return domain.Version{}, domain.ErrInvalidCommand
	}
	doc, err := s.repo.GetDocument(ctx, strings.TrimSpace(cmd.DocumentID))
	if err != nil {
		return domain.Version{}, err
	}
	allowed, err := s.isAllowed(ctx, doc, domain.CapabilityDocumentEdit)
	if err != nil {
		return domain.Version{}, err
	}
	if !allowed {
		return domain.Version{}, domain.ErrDocumentNotFound
	}
	if !isVersioningAllowed(doc) {
		return domain.Version{}, domain.ErrVersioningNotAllowed
	}

	contentPayload := cmd.Content
	if contentPayload == nil {
		contentPayload = map[string]any{}
	}
	if schema, ok, err := s.resolveDocumentProfileSchema(ctx, doc.DocumentProfile, doc.ProfileSchemaVersion); err != nil {
		return domain.Version{}, err
	} else if ok {
		if err := validateContentSchema(schema.ContentSchema, contentPayload); err != nil {
			return domain.Version{}, err
		}
	}
	rawContent, err := json.Marshal(contentPayload)
	if err != nil {
		return domain.Version{}, domain.ErrInvalidCommand
	}
	contentText := string(rawContent)

	next, err := s.repo.NextVersionNumber(ctx, doc.ID)
	if err != nil {
		return domain.Version{}, err
	}
	now := s.clock.Now()

	version := domain.Version{
		DocumentID:    doc.ID,
		Number:        next,
		Content:       contentText,
		ContentHash:   contentHash(contentText),
		ChangeSummary: fmt.Sprintf("Content version %d", next),
		ContentSource: domain.ContentSourceNative,
		NativeContent: contentPayload,
		TextContent:   contentText,
		CreatedAt:     now,
	}

	pending := &docgen.RenderRevision{
		Versao:    fmt.Sprintf("%d", next),
		Data:      now.Format("2006-01-02"),
		Descricao: fmt.Sprintf("Content version %d", next),
		Por:       doc.OwnerID,
	}
	docxBytes, err := s.generateDocxBytes(ctx, doc, version, contentPayload, cmd.TraceID, pending)
	if err != nil {
		return domain.Version{}, err
	}
	docxKey := documentContentStorageKey(doc.ID, next, "docx")
	if err := s.attachmentStore.Save(ctx, docxKey, docxBytes); err != nil {
		return domain.Version{}, err
	}
	version.DocxStorageKey = docxKey

	pdfBytes, err := s.convertDocxToPDF(ctx, docxBytes, cmd.TraceID)
	if err != nil {
		_ = s.attachmentStore.Delete(ctx, docxKey)
		return domain.Version{}, err
	}
	pdfKey := documentContentStorageKey(doc.ID, next, "pdf")
	if err := s.attachmentStore.Save(ctx, pdfKey, pdfBytes); err != nil {
		_ = s.attachmentStore.Delete(ctx, docxKey)
		return domain.Version{}, err
	}
	version.PdfStorageKey = pdfKey

	if err := s.repo.SaveVersion(ctx, version); err != nil {
		_ = s.attachmentStore.Delete(ctx, pdfKey)
		_ = s.attachmentStore.Delete(ctx, docxKey)
		return domain.Version{}, err
	}

	if s.publisher != nil {
		_ = s.publisher.Publish(ctx, messaging.Event{
			EventID:           fmt.Sprintf("evt-doc-version-create-%s-%d", doc.ID, next),
			EventType:         "document.version.created",
			AggregateType:     "document",
			AggregateID:       doc.ID,
			OccurredAtRFC3339: now.Format(time.RFC3339),
			Version:           next,
			IdempotencyKey:    fmt.Sprintf("document.version.created:%s:%d", doc.ID, next),
			Producer:          "documents",
			TraceID:           cmd.TraceID,
			Payload: map[string]any{
				"document_id": doc.ID,
				"version":     next,
				"source":      version.ContentSource,
			},
		})
	}

	return version, nil
}

func (s *Service) RenderContentPDFAuthorized(ctx context.Context, documentID, traceID string) (domain.Version, error) {
	if s.attachmentStore == nil {
		return domain.Version{}, domain.ErrAttachmentStoreUnavailable
	}
	if strings.TrimSpace(documentID) == "" {
		return domain.Version{}, domain.ErrInvalidCommand
	}
	doc, err := s.repo.GetDocument(ctx, strings.TrimSpace(documentID))
	if err != nil {
		return domain.Version{}, err
	}
	allowed, err := s.isAllowed(ctx, doc, domain.CapabilityDocumentEdit)
	if err != nil {
		return domain.Version{}, err
	}
	if !allowed {
		return domain.Version{}, domain.ErrDocumentNotFound
	}

	version, err := s.latestVersion(ctx, doc.ID)
	if err != nil {
		return domain.Version{}, err
	}

	contentSource := strings.TrimSpace(version.ContentSource)
	if contentSource == "" {
		contentSource = domain.ContentSourceNative
	}
	var pdfBytes []byte
	switch contentSource {
	case domain.ContentSourceDocxUpload:
		if strings.TrimSpace(version.DocxStorageKey) == "" {
			return domain.Version{}, domain.ErrVersionNotFound
		}
		docxPayload, err := s.OpenContentStorage(ctx, version.DocxStorageKey)
		if err != nil {
			return domain.Version{}, err
		}
		pdfBytes, err = s.convertDocxToPDF(ctx, docxPayload, traceID)
		if err != nil {
			return domain.Version{}, err
		}
	default:
		content := version.NativeContent
		if len(content) == 0 && strings.TrimSpace(version.Content) != "" {
			var parsed map[string]any
			if err := json.Unmarshal([]byte(version.Content), &parsed); err == nil {
				content = parsed
			}
		}
		var docxBytes []byte
		if strings.TrimSpace(version.DocxStorageKey) != "" {
			docxBytes, err = s.OpenContentStorage(ctx, version.DocxStorageKey)
			if err != nil {
				return domain.Version{}, err
			}
		} else {
			docxBytes, err = s.generateDocxBytes(ctx, doc, version, content, traceID, nil)
			if err != nil {
				return domain.Version{}, err
			}
			docxKey := documentContentStorageKey(doc.ID, version.Number, "docx")
			if saveErr := s.attachmentStore.Save(ctx, docxKey, docxBytes); saveErr == nil {
				if err := s.repo.UpdateVersionDocx(ctx, doc.ID, version.Number, docxKey); err != nil {
					_ = s.attachmentStore.Delete(ctx, docxKey)
					return domain.Version{}, err
				}
				version.DocxStorageKey = docxKey
			}
		}
		pdfBytes, err = s.convertDocxToPDF(ctx, docxBytes, traceID)
		if err != nil {
			return domain.Version{}, err
		}
	}

	pdfKey := strings.TrimSpace(version.PdfStorageKey)
	if pdfKey == "" {
		pdfKey = documentContentStorageKey(doc.ID, version.Number, "pdf")
	}
	if err := s.attachmentStore.Save(ctx, pdfKey, pdfBytes); err != nil {
		return domain.Version{}, err
	}
	if pdfKey != version.PdfStorageKey || version.PageCount == 0 {
		if err := s.repo.UpdateVersionPDF(ctx, doc.ID, version.Number, pdfKey, version.PageCount); err != nil {
			return domain.Version{}, err
		}
	}
	version.PdfStorageKey = pdfKey
	return version, nil
}

func (s *Service) OpenContentStorage(ctx context.Context, storageKey string) ([]byte, error) {
	if s.attachmentStore == nil {
		return nil, domain.ErrAttachmentStoreUnavailable
	}
	if strings.TrimSpace(storageKey) == "" {
		return nil, domain.ErrInvalidCommand
	}
	reader, err := s.attachmentStore.Open(ctx, storageKey)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = reader.Close()
	}()
	payload, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	return payload, nil
}

func documentContentStorageKey(documentID string, version int, extension string) string {
	return fmt.Sprintf("documents/%s/versions/%d/content.%s", documentID, version, strings.TrimPrefix(extension, "."))
}
