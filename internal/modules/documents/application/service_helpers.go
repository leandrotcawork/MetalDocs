package application

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"metaldocs/internal/modules/documents/domain"
)

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

func formatDocumentCode(profileCode string, sequence int) string {
	trimmed := strings.TrimSpace(profileCode)
	if trimmed == "" || sequence <= 0 {
		return ""
	}
	return fmt.Sprintf("%s-%03d", strings.ToUpper(trimmed), sequence)
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

func validateContentSchema(schema map[string]any, content map[string]any) error {
	if len(schema) == 0 {
		return nil
	}
	rawSections, ok := schema["sections"]
	if !ok {
		return nil
	}
	sections, ok := rawSections.([]any)
	if !ok {
		return nil
	}
	for _, rawSection := range sections {
		section, ok := rawSection.(map[string]any)
		if !ok {
			continue
		}
		sectionKey, _ := asSchemaString(section["key"])
		if sectionKey == "" {
			continue
		}
		sectionValue, _ := content[sectionKey].(map[string]any)
		if sectionValue == nil {
			sectionValue = map[string]any{}
		}
		fields, _ := section["fields"].([]any)
		for _, rawField := range fields {
			field, ok := rawField.(map[string]any)
			if !ok {
				continue
			}
			if err := validateContentField(field, sectionValue); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateContentField(field map[string]any, container map[string]any) error {
	key, _ := asSchemaString(field["key"])
	if key == "" {
		return nil
	}
	fieldType, _ := asSchemaString(field["type"])
	required, _ := field["required"].(bool)
	value, exists := container[key]
	if !exists || isEmptyContentValue(value) {
		if required {
			return domain.ErrInvalidNativeContent
		}
		return nil
	}

	switch fieldType {
	case "text", "textarea":
		if _, ok := value.(string); !ok {
			return domain.ErrInvalidNativeContent
		}
	case "number":
		if !isNumericValue(value) {
			return domain.ErrInvalidNativeContent
		}
	case "select":
		selected, ok := value.(string)
		if !ok {
			return domain.ErrInvalidNativeContent
		}
		options := normalizeSchemaStringList(field["options"])
		if len(options) > 0 && !containsSchemaOption(options, selected) {
			return domain.ErrInvalidNativeContent
		}
	case "array":
		items, ok := value.([]any)
		if !ok {
			return domain.ErrInvalidNativeContent
		}
		if required && len(items) == 0 {
			return domain.ErrInvalidNativeContent
		}
		itemType, _ := asSchemaString(field["itemType"])
		if itemType != "" {
			for _, item := range items {
				if isEmptyContentValue(item) {
					continue
				}
				if !matchesContentType(itemType, item, field) {
					return domain.ErrInvalidNativeContent
				}
			}
		}
	case "table":
		rows, ok := value.([]any)
		if !ok {
			return domain.ErrInvalidNativeContent
		}
		if required && len(rows) == 0 {
			return domain.ErrInvalidNativeContent
		}
		columns, _ := field["columns"].([]any)
		for _, rawRow := range rows {
			row, ok := rawRow.(map[string]any)
			if !ok {
				return domain.ErrInvalidNativeContent
			}
			for _, rawColumn := range columns {
				column, ok := rawColumn.(map[string]any)
				if !ok {
					continue
				}
				if err := validateContentField(column, row); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func matchesContentType(fieldType string, value any, field map[string]any) bool {
	switch fieldType {
	case "text", "textarea":
		_, ok := value.(string)
		return ok
	case "number":
		return isNumericValue(value)
	case "select":
		selected, ok := value.(string)
		if !ok {
			return false
		}
		options := normalizeSchemaStringList(field["options"])
		if len(options) == 0 {
			return true
		}
		return containsSchemaOption(options, selected)
	default:
		return true
	}
}

func isNumericValue(value any) bool {
	switch value.(type) {
	case float64, float32, int, int32, int64, uint, uint32, uint64, json.Number:
		return true
	default:
		return false
	}
}

func isEmptyContentValue(value any) bool {
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

func asSchemaString(value any) (string, bool) {
	typed, ok := value.(string)
	if !ok {
		return "", false
	}
	trimmed := strings.TrimSpace(typed)
	if trimmed == "" {
		return "", false
	}
	return trimmed, true
}

func normalizeSchemaStringList(value any) []string {
	switch typed := value.(type) {
	case []string:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if trimmed := strings.TrimSpace(item); trimmed != "" {
				out = append(out, trimmed)
			}
		}
		return out
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			str, ok := item.(string)
			if !ok {
				continue
			}
			if trimmed := strings.TrimSpace(str); trimmed != "" {
				out = append(out, trimmed)
			}
		}
		return out
	default:
		return nil
	}
}

func containsSchemaOption(options []string, value string) bool {
	for _, option := range options {
		if strings.EqualFold(option, value) {
			return true
		}
	}
	return false
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
