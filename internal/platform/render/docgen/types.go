package docgen

type RenderPayload struct {
	Document RenderDocument `json:"document"`
	Schema   RenderSchema   `json:"schema"`
	Values   map[string]any `json:"values"`
}

type RenderDocument struct {
	DocumentID string `json:"documentId"`
	Title      string `json:"title"`
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
