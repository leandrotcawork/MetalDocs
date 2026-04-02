package domain

import "time"

type DocumentTemplateVersion struct {
	TemplateKey   string
	Version       int
	ProfileCode   string
	SchemaVersion int
	Name          string
	Definition    map[string]any
	CreatedAt     time.Time
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
	Definition    map[string]any
}

func DefaultDocumentTemplateVersions() []DocumentTemplateVersion {
	return []DocumentTemplateVersion{
		{
			TemplateKey:   "po-default-canvas",
			Version:       1,
			ProfileCode:   "po",
			SchemaVersion: 3,
			Name:          "PO Governed Canvas v1",
			Definition: map[string]any{
				"type": "page",
				"id":   "po-root",
				"children": []any{
					map[string]any{
						"type":  "section-frame",
						"id":    "identificacao-processo",
						"title": "Identificacao do Processo",
						"children": []any{
							map[string]any{"type": "label", "id": "lbl-objetivo", "text": "Objetivo"},
							map[string]any{"type": "field-slot", "id": "slot-objetivo", "path": "identificacaoProcesso.objetivo", "fieldKind": "scalar"},
							map[string]any{"type": "label", "id": "lbl-descricao", "text": "Descricao do processo"},
							map[string]any{"type": "rich-slot", "id": "slot-descricao", "path": "visaoGeral.descricaoProcesso", "fieldKind": "rich"},
						},
					},
				},
			},
			CreatedAt: time.Unix(0, 0).UTC(),
		},
	}
}
