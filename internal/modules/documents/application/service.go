package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"metaldocs/internal/modules/documents/domain"
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
	repo      domain.Repository
	publisher messaging.Publisher
	clock     Clock
}

func NewService(repo domain.Repository, publisher messaging.Publisher, clock Clock) *Service {
	if clock == nil {
		clock = realClock{}
	}
	return &Service{repo: repo, publisher: publisher, clock: clock}
}

func (s *Service) CreateDocument(ctx context.Context, cmd domain.CreateDocumentCommand) (domain.Document, error) {
	if strings.TrimSpace(cmd.DocumentID) == "" || strings.TrimSpace(cmd.Title) == "" || strings.TrimSpace(cmd.OwnerID) == "" {
		return domain.Document{}, domain.ErrInvalidCommand
	}

	classification := strings.TrimSpace(cmd.Classification)
	if classification == "" {
		classification = domain.ClassificationInternal
	}

	now := s.clock.Now()
	doc := domain.Document{
		ID:             strings.TrimSpace(cmd.DocumentID),
		Title:          strings.TrimSpace(cmd.Title),
		OwnerID:        strings.TrimSpace(cmd.OwnerID),
		Classification: classification,
		Status:         domain.StatusDraft,
		CreatedAt:      now,
	}

	v1 := domain.Version{
		DocumentID: doc.ID,
		Number:     1,
		Content:    cmd.InitialContent,
		CreatedAt:  now,
	}

	if atomicRepo, ok := s.repo.(domain.AtomicCreateRepository); ok {
		if err := atomicRepo.CreateDocumentWithInitialVersion(ctx, doc, v1); err != nil {
			return domain.Document{}, err
		}
	} else {
		if err := s.repo.CreateDocument(ctx, doc); err != nil {
			return domain.Document{}, err
		}
		if err := s.repo.SaveVersion(ctx, v1); err != nil {
			return domain.Document{}, err
		}
	}

	if s.publisher != nil {
		_ = s.publisher.Publish(ctx, messaging.Event{
			EventID:           fmt.Sprintf("evt-doc-create-%s", doc.ID),
			EventType:         "document.created",
			AggregateType:     "document",
			AggregateID:       doc.ID,
			OccurredAtRFC3339: now.Format(time.RFC3339),
			Version:           1,
			IdempotencyKey:    fmt.Sprintf("doc-create-%s", doc.ID),
			Producer:          "documents",
			TraceID:           cmd.TraceID,
			Payload: map[string]any{
				"document_id": doc.ID,
				"title":       doc.Title,
			},
		})

		_ = s.publisher.Publish(ctx, messaging.Event{
			EventID:           fmt.Sprintf("evt-doc-version-create-%s-1", doc.ID),
			EventType:         "document.version.created",
			AggregateType:     "document",
			AggregateID:       doc.ID,
			OccurredAtRFC3339: now.Format(time.RFC3339),
			Version:           1,
			IdempotencyKey:    fmt.Sprintf("doc-version-create-%s-1", doc.ID),
			Producer:          "documents",
			TraceID:           cmd.TraceID,
			Payload: map[string]any{
				"document_id": doc.ID,
				"version":     1,
			},
		})
	}

	return doc, nil
}

func (s *Service) AddVersion(ctx context.Context, cmd domain.AddVersionCommand) (domain.Version, error) {
	if strings.TrimSpace(cmd.DocumentID) == "" {
		return domain.Version{}, domain.ErrInvalidCommand
	}

	doc, err := s.repo.GetDocument(ctx, strings.TrimSpace(cmd.DocumentID))
	if err != nil {
		return domain.Version{}, err
	}

	next, err := s.repo.NextVersionNumber(ctx, doc.ID)
	if err != nil {
		return domain.Version{}, err
	}

	version := domain.Version{
		DocumentID: doc.ID,
		Number:     next,
		Content:    cmd.Content,
		CreatedAt:  s.clock.Now(),
	}

	if err := s.repo.SaveVersion(ctx, version); err != nil {
		return domain.Version{}, err
	}

	if s.publisher != nil {
		_ = s.publisher.Publish(ctx, messaging.Event{
			EventID:           fmt.Sprintf("evt-doc-version-create-%s-%d", doc.ID, next),
			EventType:         "document.version.created",
			AggregateType:     "document",
			AggregateID:       doc.ID,
			OccurredAtRFC3339: version.CreatedAt.Format(time.RFC3339),
			Version:           next,
			IdempotencyKey:    fmt.Sprintf("doc-version-create-%s-%d", doc.ID, next),
			Producer:          "documents",
			TraceID:           cmd.TraceID,
			Payload: map[string]any{
				"document_id": doc.ID,
				"version":     next,
			},
		})
	}

	return version, nil
}

func (s *Service) ListDocuments(ctx context.Context) ([]domain.Document, error) {
	return s.repo.ListDocuments(ctx)
}

func (s *Service) ListVersions(ctx context.Context, documentID string) ([]domain.Version, error) {
	if strings.TrimSpace(documentID) == "" {
		return nil, domain.ErrInvalidCommand
	}
	return s.repo.ListVersions(ctx, strings.TrimSpace(documentID))
}
