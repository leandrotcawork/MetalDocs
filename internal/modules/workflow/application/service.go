package application

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	auditdomain "metaldocs/internal/modules/audit/domain"
	docdomain "metaldocs/internal/modules/documents/domain"
	workflowdomain "metaldocs/internal/modules/workflow/domain"
	"metaldocs/internal/platform/messaging"
)

type Clock interface {
	Now() time.Time
}

type realClock struct{}

func (realClock) Now() time.Time {
	return time.Now().UTC()
}

type Service struct {
	docRepo   docdomain.Repository
	audit     auditdomain.Writer
	publisher messaging.Publisher
	clock     Clock
}

func NewService(docRepo docdomain.Repository, audit auditdomain.Writer, publisher messaging.Publisher, clock Clock) *Service {
	if clock == nil {
		clock = realClock{}
	}
	return &Service{docRepo: docRepo, audit: audit, publisher: publisher, clock: clock}
}

func (s *Service) Transition(ctx context.Context, cmd workflowdomain.TransitionCommand) (workflowdomain.TransitionResult, error) {
	if strings.TrimSpace(cmd.DocumentID) == "" || strings.TrimSpace(cmd.ToStatus) == "" || strings.TrimSpace(cmd.ActorID) == "" {
		return workflowdomain.TransitionResult{}, workflowdomain.ErrInvalidCommand
	}

	doc, err := s.docRepo.GetDocument(ctx, strings.TrimSpace(cmd.DocumentID))
	if err != nil {
		return workflowdomain.TransitionResult{}, err
	}

	toStatus := strings.ToUpper(strings.TrimSpace(cmd.ToStatus))
	if !canTransition(doc.Status, toStatus) {
		return workflowdomain.TransitionResult{}, workflowdomain.ErrInvalidTransition
	}

	if err := s.docRepo.UpdateDocumentStatus(ctx, doc.ID, toStatus); err != nil {
		return workflowdomain.TransitionResult{}, err
	}

	now := s.clock.Now()
	if s.audit != nil {
		payloadBytes, err := json.Marshal(map[string]string{
			"from_status": doc.Status,
			"to_status":   toStatus,
			"reason":      strings.TrimSpace(cmd.Reason),
		})
		if err != nil {
			return workflowdomain.TransitionResult{}, fmt.Errorf("marshal audit payload: %w", err)
		}

		if err := s.audit.Record(ctx, auditdomain.Event{
			ID:           mustNewID(),
			OccurredAt:   now,
			ActorID:      strings.TrimSpace(cmd.ActorID),
			Action:       "workflow.transitioned",
			ResourceType: "document",
			ResourceID:   doc.ID,
			PayloadJSON:  string(payloadBytes),
			TraceID:      strings.TrimSpace(cmd.TraceID),
		}); err != nil {
			rollbackErr := s.docRepo.UpdateDocumentStatus(ctx, doc.ID, doc.Status)
			if rollbackErr != nil {
				return workflowdomain.TransitionResult{}, fmt.Errorf("record audit event: %w; rollback status: %v", err, rollbackErr)
			}
			return workflowdomain.TransitionResult{}, fmt.Errorf("record audit event: %w", err)
		}
	}

	if s.publisher != nil {
		_ = s.publisher.Publish(ctx, messaging.Event{
			EventID:           fmt.Sprintf("evt-workflow-transition-%s-%s", doc.ID, toStatus),
			EventType:         "workflow.transitioned",
			AggregateType:     "document",
			AggregateID:       doc.ID,
			OccurredAtRFC3339: now.Format(time.RFC3339),
			Version:           1,
			IdempotencyKey:    fmt.Sprintf("workflow-transition-%s-%s", doc.ID, toStatus),
			Producer:          "workflow",
			TraceID:           strings.TrimSpace(cmd.TraceID),
			Payload: map[string]any{
				"document_id": doc.ID,
				"from_status": doc.Status,
				"to_status":   toStatus,
				"reason":      strings.TrimSpace(cmd.Reason),
			},
		})
	}

	return workflowdomain.TransitionResult{
		DocumentID: doc.ID,
		FromStatus: doc.Status,
		ToStatus:   toStatus,
	}, nil
}

func canTransition(fromStatus, toStatus string) bool {
	switch fromStatus {
	case docdomain.StatusDraft:
		return toStatus == docdomain.StatusInReview
	case docdomain.StatusInReview:
		return toStatus == docdomain.StatusApproved || toStatus == docdomain.StatusDraft
	case docdomain.StatusApproved:
		return toStatus == docdomain.StatusPublished || toStatus == docdomain.StatusArchived
	case docdomain.StatusPublished:
		return toStatus == docdomain.StatusArchived
	case docdomain.StatusArchived:
		return false
	default:
		return false
	}
}

func mustNewID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("audit-fallback-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}
