package docgen

import "encoding/json"

type RenderPayload struct {
	DocumentType string           `json:"documentType"`
	DocumentCode string           `json:"documentCode"`
	Title        string           `json:"title"`
	Version      string           `json:"version,omitempty"`
	Status       string           `json:"status,omitempty"`
	Schema       RenderSchema     `json:"schema"`
	Values       map[string]any   `json:"values"`
	Metadata     *RenderMetadata  `json:"metadata,omitempty"`
	Revisions    []RenderRevision `json:"revisions,omitempty"`
}

type RenderMetadata struct {
	ElaboradoPor string `json:"elaboradoPor"`
	AprovadoPor  string `json:"aprovadoPor"`
	CreatedAt    string `json:"createdAt"`
	ApprovedAt   string `json:"approvedAt"`
}

type RenderRevision struct {
	Versao    string `json:"versao"`
	Data      string `json:"data"`
	Descricao string `json:"descricao"`
	Por       string `json:"por"`
}

type RenderSchema struct {
	Sections []RenderSection `json:"sections"`
}

type RenderSection struct {
	Key    string        `json:"key"`
	Num    string        `json:"num"`
	Title  string        `json:"title"`
	Color  string        `json:"color,omitempty"`
	Fields []RenderField `json:"fields"`
}

type RenderField struct {
	Key        string        `json:"key"`
	Label      string        `json:"label"`
	Type       string        `json:"type"`
	Options    []string      `json:"options,omitempty"`
	Columns    []RenderField `json:"columns,omitempty"`
	ItemFields []RenderField `json:"itemFields,omitempty"`
}

type MDDMTemplateTheme struct {
	Accent       string `json:"accent,omitempty"`
	AccentLight  string `json:"accentLight,omitempty"`
	AccentDark   string `json:"accentDark,omitempty"`
	AccentBorder string `json:"accentBorder,omitempty"`
}

type MDDMExportMetadata struct {
	DocumentCode  string `json:"document_code"`
	Title         string `json:"title"`
	RevisionLabel string `json:"revision_label"`
	Mode          string `json:"mode"`
}

type MDDMExportPayload struct {
	Envelope      json.RawMessage    `json:"envelope"`
	Metadata      MDDMExportMetadata `json:"metadata"`
	TemplateTheme *MDDMTemplateTheme `json:"templateTheme,omitempty"`
}

type BrowserRenderMargins struct {
	Top    float64 `json:"top"`
	Right  float64 `json:"right"`
	Bottom float64 `json:"bottom"`
	Left   float64 `json:"left"`
}

type BrowserRenderPayload struct {
	DocumentCode string                `json:"documentCode"`
	Title        string                `json:"title"`
	Version      string                `json:"version,omitempty"`
	HTML         string                `json:"html"`
	Margins      *BrowserRenderMargins `json:"margins,omitempty"`
}
