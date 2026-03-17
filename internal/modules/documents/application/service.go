package application

import (
	"context"
	"encoding/json"
	"fmt"
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
		{scope: domain.ResourceScopeDocumentType, id: doc.DocumentType},
		{scope: domain.ResourceScopeArea, id: areaResourceID(doc.BusinessUnit, doc.Department)},
	}
	return s.loadPoliciesForScopes(ctx, scopes, capability)
}

func (s *Service) policiesForCreate(ctx context.Context, cmd domain.CreateDocumentCommand, capability string) ([]domain.AccessPolicy, error) {
	scopes := []struct {
		scope string
		id    string
	}{
		{scope: domain.ResourceScopeDocumentType, id: strings.ToLower(strings.TrimSpace(cmd.DocumentType))},
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
