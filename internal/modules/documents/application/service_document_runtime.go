package application

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	auditdomain "metaldocs/internal/modules/audit/domain"
	"metaldocs/internal/modules/documents/domain"
	"metaldocs/internal/platform/authn"
	"metaldocs/internal/platform/messaging"
)

type DocumentRuntimeBundle struct {
	Document domain.Document
	Version  domain.Version
	Schema   domain.DocumentProfileSchemaVersion
}

func (s *Service) resolveUserDisplayName(ctx context.Context, userID string) string {
	if userID == "" {
		return "-"
	}
	if s.userResolver == nil {
		return userID
	}
	name, err := s.userResolver.ResolveDisplayName(ctx, userID)
	if err != nil || name == "" {
		return userID
	}
	return name
}

func (s *Service) resolveLatestApproval(ctx context.Context, documentID string) (approverName string, approvedAt string) {
	if s.approvalReader == nil {
		return "-", ""
	}
	approvals, err := s.approvalReader.ListApprovals(ctx, documentID)
	if err != nil || len(approvals) == 0 {
		return "-", ""
	}
	for i := len(approvals) - 1; i >= 0; i-- {
		a := approvals[i]
		if strings.EqualFold(a.Status, "APPROVED") && a.DecidedAt != nil {
			name := s.resolveUserDisplayName(ctx, a.ApproverID)
			return name, a.DecidedAt.Format("2006-01-02")
		}
	}
	return "-", ""
}

func (s *Service) GetDocumentRuntimeBundle(ctx context.Context, documentID string) (DocumentRuntimeBundle, error) {
	doc, err := s.GetDocumentAuthorized(ctx, documentID)
	if err != nil {
		return DocumentRuntimeBundle{}, err
	}

	version, err := s.latestVersion(ctx, doc.ID)
	if err != nil {
		return DocumentRuntimeBundle{}, err
	}

	runtimeSchema, err := s.resolveActiveProfileSchema(ctx, doc.DocumentProfile)
	if err != nil {
		return DocumentRuntimeBundle{}, err
	}

	return DocumentRuntimeBundle{
		Document: doc,
		Version:  version,
		Schema:   runtimeSchema,
	}, nil
}

func (s *Service) SaveDocumentValues(ctx context.Context, cmd domain.SaveDocumentValuesCommand) (domain.Version, error) {
	documentID := strings.TrimSpace(cmd.DocumentID)
	if documentID == "" {
		return domain.Version{}, domain.ErrInvalidCommand
	}

	doc, err := s.repo.GetDocument(ctx, documentID)
	if err != nil {
		return domain.Version{}, err
	}

	typeDefinition, err := s.resolveDocumentTypeDefinition(ctx, firstNonEmpty(doc.DocumentType, doc.DocumentProfile))
	if err != nil {
		return domain.Version{}, err
	}

	values := cloneRuntimeValues(cmd.Values)
	if err := validateDocumentTypeValues(typeDefinition.Schema, values); err != nil {
		return domain.Version{}, err
	}

	latest, err := s.latestVersion(ctx, documentID)
	if err != nil {
		return domain.Version{}, err
	}

	if doc.Status == domain.StatusDraft {
		updated := latest
		updated.Values = values
		if err := s.repo.UpdateVersionValues(ctx, documentID, updated.Number, values); err != nil {
			return domain.Version{}, err
		}
		return updated, nil
	}

	next, err := s.repo.NextVersionNumber(ctx, documentID)
	if err != nil {
		return domain.Version{}, err
	}

	nextVersion := latest
	nextVersion.Number = next
	nextVersion.Values = values
	nextVersion.CreatedAt = s.clock.Now()
	nextVersion.ChangeSummary = fmt.Sprintf("Runtime values update %d", next)

	if err := s.repo.SaveVersion(ctx, nextVersion); err != nil {
		return domain.Version{}, err
	}

	return nextVersion, nil
}

func (s *Service) SaveDocumentValuesAuthorized(ctx context.Context, cmd domain.SaveDocumentValuesCommand) (domain.Version, error) {
	documentID := strings.TrimSpace(cmd.DocumentID)
	if documentID == "" {
		return domain.Version{}, domain.ErrInvalidCommand
	}

	doc, err := s.repo.GetDocument(ctx, documentID)
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

	var previousValues domain.DocumentValues
	if doc.Status == domain.StatusDraft {
		previous, err := s.latestVersion(ctx, documentID)
		if err != nil {
			return domain.Version{}, err
		}
		previousValues = cloneRuntimeValues(previous.Values)
	}

	version, err := s.SaveDocumentValues(ctx, cmd)
	if err != nil {
		return domain.Version{}, err
	}

	now := s.clock.Now()
	if err := s.recordRuntimeValuesAudit(ctx, doc, version, len(cmd.Values), cmd.TraceID, now); err != nil {
		if doc.Status == domain.StatusDraft && len(previousValues) > 0 {
			_ = s.repo.UpdateVersionValues(ctx, documentID, version.Number, previousValues)
		}
		return domain.Version{}, err
	}
	s.publishRuntimeValuesUpdated(ctx, doc, version, len(cmd.Values), cmd.TraceID, now)

	return version, nil
}

func (s *Service) ExportDocumentDocxAuthorized(ctx context.Context, documentID, traceID string) ([]byte, error) {
	return nil, domain.ErrRenderUnavailable
}

func (s *Service) generateDocxBytes(ctx context.Context, doc domain.Document, version domain.Version, content map[string]any, traceID string, pendingRevision any) ([]byte, error) {
	return nil, domain.ErrRenderUnavailable
}

func (s *Service) recordRuntimeValuesAudit(ctx context.Context, doc domain.Document, version domain.Version, valueCount int, traceID string, now time.Time) error {
	if s.audit == nil {
		return nil
	}

	payload, err := json.Marshal(map[string]any{
		"document_id":    doc.ID,
		"version_number": version.Number,
		"value_count":    valueCount,
	})
	if err != nil {
		return fmt.Errorf("marshal runtime values audit payload: %w", err)
	}

	return s.audit.Record(ctx, auditdomain.Event{
		ID:           mustNewID(),
		OccurredAt:   now,
		ActorID:      strings.TrimSpace(authn.UserIDFromContext(ctx)),
		Action:       "document.runtime.values.updated",
		ResourceType: "document",
		ResourceID:   doc.ID,
		PayloadJSON:  string(payload),
		TraceID:      strings.TrimSpace(traceID),
	})
}

func (s *Service) publishRuntimeValuesUpdated(ctx context.Context, doc domain.Document, version domain.Version, valueCount int, traceID string, now time.Time) {
	if s.publisher == nil {
		return
	}

	_ = s.publisher.Publish(ctx, messaging.Event{
		EventID:           fmt.Sprintf("evt-document-runtime-values-updated-%s-%d", doc.ID, version.Number),
		EventType:         "document.runtime.values.updated",
		AggregateType:     "document",
		AggregateID:       doc.ID,
		OccurredAtRFC3339: now.Format(time.RFC3339),
		Version:           version.Number,
		IdempotencyKey:    fmt.Sprintf("document.runtime.values.updated:%s", doc.ID),
		Producer:          "documents",
		TraceID:           strings.TrimSpace(traceID),
		Payload: map[string]any{
			"document_id":    doc.ID,
			"version_number": version.Number,
			"value_count":    valueCount,
		},
	})
}
