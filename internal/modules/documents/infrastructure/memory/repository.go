package memory

import (
	"context"
	"sort"
	"sync"
	"time"

	"metaldocs/internal/modules/documents/domain"
)

type Repository struct {
	mu                  sync.RWMutex
	documents           map[string]domain.Document
	versions            map[string][]domain.Version
	attachments         map[string]domain.Attachment
	documentAttachments map[string][]domain.Attachment
	families            []domain.DocumentFamily
	profiles            []domain.DocumentProfile
	profileSchemas      []domain.DocumentProfileSchemaVersion
	profileGovernance   map[string]domain.DocumentProfileGovernance
	processAreas        []domain.ProcessArea
	subjects            []domain.Subject
	types               []domain.DocumentType
	policies            map[string][]domain.AccessPolicy
}

func NewRepository() *Repository {
	return &Repository{
		documents:           map[string]domain.Document{},
		versions:            map[string][]domain.Version{},
		attachments:         map[string]domain.Attachment{},
		documentAttachments: map[string][]domain.Attachment{},
		families:            domain.DefaultDocumentFamilies(),
		profiles:            domain.DefaultDocumentProfiles(),
		profileSchemas:      domain.DefaultDocumentProfileSchemas(),
		profileGovernance:   domain.DefaultDocumentProfileGovernanceByCode(),
		processAreas:        domain.DefaultProcessAreas(),
		subjects:            domain.DefaultSubjects(),
		types:               domain.DefaultDocumentTypes(),
		policies:            map[string][]domain.AccessPolicy{},
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

func (r *Repository) ListDocumentsForReviewReminder(_ context.Context, fromInclusive, toInclusive time.Time) ([]domain.Document, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	from := fromInclusive.UTC()
	to := toInclusive.UTC()
	docs := make([]domain.Document, 0, len(r.documents))
	for _, doc := range r.documents {
		if doc.ExpiryAt == nil {
			continue
		}
		if doc.Status != domain.StatusPublished && doc.Status != domain.StatusApproved {
			continue
		}
		expiryUTC := doc.ExpiryAt.UTC()
		if expiryUTC.Before(from) || expiryUTC.After(to) {
			continue
		}
		docs = append(docs, doc)
	}

	sort.Slice(docs, func(i, j int) bool {
		left := docs[i].ExpiryAt.UTC()
		right := docs[j].ExpiryAt.UTC()
		if left.Equal(right) {
			return docs[i].CreatedAt.Before(docs[j].CreatedAt)
		}
		return left.Before(right)
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

func (r *Repository) ListDocumentFamilies(_ context.Context) ([]domain.DocumentFamily, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]domain.DocumentFamily, len(r.families))
	copy(out, r.families)
	return out, nil
}

func (r *Repository) ListDocumentProfiles(_ context.Context) ([]domain.DocumentProfile, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]domain.DocumentProfile, len(r.profiles))
	copy(out, r.profiles)
	return out, nil
}

func (r *Repository) UpsertDocumentProfile(_ context.Context, item domain.DocumentProfile) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for index := range r.profiles {
		if r.profiles[index].Code == item.Code {
			r.profiles[index] = item
			return nil
		}
	}
	r.profiles = append(r.profiles, item)
	return nil
}

func (r *Repository) DeactivateDocumentProfile(_ context.Context, code string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	filtered := make([]domain.DocumentProfile, 0, len(r.profiles))
	found := false
	for _, item := range r.profiles {
		if item.Code == code {
			found = true
			continue
		}
		filtered = append(filtered, item)
	}
	if !found {
		return domain.ErrInvalidCommand
	}
	r.profiles = filtered
	delete(r.profileGovernance, code)
	return nil
}

func (r *Repository) ListDocumentProfileSchemas(_ context.Context, profileCode string) ([]domain.DocumentProfileSchemaVersion, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	filtered := make([]domain.DocumentProfileSchemaVersion, 0, len(r.profileSchemas))
	for _, item := range r.profileSchemas {
		if profileCode != "" && item.ProfileCode != profileCode {
			continue
		}
		copiedRules := make([]domain.MetadataFieldRule, len(item.MetadataRules))
		copy(copiedRules, item.MetadataRules)
		filtered = append(filtered, domain.DocumentProfileSchemaVersion{
			ProfileCode:   item.ProfileCode,
			Version:       item.Version,
			IsActive:      item.IsActive,
			MetadataRules: copiedRules,
		})
	}
	return filtered, nil
}

func (r *Repository) UpsertDocumentProfileSchemaVersion(_ context.Context, item domain.DocumentProfileSchemaVersion) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for index := range r.profileSchemas {
		if r.profileSchemas[index].ProfileCode == item.ProfileCode && r.profileSchemas[index].Version == item.Version {
			r.profileSchemas[index] = item
			return nil
		}
	}
	r.profileSchemas = append(r.profileSchemas, item)
	return nil
}

func (r *Repository) ActivateDocumentProfileSchemaVersion(_ context.Context, profileCode string, version int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	found := false
	for index := range r.profileSchemas {
		if r.profileSchemas[index].ProfileCode != profileCode {
			continue
		}
		if r.profileSchemas[index].Version == version {
			found = true
			r.profileSchemas[index].IsActive = true
		} else {
			r.profileSchemas[index].IsActive = false
		}
	}
	if !found {
		return domain.ErrInvalidCommand
	}
	return nil
}

func (r *Repository) GetDocumentProfileGovernance(_ context.Context, profileCode string) (domain.DocumentProfileGovernance, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	item, ok := r.profileGovernance[profileCode]
	if !ok {
		return domain.DocumentProfileGovernance{}, domain.ErrInvalidCommand
	}
	return item, nil
}

func (r *Repository) UpsertDocumentProfileGovernance(_ context.Context, item domain.DocumentProfileGovernance) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.profileGovernance[item.ProfileCode] = item
	return nil
}

func (r *Repository) ListProcessAreas(_ context.Context) ([]domain.ProcessArea, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]domain.ProcessArea, len(r.processAreas))
	copy(out, r.processAreas)
	return out, nil
}

func (r *Repository) UpsertProcessArea(_ context.Context, item domain.ProcessArea) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for index := range r.processAreas {
		if r.processAreas[index].Code == item.Code {
			r.processAreas[index] = item
			return nil
		}
	}
	r.processAreas = append(r.processAreas, item)
	return nil
}

func (r *Repository) DeactivateProcessArea(_ context.Context, code string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	filtered := make([]domain.ProcessArea, 0, len(r.processAreas))
	found := false
	for _, item := range r.processAreas {
		if item.Code == code {
			found = true
			continue
		}
		filtered = append(filtered, item)
	}
	if !found {
		return domain.ErrInvalidCommand
	}
	r.processAreas = filtered
	return nil
}

func (r *Repository) ListSubjects(_ context.Context) ([]domain.Subject, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]domain.Subject, len(r.subjects))
	copy(out, r.subjects)
	return out, nil
}

func (r *Repository) UpsertSubject(_ context.Context, item domain.Subject) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for index := range r.subjects {
		if r.subjects[index].Code == item.Code {
			r.subjects[index] = item
			return nil
		}
	}
	r.subjects = append(r.subjects, item)
	return nil
}

func (r *Repository) DeactivateSubject(_ context.Context, code string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	filtered := make([]domain.Subject, 0, len(r.subjects))
	found := false
	for _, item := range r.subjects {
		if item.Code == code {
			found = true
			continue
		}
		filtered = append(filtered, item)
	}
	if !found {
		return domain.ErrInvalidCommand
	}
	r.subjects = filtered
	return nil
}

func (r *Repository) ListAccessPolicies(_ context.Context, resourceScope, resourceID string) ([]domain.AccessPolicy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	key := resourceScope + ":" + resourceID
	items := append([]domain.AccessPolicy(nil), r.policies[key]...)
	return items, nil
}

func (r *Repository) ReplaceAccessPolicies(_ context.Context, resourceScope, resourceID string, policies []domain.AccessPolicy) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := resourceScope + ":" + resourceID
	items := make([]domain.AccessPolicy, len(policies))
	copy(items, policies)
	r.policies[key] = items
	return nil
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

func (r *Repository) GetVersion(_ context.Context, documentID string, versionNumber int) (domain.Version, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if _, exists := r.documents[documentID]; !exists {
		return domain.Version{}, domain.ErrDocumentNotFound
	}
	for _, version := range r.versions[documentID] {
		if version.Number == versionNumber {
			return version, nil
		}
	}
	return domain.Version{}, domain.ErrVersionNotFound
}

func (r *Repository) NextVersionNumber(_ context.Context, documentID string) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if _, exists := r.documents[documentID]; !exists {
		return 0, domain.ErrDocumentNotFound
	}

	return len(r.versions[documentID]) + 1, nil
}

func (r *Repository) CreateAttachment(_ context.Context, attachment domain.Attachment) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.documents[attachment.DocumentID]; !exists {
		return domain.ErrDocumentNotFound
	}
	r.attachments[attachment.ID] = attachment
	r.documentAttachments[attachment.DocumentID] = append(r.documentAttachments[attachment.DocumentID], attachment)
	return nil
}

func (r *Repository) GetAttachment(_ context.Context, attachmentID string) (domain.Attachment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	attachment, exists := r.attachments[attachmentID]
	if !exists {
		return domain.Attachment{}, domain.ErrAttachmentNotFound
	}
	return attachment, nil
}

func (r *Repository) ListAttachments(_ context.Context, documentID string) ([]domain.Attachment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if _, exists := r.documents[documentID]; !exists {
		return nil, domain.ErrDocumentNotFound
	}
	items := append([]domain.Attachment(nil), r.documentAttachments[documentID]...)
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.Before(items[j].CreatedAt)
	})
	return items, nil
}
