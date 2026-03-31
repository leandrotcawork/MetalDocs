package domain

import "strings"

type Block struct {
	Type     string   `json:"type"`
	Content  []Inline `json:"content,omitempty"`
	Base64   string   `json:"base64,omitempty"`
	MimeType string   `json:"mimeType,omitempty"`
	Width    int      `json:"width,omitempty"`
	Caption  string   `json:"caption,omitempty"`
}

type Inline struct {
	Type     string `json:"type"`
	Text     string `json:"text"`
	Bold     bool   `json:"bold,omitempty"`
	Italic   bool   `json:"italic,omitempty"`
	FontSize int    `json:"fontSize,omitempty"`
	Color    string `json:"color,omitempty"`
}

func ValidateBlocks(blocks []Block) error {
	for _, block := range blocks {
		switch strings.TrimSpace(block.Type) {
		case "paragraph":
			if len(block.Content) == 0 {
				return ErrInvalidNativeContent
			}
			for _, inline := range block.Content {
				if strings.TrimSpace(inline.Type) != "text" || strings.TrimSpace(inline.Text) == "" {
					return ErrInvalidNativeContent
				}
			}
		case "image":
			if strings.TrimSpace(block.Base64) == "" || strings.TrimSpace(block.MimeType) == "" || block.Width <= 0 {
				return ErrInvalidNativeContent
			}
		default:
			return ErrInvalidNativeContent
		}
	}
	return nil
}
