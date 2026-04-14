package memory

import (
	"context"
	"fmt"
	"sort"
	"strings"
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
	departments         []domain.DocumentDepartment
	subjects            []domain.Subject
	types               []domain.DocumentType
	typeDefinitions     []domain.DocumentTypeDefinition
	templateVersions    map[string]map[int]domain.DocumentTemplateVersion
	templateDefaults    map[string]domain.DocumentTemplateVersion
	templateAssignments map[string]domain.DocumentTemplateAssignment
	policies            map[string][]domain.AccessPolicy
	collabPresence      map[string]map[string]domain.CollaborationPresence
	editLocks           map[string]domain.DocumentEditLock
	documentSequences   map[string]int
	templateDrafts      map[string]domain.TemplateDraft
	templateAuditLog    []domain.TemplateAuditEvent
}

func NewRepository() *Repository {
	repo := &Repository{
		documents:           map[string]domain.Document{},
		versions:            map[string][]domain.Version{},
		attachments:         map[string]domain.Attachment{},
		documentAttachments: map[string][]domain.Attachment{},
		families:            domain.DefaultDocumentFamilies(),
		profiles:            domain.DefaultDocumentProfiles(),
		profileSchemas:      domain.DefaultDocumentProfileSchemas(),
		profileGovernance:   domain.DefaultDocumentProfileGovernanceByCode(),
		processAreas:        domain.DefaultProcessAreas(),
		departments:         domain.DefaultDocumentDepartments(),
		subjects:            domain.DefaultSubjects(),
		types:               domain.DefaultDocumentTypes(),
		typeDefinitions:     domain.DefaultDocumentTypeDefinitions(),
		templateVersions:    map[string]map[int]domain.DocumentTemplateVersion{},
		templateDefaults:    map[string]domain.DocumentTemplateVersion{},
		templateAssignments: map[string]domain.DocumentTemplateAssignment{},
		policies:            map[string][]domain.AccessPolicy{},
		collabPresence:      map[string]map[string]domain.CollaborationPresence{},
		editLocks:           map[string]domain.DocumentEditLock{},
		documentSequences:   map[string]int{},
		templateDrafts:      map[string]domain.TemplateDraft{},
		templateAuditLog:    nil,
	}

	for _, item := range domain.DefaultDocumentTemplateVersions() {
		if _, ok := repo.templateVersions[item.TemplateKey]; !ok {
			repo.templateVersions[item.TemplateKey] = map[int]domain.DocumentTemplateVersion{}
		}
		repo.templateVersions[item.TemplateKey][item.Version] = cloneDocumentTemplateVersion(item)
		if item.ProfileCode != "" {
			repo.templateDefaults[item.ProfileCode] = cloneDocumentTemplateVersion(item)
		}
	}

	return repo
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

func (r *Repository) CreateDocumentWithInitialVersionAndPolicies(ctx context.Context, document domain.Document, version domain.Version, policies []domain.AccessPolicy) error {
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
	if len(policies) > 0 {
		key := policies[0].ResourceScope + ":" + policies[0].ResourceID
		items := make([]domain.AccessPolicy, len(policies))
		copy(items, policies)
		r.policies[key] = items
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

func (r *Repository) ReserveNextDocumentSequence(_ context.Context, profileCode string) (int, error) {
	code := strings.TrimSpace(profileCode)
	if code == "" {
		return 0, domain.ErrInvalidCommand
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	next := r.documentSequences[code]
	if next <= 0 {
		next = 1
	}
	r.documentSequences[code] = next + 1
	return next, nil
}

func (r *Repository) ListDocumentTypes(_ context.Context) ([]domain.DocumentType, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]domain.DocumentType, len(r.types))
	copy(out, r.types)
	return out, nil
}

func (r *Repository) ListDocumentTypeDefinitions(_ context.Context) ([]domain.DocumentTypeDefinition, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]domain.DocumentTypeDefinition, len(r.typeDefinitions))
	for i, item := range r.typeDefinitions {
		out[i] = cloneDocumentTypeDefinition(item)
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(strings.TrimSpace(out[i].Key)) < strings.ToLower(strings.TrimSpace(out[j].Key))
	})
	return out, nil
}

func (r *Repository) GetDocumentTypeDefinition(_ context.Context, key string) (domain.DocumentTypeDefinition, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	normalizedKey := strings.ToLower(strings.TrimSpace(key))
	for _, item := range r.typeDefinitions {
		if strings.EqualFold(item.Key, normalizedKey) {
			return cloneDocumentTypeDefinition(item), nil
		}
	}
	return domain.DocumentTypeDefinition{}, domain.ErrInvalidDocumentType
}

func (r *Repository) UpsertDocumentTypeDefinition(_ context.Context, item domain.DocumentTypeDefinition) error {
	normalized, err := normalizeDocumentTypeDefinition(item)
	if err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for index := range r.typeDefinitions {
		if strings.EqualFold(r.typeDefinitions[index].Key, normalized.Key) {
			r.typeDefinitions[index] = cloneDocumentTypeDefinition(normalized)
			return nil
		}
	}
	r.typeDefinitions = append(r.typeDefinitions, cloneDocumentTypeDefinition(normalized))
	return nil
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
			ContentSchema: cloneContentSchema(item.ContentSchema),
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

func cloneContentSchema(value map[string]any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(value))
	for key, item := range value {
		out[key] = item
	}
	return out
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

func (r *Repository) ListDocumentDepartments(_ context.Context) ([]domain.DocumentDepartment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]domain.DocumentDepartment, len(r.departments))
	copy(out, r.departments)
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

func (r *Repository) UpsertDocumentDepartment(_ context.Context, item domain.DocumentDepartment) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for index := range r.departments {
		if r.departments[index].Code == item.Code {
			r.departments[index] = item
			return nil
		}
	}
	r.departments = append(r.departments, item)
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

func (r *Repository) DeactivateDocumentDepartment(_ context.Context, code string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	filtered := make([]domain.DocumentDepartment, 0, len(r.departments))
	found := false
	for _, item := range r.departments {
		if item.Code == code {
			found = true
			continue
		}
		filtered = append(filtered, item)
	}
	if !found {
		return domain.ErrInvalidCommand
	}
	r.departments = filtered
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

func (r *Repository) UpdateVersionPDF(_ context.Context, documentID string, versionNumber int, pdfStorageKey string, pageCount int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.documents[documentID]; !exists {
		return domain.ErrDocumentNotFound
	}
	versions := r.versions[documentID]
	for i := range versions {
		if versions[i].Number == versionNumber {
			versions[i].PdfStorageKey = pdfStorageKey
			if pageCount > 0 {
				versions[i].PageCount = pageCount
			}
			r.versions[documentID] = versions
			return nil
		}
	}
	return domain.ErrVersionNotFound
}

func (r *Repository) UpdateVersionBodyBlocks(_ context.Context, documentID string, versionNumber int, bodyBlocks []domain.EtapaBody) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.documents[documentID]; !exists {
		return domain.ErrDocumentNotFound
	}
	versions := r.versions[documentID]
	for i := range versions {
		if versions[i].Number == versionNumber {
			versions[i].BodyBlocks = bodyBlocks
			r.versions[documentID] = versions
			return nil
		}
	}
	return domain.ErrVersionNotFound
}

func (r *Repository) UpdateVersionValues(_ context.Context, documentID string, versionNumber int, values domain.DocumentValues) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.documents[documentID]; !exists {
		return domain.ErrDocumentNotFound
	}
	versions := r.versions[documentID]
	for i := range versions {
		if versions[i].Number == versionNumber {
			versions[i].Values = cloneRuntimeValues(values)
			r.versions[documentID] = versions
			return nil
		}
	}
	return domain.ErrVersionNotFound
}

func (r *Repository) GetDocumentTemplateVersion(_ context.Context, templateKey string, version int) (domain.DocumentTemplateVersion, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	key := strings.TrimSpace(templateKey)
	if key == "" || version <= 0 {
		return domain.DocumentTemplateVersion{}, domain.ErrDocumentTemplateNotFound
	}
	versions, ok := r.templateVersions[key]
	if !ok {
		return domain.DocumentTemplateVersion{}, domain.ErrDocumentTemplateNotFound
	}
	item, ok := versions[version]
	if !ok {
		return domain.DocumentTemplateVersion{}, domain.ErrDocumentTemplateNotFound
	}
	return cloneDocumentTemplateVersion(item), nil
}

func (r *Repository) ListDocumentTemplateVersions(_ context.Context, profileCode string) ([]domain.DocumentTemplateVersion, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	normalizedProfileCode := strings.ToLower(strings.TrimSpace(profileCode))
	templateKeys := make([]string, 0, len(r.templateVersions))
	for templateKey := range r.templateVersions {
		templateKeys = append(templateKeys, templateKey)
	}
	sort.Strings(templateKeys)

	items := make([]domain.DocumentTemplateVersion, 0)
	for _, templateKey := range templateKeys {
		versions := r.templateVersions[templateKey]
		versionNumbers := make([]int, 0, len(versions))
		for version := range versions {
			versionNumbers = append(versionNumbers, version)
		}
		sort.Sort(sort.Reverse(sort.IntSlice(versionNumbers)))
		for _, version := range versionNumbers {
			item := versions[version]
			if normalizedProfileCode != "" && !strings.EqualFold(item.ProfileCode, normalizedProfileCode) {
				continue
			}
			items = append(items, cloneDocumentTemplateVersion(item))
		}
	}
	return items, nil
}

func (r *Repository) GetDefaultDocumentTemplate(_ context.Context, profileCode string) (domain.DocumentTemplateVersion, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	item, ok := r.templateDefaults[strings.TrimSpace(profileCode)]
	if !ok {
		return domain.DocumentTemplateVersion{}, domain.ErrDocumentTemplateNotFound
	}
	return cloneDocumentTemplateVersion(item), nil
}

func (r *Repository) GetDocumentTemplateAssignment(_ context.Context, documentID string) (domain.DocumentTemplateAssignment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	item, ok := r.templateAssignments[strings.TrimSpace(documentID)]
	if !ok {
		return domain.DocumentTemplateAssignment{}, domain.ErrDocumentTemplateAssignmentNotFound
	}
	return item, nil
}

func (r *Repository) UpsertDocumentTemplateAssignment(_ context.Context, item domain.DocumentTemplateAssignment) error {
	normalized := domain.DocumentTemplateAssignment{
		DocumentID:      strings.TrimSpace(item.DocumentID),
		TemplateKey:     strings.TrimSpace(item.TemplateKey),
		TemplateVersion: item.TemplateVersion,
		AssignedAt:      item.AssignedAt,
	}
	if normalized.DocumentID == "" || normalized.TemplateKey == "" || normalized.TemplateVersion <= 0 {
		return domain.ErrInvalidCommand
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.templateAssignments[normalized.DocumentID] = normalized
	return nil
}

func (r *Repository) UpsertDocumentTemplateVersionForTest(_ context.Context, item domain.DocumentTemplateVersion) error {
	normalized := cloneDocumentTemplateVersion(item)
	if strings.TrimSpace(normalized.TemplateKey) == "" || normalized.Version <= 0 || strings.TrimSpace(normalized.ProfileCode) == "" {
		return domain.ErrInvalidCommand
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.templateVersions[normalized.TemplateKey]; !ok {
		r.templateVersions[normalized.TemplateKey] = map[int]domain.DocumentTemplateVersion{}
	}
	r.templateVersions[normalized.TemplateKey][normalized.Version] = normalized
	return nil
}

func (r *Repository) UpdateDraftVersionContentCAS(_ context.Context, version domain.Version, expectedContentHash string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.documents[version.DocumentID]; !exists {
		return domain.ErrDocumentNotFound
	}
	versions := r.versions[version.DocumentID]
	for i := range versions {
		if versions[i].Number != version.Number {
			continue
		}
		if versions[i].ContentHash != expectedContentHash {
			return domain.ErrDraftConflict
		}
		versions[i] = cloneVersion(version)
		r.versions[version.DocumentID] = versions
		return nil
	}
	return domain.ErrVersionNotFound
}

func (r *Repository) UpdateVersionDocx(_ context.Context, documentID string, versionNumber int, docxStorageKey string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.documents[documentID]; !exists {
		return domain.ErrDocumentNotFound
	}
	versions := r.versions[documentID]
	for i := range versions {
		if versions[i].Number == versionNumber {
			versions[i].DocxStorageKey = docxStorageKey
			r.versions[documentID] = versions
			return nil
		}
	}
	return domain.ErrVersionNotFound
}

func (r *Repository) SetVersionRendererPin(_ context.Context, documentID string, versionNumber int, pin *domain.RendererPin) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	versions, ok := r.versions[documentID]
	if !ok {
		return fmt.Errorf("document %s not found", documentID)
	}
	for i, v := range versions {
		if v.Number == versionNumber {
			if pin != nil {
				if err := pin.Validate(); err != nil {
					return fmt.Errorf("invalid renderer pin: %w", err)
				}
			}
			versions[i].RendererPin = pin
			r.versions[documentID] = versions
			return nil
		}
	}
	return fmt.Errorf("version %d of document %s not found", versionNumber, documentID)
}

func (r *Repository) saveVersionLocked(_ context.Context, version domain.Version) error {
	if _, exists := r.documents[version.DocumentID]; !exists {
		return domain.ErrDocumentNotFound
	}
	r.versions[version.DocumentID] = append(r.versions[version.DocumentID], cloneVersion(version))
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
	for i := range versions {
		versions[i].Values = cloneRuntimeValues(versions[i].Values)
		versions[i].NativeContent = cloneRuntimeValues(versions[i].NativeContent)
	}

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
			version.Values = cloneRuntimeValues(version.Values)
			version.NativeContent = cloneRuntimeValues(version.NativeContent)
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

func (r *Repository) UpsertCollaborationPresence(_ context.Context, item domain.CollaborationPresence) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.documents[item.DocumentID]; !exists {
		return domain.ErrDocumentNotFound
	}
	if _, ok := r.collabPresence[item.DocumentID]; !ok {
		r.collabPresence[item.DocumentID] = map[string]domain.CollaborationPresence{}
	}
	r.collabPresence[item.DocumentID][item.UserID] = item
	return nil
}

func (r *Repository) ListCollaborationPresence(_ context.Context, documentID string, activeSince time.Time) ([]domain.CollaborationPresence, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if _, exists := r.documents[documentID]; !exists {
		return nil, domain.ErrDocumentNotFound
	}
	items := make([]domain.CollaborationPresence, 0)
	for _, presence := range r.collabPresence[documentID] {
		if presence.LastSeenAt.Before(activeSince) {
			continue
		}
		items = append(items, presence)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].LastSeenAt.After(items[j].LastSeenAt)
	})
	return items, nil
}

func (r *Repository) AcquireDocumentEditLock(_ context.Context, item domain.DocumentEditLock, now time.Time) (domain.DocumentEditLock, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.documents[item.DocumentID]; !exists {
		return domain.DocumentEditLock{}, domain.ErrDocumentNotFound
	}
	if current, ok := r.editLocks[item.DocumentID]; ok {
		if current.ExpiresAt.After(now) && !strings.EqualFold(current.LockedBy, item.LockedBy) {
			return domain.DocumentEditLock{}, domain.ErrEditLockActive
		}
	}
	r.editLocks[item.DocumentID] = item
	return item, nil
}

func (r *Repository) GetDocumentEditLock(_ context.Context, documentID string, now time.Time) (domain.DocumentEditLock, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if _, exists := r.documents[documentID]; !exists {
		return domain.DocumentEditLock{}, domain.ErrDocumentNotFound
	}
	lock, ok := r.editLocks[documentID]
	if !ok || !lock.ExpiresAt.After(now) {
		return domain.DocumentEditLock{}, domain.ErrEditLockNotFound
	}
	return lock, nil
}

func (r *Repository) ReleaseDocumentEditLock(_ context.Context, documentID, lockedBy string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	lock, ok := r.editLocks[documentID]
	if !ok {
		return domain.ErrEditLockNotFound
	}
	if !strings.EqualFold(lock.LockedBy, lockedBy) {
		return domain.ErrEditLockActive
	}
	delete(r.editLocks, documentID)
	return nil
}

func normalizeDocumentTypeDefinition(item domain.DocumentTypeDefinition) (domain.DocumentTypeDefinition, error) {
	item.Key = strings.ToLower(strings.TrimSpace(item.Key))
	item.Name = strings.TrimSpace(item.Name)
	if item.Key == "" || item.Name == "" {
		return domain.DocumentTypeDefinition{}, domain.ErrInvalidCommand
	}
	if item.ActiveVersion <= 0 {
		item.ActiveVersion = 1
	}
	item.Schema = cloneDocumentTypeSchema(item.Schema)
	return item, nil
}

func cloneDocumentTypeDefinition(item domain.DocumentTypeDefinition) domain.DocumentTypeDefinition {
	item.Schema = cloneDocumentTypeSchema(item.Schema)
	return item
}

func cloneDocumentTypeSchema(schema domain.DocumentTypeSchema) domain.DocumentTypeSchema {
	if len(schema.Sections) == 0 {
		return domain.DocumentTypeSchema{}
	}
	out := domain.DocumentTypeSchema{Sections: make([]domain.SectionDef, len(schema.Sections))}
	for i, section := range schema.Sections {
		out.Sections[i] = cloneSectionDef(section)
	}
	return out
}

func cloneSectionDef(section domain.SectionDef) domain.SectionDef {
	out := domain.SectionDef{
		Key:   section.Key,
		Num:   section.Num,
		Title: section.Title,
		Color: section.Color,
	}
	if len(section.Fields) > 0 {
		out.Fields = make([]domain.FieldDef, len(section.Fields))
		for i, field := range section.Fields {
			out.Fields[i] = cloneFieldDef(field)
		}
	}
	return out
}

func cloneFieldDef(field domain.FieldDef) domain.FieldDef {
	out := domain.FieldDef{
		Key:   field.Key,
		Label: field.Label,
		Type:  field.Type,
	}
	if len(field.Options) > 0 {
		out.Options = append([]string(nil), field.Options...)
	}
	if len(field.Columns) > 0 {
		out.Columns = make([]domain.FieldDef, len(field.Columns))
		for i, column := range field.Columns {
			out.Columns[i] = cloneFieldDef(column)
		}
	}
	if len(field.ItemFields) > 0 {
		out.ItemFields = make([]domain.FieldDef, len(field.ItemFields))
		for i, itemField := range field.ItemFields {
			out.ItemFields[i] = cloneFieldDef(itemField)
		}
	}
	return out
}

func cloneRuntimeValues(values map[string]any) map[string]any {
	if len(values) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(values))
	for key, value := range values {
		out[key] = cloneRuntimeValue(value)
	}
	return out
}

func cloneRuntimeValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return cloneRuntimeValues(typed)
	case []any:
		out := make([]any, len(typed))
		for i, item := range typed {
			out[i] = cloneRuntimeValue(item)
		}
		return out
	case []map[string]any:
		out := make([]map[string]any, len(typed))
		for i, item := range typed {
			out[i] = cloneRuntimeValues(item)
		}
		return out
	default:
		return typed
	}
}

func cloneVersion(item domain.Version) domain.Version {
	item.NativeContent = cloneRuntimeValues(item.NativeContent)
	item.Values = cloneRuntimeValues(item.Values)
	if len(item.BodyBlocks) > 0 {
		item.BodyBlocks = append([]domain.EtapaBody(nil), item.BodyBlocks...)
	}
	return item
}

func cloneDocumentTemplateVersion(item domain.DocumentTemplateVersion) domain.DocumentTemplateVersion {
	item.Definition = cloneRuntimeValues(item.Definition)
	return item
}
