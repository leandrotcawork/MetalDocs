package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"metaldocs/internal/modules/documents/domain"
)

func (s *Service) ResolveDocumentTemplate(ctx context.Context, documentID, profileCode string) (domain.DocumentTemplateVersion, error) {
	assignment, err := s.repo.GetDocumentTemplateAssignment(ctx, documentID)
	var templateVersion domain.DocumentTemplateVersion
	if err == nil {
		templateVersion, err = s.repo.GetDocumentTemplateVersion(ctx, assignment.TemplateKey, assignment.TemplateVersion)
	} else if errors.Is(err, domain.ErrDocumentTemplateAssignmentNotFound) {
		templateVersion, err = s.repo.GetDefaultDocumentTemplate(ctx, profileCode)
	} else {
		return domain.DocumentTemplateVersion{}, err
	}

	if err != nil {
		return domain.DocumentTemplateVersion{}, err
	}

	if err := s.validateDocumentTemplateCompatibility(ctx, templateVersion); err != nil {
		return domain.DocumentTemplateVersion{}, err
	}

	return templateVersion, nil
}

func (s *Service) ListDocumentTemplates(ctx context.Context, profileCode string) ([]domain.DocumentTemplateVersion, error) {
	normalizedProfileCode := strings.ToLower(strings.TrimSpace(profileCode))
	profiles, err := s.ListDocumentProfiles(ctx)
	if err != nil {
		return nil, err
	}

	allowedProfiles := make(map[string]struct{}, len(profiles))
	if normalizedProfileCode != "" {
		found := false
		for _, item := range profiles {
			if strings.EqualFold(item.Code, normalizedProfileCode) {
				allowedProfiles[strings.ToLower(strings.TrimSpace(item.Code))] = struct{}{}
				found = true
				break
			}
		}
		if !found {
			return nil, domain.ErrInvalidCommand
		}
	} else {
		for _, profile := range profiles {
			allowedProfiles[strings.ToLower(strings.TrimSpace(profile.Code))] = struct{}{}
		}
	}

	items, err := s.repo.ListDocumentTemplateVersions(ctx, normalizedProfileCode)
	if err != nil {
		return nil, err
	}

	filtered := make([]domain.DocumentTemplateVersion, 0, len(items))
	for _, item := range items {
		if _, ok := allowedProfiles[strings.ToLower(strings.TrimSpace(item.ProfileCode))]; !ok {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered, nil
}

func (s *Service) AssignDocumentTemplateAuthorized(ctx context.Context, item domain.DocumentTemplateAssignment) (domain.DocumentTemplateAssignment, error) {
	normalizedDocumentID := strings.TrimSpace(item.DocumentID)
	normalizedTemplateKey := strings.TrimSpace(item.TemplateKey)
	if normalizedDocumentID == "" || normalizedTemplateKey == "" || item.TemplateVersion <= 0 {
		return domain.DocumentTemplateAssignment{}, domain.ErrInvalidCommand
	}

	doc, err := s.repo.GetDocument(ctx, normalizedDocumentID)
	if err != nil {
		return domain.DocumentTemplateAssignment{}, err
	}
	allowed, err := s.isAllowed(ctx, doc, domain.CapabilityDocumentEdit)
	if err != nil {
		return domain.DocumentTemplateAssignment{}, err
	}
	if !allowed {
		return domain.DocumentTemplateAssignment{}, domain.ErrDocumentNotFound
	}

	templateVersion, err := s.repo.GetDocumentTemplateVersion(ctx, normalizedTemplateKey, item.TemplateVersion)
	if err != nil {
		return domain.DocumentTemplateAssignment{}, err
	}
	if !strings.EqualFold(templateVersion.ProfileCode, doc.DocumentProfile) {
		return domain.DocumentTemplateAssignment{}, domain.ErrInvalidCommand
	}
	if err := s.validateDocumentTemplateCompatibility(ctx, templateVersion); err != nil {
		return domain.DocumentTemplateAssignment{}, err
	}

	assignment := domain.DocumentTemplateAssignment{
		DocumentID:      normalizedDocumentID,
		TemplateKey:     normalizedTemplateKey,
		TemplateVersion: item.TemplateVersion,
		AssignedAt:      item.AssignedAt,
	}
	if assignment.AssignedAt.IsZero() {
		assignment.AssignedAt = s.clock.Now()
	}
	if err := s.repo.UpsertDocumentTemplateAssignment(ctx, assignment); err != nil {
		return domain.DocumentTemplateAssignment{}, err
	}
	return assignment, nil
}

func (s *Service) resolveDocumentTemplateOptional(ctx context.Context, documentID, profileCode string) (domain.DocumentTemplateVersion, bool, error) {
	templateVersion, err := s.ResolveDocumentTemplate(ctx, documentID, profileCode)
	if err == nil {
		return templateVersion, true, nil
	}
	if errors.Is(err, domain.ErrDocumentTemplateNotFound) {
		return domain.DocumentTemplateVersion{}, false, nil
	}
	return domain.DocumentTemplateVersion{}, false, err
}

func (s *Service) validateDocumentTemplateCompatibility(ctx context.Context, templateVersion domain.DocumentTemplateVersion) error {
	profileCode := strings.TrimSpace(templateVersion.ProfileCode)
	if profileCode == "" {
		return domain.ErrInvalidCommand
	}

	schema, ok, err := s.resolveDocumentProfileSchemaExact(ctx, profileCode, templateVersion.SchemaVersion)
	if err != nil {
		return err
	}
	if !ok {
		return domain.ErrInvalidCommand
	}
	if templateVersion.IsBrowserHTML() {
		return nil
	}

	if err := validateTemplateCompatibility(schema, templateVersion); err != nil {
		return err
	}
	return nil
}

func (s *Service) resolveDocumentProfileSchemaExact(ctx context.Context, profileCode string, version int) (domain.DocumentProfileSchemaVersion, bool, error) {
	if version <= 0 {
		return domain.DocumentProfileSchemaVersion{}, false, domain.ErrInvalidCommand
	}

	items, err := s.ListDocumentProfileSchemas(ctx, profileCode)
	if err != nil {
		return domain.DocumentProfileSchemaVersion{}, false, err
	}
	for _, item := range items {
		if item.Version == version {
			return item, true, nil
		}
	}
	return domain.DocumentProfileSchemaVersion{}, false, nil
}

func validateTemplateCompatibility(schema domain.DocumentProfileSchemaVersion, templateVersion domain.DocumentTemplateVersion) error {
	if len(schema.ContentSchema) == 0 || len(templateVersion.Definition) == 0 {
		return domain.ErrInvalidCommand
	}

	schemaNodes, err := decodeDocumentTypeSchema(schema.ContentSchema)
	if err != nil {
		return err
	}
	schemaPaths := collectTemplateCompatibleSchemaPaths(schemaNodes)
	if len(schemaPaths) == 0 {
		return domain.ErrInvalidCommand
	}

	seen := map[string]string{}
	return walkTemplateNodes(templateVersion.Definition, func(node map[string]any) error {
		typ, _ := node["type"].(string)
		if !isEditableTemplateNodeType(typ) {
			return nil
		}

		path, ok := node["path"].(string)
		if !ok || strings.TrimSpace(path) == "" {
			return domain.ErrInvalidCommand
		}
		fieldKind, ok := node["fieldKind"].(string)
		if !ok || strings.TrimSpace(fieldKind) == "" {
			return domain.ErrInvalidCommand
		}

		normalizedPath := normalizeTemplatePath(path)
		if previousKind, exists := seen[normalizedPath]; exists {
			if !strings.EqualFold(previousKind, fieldKind) {
				return domain.ErrInvalidCommand
			}
			return domain.ErrInvalidCommand
		}
		seen[normalizedPath] = strings.ToLower(strings.TrimSpace(fieldKind))

		resolvedKind, ok := schemaPaths[normalizedPath]
		if !ok {
			return domain.ErrInvalidCommand
		}
		if !templateFieldKindCompatible(resolvedKind, fieldKind) {
			return domain.ErrInvalidCommand
		}
		return nil
	})
}

func decodeDocumentTypeSchema(content map[string]any) (domain.DocumentTypeSchema, error) {
	raw, err := json.Marshal(content)
	if err != nil {
		return domain.DocumentTypeSchema{}, domain.ErrInvalidCommand
	}

	var schema domain.DocumentTypeSchema
	if err := json.Unmarshal(raw, &schema); err != nil {
		return domain.DocumentTypeSchema{}, domain.ErrInvalidCommand
	}
	return schema, nil
}

func collectTemplateCompatibleSchemaPaths(schema domain.DocumentTypeSchema) map[string]string {
	out := make(map[string]string)
	for _, section := range schema.Sections {
		sectionKey := strings.TrimSpace(section.Key)
		if sectionKey == "" {
			continue
		}
		for _, field := range section.Fields {
			collectTemplateCompatibleFieldPaths(out, sectionKey, "", field)
		}
	}
	return out
}

func collectTemplateCompatibleFieldPaths(out map[string]string, sectionKey, prefix string, field domain.FieldDef) {
	fieldKey := strings.TrimSpace(field.Key)
	if fieldKey == "" {
		return
	}

	base := fieldKey
	if prefix != "" {
		base = prefix + "." + fieldKey
	} else {
		base = sectionKey + "." + fieldKey
	}

	kind := schemaFieldTemplateKind(field.Type)
	if kind == "" {
		return
	}

	out[base] = kind
	if kind == "repeat" || kind == "table" {
		out[base+"[]"] = kind
	}

	switch kind {
	case "repeat":
		for _, itemField := range field.ItemFields {
			collectTemplateCompatibleFieldPaths(out, sectionKey, base, itemField)
			collectTemplateCompatibleFieldPaths(out, sectionKey, base+"[]", itemField)
		}
	case "table":
		for _, column := range field.Columns {
			collectTemplateCompatibleFieldPaths(out, sectionKey, base, column)
			collectTemplateCompatibleFieldPaths(out, sectionKey, base+"[]", column)
		}
	}
}

func schemaFieldTemplateKind(fieldType string) string {
	switch strings.ToLower(strings.TrimSpace(fieldType)) {
	case "rich":
		return "rich"
	case "repeat":
		return "repeat"
	case "table":
		return "table"
	case "text", "textarea", "number", "date", "select", "checkbox":
		return "scalar"
	default:
		return ""
	}
}

func templateFieldKindCompatible(schemaKind, templateKind string) bool {
	schemaKind = strings.ToLower(strings.TrimSpace(schemaKind))
	templateKind = strings.ToLower(strings.TrimSpace(templateKind))
	if schemaKind == "" || templateKind == "" {
		return false
	}
	return schemaKind == templateKind || (schemaKind == "scalar" && templateKind == "metadata")
}

func normalizeTemplatePath(path string) string {
	parts := strings.Split(strings.TrimSpace(path), ".")
	normalized := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		normalized = append(normalized, strings.TrimSuffix(part, "[]"))
	}
	return strings.Join(normalized, ".")
}

func walkTemplateNodes(node any, visit func(map[string]any) error) error {
	switch typed := node.(type) {
	case map[string]any:
		if err := visit(typed); err != nil {
			return err
		}
		for key, value := range typed {
			switch key {
			case "children", "columns":
				if err := walkTemplateNodes(value, visit); err != nil {
					return err
				}
			}
		}
	case []any:
		for _, item := range typed {
			if err := walkTemplateNodes(item, visit); err != nil {
				return err
			}
		}
	}
	return nil
}

func isEditableTemplateNodeType(nodeType string) bool {
	switch strings.ToLower(strings.TrimSpace(nodeType)) {
	case "field-slot", "rich-slot", "repeat-slot", "table-slot", "image-slot", "metadata-cell":
		return true
	default:
		return false
	}
}

func draftTokenForVersion(version domain.Version) string {
	hash := strings.TrimSpace(version.ContentHash)
	if hash == "" {
		hash = contentHash(version.Content)
	}
	return fmt.Sprintf("v%d:%s", version.Number, hash)
}
