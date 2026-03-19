package application

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	docdomain "metaldocs/internal/modules/documents/domain"
	notificationdomain "metaldocs/internal/modules/notifications/domain"
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
	repo    notificationdomain.Repository
	docRepo docdomain.Repository
	clock   Clock
}

type OperationsSnapshot struct {
	PendingNotifications int       `json:"pendingNotifications"`
	PendingApprovals     int       `json:"pendingApprovals"`
	DocumentsInReview    int       `json:"documentsInReview"`
	TotalDocuments       int       `json:"totalDocuments"`
	GeneratedAt          time.Time `json:"generatedAt"`
}

var ErrNotificationNotFound = errors.New("notification not found")

func NewService(repo notificationdomain.Repository, docRepo docdomain.Repository, clock Clock) *Service {
	if clock == nil {
		clock = realClock{}
	}
	return &Service{repo: repo, docRepo: docRepo, clock: clock}
}

func (s *Service) ListNotifications(ctx context.Context, query notificationdomain.ListNotificationsQuery) ([]notificationdomain.Notification, error) {
	if query.Limit <= 0 {
		query.Limit = 50
	}
	return s.repo.List(ctx, query)
}

func (s *Service) BuildOperationsSnapshot(ctx context.Context, recipientUserID string) (OperationsSnapshot, error) {
	notifications, err := s.repo.List(ctx, notificationdomain.ListNotificationsQuery{
		RecipientUserID: strings.TrimSpace(recipientUserID),
		Status:          notificationdomain.StatusPending,
		Limit:           200,
	})
	if err != nil {
		return OperationsSnapshot{}, err
	}

	documents, err := s.docRepo.ListDocuments(ctx)
	if err != nil {
		return OperationsSnapshot{}, err
	}

	inReview := 0
	for _, document := range documents {
		if strings.EqualFold(document.Status, string(docdomain.StatusInReview)) {
			inReview++
		}
	}
	pendingApprovals := 0
	for _, notification := range notifications {
		if strings.EqualFold(notification.EventType, "workflow.approval.requested") {
			pendingApprovals++
		}
	}

	return OperationsSnapshot{
		PendingNotifications: len(notifications),
		PendingApprovals:     pendingApprovals,
		DocumentsInReview:    inReview,
		TotalDocuments:       len(documents),
		GeneratedAt:          s.clock.Now(),
	}, nil
}

func (s *Service) MarkNotificationRead(ctx context.Context, notificationID, recipientUserID string) error {
	if strings.TrimSpace(notificationID) == "" || strings.TrimSpace(recipientUserID) == "" {
		return ErrNotificationNotFound
	}
	return s.repo.MarkRead(ctx, strings.TrimSpace(notificationID), strings.TrimSpace(recipientUserID), s.clock.Now())
}

func (s *Service) HandleEvent(ctx context.Context, event messaging.Event) error {
	switch strings.TrimSpace(event.EventType) {
	case "workflow.approval.requested":
		return s.handleApprovalRequested(ctx, event)
	case "workflow.approval.decisioned":
		return s.handleApprovalDecisioned(ctx, event)
	default:
		return nil
	}
}

func (s *Service) EmitReviewReminders(ctx context.Context, withinDays int) error {
	now := s.clock.Now()
	deadline := now.Add(time.Duration(withinDays) * 24 * time.Hour)
	docs, err := s.docRepo.ListDocumentsForReviewReminder(ctx, now, deadline)
	if err != nil {
		return err
	}
	for _, doc := range docs {
		expiryUTC := doc.ExpiryAt.UTC()

		notification := notificationdomain.Notification{
			ID:              newNotificationID(),
			RecipientUserID: doc.OwnerID,
			EventType:       "document.review.reminder",
			ResourceType:    "document",
			ResourceID:      doc.ID,
			Title:           "Document review reminder",
			Message:         fmt.Sprintf("Document %s is approaching expiry/review at %s", doc.Title, expiryUTC.Format(time.RFC3339)),
			Status:          notificationdomain.StatusPending,
			IdempotencyKey:  fmt.Sprintf("review-reminder:%s:%s", doc.ID, expiryUTC.Format("2006-01-02")),
			CreatedAt:       now,
		}
		if err := s.repo.Create(ctx, notification); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) handleApprovalRequested(ctx context.Context, event messaging.Event) error {
	recipient, _ := event.Payload["assigned_reviewer"].(string)
	documentID, _ := event.Payload["document_id"].(string)
	if strings.TrimSpace(recipient) == "" || strings.TrimSpace(documentID) == "" {
		return nil
	}

	notification := notificationdomain.Notification{
		ID:              newNotificationID(),
		RecipientUserID: strings.TrimSpace(recipient),
		EventType:       event.EventType,
		ResourceType:    "document",
		ResourceID:      strings.TrimSpace(documentID),
		Title:           "Document approval requested",
		Message:         fmt.Sprintf("A document requires your review: %s", strings.TrimSpace(documentID)),
		Status:          notificationdomain.StatusPending,
		IdempotencyKey:  strings.TrimSpace(event.IdempotencyKey),
		CreatedAt:       s.clock.Now(),
	}
	return s.repo.Create(ctx, notification)
}

func (s *Service) handleApprovalDecisioned(ctx context.Context, event messaging.Event) error {
	recipient, _ := event.Payload["requested_by"].(string)
	documentID, _ := event.Payload["document_id"].(string)
	status, _ := event.Payload["approval_status"].(string)
	if strings.TrimSpace(recipient) == "" || strings.TrimSpace(documentID) == "" {
		return nil
	}

	title := "Document approval decided"
	message := fmt.Sprintf("Approval decision for document %s: %s", strings.TrimSpace(documentID), strings.TrimSpace(status))
	notification := notificationdomain.Notification{
		ID:              newNotificationID(),
		RecipientUserID: strings.TrimSpace(recipient),
		EventType:       event.EventType,
		ResourceType:    "document",
		ResourceID:      strings.TrimSpace(documentID),
		Title:           title,
		Message:         message,
		Status:          notificationdomain.StatusPending,
		IdempotencyKey:  strings.TrimSpace(event.IdempotencyKey),
		CreatedAt:       s.clock.Now(),
	}
	return s.repo.Create(ctx, notification)
}

func newNotificationID() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("notif-%d", time.Now().UTC().UnixNano())
	}
	return "notif_" + hex.EncodeToString(buf)
}
