package application

import (
	"context"
	"strings"

	"metaldocs/internal/modules/documents/domain"
	"metaldocs/internal/platform/authn"
)

func buildAudiencePolicies(cmd domain.CreateDocumentCommand, classification string) ([]domain.AccessPolicy, error) {
	audience, err := normalizeAudience(cmd, classification)
	if err != nil {
		return nil, err
	}
	if audience == nil {
		return nil, nil
	}

	policies := make([]domain.AccessPolicy, 0)
	seen := map[string]struct{}{}
	addPolicy := func(subjectType, subjectID, capability string) error {
		item, ok := normalizeAccessPolicy(domain.ResourceScopeDocument, cmd.DocumentID, domain.AccessPolicy{
			SubjectType: subjectType,
			SubjectID:   subjectID,
			Capability:  capability,
			Effect:      domain.PolicyEffectAllow,
		})
		if !ok {
			return domain.ErrInvalidAccessPolicy
		}
		key := item.SubjectType + "|" + item.SubjectID + "|" + item.Capability
		if _, exists := seen[key]; exists {
			return nil
		}
		seen[key] = struct{}{}
		policies = append(policies, item)
		return nil
	}

	ownerID := strings.TrimSpace(cmd.OwnerID)
	if ownerID == "" {
		return nil, domain.ErrInvalidAccessPolicy
	}
	for _, capability := range []string{
		domain.CapabilityDocumentView,
		domain.CapabilityDocumentEdit,
		domain.CapabilityDocumentUploadAttachment,
	} {
		if err := addPolicy(domain.SubjectTypeUser, ownerID, capability); err != nil {
			return nil, err
		}
	}

	const adminRoleCode = "admin"
	for _, capability := range []string{
		domain.CapabilityDocumentView,
		domain.CapabilityDocumentEdit,
		domain.CapabilityDocumentUploadAttachment,
	} {
		if err := addPolicy(domain.SubjectTypeRole, adminRoleCode, capability); err != nil {
			return nil, err
		}
	}

	switch audience.Mode {
	case domain.AudienceModeDepartment:
		for _, dept := range audience.DepartmentCodes {
			if err := addPolicy(domain.SubjectTypeRole, "dept:"+dept, domain.CapabilityDocumentView); err != nil {
				return nil, err
			}
		}
	case domain.AudienceModeAreas:
		for _, dept := range audience.DepartmentCodes {
			for _, area := range audience.ProcessAreaCodes {
				compoundRole := "dept:" + dept + ":area:" + area
				if err := addPolicy(domain.SubjectTypeRole, compoundRole, domain.CapabilityDocumentView); err != nil {
					return nil, err
				}
			}
		}
	case domain.AudienceModeExplicit:
		for _, role := range audience.RoleCodes {
			if err := addPolicy(domain.SubjectTypeRole, role, domain.CapabilityDocumentView); err != nil {
				return nil, err
			}
		}
		for _, userID := range audience.UserIDs {
			if err := addPolicy(domain.SubjectTypeUser, userID, domain.CapabilityDocumentView); err != nil {
				return nil, err
			}
		}
	}

	return policies, nil
}

func normalizeAudience(cmd domain.CreateDocumentCommand, classification string) (*domain.DocumentAudience, error) {
	mode := ""
	if cmd.Audience != nil {
		mode = strings.ToUpper(strings.TrimSpace(cmd.Audience.Mode))
	}

	if mode == "" {
		switch classification {
		case domain.ClassificationConfidential, domain.ClassificationRestricted:
			dept := strings.ToLower(strings.TrimSpace(cmd.Department))
			if dept == "" {
				return nil, domain.ErrInvalidAccessPolicy
			}
			if classification == domain.ClassificationRestricted {
				area := strings.ToLower(strings.TrimSpace(cmd.ProcessArea))
				if area == "" {
					return nil, domain.ErrInvalidAccessPolicy
				}
				return &domain.DocumentAudience{
					Mode:             domain.AudienceModeAreas,
					DepartmentCodes:  []string{dept},
					ProcessAreaCodes: []string{area},
				}, nil
			}
			return &domain.DocumentAudience{
				Mode:            domain.AudienceModeDepartment,
				DepartmentCodes: []string{dept},
			}, nil
		default:
			return nil, nil
		}
	}

	switch mode {
	case domain.AudienceModeInternal:
		return nil, nil
	case domain.AudienceModeDepartment:
		if classification == domain.ClassificationRestricted {
			return nil, domain.ErrInvalidAccessPolicy
		}
		departments := normalizeCodeList(cmd.Audience.DepartmentCodes)
		if len(departments) == 0 {
			if dept := strings.ToLower(strings.TrimSpace(cmd.Department)); dept != "" {
				departments = []string{dept}
			}
		}
		if len(departments) == 0 {
			return nil, domain.ErrInvalidAccessPolicy
		}
		return &domain.DocumentAudience{
			Mode:            domain.AudienceModeDepartment,
			DepartmentCodes: departments,
		}, nil
	case domain.AudienceModeAreas:
		departments := normalizeCodeList(cmd.Audience.DepartmentCodes)
		if len(departments) == 0 {
			if dept := strings.ToLower(strings.TrimSpace(cmd.Department)); dept != "" {
				departments = []string{dept}
			}
		}
		areas := normalizeCodeList(cmd.Audience.ProcessAreaCodes)
		if len(areas) == 0 {
			if area := strings.ToLower(strings.TrimSpace(cmd.ProcessArea)); area != "" {
				areas = []string{area}
			}
		}
		if len(departments) == 0 || len(areas) == 0 {
			return nil, domain.ErrInvalidAccessPolicy
		}
		return &domain.DocumentAudience{
			Mode:             domain.AudienceModeAreas,
			DepartmentCodes:  departments,
			ProcessAreaCodes: areas,
		}, nil
	case domain.AudienceModeExplicit:
		roleCodes := normalizeCodeList(cmd.Audience.RoleCodes)
		userIDs := normalizeUserIDList(cmd.Audience.UserIDs)
		if len(roleCodes) == 0 && len(userIDs) == 0 {
			return nil, domain.ErrInvalidAccessPolicy
		}
		return &domain.DocumentAudience{
			Mode:      domain.AudienceModeExplicit,
			RoleCodes: roleCodes,
			UserIDs:   userIDs,
		}, nil
	default:
		return nil, domain.ErrInvalidAccessPolicy
	}
}

func normalizeCodeList(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, raw := range values {
		value := strings.ToLower(strings.TrimSpace(raw))
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func normalizeUserIDList(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, raw := range values {
		value := strings.TrimSpace(raw)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
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
	userID := authn.UserIDFromContext(ctx)
	roles := authn.RolesFromContext(ctx)
	rolesSet := map[string]struct{}{}
	for _, role := range roles {
		rolesSet[strings.ToLower(strings.TrimSpace(role))] = struct{}{}
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
	return authn.UserIDFromContext(ctx) == "" && len(authn.RolesFromContext(ctx)) == 0
}

func isKnownEffect(raw string) bool {
	switch raw {
	case domain.PolicyEffectAllow, domain.PolicyEffectDeny:
		return true
	default:
		return false
	}
}
