package application

import (
	"context"
	"encoding/json"
	"strings"

	"metaldocs/internal/modules/documents/domain"
)

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
		items = domain.DefaultDocumentProfiles()
	}
	out := make([]domain.DocumentProfile, 0, len(items))
	for _, item := range items {
		normalized, err := domain.NormalizeDocumentProfile(item)
		if err != nil {
			return nil, err
		}
		out = append(out, normalized)
	}
	return out, nil
}

func (s *Service) UpsertDocumentProfile(ctx context.Context, item domain.DocumentProfile) error {
	normalized, err := domain.NormalizeDocumentProfile(item)
	if err != nil {
		return err
	}
	families, err := s.ListDocumentFamilies(ctx)
	if err != nil {
		return err
	}
	hasFamily := false
	for _, family := range families {
		if strings.EqualFold(family.Code, normalized.FamilyCode) {
			hasFamily = true
			break
		}
	}
	if !hasFamily {
		return domain.ErrInvalidCommand
	}
	return s.repo.UpsertDocumentProfile(ctx, normalized)
}

func (s *Service) DeactivateDocumentProfile(ctx context.Context, code string) error {
	normalizedCode := strings.ToLower(strings.TrimSpace(code))
	if normalizedCode == "" {
		return domain.ErrInvalidCommand
	}
	return s.repo.DeactivateDocumentProfile(ctx, normalizedCode)
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

	// Resolve ContentSchema from document_type_schema_versions if a matching type exists.
	// This is the single point where profile schemas are enriched with the type schema.
	// All downstream consumers (resolveActiveProfileSchema, resolveDocumentProfileSchema,
	// editor bundle, profile bundle, runtime bundle, export) call this method.
	if profileCode != "" {
		typeDef, err := s.repo.GetDocumentTypeDefinition(ctx, profileCode)
		if err == nil && typeDef.Schema.Sections != nil {
			schemaMap, marshalErr := typeSchemaToMap(typeDef.Schema)
			if marshalErr == nil {
				for i := range items {
					items[i].ContentSchema = schemaMap
				}
			}
		}
		// If type not found or marshal fails, fall back to profile's own content_schema_json (no-op).
	}

	return items, nil
}

func (s *Service) UpsertDocumentProfileSchemaVersion(ctx context.Context, item domain.DocumentProfileSchemaVersion) error {
	normalized, err := domain.NormalizeDocumentProfileSchemaVersion(item)
	if err != nil {
		return err
	}
	profiles, err := s.ListDocumentProfiles(ctx)
	if err != nil {
		return err
	}
	hasProfile := false
	for _, profile := range profiles {
		if strings.EqualFold(profile.Code, normalized.ProfileCode) {
			hasProfile = true
			break
		}
	}
	if !hasProfile {
		return domain.ErrInvalidCommand
	}
	return s.repo.UpsertDocumentProfileSchemaVersion(ctx, normalized)
}

func (s *Service) ActivateDocumentProfileSchemaVersion(ctx context.Context, profileCode string, version int) error {
	normalizedCode := strings.ToLower(strings.TrimSpace(profileCode))
	if normalizedCode == "" || version <= 0 {
		return domain.ErrInvalidCommand
	}
	items, err := s.ListDocumentProfileSchemas(ctx, normalizedCode)
	if err != nil {
		return err
	}
	found := false
	for _, item := range items {
		if item.Version == version {
			found = true
			break
		}
	}
	if !found {
		return domain.ErrInvalidCommand
	}
	return s.repo.ActivateDocumentProfileSchemaVersion(ctx, normalizedCode, version)
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

func (s *Service) UpsertDocumentProfileGovernance(ctx context.Context, item domain.DocumentProfileGovernance) error {
	normalized, err := domain.NormalizeDocumentProfileGovernance(item)
	if err != nil {
		return err
	}
	profiles, err := s.ListDocumentProfiles(ctx)
	if err != nil {
		return err
	}
	hasProfile := false
	for _, profile := range profiles {
		if strings.EqualFold(profile.Code, normalized.ProfileCode) {
			hasProfile = true
			break
		}
	}
	if !hasProfile {
		return domain.ErrInvalidCommand
	}
	return s.repo.UpsertDocumentProfileGovernance(ctx, normalized)
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

func (s *Service) ListDocumentDepartments(ctx context.Context) ([]domain.DocumentDepartment, error) {
	items, err := s.repo.ListDocumentDepartments(ctx)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return domain.DefaultDocumentDepartments(), nil
	}
	return items, nil
}

func (s *Service) UpsertProcessArea(ctx context.Context, item domain.ProcessArea) error {
	normalized, err := domain.NormalizeProcessArea(item)
	if err != nil {
		return err
	}
	return s.repo.UpsertProcessArea(ctx, normalized)
}

func (s *Service) UpsertDocumentDepartment(ctx context.Context, item domain.DocumentDepartment) error {
	normalized, err := domain.NormalizeDocumentDepartment(item)
	if err != nil {
		return err
	}
	return s.repo.UpsertDocumentDepartment(ctx, normalized)
}

func (s *Service) DeactivateProcessArea(ctx context.Context, code string) error {
	normalizedCode := strings.ToLower(strings.TrimSpace(code))
	if normalizedCode == "" {
		return domain.ErrInvalidCommand
	}
	return s.repo.DeactivateProcessArea(ctx, normalizedCode)
}

func (s *Service) DeactivateDocumentDepartment(ctx context.Context, code string) error {
	normalizedCode := strings.ToLower(strings.TrimSpace(code))
	if normalizedCode == "" {
		return domain.ErrInvalidCommand
	}
	return s.repo.DeactivateDocumentDepartment(ctx, normalizedCode)
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

func (s *Service) UpsertSubject(ctx context.Context, item domain.Subject) error {
	normalized, err := domain.NormalizeSubject(item)
	if err != nil {
		return err
	}
	areas, err := s.ListProcessAreas(ctx)
	if err != nil {
		return err
	}
	hasArea := false
	for _, area := range areas {
		if strings.EqualFold(area.Code, normalized.ProcessAreaCode) {
			hasArea = true
			break
		}
	}
	if !hasArea {
		return domain.ErrInvalidCommand
	}
	return s.repo.UpsertSubject(ctx, normalized)
}

func (s *Service) DeactivateSubject(ctx context.Context, code string) error {
	normalizedCode := strings.ToLower(strings.TrimSpace(code))
	if normalizedCode == "" {
		return domain.ErrInvalidCommand
	}
	return s.repo.DeactivateSubject(ctx, normalizedCode)
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

func (s *Service) resolveDocumentProfileSchema(ctx context.Context, profileCode string, version int) (domain.DocumentProfileSchemaVersion, bool, error) {
	items, err := s.ListDocumentProfileSchemas(ctx, profileCode)
	if err != nil {
		return domain.DocumentProfileSchemaVersion{}, false, err
	}
	if version > 0 {
		for _, item := range items {
			if item.Version == version {
				return item, true, nil
			}
		}
	}
	for _, item := range items {
		if item.IsActive {
			return item, true, nil
		}
	}
	if len(items) == 0 {
		return domain.DocumentProfileSchemaVersion{}, false, nil
	}
	return items[len(items)-1], true, nil
}

func typeSchemaToMap(schema domain.DocumentTypeSchema) (map[string]any, error) {
	raw, err := json.Marshal(schema)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
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
