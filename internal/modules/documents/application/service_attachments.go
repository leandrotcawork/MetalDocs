package application

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"metaldocs/internal/modules/documents/domain"
	"metaldocs/internal/platform/authn"
	"metaldocs/internal/platform/messaging"
)

var attachmentSafeChars = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

func (s *Service) UploadAttachmentAuthorized(ctx context.Context, cmd domain.UploadAttachmentCommand) (domain.Attachment, error) {
	if s.attachmentStore == nil {
		return domain.Attachment{}, domain.ErrAttachmentStoreUnavailable
	}
	if strings.TrimSpace(cmd.DocumentID) == "" || strings.TrimSpace(cmd.FileName) == "" || len(cmd.Content) == 0 {
		return domain.Attachment{}, domain.ErrInvalidAttachment
	}
	if len(cmd.Content) > 10*1024*1024 {
		return domain.Attachment{}, domain.ErrInvalidAttachment
	}

	doc, err := s.repo.GetDocument(ctx, strings.TrimSpace(cmd.DocumentID))
	if err != nil {
		return domain.Attachment{}, err
	}
	allowed, err := s.isAllowed(ctx, doc, domain.CapabilityDocumentUploadAttachment)
	if err != nil {
		return domain.Attachment{}, err
	}
	if !allowed {
		return domain.Attachment{}, domain.ErrDocumentNotFound
	}

	attachmentID := newAttachmentID()
	storageKey := attachmentStorageKey(doc.ID, attachmentID, cmd.FileName)
	attachment := domain.Attachment{
		ID:          attachmentID,
		DocumentID:  doc.ID,
		FileName:    strings.TrimSpace(cmd.FileName),
		ContentType: normalizeContentType(cmd.ContentType),
		SizeBytes:   int64(len(cmd.Content)),
		StorageKey:  storageKey,
		UploadedBy:  strings.TrimSpace(cmd.UploadedBy),
		CreatedAt:   s.clock.Now(),
	}
	if attachment.UploadedBy == "" {
		attachment.UploadedBy = authn.UserIDFromContext(ctx)
	}

	if err := s.attachmentStore.Save(ctx, storageKey, cmd.Content); err != nil {
		return domain.Attachment{}, err
	}
	if err := s.repo.CreateAttachment(ctx, attachment); err != nil {
		_ = s.attachmentStore.Delete(ctx, storageKey)
		return domain.Attachment{}, err
	}

	if s.publisher != nil {
		_ = s.publisher.Publish(ctx, messaging.Event{
			EventID:           fmt.Sprintf("evt-doc-attachment-create-%s", attachment.ID),
			EventType:         "document.attachment.created",
			AggregateType:     "document",
			AggregateID:       doc.ID,
			OccurredAtRFC3339: attachment.CreatedAt.Format(time.RFC3339),
			Version:           1,
			IdempotencyKey:    fmt.Sprintf("doc-attachment-create-%s", attachment.ID),
			Producer:          "documents",
			TraceID:           cmd.TraceID,
			Payload: map[string]any{
				"document_id":   doc.ID,
				"attachment_id": attachment.ID,
				"file_name":     attachment.FileName,
				"content_type":  attachment.ContentType,
				"size_bytes":    attachment.SizeBytes,
			},
		})
	}

	return attachment, nil
}

func (s *Service) ListAttachmentsAuthorized(ctx context.Context, documentID string) ([]domain.Attachment, error) {
	doc, err := s.GetDocumentAuthorized(ctx, documentID)
	if err != nil {
		return nil, err
	}
	return s.repo.ListAttachments(ctx, doc.ID)
}

func (s *Service) GetAttachmentAuthorized(ctx context.Context, documentID, attachmentID string) (domain.Attachment, error) {
	doc, err := s.GetDocumentAuthorized(ctx, documentID)
	if err != nil {
		return domain.Attachment{}, err
	}
	attachment, err := s.repo.GetAttachment(ctx, strings.TrimSpace(attachmentID))
	if err != nil {
		return domain.Attachment{}, err
	}
	if attachment.DocumentID != doc.ID {
		return domain.Attachment{}, domain.ErrAttachmentNotFound
	}
	return attachment, nil
}

func (s *Service) OpenAttachmentContent(ctx context.Context, attachmentID string) (domain.Attachment, []byte, error) {
	if s.attachmentStore == nil {
		return domain.Attachment{}, nil, domain.ErrAttachmentStoreUnavailable
	}
	attachment, err := s.repo.GetAttachment(ctx, strings.TrimSpace(attachmentID))
	if err != nil {
		return domain.Attachment{}, nil, err
	}
	reader, err := s.attachmentStore.Open(ctx, attachment.StorageKey)
	if err != nil {
		return domain.Attachment{}, nil, err
	}
	defer reader.Close()
	content, err := io.ReadAll(reader)
	if err != nil {
		return domain.Attachment{}, nil, err
	}
	return attachment, content, nil
}

func newAttachmentID() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("att_%d", time.Now().UTC().UnixNano())
	}
	return "att_" + hex.EncodeToString(buf)
}

func attachmentStorageKey(documentID, attachmentID, fileName string) string {
	safeName := attachmentSafeChars.ReplaceAllString(strings.TrimSpace(filepath.Base(fileName)), "_")
	if safeName == "" {
		safeName = "attachment.bin"
	}
	return strings.TrimSpace(documentID) + "/" + attachmentID + "/" + safeName
}

func normalizeContentType(value string) string {
	normalized := strings.TrimSpace(value)
	if normalized == "" {
		return "application/octet-stream"
	}
	return normalized
}
