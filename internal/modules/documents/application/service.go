package application

import (
	"context"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"metaldocs/internal/modules/documents/domain"
	iamdomain "metaldocs/internal/modules/iam/domain"
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
	repo            domain.Repository
	attachmentStore domain.AttachmentStore
	publisher       messaging.Publisher
	clock           Clock
}

func NewService(repo domain.Repository, publisher messaging.Publisher, clock Clock) *Service {
	if clock == nil {
		clock = realClock{}
	}
	return &Service{repo: repo, publisher: publisher, clock: clock}
}

func (s *Service) WithAttachmentStore(store domain.AttachmentStore) *Service {
	s.attachmentStore = store
	return s
}

func (s *Service) CreateDocument(ctx context.Context, cmd domain.CreateDocumentCommand) (domain.Document, error) {
	if strings.TrimSpace(cmd.DocumentID) == "" ||
		strings.TrimSpace(cmd.Title) == "" ||
		strings.TrimSpace(cmd.OwnerID) == "" ||
		strings.TrimSpace(cmd.BusinessUnit) == "" ||
		strings.TrimSpace(cmd.Department) == "" {
		return domain.Document{}, domain.ErrInvalidCommand
	}

	classification := strings.TrimSpace(cmd.Classification)
	if classification == "" {
		classification = domain.ClassificationInternal
	}

	profile, err := s.resolveDocumentProfile(ctx, cmd.DocumentProfile, cmd.DocumentType)
	if err != nil {
		return domain.Document{}, domain.ErrInvalidDocumentType
	}
	activeSchema, err := s.resolveActiveProfileSchema(ctx, profile.Code)
	if err != nil {
		return domain.Document{}, domain.ErrInvalidCommand
	}
	processArea, subject, err := s.resolveTaxonomy(ctx, cmd.ProcessArea, cmd.Subject)
	if err != nil {
		return domain.Document{}, domain.ErrInvalidCommand
	}

	metadata := normalizeMetadata(cmd.MetadataJSON)
	if _, err := json.Marshal(metadata); err != nil {
		return domain.Document{}, domain.ErrInvalidCommand
	}
	if err := validateMetadata(activeSchema.MetadataRules, metadata); err != nil {
		return domain.Document{}, err
	}

	now := s.clock.Now()
	doc := domain.Document{
		ID:                   strings.TrimSpace(cmd.DocumentID),
		Title:                strings.TrimSpace(cmd.Title),
		DocumentType:         profile.Code,
		DocumentProfile:      profile.Code,
		DocumentFamily:       profile.FamilyCode,
		ProfileSchemaVersion: activeSchema.Version,
		ProcessArea:          processArea,
		Subject:              subject,
		OwnerID:              strings.TrimSpace(cmd.OwnerID),
		BusinessUnit:         strings.TrimSpace(cmd.BusinessUnit),
		Department:           strings.TrimSpace(cmd.Department),
		Classification:       classification,
		Status:               domain.StatusDraft,
		Tags:                 normalizeTags(cmd.Tags),
		EffectiveAt:          cloneTimePtr(cmd.EffectiveAt),
		ExpiryAt:             cloneTimePtr(cmd.ExpiryAt),
		MetadataJSON:         metadata,
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	v1 := domain.Version{
		DocumentID:    doc.ID,
		Number:        1,
		Content:       cmd.InitialContent,
		ContentHash:   contentHash(cmd.InitialContent),
		ChangeSummary: "Initial version",
		CreatedAt:     now,
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
				"document_id":      doc.ID,
				"title":            doc.Title,
				"document_type":    doc.DocumentType,
				"document_profile": doc.DocumentProfile,
				"document_family":  doc.DocumentFamily,
				"process_area":     doc.ProcessArea,
				"subject":          doc.Subject,
				"business_unit":    doc.BusinessUnit,
				"department":       doc.Department,
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

func (s *Service) CreateDocumentAuthorized(ctx context.Context, cmd domain.CreateDocumentCommand) (domain.Document, error) {
	if !s.isAllowedForCreate(ctx, cmd) {
		return domain.Document{}, domain.ErrDocumentNotFound
	}
	return s.CreateDocument(ctx, cmd)
}

func (s *Service) AddVersion(ctx context.Context, cmd domain.AddVersionCommand) (domain.Version, error) {
	if strings.TrimSpace(cmd.DocumentID) == "" {
		return domain.Version{}, domain.ErrInvalidCommand
	}

	doc, err := s.repo.GetDocument(ctx, strings.TrimSpace(cmd.DocumentID))
	if err != nil {
		return domain.Version{}, err
	}
	if doc.Status != domain.StatusDraft && doc.Status != domain.StatusInReview {
		return domain.Version{}, domain.ErrVersioningNotAllowed
	}

	next, err := s.repo.NextVersionNumber(ctx, doc.ID)
	if err != nil {
		return domain.Version{}, err
	}

	version := domain.Version{
		DocumentID:    doc.ID,
		Number:        next,
		Content:       cmd.Content,
		ContentHash:   contentHash(cmd.Content),
		ChangeSummary: strings.TrimSpace(cmd.ChangeSummary),
		CreatedAt:     s.clock.Now(),
	}
	if version.ChangeSummary == "" {
		version.ChangeSummary = fmt.Sprintf("Version %d update", next)
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

func (s *Service) AddVersionAuthorized(ctx context.Context, cmd domain.AddVersionCommand) (domain.Version, error) {
	if strings.TrimSpace(cmd.DocumentID) == "" {
		return domain.Version{}, domain.ErrInvalidCommand
	}
	doc, err := s.repo.GetDocument(ctx, strings.TrimSpace(cmd.DocumentID))
	if err != nil {
		return domain.Version{}, err
	}
	allowed, err := s.isAllowed(ctx, doc, domain.CapabilityDocumentEdit)
	if err != nil {
		return domain.Version{}, err
	}
	if !allowed {
		return domain.Version{}, domain.ErrDocumentNotFound
	}
	return s.AddVersion(ctx, cmd)
}

func (s *Service) DiffVersions(ctx context.Context, documentID string, fromVersion, toVersion int) (domain.VersionDiff, error) {
	if strings.TrimSpace(documentID) == "" || fromVersion < 1 || toVersion < 1 || fromVersion == toVersion {
		return domain.VersionDiff{}, domain.ErrInvalidCommand
	}
	doc, err := s.repo.GetDocument(ctx, strings.TrimSpace(documentID))
	if err != nil {
		return domain.VersionDiff{}, err
	}
	allowed, err := s.isAllowed(ctx, doc, domain.CapabilityDocumentView)
	if err != nil {
		return domain.VersionDiff{}, err
	}
	if !allowed {
		return domain.VersionDiff{}, domain.ErrDocumentNotFound
	}

	fromItem, err := s.repo.GetVersion(ctx, documentID, fromVersion)
	if err != nil {
		return domain.VersionDiff{}, err
	}
	toItem, err := s.repo.GetVersion(ctx, documentID, toVersion)
	if err != nil {
		return domain.VersionDiff{}, err
	}

	return domain.VersionDiff{
		DocumentID:            documentID,
		FromVersion:           fromVersion,
		ToVersion:             toVersion,
		ContentChanged:        fromItem.ContentHash != toItem.ContentHash,
		MetadataChanged:       []string{},
		ClassificationChanged: false,
		EffectiveAtChanged:    false,
		ExpiryAtChanged:       false,
	}, nil
}

func (s *Service) ListDocuments(ctx context.Context) ([]domain.Document, error) {
	return s.repo.ListDocuments(ctx)
}

func (s *Service) ListDocumentsAuthorized(ctx context.Context) ([]domain.Document, error) {
	docs, err := s.repo.ListDocuments(ctx)
	if err != nil {
		return nil, err
	}
	if shouldBypassPolicy(ctx) {
		return docs, nil
	}

	filtered := make([]domain.Document, 0, len(docs))
	for _, doc := range docs {
		allowed, err := s.isAllowed(ctx, doc, domain.CapabilityDocumentView)
		if err != nil {
			return nil, err
		}
		if allowed {
			filtered = append(filtered, doc)
		}
	}
	return filtered, nil
}

func (s *Service) ListDocumentTypes(ctx context.Context) ([]domain.DocumentType, error) {
	items, err := s.repo.ListDocumentTypes(ctx)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return domain.DefaultDocumentTypes(), nil
	}
	return items, nil
}

func (s *Service) ListDocumentFamilies(ctx context.Context) ([]domain.DocumentFamily, error) {
	items, err := s.repo.ListDocumentFamilies(ctx)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return domain.DefaultDocumentFamilies(), nil
	}
	return items, nil
}

func (s *Service) ListDocumentProfiles(ctx context.Context) ([]domain.DocumentProfile, error) {
	items, err := s.repo.ListDocumentProfiles(ctx)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return domain.DefaultDocumentProfiles(), nil
	}
	return items, nil
}

func (s *Service) ListDocumentProfileSchemas(ctx context.Context, profileCode string) ([]domain.DocumentProfileSchemaVersion, error) {
	profileCode = strings.ToLower(strings.TrimSpace(profileCode))
	items, err := s.repo.ListDocumentProfileSchemas(ctx, profileCode)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return filterDefaultSchemas(profileCode), nil
	}
	return items, nil
}

func (s *Service) GetDocumentProfileGovernance(ctx context.Context, profileCode string) (domain.DocumentProfileGovernance, error) {
	profileCode = strings.ToLower(strings.TrimSpace(profileCode))
	item, err := s.repo.GetDocumentProfileGovernance(ctx, profileCode)
	if err == nil {
		return item, nil
	}
	for _, fallback := range domain.DefaultDocumentProfileGovernance() {
		if strings.EqualFold(fallback.ProfileCode, profileCode) {
			return fallback, nil
		}
	}
	return domain.DocumentProfileGovernance{}, err
}

func (s *Service) ListProcessAreas(ctx context.Context) ([]domain.ProcessArea, error) {
	items, err := s.repo.ListProcessAreas(ctx)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return domain.DefaultProcessAreas(), nil
	}
	return items, nil
}

func (s *Service) ListSubjects(ctx context.Context) ([]domain.Subject, error) {
	items, err := s.repo.ListSubjects(ctx)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return domain.DefaultSubjects(), nil
	}
	return items, nil
}

func (s *Service) ListVersions(ctx context.Context, documentID string) ([]domain.Version, error) {
	if strings.TrimSpace(documentID) == "" {
		return nil, domain.ErrInvalidCommand
	}
	doc, err := s.repo.GetDocument(ctx, strings.TrimSpace(documentID))
	if err != nil {
		return nil, err
	}
	allowed, err := s.isAllowed(ctx, doc, domain.CapabilityDocumentView)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, domain.ErrDocumentNotFound
	}
	return s.repo.ListVersions(ctx, strings.TrimSpace(documentID))
}

func (s *Service) GetDocumentAuthorized(ctx context.Context, documentID string) (domain.Document, error) {
	if strings.TrimSpace(documentID) == "" {
		return domain.Document{}, domain.ErrInvalidCommand
	}
	doc, err := s.repo.GetDocument(ctx, strings.TrimSpace(documentID))
	if err != nil {
		return domain.Document{}, err
	}
	allowed, err := s.isAllowed(ctx, doc, domain.CapabilityDocumentView)
	if err != nil {
		return domain.Document{}, err
	}
	if !allowed {
		return domain.Document{}, domain.ErrDocumentNotFound
	}
	return doc, nil
}

func (s *Service) UploadAttachmentAuthorized(ctx context.Context, cmd domain.UploadAttachmentCommand) (domain.Attachment, error) {
	if s.attachmentStore == nil {
		return domain.Attachment{}, domain.ErrAttachmentStoreUnavailable
	}
	if strings.TrimSpace(cmd.DocumentID) == "" || strings.TrimSpace(cmd.FileName) == "" || len(cmd.Content) == 0 {
		return domain.Attachment{}, domain.ErrInvalidAttachment
	}
	if len(cmd.Content) > 10*1024*1024 {
		return domain.Attachment{}, domain.ErrInvalidAttachment
	}

	doc, err := s.repo.GetDocument(ctx, strings.TrimSpace(cmd.DocumentID))
	if err != nil {
		return domain.Attachment{}, err
	}
	allowed, err := s.isAllowed(ctx, doc, domain.CapabilityDocumentUploadAttachment)
	if err != nil {
		return domain.Attachment{}, err
	}
	if !allowed {
		return domain.Attachment{}, domain.ErrDocumentNotFound
	}

	attachmentID := newAttachmentID()
	storageKey := attachmentStorageKey(doc.ID, attachmentID, cmd.FileName)
	attachment := domain.Attachment{
		ID:          attachmentID,
		DocumentID:  doc.ID,
		FileName:    strings.TrimSpace(cmd.FileName),
		ContentType: normalizeContentType(cmd.ContentType),
		SizeBytes:   int64(len(cmd.Content)),
		StorageKey:  storageKey,
		UploadedBy:  strings.TrimSpace(cmd.UploadedBy),
		CreatedAt:   s.clock.Now(),
	}
	if attachment.UploadedBy == "" {
		attachment.UploadedBy = iamdomain.UserIDFromContext(ctx)
	}

	if err := s.attachmentStore.Save(ctx, storageKey, cmd.Content); err != nil {
		return domain.Attachment{}, err
	}
	if err := s.repo.CreateAttachment(ctx, attachment); err != nil {
		_ = s.attachmentStore.Delete(ctx, storageKey)
		return domain.Attachment{}, err
	}

	if s.publisher != nil {
		_ = s.publisher.Publish(ctx, messaging.Event{
			EventID:           fmt.Sprintf("evt-doc-attachment-create-%s", attachment.ID),
			EventType:         "document.attachment.created",
			AggregateType:     "document",
			AggregateID:       doc.ID,
			OccurredAtRFC3339: attachment.CreatedAt.Format(time.RFC3339),
			Version:           1,
			IdempotencyKey:    fmt.Sprintf("doc-attachment-create-%s", attachment.ID),
			Producer:          "documents",
			TraceID:           cmd.TraceID,
			Payload: map[string]any{
				"document_id":   doc.ID,
				"attachment_id": attachment.ID,
				"file_name":     attachment.FileName,
				"content_type":  attachment.ContentType,
				"size_bytes":    attachment.SizeBytes,
			},
		})
	}

	return attachment, nil
}

func (s *Service) ListAttachmentsAuthorized(ctx context.Context, documentID string) ([]domain.Attachment, error) {
	doc, err := s.GetDocumentAuthorized(ctx, documentID)
	if err != nil {
		return nil, err
	}
	return s.repo.ListAttachments(ctx, doc.ID)
}

func (s *Service) GetAttachmentAuthorized(ctx context.Context, documentID, attachmentID string) (domain.Attachment, error) {
	doc, err := s.GetDocumentAuthorized(ctx, documentID)
	if err != nil {
		return domain.Attachment{}, err
	}
	attachment, err := s.repo.GetAttachment(ctx, strings.TrimSpace(attachmentID))
	if err != nil {
		return domain.Attachment{}, err
	}
	if attachment.DocumentID != doc.ID {
		return domain.Attachment{}, domain.ErrAttachmentNotFound
	}
	return attachment, nil
}

func (s *Service) OpenAttachmentContent(ctx context.Context, attachmentID string) (domain.Attachment, []byte, error) {
	if s.attachmentStore == nil {
		return domain.Attachment{}, nil, domain.ErrAttachmentStoreUnavailable
	}
	attachment, err := s.repo.GetAttachment(ctx, strings.TrimSpace(attachmentID))
	if err != nil {
		return domain.Attachment{}, nil, err
	}
	reader, err := s.attachmentStore.Open(ctx, attachment.StorageKey)
	if err != nil {
		return domain.Attachment{}, nil, err
	}
	defer reader.Close()
	content, err := io.ReadAll(reader)
	if err != nil {
		return domain.Attachment{}, nil, err
	}
	return attachment, content, nil
}

func (s *Service) ListAccessPolicies(ctx context.Context, resourceScope, resourceID string) ([]domain.AccessPolicy, error) {
	resourceScope = normalizeResourceScope(resourceScope)
	resourceID = strings.TrimSpace(resourceID)
	if resourceScope == "" || resourceID == "" {
		return nil, domain.ErrInvalidCommand
	}
	if !isKnownResourceScope(resourceScope) {
		return nil, domain.ErrInvalidAccessPolicy
	}
	return s.repo.ListAccessPolicies(ctx, resourceScope, resourceID)
}

func (s *Service) ReplaceAccessPolicies(ctx context.Context, resourceScope, resourceID string, policies []domain.AccessPolicy) error {
	resourceScope = normalizeResourceScope(resourceScope)
	resourceID = strings.TrimSpace(resourceID)
	if resourceScope == "" || resourceID == "" {
		return domain.ErrInvalidCommand
	}
	if !isKnownResourceScope(resourceScope) {
		return domain.ErrInvalidAccessPolicy
	}

	normalized := make([]domain.AccessPolicy, 0, len(policies))
	for _, policy := range policies {
		item, ok := normalizeAccessPolicy(resourceScope, resourceID, policy)
		if !ok {
			return domain.ErrInvalidAccessPolicy
		}
		normalized = append(normalized, item)
	}
	return s.repo.ReplaceAccessPolicies(ctx, resourceScope, resourceID, normalized)
}

func (s *Service) isKnownDocumentType(ctx context.Context, code string) bool {
	items, err := s.ListDocumentProfiles(ctx)
	if err != nil {
		return false
	}
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(item.Code), code) {
			return true
		}
	}
	return false
}

func (s *Service) resolveDocumentProfile(ctx context.Context, preferredProfile, legacyDocumentType string) (domain.DocumentProfile, error) {
	code := strings.ToLower(strings.TrimSpace(preferredProfile))
	if code == "" {
		code = strings.ToLower(strings.TrimSpace(legacyDocumentType))
	}
	if code == "" {
		return domain.DocumentProfile{}, domain.ErrInvalidDocumentType
	}

	items, err := s.ListDocumentProfiles(ctx)
	if err != nil {
		return domain.DocumentProfile{}, err
	}
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(item.Code), code) {
			return item, nil
		}
	}
	return domain.DocumentProfile{}, domain.ErrInvalidDocumentType
}

func (s *Service) resolveTaxonomy(ctx context.Context, processAreaCode, subjectCode string) (string, string, error) {
	processAreaCode = strings.ToLower(strings.TrimSpace(processAreaCode))
	subjectCode = strings.ToLower(strings.TrimSpace(subjectCode))

	if processAreaCode == "" && subjectCode == "" {
		return "", "", nil
	}

	var selectedArea domain.ProcessArea
	if processAreaCode != "" {
		areas, err := s.ListProcessAreas(ctx)
		if err != nil {
			return "", "", err
		}
		found := false
		for _, item := range areas {
			if strings.EqualFold(strings.TrimSpace(item.Code), processAreaCode) {
				selectedArea = item
				found = true
				break
			}
		}
		if !found {
			return "", "", domain.ErrInvalidCommand
		}
	}

	if subjectCode == "" {
		return processAreaCode, "", nil
	}

	subjects, err := s.ListSubjects(ctx)
	if err != nil {
		return "", "", err
	}
	for _, item := range subjects {
		if !strings.EqualFold(strings.TrimSpace(item.Code), subjectCode) {
			continue
		}
		if processAreaCode != "" && !strings.EqualFold(strings.TrimSpace(item.ProcessAreaCode), selectedArea.Code) {
			return "", "", domain.ErrInvalidCommand
		}
		if processAreaCode == "" {
			processAreaCode = strings.ToLower(strings.TrimSpace(item.ProcessAreaCode))
		}
		return processAreaCode, subjectCode, nil
	}
	return "", "", domain.ErrInvalidCommand
}

func (s *Service) resolveActiveProfileSchema(ctx context.Context, profileCode string) (domain.DocumentProfileSchemaVersion, error) {
	items, err := s.ListDocumentProfileSchemas(ctx, profileCode)
	if err != nil {
		return domain.DocumentProfileSchemaVersion{}, err
	}
	for _, item := range items {
		if item.IsActive {
			return item, nil
		}
	}
	if len(items) == 0 {
		return domain.DocumentProfileSchemaVersion{}, domain.ErrInvalidCommand
	}
	return items[len(items)-1], nil
}

func normalizeTags(tags []string) []string {
	if len(tags) == 0 {
		return []string{}
	}
	out := make([]string, 0, len(tags))
	seen := map[string]struct{}{}
	for _, tag := range tags {
		normalized := strings.TrimSpace(tag)
		if normalized == "" {
			continue
		}
		key := strings.ToLower(normalized)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, normalized)
	}
	return out
}

func normalizeMetadata(metadata map[string]any) map[string]any {
	if len(metadata) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(metadata))
	for key, value := range metadata {
		trimmed := strings.TrimSpace(key)
		if trimmed == "" {
			continue
		}
		out[trimmed] = value
	}
	return out
}

func validateMetadata(rules []domain.MetadataFieldRule, metadata map[string]any) error {
	if len(rules) == 0 {
		return nil
	}
	for _, rule := range rules {
		value, exists := metadata[rule.Name]
		if rule.Required && (!exists || isEmptyMetadataValue(value)) {
			return domain.ErrInvalidMetadata
		}
		if exists && !matchesMetadataType(rule.Type, value) {
			return domain.ErrInvalidMetadata
		}
	}
	return nil
}

func filterDefaultSchemas(profileCode string) []domain.DocumentProfileSchemaVersion {
	if profileCode == "" {
		return domain.DefaultDocumentProfileSchemas()
	}
	items := domain.DefaultDocumentProfileSchemas()
	filtered := make([]domain.DocumentProfileSchemaVersion, 0, len(items))
	for _, item := range items {
		if strings.EqualFold(item.ProfileCode, profileCode) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func cloneTimePtr(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	cloned := value.UTC()
	return &cloned
}

func contentHash(value string) string {
	sum := md5.Sum([]byte(value))
	return fmt.Sprintf("%x", sum[:])
}

func normalizeAccessPolicy(resourceScope, resourceID string, policy domain.AccessPolicy) (domain.AccessPolicy, bool) {
	subjectType := strings.ToLower(strings.TrimSpace(policy.SubjectType))
	subjectID := strings.TrimSpace(policy.SubjectID)
	capability := strings.TrimSpace(policy.Capability)
	effect := strings.ToLower(strings.TrimSpace(policy.Effect))

	if !isKnownSubjectType(subjectType) || subjectID == "" || !isKnownCapability(capability) || !isKnownEffect(effect) {
		return domain.AccessPolicy{}, false
	}

	return domain.AccessPolicy{
		SubjectType:   subjectType,
		SubjectID:     subjectID,
		ResourceScope: resourceScope,
		ResourceID:    resourceID,
		Capability:    capability,
		Effect:        effect,
	}, true
}

func normalizeResourceScope(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}

func isKnownResourceScope(raw string) bool {
	switch raw {
	case domain.ResourceScopeDocument, domain.ResourceScopeDocumentType, domain.ResourceScopeArea:
		return true
	default:
		return false
	}
}

func isKnownSubjectType(raw string) bool {
	switch raw {
	case domain.SubjectTypeUser, domain.SubjectTypeRole, domain.SubjectTypeGroup:
		return true
	default:
		return false
	}
}

func isKnownCapability(raw string) bool {
	switch raw {
	case domain.CapabilityDocumentCreate,
		domain.CapabilityDocumentView,
		domain.CapabilityDocumentEdit,
		domain.CapabilityDocumentUploadAttachment,
		domain.CapabilityDocumentChangeWorkflow,
		domain.CapabilityDocumentManagePermissions:
		return true
	default:
		return false
	}
}

func (s *Service) isAllowedForCreate(ctx context.Context, cmd domain.CreateDocumentCommand) bool {
	if shouldBypassPolicy(ctx) {
		return true
	}
	items, err := s.policiesForCreate(ctx, cmd, domain.CapabilityDocumentCreate)
	if err != nil {
		return false
	}
	return decidePolicies(ctx, items)
}

func (s *Service) isAllowed(ctx context.Context, doc domain.Document, capability string) (bool, error) {
	if shouldBypassPolicy(ctx) {
		return true, nil
	}
	items, err := s.policiesForDocument(ctx, doc, capability)
	if err != nil {
		return false, err
	}
	return decidePolicies(ctx, items), nil
}

func (s *Service) policiesForDocument(ctx context.Context, doc domain.Document, capability string) ([]domain.AccessPolicy, error) {
	scopes := []struct {
		scope string
		id    string
	}{
		{scope: domain.ResourceScopeDocument, id: doc.ID},
		{scope: domain.ResourceScopeDocumentType, id: doc.DocumentProfile},
		{scope: domain.ResourceScopeArea, id: areaResourceID(doc.BusinessUnit, doc.Department)},
	}
	return s.loadPoliciesForScopes(ctx, scopes, capability)
}

func (s *Service) policiesForCreate(ctx context.Context, cmd domain.CreateDocumentCommand, capability string) ([]domain.AccessPolicy, error) {
	scopes := []struct {
		scope string
		id    string
	}{
		{scope: domain.ResourceScopeDocumentType, id: strings.ToLower(strings.TrimSpace(firstNonEmpty(cmd.DocumentProfile, cmd.DocumentType)))},
		{scope: domain.ResourceScopeArea, id: areaResourceID(cmd.BusinessUnit, cmd.Department)},
	}
	return s.loadPoliciesForScopes(ctx, scopes, capability)
}

func (s *Service) loadPoliciesForScopes(ctx context.Context, scopes []struct {
	scope string
	id    string
}, capability string) ([]domain.AccessPolicy, error) {
	var out []domain.AccessPolicy
	for _, scope := range scopes {
		if strings.TrimSpace(scope.id) == "" {
			continue
		}
		items, err := s.repo.ListAccessPolicies(ctx, scope.scope, scope.id)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			if item.Capability == capability {
				out = append(out, item)
			}
		}
	}
	return out, nil
}

func decidePolicies(ctx context.Context, items []domain.AccessPolicy) bool {
	if len(items) == 0 {
		return true
	}
	userID := iamdomain.UserIDFromContext(ctx)
	roles := iamdomain.RolesFromContext(ctx)
	rolesSet := map[string]struct{}{}
	for _, role := range roles {
		rolesSet[strings.ToLower(strings.TrimSpace(string(role)))] = struct{}{}
	}

	matchedAny := false
	for _, item := range items {
		if !matchesPolicySubject(item, userID, rolesSet) {
			continue
		}
		matchedAny = true
		if item.Effect == domain.PolicyEffectDeny {
			return false
		}
	}
	if matchedAny {
		return true
	}
	return false
}

func matchesPolicySubject(item domain.AccessPolicy, userID string, rolesSet map[string]struct{}) bool {
	switch item.SubjectType {
	case domain.SubjectTypeUser:
		return strings.EqualFold(item.SubjectID, userID)
	case domain.SubjectTypeRole:
		_, ok := rolesSet[strings.ToLower(strings.TrimSpace(item.SubjectID))]
		return ok
	default:
		return false
	}
}

func areaResourceID(businessUnit, department string) string {
	return strings.TrimSpace(businessUnit) + ":" + strings.TrimSpace(department)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func shouldBypassPolicy(ctx context.Context) bool {
	return iamdomain.UserIDFromContext(ctx) == "" && len(iamdomain.RolesFromContext(ctx)) == 0
}

func isKnownEffect(raw string) bool {
	switch raw {
	case domain.PolicyEffectAllow, domain.PolicyEffectDeny:
		return true
	default:
		return false
	}
}

func isEmptyMetadataValue(value any) bool {
	switch typed := value.(type) {
	case nil:
		return true
	case string:
		return strings.TrimSpace(typed) == ""
	default:
		return false
	}
}

func matchesMetadataType(expected string, value any) bool {
	switch expected {
	case "string":
		_, ok := value.(string)
		return ok && !isEmptyMetadataValue(value)
	case "date":
		raw, ok := value.(string)
		if !ok {
			return false
		}
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return false
		}
		if _, err := time.Parse("2006-01-02", raw); err == nil {
			return true
		}
		_, err := time.Parse(time.RFC3339, raw)
		return err == nil
	default:
		return true
	}
}

var attachmentSafeChars = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

func newAttachmentID() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("att_%d", time.Now().UTC().UnixNano())
	}
	return "att_" + hex.EncodeToString(buf)
}

func attachmentStorageKey(documentID, attachmentID, fileName string) string {
	safeName := attachmentSafeChars.ReplaceAllString(strings.TrimSpace(filepath.Base(fileName)), "_")
	if safeName == "" {
		safeName = "attachment.bin"
	}
	return strings.TrimSpace(documentID) + "/" + attachmentID + "/" + safeName
}

func normalizeContentType(value string) string {
	normalized := strings.TrimSpace(value)
	if normalized == "" {
		return "application/octet-stream"
	}
	return normalized
}
