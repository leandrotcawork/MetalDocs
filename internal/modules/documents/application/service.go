package application

import (
	"context"
	"encoding/json"
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
	if strings.TrimSpace(cmd.DocumentID) == "" ||
		strings.TrimSpace(cmd.Title) == "" ||
		strings.TrimSpace(cmd.OwnerID) == "" ||
		strings.TrimSpace(cmd.DocumentType) == "" ||
		strings.TrimSpace(cmd.BusinessUnit) == "" ||
		strings.TrimSpace(cmd.Department) == "" {
		return domain.Document{}, domain.ErrInvalidCommand
	}

	classification := strings.TrimSpace(cmd.Classification)
	if classification == "" {
		classification = domain.ClassificationInternal
	}

	documentType := strings.TrimSpace(strings.ToLower(cmd.DocumentType))
	if !s.isKnownDocumentType(ctx, documentType) {
		return domain.Document{}, domain.ErrInvalidDocumentType
	}

	metadata := normalizeMetadata(cmd.MetadataJSON)
	if _, err := json.Marshal(metadata); err != nil {
		return domain.Document{}, domain.ErrInvalidCommand
	}

	now := s.clock.Now()
	doc := domain.Document{
		ID:             strings.TrimSpace(cmd.DocumentID),
		Title:          strings.TrimSpace(cmd.Title),
		DocumentType:   documentType,
		OwnerID:        strings.TrimSpace(cmd.OwnerID),
		BusinessUnit:   strings.TrimSpace(cmd.BusinessUnit),
		Department:     strings.TrimSpace(cmd.Department),
		Classification: classification,
		Status:         domain.StatusDraft,
		Tags:           normalizeTags(cmd.Tags),
		EffectiveAt:    cloneTimePtr(cmd.EffectiveAt),
		ExpiryAt:       cloneTimePtr(cmd.ExpiryAt),
		MetadataJSON:   metadata,
		CreatedAt:      now,
		UpdatedAt:      now,
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
				"document_id":   doc.ID,
				"title":         doc.Title,
				"document_type": doc.DocumentType,
				"business_unit": doc.BusinessUnit,
				"department":    doc.Department,
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

func (s *Service) ListVersions(ctx context.Context, documentID string) ([]domain.Version, error) {
	if strings.TrimSpace(documentID) == "" {
		return nil, domain.ErrInvalidCommand
	}
	return s.repo.ListVersions(ctx, strings.TrimSpace(documentID))
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
	items, err := s.ListDocumentTypes(ctx)
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

func cloneTimePtr(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	cloned := value.UTC()
	return &cloned
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
	case domain.CapabilityDocumentView,
		domain.CapabilityDocumentEdit,
		domain.CapabilityDocumentUploadAttachment,
		domain.CapabilityDocumentChangeWorkflow,
		domain.CapabilityDocumentManagePermissions:
		return true
	default:
		return false
	}
}

func isKnownEffect(raw string) bool {
	switch raw {
	case domain.PolicyEffectAllow, domain.PolicyEffectDeny:
		return true
	default:
		return false
	}
}
