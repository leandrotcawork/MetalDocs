package application

import (
	"context"
	"sort"
	"strings"

	"metaldocs/internal/modules/search/domain"
)

const (
	defaultLimit = 20
	maxLimit     = 100
)

type Service struct {
	reader domain.Reader
}

func NewService(reader domain.Reader) *Service {
	return &Service{reader: reader}
}

func (s *Service) SearchDocuments(ctx context.Context, q domain.Query) ([]domain.Document, error) {
	docs, err := s.reader.ListDocuments(ctx)
	if err != nil {
		return nil, err
	}

	text := strings.ToLower(strings.TrimSpace(q.Text))
	ownerID := strings.TrimSpace(q.OwnerID)
	classification := strings.ToUpper(strings.TrimSpace(q.Classification))
	status := strings.ToUpper(strings.TrimSpace(q.Status))

	filtered := make([]domain.Document, 0, len(docs))
	for _, doc := range docs {
		if text != "" && !strings.Contains(strings.ToLower(doc.Title), text) {
			continue
		}
		if ownerID != "" && doc.OwnerID != ownerID {
			continue
		}
		if classification != "" && strings.ToUpper(doc.Classification) != classification {
			continue
		}
		if status != "" && strings.ToUpper(doc.Status) != status {
			continue
		}
		filtered = append(filtered, doc)
	}

	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].CreatedAt.After(filtered[j].CreatedAt)
	})

	limit := q.Limit
	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	if len(filtered) > limit {
		filtered = filtered[:limit]
	}

	return filtered, nil
}
