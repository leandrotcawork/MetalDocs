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

func (r *Reader) ListAccessPolicies(ctx context.Context, resourceScope, resourceID string) ([]searchdomain.AccessPolicy, error) {
	items, err := r.repo.ListAccessPolicies(ctx, resourceScope, resourceID)
	if err != nil {
		return nil, err
	}

	out := make([]searchdomain.AccessPolicy, 0, len(items))
	for _, item := range items {
		out = append(out, searchdomain.AccessPolicy{
			SubjectType:   item.SubjectType,
			SubjectID:     item.SubjectID,
			ResourceScope: item.ResourceScope,
			ResourceID:    item.ResourceID,
			Capability:    item.Capability,
			Effect:        item.Effect,
		})
	}
	return out, nil
}
