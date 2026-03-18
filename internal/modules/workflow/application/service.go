package application

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
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

	var approval docdomain.WorkflowApproval
	var previousApproval *docdomain.WorkflowApproval
	now := s.clock.Now()

	switch {
	case doc.Status == docdomain.StatusDraft && toStatus == docdomain.StatusInReview:
		assignedReviewer := strings.TrimSpace(cmd.AssignedReviewer)
		if assignedReviewer == "" {
			return workflowdomain.TransitionResult{}, workflowdomain.ErrInvalidCommand
		}
		approval = docdomain.WorkflowApproval{
			ID:               mustNewID(),
			DocumentID:       doc.ID,
			RequestedBy:      strings.TrimSpace(cmd.ActorID),
			AssignedReviewer: assignedReviewer,
			Status:           docdomain.WorkflowApprovalStatusPending,
			RequestReason:    strings.TrimSpace(cmd.Reason),
			RequestedAt:      now,
		}
		if err := s.docRepo.CreateWorkflowApproval(ctx, approval); err != nil {
			return workflowdomain.TransitionResult{}, mapApprovalError(err)
		}
	case doc.Status == docdomain.StatusInReview && (toStatus == docdomain.StatusApproved || toStatus == docdomain.StatusDraft):
		latestApproval, err := s.docRepo.GetLatestWorkflowApproval(ctx, doc.ID)
		if err != nil {
			return workflowdomain.TransitionResult{}, mapApprovalError(err)
		}
		copyApproval := latestApproval
		previousApproval = &copyApproval
		if latestApproval.Status != docdomain.WorkflowApprovalStatusPending {
			return workflowdomain.TransitionResult{}, workflowdomain.ErrApprovalNotFound
		}
		if !strings.EqualFold(strings.TrimSpace(latestApproval.AssignedReviewer), strings.TrimSpace(cmd.ActorID)) {
			return workflowdomain.TransitionResult{}, workflowdomain.ErrApprovalReviewerDenied
		}

		decisionStatus := docdomain.WorkflowApprovalStatusApproved
		if toStatus == docdomain.StatusDraft {
			decisionStatus = docdomain.WorkflowApprovalStatusRejected
		}
		if err := s.docRepo.UpdateWorkflowApprovalDecision(ctx, latestApproval.ID, decisionStatus, cmd.ActorID, strings.TrimSpace(cmd.Reason), now); err != nil {
			return workflowdomain.TransitionResult{}, mapApprovalError(err)
		}
		latestApproval.Status = decisionStatus
		latestApproval.DecisionBy = strings.TrimSpace(cmd.ActorID)
		latestApproval.DecisionReason = strings.TrimSpace(cmd.Reason)
		latestApproval.DecidedAt = &now
		approval = latestApproval
	}

	if err := s.docRepo.UpdateDocumentStatus(ctx, doc.ID, toStatus); err != nil {
		return workflowdomain.TransitionResult{}, err
	}

	if err := s.recordAudit(ctx, doc, toStatus, cmd, approval, now); err != nil {
		rollbackErr := s.docRepo.UpdateDocumentStatus(ctx, doc.ID, doc.Status)
		rollbackApprovalErr := s.rollbackApproval(ctx, doc, approval, previousApproval)
		if rollbackErr != nil {
			return workflowdomain.TransitionResult{}, fmt.Errorf("record audit event: %w; rollback status: %v", err, rollbackErr)
		}
		if rollbackApprovalErr != nil {
			return workflowdomain.TransitionResult{}, fmt.Errorf("record audit event: %w; rollback approval: %v", err, rollbackApprovalErr)
		}
		return workflowdomain.TransitionResult{}, fmt.Errorf("record audit event: %w", err)
	}

	s.publishTransitionEvents(ctx, doc, toStatus, cmd, approval, now)

	return workflowdomain.TransitionResult{
		DocumentID:       doc.ID,
		FromStatus:       doc.Status,
		ToStatus:         toStatus,
		ApprovalID:       approval.ID,
		ApprovalStatus:   approval.Status,
		AssignedReviewer: approval.AssignedReviewer,
	}, nil
}

func (s *Service) rollbackApproval(ctx context.Context, doc docdomain.Document, approval docdomain.WorkflowApproval, previousApproval *docdomain.WorkflowApproval) error {
	if approval.ID == "" {
		return nil
	}
	if doc.Status == docdomain.StatusDraft && approval.Status == docdomain.WorkflowApprovalStatusPending && previousApproval == nil {
		return s.docRepo.DeleteWorkflowApproval(ctx, approval.ID)
	}
	if previousApproval != nil {
		return s.docRepo.SaveWorkflowApprovalState(ctx, *previousApproval)
	}
	return nil
}

func (s *Service) ListApprovals(ctx context.Context, documentID string) ([]workflowdomain.Approval, error) {
	if strings.TrimSpace(documentID) == "" {
		return nil, workflowdomain.ErrInvalidCommand
	}
	approvals, err := s.docRepo.ListWorkflowApprovals(ctx, strings.TrimSpace(documentID))
	if err != nil {
		return nil, mapApprovalError(err)
	}
	out := make([]workflowdomain.Approval, 0, len(approvals))
	for _, approval := range approvals {
		out = append(out, mapDocApprovalToWorkflowApproval(approval))
	}
	return out, nil
}

func (s *Service) recordAudit(ctx context.Context, doc docdomain.Document, toStatus string, cmd workflowdomain.TransitionCommand, approval docdomain.WorkflowApproval, now time.Time) error {
	if s.audit == nil {
		return nil
	}

	payloadBytes, err := json.Marshal(map[string]any{
		"from_status":       doc.Status,
		"to_status":         toStatus,
		"reason":            strings.TrimSpace(cmd.Reason),
		"approval_id":       approval.ID,
		"approval_status":   approval.Status,
		"assigned_reviewer": approval.AssignedReviewer,
		"decision_by":       approval.DecisionBy,
	})
	if err != nil {
		return fmt.Errorf("marshal audit payload: %w", err)
	}

	return s.audit.Record(ctx, auditdomain.Event{
		ID:           mustNewID(),
		OccurredAt:   now,
		ActorID:      strings.TrimSpace(cmd.ActorID),
		Action:       "workflow.transitioned",
		ResourceType: "document",
		ResourceID:   doc.ID,
		PayloadJSON:  string(payloadBytes),
		TraceID:      strings.TrimSpace(cmd.TraceID),
	})
}

func (s *Service) publishTransitionEvents(ctx context.Context, doc docdomain.Document, toStatus string, cmd workflowdomain.TransitionCommand, approval docdomain.WorkflowApproval, now time.Time) {
	if s.publisher == nil {
		return
	}

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
			"document_id":       doc.ID,
			"from_status":       doc.Status,
			"to_status":         toStatus,
			"reason":            strings.TrimSpace(cmd.Reason),
			"approval_id":       approval.ID,
			"approval_status":   approval.Status,
			"assigned_reviewer": approval.AssignedReviewer,
		},
	})

	if approval.ID == "" {
		return
	}

	eventType := "workflow.approval.decisioned"
	idempotencyKey := fmt.Sprintf("workflow-approval-decisioned-%s", approval.ID)
	if approval.Status == docdomain.WorkflowApprovalStatusPending {
		eventType = "workflow.approval.requested"
		idempotencyKey = fmt.Sprintf("workflow-approval-requested-%s", approval.ID)
	}

	_ = s.publisher.Publish(ctx, messaging.Event{
		EventID:           fmt.Sprintf("evt-%s", idempotencyKey),
		EventType:         eventType,
		AggregateType:     "document",
		AggregateID:       doc.ID,
		OccurredAtRFC3339: now.Format(time.RFC3339),
		Version:           1,
		IdempotencyKey:    idempotencyKey,
		Producer:          "workflow",
		TraceID:           strings.TrimSpace(cmd.TraceID),
		Payload: map[string]any{
			"document_id":       doc.ID,
			"approval_id":       approval.ID,
			"approval_status":   approval.Status,
			"requested_by":      approval.RequestedBy,
			"assigned_reviewer": approval.AssignedReviewer,
			"decision_by":       approval.DecisionBy,
			"request_reason":    approval.RequestReason,
			"decision_reason":   approval.DecisionReason,
		},
	})
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
		return fmt.Sprintf("workflow-fallback-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}

func mapApprovalError(err error) error {
	if errors.Is(err, docdomain.ErrWorkflowApprovalNotFound) {
		return workflowdomain.ErrApprovalNotFound
	}
	return err
}

func mapDocApprovalToWorkflowApproval(approval docdomain.WorkflowApproval) workflowdomain.Approval {
	return workflowdomain.Approval{
		ID:               approval.ID,
		DocumentID:       approval.DocumentID,
		RequestedBy:      approval.RequestedBy,
		AssignedReviewer: approval.AssignedReviewer,
		DecisionBy:       approval.DecisionBy,
		Status:           approval.Status,
		RequestReason:    approval.RequestReason,
		DecisionReason:   approval.DecisionReason,
		RequestedAt:      approval.RequestedAt,
		DecidedAt:        approval.DecidedAt,
	}
}
