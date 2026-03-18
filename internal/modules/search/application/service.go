package application

import (
	"context"
	"sort"
	"strings"
	"time"

	iamdomain "metaldocs/internal/modules/iam/domain"
	"metaldocs/internal/modules/search/domain"
)

const (
	defaultLimit            = 20
	maxLimit                = 100
	searchCapabilityView    = "document.view"
	searchPolicyEffectDeny  = "deny"
	searchSubjectTypeUser   = "user"
	searchSubjectTypeRole   = "role"
	searchScopeDocument     = "document"
	searchScopeDocumentType = "document_type"
	searchScopeArea         = "area"
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
	documentType := strings.ToLower(strings.TrimSpace(q.DocumentType))
	documentProfile := strings.ToLower(strings.TrimSpace(q.DocumentProfile))
	documentFamily := strings.ToLower(strings.TrimSpace(q.DocumentFamily))
	processArea := strings.ToLower(strings.TrimSpace(q.ProcessArea))
	subject := strings.ToLower(strings.TrimSpace(q.Subject))
	ownerID := strings.TrimSpace(q.OwnerID)
	businessUnit := strings.TrimSpace(q.BusinessUnit)
	department := strings.TrimSpace(q.Department)
	classification := strings.ToUpper(strings.TrimSpace(q.Classification))
	status := strings.ToUpper(strings.TrimSpace(q.Status))
	tag := strings.ToLower(strings.TrimSpace(q.Tag))

	filtered := make([]domain.Document, 0, len(docs))
	for _, doc := range docs {
		allowed, err := s.canView(ctx, doc)
		if err != nil {
			return nil, err
		}
		if !allowed {
			continue
		}
		if text != "" && !strings.Contains(strings.ToLower(doc.Title), text) {
			continue
		}
		if documentType != "" && strings.ToLower(doc.DocumentType) != documentType {
			continue
		}
		if documentProfile != "" && strings.ToLower(doc.DocumentProfile) != documentProfile {
			continue
		}
		if documentFamily != "" && strings.ToLower(doc.DocumentFamily) != documentFamily {
			continue
		}
		if processArea != "" && strings.ToLower(doc.ProcessArea) != processArea {
			continue
		}
		if subject != "" && strings.ToLower(doc.Subject) != subject {
			continue
		}
		if ownerID != "" && doc.OwnerID != ownerID {
			continue
		}
		if businessUnit != "" && doc.BusinessUnit != businessUnit {
			continue
		}
		if department != "" && doc.Department != department {
			continue
		}
		if classification != "" && strings.ToUpper(doc.Classification) != classification {
			continue
		}
		if status != "" && strings.ToUpper(doc.Status) != status {
			continue
		}
		if tag != "" && !hasTag(doc.Tags, tag) {
			continue
		}
		if q.ExpiryBefore != nil {
			if doc.ExpiryAt == nil || doc.ExpiryAt.After(q.ExpiryBefore.UTC()) {
				continue
			}
		}
		if q.ExpiryAfter != nil {
			if doc.ExpiryAt == nil || doc.ExpiryAt.Before(q.ExpiryAfter.UTC()) {
				continue
			}
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

func (s *Service) canView(ctx context.Context, doc domain.Document) (bool, error) {
	if shouldBypassPolicy(ctx) {
		return true, nil
	}

	policies, err := s.policiesForDocument(ctx, doc)
	if err != nil {
		return false, err
	}
	return decidePolicies(ctx, policies), nil
}

func (s *Service) policiesForDocument(ctx context.Context, doc domain.Document) ([]domain.AccessPolicy, error) {
	scopes := []struct {
		scope string
		id    string
	}{
		{scope: searchScopeDocument, id: doc.ID},
		{scope: searchScopeDocumentType, id: doc.DocumentProfile},
		{scope: searchScopeArea, id: areaResourceID(doc.BusinessUnit, doc.Department)},
	}

	var out []domain.AccessPolicy
	for _, scope := range scopes {
		if strings.TrimSpace(scope.id) == "" {
			continue
		}
		items, err := s.reader.ListAccessPolicies(ctx, scope.scope, scope.id)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			if item.Capability == searchCapabilityView {
				out = append(out, item)
			}
		}
	}
	return out, nil
}

func shouldBypassPolicy(ctx context.Context) bool {
	return iamdomain.UserIDFromContext(ctx) == "" && len(iamdomain.RolesFromContext(ctx)) == 0
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
		if item.Effect == searchPolicyEffectDeny {
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
	case searchSubjectTypeUser:
		return strings.EqualFold(item.SubjectID, userID)
	case searchSubjectTypeRole:
		_, ok := rolesSet[strings.ToLower(strings.TrimSpace(item.SubjectID))]
		return ok
	default:
		return false
	}
}

func areaResourceID(businessUnit, department string) string {
	return strings.TrimSpace(businessUnit) + ":" + strings.TrimSpace(department)
}

func hasTag(tags []string, expected string) bool {
	for _, tag := range tags {
		if strings.EqualFold(strings.TrimSpace(tag), expected) {
			return true
		}
	}
	return false
}

func cloneOptionalUTC(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	cloned := value.UTC()
	return &cloned
}
