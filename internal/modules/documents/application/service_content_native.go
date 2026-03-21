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

	pdfBytes, err := s.renderDocumentPDF(ctx, doc, version, cmd.Content, cmd.TraceID)
	if err != nil {
		return domain.Version{}, err
	}
	pdfKey := documentContentStorageKey(doc.ID, next, "pdf")
	if err := s.attachmentStore.Save(ctx, pdfKey, pdfBytes); err != nil {
		return domain.Version{}, err
	}
	version.PdfStorageKey = pdfKey

	if err := s.repo.SaveVersion(ctx, version); err != nil {
		_ = s.attachmentStore.Delete(ctx, pdfKey)
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
		pdfBytes, err = s.renderDocumentPDF(ctx, doc, version, content, traceID)
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

func (s *Service) RenderProfileTemplateDocx(ctx context.Context, profileCode string) ([]byte, error) {
	return s.renderProfileTemplate(ctx, profileCode, map[string]any{}, "docx", "profile-template")
}

func (s *Service) RenderDocumentTemplateDocxAuthorized(ctx context.Context, documentID string) ([]byte, error) {
	if strings.TrimSpace(documentID) == "" {
		return nil, domain.ErrInvalidCommand
	}
	doc, err := s.repo.GetDocument(ctx, strings.TrimSpace(documentID))
	if err != nil {
		return nil, err
	}
	allowed, err := s.isAllowed(ctx, doc, domain.CapabilityDocumentView)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, domain.ErrDocumentNotFound
	}
	data := buildDocumentTemplateData(doc, nil)
	return s.renderProfileTemplate(ctx, doc.DocumentProfile, data, "docx", "document-template")
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

func (s *Service) renderProfileTemplate(ctx context.Context, profileCode string, data map[string]any, convertTo, traceID string) ([]byte, error) {
	if s.carboneClient == nil || s.carboneTemplates == nil {
		return nil, domain.ErrRenderUnavailable
	}
	normalized := strings.ToLower(strings.TrimSpace(profileCode))
	templateID, ok := s.carboneTemplates.TemplateID(normalized)
	if !ok {
		return nil, domain.ErrRenderUnavailable
	}
	renderID, err := s.carboneClient.RenderTemplate(ctx, traceID, templateID, data, convertTo)
	if err != nil {
		return nil, err
	}
	return s.carboneClient.DownloadRender(ctx, traceID, renderID)
}

func (s *Service) renderDocumentPDF(ctx context.Context, doc domain.Document, version domain.Version, content map[string]any, traceID string) ([]byte, error) {
	data := buildDocumentTemplateData(doc, content)
	data["version"] = version.Number
	data["createdAt"] = version.CreatedAt.Format(time.RFC3339)
	return s.renderProfileTemplate(ctx, doc.DocumentProfile, data, "pdf", traceID)
}

func buildDocumentTemplateData(doc domain.Document, content map[string]any) map[string]any {
	data := map[string]any{
		"documentId":     doc.ID,
		"title":          doc.Title,
		"profile":        doc.DocumentProfile,
		"family":         doc.DocumentFamily,
		"processArea":    doc.ProcessArea,
		"subject":        doc.Subject,
		"owner":          doc.OwnerID,
		"businessUnit":   doc.BusinessUnit,
		"department":     doc.Department,
		"classification": doc.Classification,
	}
	if content != nil {
		data["content"] = content
	}
	return data
}

func documentContentStorageKey(documentID string, version int, extension string) string {
	return fmt.Sprintf("documents/%s/versions/%d/content.%s", documentID, version, strings.TrimPrefix(extension, "."))
}
