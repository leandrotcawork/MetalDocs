package documents

import (
	"context"

	docdomain "metaldocs/internal/modules/documents/domain"
	searchdomain "metaldocs/internal/modules/search/domain"
)

type Reader struct {
	repo docdomain.Repository
}

func NewReader(repo docdomain.Repository) *Reader {
	return &Reader{repo: repo}
}

func (r *Reader) ListDocuments(ctx context.Context) ([]searchdomain.Document, error) {
	docs, err := r.repo.ListDocuments(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]searchdomain.Document, 0, len(docs))
	for _, doc := range docs {
		out = append(out, searchdomain.Document{
			ID:             doc.ID,
			Title:          doc.Title,
			DocumentType:   doc.DocumentType,
			OwnerID:        doc.OwnerID,
			BusinessUnit:   doc.BusinessUnit,
			Department:     doc.Department,
			Classification: doc.Classification,
			Status:         doc.Status,
			Tags:           append([]string(nil), doc.Tags...),
			EffectiveAt:    doc.EffectiveAt,
			ExpiryAt:       doc.ExpiryAt,
			CreatedAt:      doc.CreatedAt,
		})
	}
	return out, nil
}
