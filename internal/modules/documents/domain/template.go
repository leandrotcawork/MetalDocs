package domain

import (
	"strings"
	"time"
)

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
}

func (v DocumentTemplateVersion) IsBrowserHTML() bool {
	return strings.EqualFold(v.Editor, "ckeditor5") && strings.EqualFold(v.ContentFormat, "html")
}

func (s DocumentTemplateSnapshot) IsBrowserHTML() bool {
	return strings.EqualFold(s.Editor, "ckeditor5") && strings.EqualFold(s.ContentFormat, "html")
}

func DefaultDocumentTemplateVersions() []DocumentTemplateVersion {
	return []DocumentTemplateVersion{
		{
			TemplateKey:   "po-default-canvas",
			Version:       1,
			ProfileCode:   "po",
			SchemaVersion: 3,
			Name:          "PO Governed Canvas v1",
			Editor:        "ckeditor5",
			ContentFormat: "html",
			Body: `<section class="md-doc-shell">
  <h1>Procedimento Operacional</h1>
  <p><strong>Objetivo</strong></p>
  <p><span class="restricted-editing-exception">Preencha o objetivo.</span></p>
  <p><strong>Descricao do processo</strong></p>
  <div class="restricted-editing-exception"><p>Descreva o processo.</p></div>
</section>`,
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
