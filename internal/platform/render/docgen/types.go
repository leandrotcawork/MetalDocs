package docgen

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
