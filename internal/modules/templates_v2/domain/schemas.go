package domain

type MetadataSchema struct {
	DocCodePattern      string   `json:"doc_code_pattern"`
	RetentionDays       int      `json:"retention_days"`
	DistributionDefault []string `json:"distribution_default"`
	RequiredMetadata    []string `json:"required_metadata"`
}

type PlaceholderType string

const (
	PHText   PlaceholderType = "text"
	PHDate   PlaceholderType = "date"
	PHNumber PlaceholderType = "number"
	PHSelect PlaceholderType = "select"
	PHUser   PlaceholderType = "user"
)

type Placeholder struct {
	ID       string          `json:"id"`
	Label    string          `json:"label"`
	Type     PlaceholderType `json:"type"`
	Required bool            `json:"required"`
	Default  any             `json:"default,omitempty"`
	Options  []string        `json:"options,omitempty"`
}

type EditableZone struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Required bool   `json:"required"`
}
