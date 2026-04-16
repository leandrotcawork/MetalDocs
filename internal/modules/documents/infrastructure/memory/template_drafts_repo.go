package memory

import (
	"context"
	"encoding/json"
	"time"

	"metaldocs/internal/modules/documents/domain"
)

// GetTemplateDraft returns the draft for templateKey, or ErrTemplateDraftNotFound.
func (r *Repository) GetTemplateDraft(_ context.Context, templateKey string) (*domain.TemplateDraft, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	d, ok := r.templateDrafts[templateKey]
	if !ok {
		return nil, domain.ErrTemplateDraftNotFound
	}
	cp := cloneTemplateDraft(d)
	return &cp, nil
}

// UpsertTemplateDraftCAS saves the draft with compare-and-swap on LockVersion.
//
//   - expectedLockVersion == 0: creates a new draft (LockVersion set to 1).
//   - expectedLockVersion > 0: updates existing draft only if stored LockVersion matches.
//     Returns ErrTemplateLockConflict on mismatch; ErrTemplateDraftNotFound if not found.
func (r *Repository) UpsertTemplateDraftCAS(_ context.Context, draft *domain.TemplateDraft, expectedLockVersion int) (*domain.TemplateDraft, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if expectedLockVersion == 0 {
		// New draft.
		now := time.Now().UTC()
		saved := cloneTemplateDraft(*draft)
		saved.TemplateKey = draft.TemplateKey
		saved.LockVersion = 1
		saved.CreatedAt = now
		saved.UpdatedAt = now
		r.templateDrafts[draft.TemplateKey] = saved
		out := cloneTemplateDraft(saved)
		return &out, nil
	}

	// Update with CAS.
	existing, ok := r.templateDrafts[draft.TemplateKey]
	if !ok {
		return nil, domain.ErrTemplateDraftNotFound
	}
	if existing.LockVersion != expectedLockVersion {
		return nil, domain.ErrTemplateLockConflict
	}

	saved := cloneTemplateDraft(*draft)
	saved.TemplateKey = draft.TemplateKey
	saved.LockVersion = existing.LockVersion + 1
	saved.CreatedAt = existing.CreatedAt
	saved.CreatedBy = existing.CreatedBy
	saved.UpdatedAt = time.Now().UTC()
	r.templateDrafts[draft.TemplateKey] = saved

	out := cloneTemplateDraft(saved)
	return &out, nil
}

// DeleteTemplateDraft removes the draft. Returns ErrTemplateDraftNotFound if absent.
func (r *Repository) DeleteTemplateDraft(_ context.Context, templateKey string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.templateDrafts[templateKey]; !ok {
		return domain.ErrTemplateDraftNotFound
	}
	delete(r.templateDrafts, templateKey)
	return nil
}

func (r *Repository) UpdateTemplateDraftStatus(_ context.Context, templateKey string, newStatus domain.TemplateStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	d, ok := r.templateDrafts[templateKey]
	if !ok {
		return domain.ErrTemplateDraftNotFound
	}
	d.DraftStatus = newStatus
	r.templateDrafts[templateKey] = d
	return nil
}

func (r *Repository) SetTemplateDraftPublished(_ context.Context, templateKey string, publishedHTML string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	d, ok := r.templateDrafts[templateKey]
	if !ok {
		return domain.ErrTemplateDraftNotFound
	}
	d.DraftStatus = domain.TemplateStatusPublished
	d.PublishedHTML = &publishedHTML
	r.templateDrafts[templateKey] = d
	return nil
}

// InsertTemplateVersion inserts a new version into the in-memory store.
func (r *Repository) InsertTemplateVersion(_ context.Context, version domain.DocumentTemplateVersion) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.templateVersions[version.TemplateKey]; !ok {
		r.templateVersions[version.TemplateKey] = make(map[int]domain.DocumentTemplateVersion)
	}
	r.templateVersions[version.TemplateKey][version.Version] = cloneDocumentTemplateVersion(version)
	return nil
}

// PublishTemplateAtomic publishes a version and removes its draft under one lock.
func (r *Repository) PublishTemplateAtomic(_ context.Context, version *domain.DocumentTemplateVersion, draftKey domain.TemplateDraftKey) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := string(draftKey)
	if _, ok := r.templateDrafts[key]; !ok {
		return domain.ErrTemplateDraftNotFound
	}
	if _, ok := r.templateVersions[version.TemplateKey]; !ok {
		r.templateVersions[version.TemplateKey] = make(map[int]domain.DocumentTemplateVersion)
	}
	r.templateVersions[version.TemplateKey][version.Version] = cloneDocumentTemplateVersion(*version)
	delete(r.templateDrafts, key)
	return nil
}

// UpdateTemplateVersionStatus updates the status of a published template version.
func (r *Repository) UpdateTemplateVersionStatus(_ context.Context, templateKey string, version int, status domain.TemplateStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	versions, ok := r.templateVersions[templateKey]
	if !ok {
		return domain.ErrTemplateNotFound
	}
	tv, ok := versions[version]
	if !ok {
		return domain.ErrTemplateNotFound
	}
	tv.Status = string(status)
	versions[version] = tv
	return nil
}

// WriteTemplateAuditEvent appends an audit event to the in-memory audit log.
func (r *Repository) WriteTemplateAuditEvent(_ context.Context, event domain.TemplateAuditEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.templateAuditLog = append(r.templateAuditLog, event)
	return nil
}

// ListTemplateAuditEvents returns all audit events for a template key.
func (r *Repository) ListTemplateAuditEvents(_ context.Context, templateKey string) ([]domain.TemplateAuditEvent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var out []domain.TemplateAuditEvent
	for _, e := range r.templateAuditLog {
		if e.TemplateKey == templateKey {
			out = append(out, e)
		}
	}
	return out, nil
}

// cloneTemplateDraft deep-copies a TemplateDraft to prevent aliasing.
func cloneTemplateDraft(d domain.TemplateDraft) domain.TemplateDraft {
	out := d
	if len(d.ThemeJSON) > 0 {
		out.ThemeJSON = cloneJSON(d.ThemeJSON)
	}
	if len(d.MetaJSON) > 0 {
		out.MetaJSON = cloneJSON(d.MetaJSON)
	}
	if len(d.BlocksJSON) > 0 {
		out.BlocksJSON = cloneJSON(d.BlocksJSON)
	}
	if len(d.StrippedFieldsJSON) > 0 {
		out.StrippedFieldsJSON = cloneJSON(d.StrippedFieldsJSON)
	}
	if d.PublishedHTML != nil {
		s := *d.PublishedHTML
		out.PublishedHTML = &s
	}
	return out
}

func cloneJSON(b json.RawMessage) json.RawMessage {
	if len(b) == 0 {
		return nil
	}
	cp := make(json.RawMessage, len(b))
	copy(cp, b)
	return cp
}
