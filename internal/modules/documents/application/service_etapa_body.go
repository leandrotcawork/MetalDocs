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

func (s *Service) GetEtapaBodyAuthorized(ctx context.Context, documentID string, versionNumber, stepIndex int) (domain.EtapaBody, error) {
	if strings.TrimSpace(documentID) == "" || versionNumber < 1 || stepIndex < 0 {
		return domain.EtapaBody{}, domain.ErrInvalidCommand
	}

	doc, err := s.repo.GetDocument(ctx, strings.TrimSpace(documentID))
	if err != nil {
		return domain.EtapaBody{}, err
	}
	allowed, err := s.isAllowed(ctx, doc, domain.CapabilityDocumentView)
	if err != nil {
		return domain.EtapaBody{}, err
	}
	if !allowed {
		return domain.EtapaBody{}, domain.ErrDocumentNotFound
	}

	version, err := s.repo.GetVersion(ctx, doc.ID, versionNumber)
	if err != nil {
		return domain.EtapaBody{}, err
	}
	if stepIndex >= len(version.BodyBlocks) {
		return domain.EtapaBody{}, domain.ErrVersionNotFound
	}

	return cloneEtapaBody(version.BodyBlocks[stepIndex]), nil
}

func (s *Service) SaveEtapaBodyAuthorized(ctx context.Context, cmd domain.SaveEtapaBodyCommand) (domain.Version, error) {
	if strings.TrimSpace(cmd.DocumentID) == "" || cmd.VersionNumber < 1 || cmd.StepIndex < 0 {
		return domain.Version{}, domain.ErrInvalidCommand
	}
	if err := validateEtapaBodyBlocks(cmd.Blocks); err != nil {
		return domain.Version{}, err
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

	version, err := s.repo.GetVersion(ctx, doc.ID, cmd.VersionNumber)
	if err != nil {
		return domain.Version{}, err
	}

	updatedBodyBlocks := updateEtapaBodyBlocks(version.BodyBlocks, cmd.StepIndex, cmd.Blocks)
	now := s.clock.Now()

	if doc.Status == domain.StatusDraft {
		previousBodyBlocks := cloneEtapaBodies(version.BodyBlocks)
		if err := s.repo.UpdateVersionBodyBlocks(ctx, doc.ID, version.Number, updatedBodyBlocks); err != nil {
			return domain.Version{}, err
		}

		updated := version
		updated.BodyBlocks = updatedBodyBlocks
		if err := s.recordEtapaBodyAudit(ctx, doc, updated, cmd.StepIndex, len(cmd.Blocks), now, cmd.TraceID); err != nil {
			_ = s.repo.UpdateVersionBodyBlocks(ctx, doc.ID, version.Number, previousBodyBlocks)
			return domain.Version{}, err
		}
		s.publishEtapaBodyUpdated(ctx, doc, updated, cmd.StepIndex, len(cmd.Blocks), cmd.TraceID, now)
		return updated, nil
	}

	nextVersionNumber, err := s.repo.NextVersionNumber(ctx, doc.ID)
	if err != nil {
		return domain.Version{}, err
	}

	nextVersion := version
	nextVersion.Number = nextVersionNumber
	nextVersion.BodyBlocks = updatedBodyBlocks
	nextVersion.CreatedAt = now

	if err := s.repo.SaveVersion(ctx, nextVersion); err != nil {
		return domain.Version{}, err
	}
	if err := s.recordEtapaBodyAudit(ctx, doc, nextVersion, cmd.StepIndex, len(cmd.Blocks), now, cmd.TraceID); err != nil {
		return domain.Version{}, err
	}
	s.publishEtapaBodyUpdated(ctx, doc, nextVersion, cmd.StepIndex, len(cmd.Blocks), cmd.TraceID, now)

	return nextVersion, nil
}

func (s *Service) recordEtapaBodyAudit(ctx context.Context, doc domain.Document, version domain.Version, stepIndex, blockCount int, now time.Time, traceID string) error {
	if s.audit == nil {
		return nil
	}

	payload, err := json.Marshal(map[string]any{
		"document_id":    doc.ID,
		"version_number": version.Number,
		"step_index":     stepIndex,
		"block_count":    blockCount,
	})
	if err != nil {
		return fmt.Errorf("marshal etapa body audit payload: %w", err)
	}

	return s.audit.Record(ctx, auditdomain.Event{
		ID:           mustNewID(),
		OccurredAt:   now,
		ActorID:      strings.TrimSpace(authn.UserIDFromContext(ctx)),
		Action:       "document.etapa_body.updated",
		ResourceType: "document",
		ResourceID:   doc.ID,
		PayloadJSON:  string(payload),
		TraceID:      strings.TrimSpace(traceID),
	})
}

func (s *Service) publishEtapaBodyUpdated(ctx context.Context, doc domain.Document, version domain.Version, stepIndex, blockCount int, traceID string, now time.Time) {
	if s.publisher == nil {
		return
	}

	_ = s.publisher.Publish(ctx, messaging.Event{
		EventID:           fmt.Sprintf("evt-document-etapa-body-updated-%s-%d-%d", doc.ID, version.Number, stepIndex),
		EventType:         "document.etapa_body.updated",
		AggregateType:     "document",
		AggregateID:       doc.ID,
		OccurredAtRFC3339: now.Format(time.RFC3339),
		Version:           version.Number,
		IdempotencyKey:    fmt.Sprintf("etapa_body_updated:%s:%d:%d", doc.ID, version.Number, stepIndex),
		Producer:          "documents",
		TraceID:           strings.TrimSpace(traceID),
		Payload: map[string]any{
			"document_id":    doc.ID,
			"version_number": version.Number,
			"step_index":     stepIndex,
			"block_count":    blockCount,
		},
	})
}

func validateEtapaBodyBlocks(blocks []json.RawMessage) error {
	for _, block := range blocks {
		if len(block) == 0 || !json.Valid(block) {
			return domain.ErrInvalidNativeContent
		}
	}
	return nil
}

func updateEtapaBodyBlocks(existing []domain.EtapaBody, stepIndex int, blocks []json.RawMessage) []domain.EtapaBody {
	out := cloneEtapaBodies(existing)
	for len(out) <= stepIndex {
		out = append(out, domain.EtapaBody{})
	}
	out[stepIndex] = domain.EtapaBody{Blocks: cloneRawMessages(blocks)}
	return out
}

func cloneEtapaBodies(items []domain.EtapaBody) []domain.EtapaBody {
	if len(items) == 0 {
		return []domain.EtapaBody{}
	}
	out := make([]domain.EtapaBody, len(items))
	for i, item := range items {
		out[i] = cloneEtapaBody(item)
	}
	return out
}

func cloneEtapaBody(item domain.EtapaBody) domain.EtapaBody {
	return domain.EtapaBody{Blocks: cloneRawMessages(item.Blocks)}
}

func cloneRawMessages(items []json.RawMessage) []json.RawMessage {
	if len(items) == 0 {
		return []json.RawMessage{}
	}
	out := make([]json.RawMessage, len(items))
	for i, item := range items {
		out[i] = append(json.RawMessage(nil), item...)
	}
	return out
}
