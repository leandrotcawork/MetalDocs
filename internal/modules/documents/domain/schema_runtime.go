package domain

import (
	"encoding/json"
	"strings"
)

type DocumentValues = map[string]any
type RichBlock = json.RawMessage

var allowedFieldTypes = map[string]struct{}{
	"text":     {},
	"textarea": {},
	"number":   {},
	"date":     {},
	"select":   {},
	"checkbox": {},
	"table":    {},
	"rich":     {},
	"repeat":   {},
}

type DocumentTypeSchema struct {
	Sections []SectionDef `json:"sections"`
}

type SectionDef struct {
	Key    string     `json:"key"`
	Num    string     `json:"num"`
	Title  string     `json:"title"`
	Color  string     `json:"color,omitempty"`
	Fields []FieldDef `json:"fields"`
}

type FieldDef struct {
	Key        string     `json:"key"`
	Label      string     `json:"label"`
	Type       string     `json:"type"`
	Options    []string   `json:"options,omitempty"`
	Columns    []FieldDef `json:"columns,omitempty"`
	ItemFields []FieldDef `json:"itemFields,omitempty"`
}

func ValidateDocumentTypeSchema(schema DocumentTypeSchema) error {
	if len(schema.Sections) == 0 {
		return ErrDocumentSchemaInvalid
	}
	for _, section := range schema.Sections {
		if err := validateSectionDef(section); err != nil {
			return err
		}
	}
	return nil
}

func validateSectionDef(section SectionDef) error {
	if strings.TrimSpace(section.Key) == "" || strings.TrimSpace(section.Num) == "" || strings.TrimSpace(section.Title) == "" {
		return ErrDocumentSchemaInvalidSection
	}
	if len(section.Fields) == 0 {
		return ErrDocumentSchemaInvalidSection
	}
	for _, field := range section.Fields {
		if err := validateFieldDef(field); err != nil {
			return err
		}
	}
	return nil
}

func validateFieldDef(field FieldDef) error {
	if strings.TrimSpace(field.Key) == "" || strings.TrimSpace(field.Label) == "" {
		return ErrDocumentSchemaInvalidField
	}

	fieldType := strings.ToLower(strings.TrimSpace(field.Type))
	if fieldType == "" {
		return ErrDocumentSchemaInvalidField
	}
	if _, ok := allowedFieldTypes[fieldType]; !ok {
		return ErrDocumentSchemaInvalidField
	}

	switch fieldType {
	case "table":
		if len(field.Columns) == 0 {
			return ErrDocumentSchemaInvalidField
		}
		for _, column := range field.Columns {
			if err := validateFieldDef(column); err != nil {
				return err
			}
		}
	case "repeat":
		if len(field.ItemFields) == 0 {
			return ErrDocumentSchemaInvalidField
		}
		for _, itemField := range field.ItemFields {
			if err := validateFieldDef(itemField); err != nil {
				return err
			}
		}
	}

	return nil
}
