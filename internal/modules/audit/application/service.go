package application

import (
	"context"
	"strings"

	"metaldocs/internal/modules/audit/domain"
)

type Service struct {
	reader domain.Reader
}

func NewService(reader domain.Reader) *Service {
	return &Service{reader: reader}
}

func (s *Service) ListEvents(ctx context.Context, query domain.ListEventsQuery) ([]domain.Event, error) {
	if s == nil || s.reader == nil {
		return []domain.Event{}, nil
	}

	normalized := domain.ListEventsQuery{
		ResourceType: strings.TrimSpace(query.ResourceType),
		ResourceID:   strings.TrimSpace(query.ResourceID),
		Limit:        query.Limit,
	}
	if normalized.Limit <= 0 {
		normalized.Limit = 50
	}
	if normalized.Limit > 200 {
		normalized.Limit = 200
	}

	return s.reader.ListEvents(ctx, normalized)
}
