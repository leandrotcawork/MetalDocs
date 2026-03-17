package memory

import (
	"context"
	"sort"
	"sync"

	"metaldocs/internal/modules/documents/domain"
)

type Repository struct {
	mu        sync.RWMutex
	documents map[string]domain.Document
	versions  map[string][]domain.Version
	types     []domain.DocumentType
}

func NewRepository() *Repository {
	return &Repository{
		documents: map[string]domain.Document{},
		versions:  map[string][]domain.Version{},
		types:     domain.DefaultDocumentTypes(),
	}
}

func (r *Repository) CreateDocument(ctx context.Context, document domain.Document) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.createDocumentLocked(ctx, document)
}

func (r *Repository) CreateDocumentWithInitialVersion(ctx context.Context, document domain.Document, version domain.Version) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := r.createDocumentLocked(ctx, document); err != nil {
		return err
	}
	if err := r.saveVersionLocked(ctx, version); err != nil {
		delete(r.documents, document.ID)
		delete(r.versions, document.ID)
		return err
	}
	return nil
}

func (r *Repository) createDocumentLocked(_ context.Context, document domain.Document) error {
	if _, exists := r.documents[document.ID]; exists {
		return domain.ErrDocumentAlreadyExists
	}
	r.documents[document.ID] = document
	return nil
}

func (r *Repository) GetDocument(_ context.Context, documentID string) (domain.Document, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	doc, exists := r.documents[documentID]
	if !exists {
		return domain.Document{}, domain.ErrDocumentNotFound
	}

	return doc, nil
}

func (r *Repository) ListDocuments(_ context.Context) ([]domain.Document, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	docs := make([]domain.Document, 0, len(r.documents))
	for _, doc := range r.documents {
		docs = append(docs, doc)
	}

	sort.Slice(docs, func(i, j int) bool {
		return docs[i].CreatedAt.Before(docs[j].CreatedAt)
	})

	return docs, nil
}

func (r *Repository) ListDocumentTypes(_ context.Context) ([]domain.DocumentType, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]domain.DocumentType, len(r.types))
	copy(out, r.types)
	return out, nil
}

func (r *Repository) UpdateDocumentStatus(_ context.Context, documentID, status string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	doc, exists := r.documents[documentID]
	if !exists {
		return domain.ErrDocumentNotFound
	}
	doc.Status = status
	r.documents[documentID] = doc
	return nil
}

func (r *Repository) SaveVersion(ctx context.Context, version domain.Version) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.saveVersionLocked(ctx, version)
}

func (r *Repository) saveVersionLocked(_ context.Context, version domain.Version) error {
	if _, exists := r.documents[version.DocumentID]; !exists {
		return domain.ErrDocumentNotFound
	}
	r.versions[version.DocumentID] = append(r.versions[version.DocumentID], version)
	return nil
}

func (r *Repository) ListVersions(_ context.Context, documentID string) ([]domain.Version, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if _, exists := r.documents[documentID]; !exists {
		return nil, domain.ErrDocumentNotFound
	}

	versions := append([]domain.Version(nil), r.versions[documentID]...)
	sort.Slice(versions, func(i, j int) bool {
		return versions[i].Number < versions[j].Number
	})

	return versions, nil
}

func (r *Repository) NextVersionNumber(_ context.Context, documentID string) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if _, exists := r.documents[documentID]; !exists {
		return 0, domain.ErrDocumentNotFound
	}

	return len(r.versions[documentID]) + 1, nil
}
