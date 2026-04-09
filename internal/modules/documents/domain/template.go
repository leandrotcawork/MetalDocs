package domain

import (
	"strings"
	"time"
)

type TemplateExportConfig struct {
	MarginTop    float64 `json:"marginTop"`
	MarginRight  float64 `json:"marginRight"`
	MarginBottom float64 `json:"marginBottom"`
	MarginLeft   float64 `json:"marginLeft"`
}

type DocumentTemplateVersion struct {
	TemplateKey   string
	Version       int
	ProfileCode   string
	SchemaVersion int
	Name          string
	Editor        string
	ContentFormat string
	Body          string
	Definition    map[string]any
	CreatedAt     time.Time
	ExportConfig  *TemplateExportConfig
}

type DocumentTemplateAssignment struct {
	DocumentID      string
	TemplateKey     string
	TemplateVersion int
	AssignedAt      time.Time
}

type DocumentTemplateSnapshot struct {
	TemplateKey   string
	Version       int
	ProfileCode   string
	SchemaVersion int
	Editor        string
	ContentFormat string
	Body          string
	Definition    map[string]any
	ExportConfig  *TemplateExportConfig
}

func (v DocumentTemplateVersion) IsBrowserHTML() bool {
	return strings.EqualFold(v.Editor, "ckeditor5") && strings.EqualFold(v.ContentFormat, "html")
}

func (v DocumentTemplateVersion) IsMDDMEditor() bool {
	return strings.EqualFold(v.Editor, "mddm-blocknote") && strings.EqualFold(v.ContentFormat, "mddm")
}

func (v DocumentTemplateVersion) IsBrowserEditor() bool {
	return v.IsBrowserHTML() || v.IsMDDMEditor()
}

func (s DocumentTemplateSnapshot) IsBrowserHTML() bool {
	return strings.EqualFold(s.Editor, "ckeditor5") && strings.EqualFold(s.ContentFormat, "html")
}

func DefaultDocumentTemplateVersions() []DocumentTemplateVersion {
	return []DocumentTemplateVersion{
		{
			TemplateKey:   "po-mddm-canvas",
			Version:       1,
			ProfileCode:   "po",
			SchemaVersion: 3,
			Name:          "PO MDDM Canvas v1",
			Editor:        "mddm-blocknote",
			ContentFormat: "mddm",
			Body:          "",
			Definition:    map[string]any{"type": "page", "id": "po-mddm-root", "children": []any{}},
			CreatedAt:     time.Unix(0, 0).UTC(),
		},
	}
}
