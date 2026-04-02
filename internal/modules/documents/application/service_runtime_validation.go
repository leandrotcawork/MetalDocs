package application

import (
	"encoding/json"
	"strings"
	"time"

	"metaldocs/internal/modules/documents/domain"
)

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

func validateDocumentTypeDefinitionSchema(schema domain.DocumentTypeSchema) error {
	for _, section := range schema.Sections {
		if strings.TrimSpace(section.Key) == "" || strings.TrimSpace(section.Num) == "" || strings.TrimSpace(section.Title) == "" {
			return domain.ErrDocumentSchemaInvalidSection
		}
		if len(section.Fields) == 0 {
			continue
		}
		for _, field := range section.Fields {
			if err := validateDocumentTypeFieldDefinition(field); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateDocumentTypeFieldDefinition(field domain.FieldDef) error {
	if strings.TrimSpace(field.Key) == "" || strings.TrimSpace(field.Label) == "" {
		return domain.ErrDocumentSchemaInvalidField
	}
	fieldType := strings.ToLower(strings.TrimSpace(field.Type))
	if fieldType == "" {
		return domain.ErrDocumentSchemaInvalidField
	}
	switch fieldType {
	case "text", "textarea", "number", "date", "select", "checkbox", "table", "rich", "repeat", "array", "rich_blocks":
		if fieldType == "table" {
			if len(field.Columns) == 0 {
				return domain.ErrDocumentSchemaInvalidField
			}
			for _, column := range field.Columns {
				if err := validateDocumentTypeFieldDefinition(column); err != nil {
					return err
				}
			}
		}
		if fieldType == "repeat" {
			if len(field.ItemFields) == 0 {
				return domain.ErrDocumentSchemaInvalidField
			}
			for _, itemField := range field.ItemFields {
				if err := validateDocumentTypeFieldDefinition(itemField); err != nil {
					return err
				}
			}
		}
		return nil
	default:
		return domain.ErrDocumentSchemaInvalidField
	}
}

func validateDocumentTypeValues(schema domain.DocumentTypeSchema, values map[string]any) error {
	if len(schema.Sections) == 0 || len(values) == 0 {
		return nil
	}
	hasFields := false
	for _, section := range schema.Sections {
		if len(section.Fields) > 0 {
			hasFields = true
			break
		}
	}
	if !hasFields {
		return nil
	}

	for _, section := range schema.Sections {
		if len(section.Fields) == 0 {
			continue
		}
		container := values
		allowRootFallback := false
		if rawSection, ok := values[section.Key]; ok {
			if nested, ok := rawSection.(map[string]any); ok {
				container = nested
				allowRootFallback = true
			}
		}
		for _, field := range section.Fields {
			if err := validateDocumentTypeValueField(field, values, container, allowRootFallback); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateDocumentTypeValueField(field domain.FieldDef, root, container map[string]any, allowRootFallback bool) error {
	key := strings.TrimSpace(field.Key)
	if key == "" {
		return nil
	}

	value, exists := container[key]
	if !exists && allowRootFallback {
		value, exists = root[key]
	}
	if !exists || isEmptyRuntimeValue(value) {
		return nil
	}

	switch strings.ToLower(strings.TrimSpace(field.Type)) {
	case "text", "textarea":
		if _, ok := value.(string); !ok {
			return domain.ErrInvalidNativeContent
		}
	case "number":
		if !isRuntimeNumericValue(value) {
			return domain.ErrInvalidNativeContent
		}
	case "date":
		if !isRuntimeDateValue(value) {
			return domain.ErrInvalidNativeContent
		}
	case "select":
		if _, ok := value.(string); !ok {
			return domain.ErrInvalidNativeContent
		}
	case "checkbox":
		if _, ok := value.(bool); !ok {
			return domain.ErrInvalidNativeContent
		}
	case "table":
		rows, ok := toRuntimeSlice(value)
		if !ok {
			return domain.ErrInvalidNativeContent
		}
		for _, row := range rows {
			rowMap, ok := row.(map[string]any)
			if !ok {
				return domain.ErrInvalidNativeContent
			}
			for _, column := range field.Columns {
				if err := validateDocumentTypeValueField(column, rowMap, rowMap, false); err != nil {
					return err
				}
			}
		}
	case "repeat":
		items, ok := toRuntimeSlice(value)
		if !ok {
			return domain.ErrInvalidNativeContent
		}
		for _, item := range items {
			itemMap, ok := item.(map[string]any)
			if !ok {
				return domain.ErrInvalidNativeContent
			}
			for _, itemField := range field.ItemFields {
				if err := validateDocumentTypeValueField(itemField, itemMap, itemMap, false); err != nil {
					return err
				}
			}
		}
	case "rich":
		if err := validateRichEnvelopeValue(value); err != nil {
			return domain.ErrInvalidNativeContent
		}
		return nil
	case "array":
		if _, ok := toRuntimeSlice(value); !ok {
			return domain.ErrInvalidNativeContent
		}
	case "rich_blocks":
		if _, ok := toRuntimeSlice(value); ok {
			return nil
		}
		if _, ok := value.([]json.RawMessage); ok {
			return nil
		}
		return domain.ErrInvalidNativeContent
	default:
		return domain.ErrInvalidNativeContent
	}

	return nil
}

func isRuntimeNumericValue(value any) bool {
	switch value.(type) {
	case float64, float32, int, int32, int64, uint, uint32, uint64, json.Number:
		return true
	default:
		return false
	}
}

func isRuntimeDateValue(value any) bool {
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
}

func isEmptyRuntimeValue(value any) bool {
	switch typed := value.(type) {
	case nil:
		return true
	case string:
		return strings.TrimSpace(typed) == ""
	case []any:
		return len(typed) == 0
	case map[string]any:
		return len(typed) == 0
	default:
		return false
	}
}

func toRuntimeSlice(value any) ([]any, bool) {
	switch typed := value.(type) {
	case []any:
		return typed, true
	case []string:
		out := make([]any, len(typed))
		for i, item := range typed {
			out[i] = item
		}
		return out, true
	case []map[string]any:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, item)
		}
		return out, true
	case []json.RawMessage:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, item)
		}
		return out, true
	default:
		return nil, false
	}
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
	case json.RawMessage:
		return append(json.RawMessage(nil), typed...)
	default:
		return typed
	}
}
