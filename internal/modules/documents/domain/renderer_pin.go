package domain

import (
	"errors"
	"strings"
	"time"
)

// RendererPin freezes the inputs used to render a document version for
// DOCX and PDF export. When a version transitions from DRAFT to RELEASED,
// the application captures the current renderer version, Layout IR hash,
// and the specific template (key + version) that was active at release time.
// From that moment on, any export of the version MUST load the matching
// historical renderer bundle — see the frontend registry for the mechanism.
type RendererPin struct {
	RendererVersion string    `json:"renderer_version"`
	LayoutIRHash    string    `json:"layout_ir_hash"`
	TemplateKey     string    `json:"template_key"`
	TemplateVersion int       `json:"template_version"`
	PinnedAt        time.Time `json:"pinned_at"`
}

// IsComplete reports whether every required field is populated.
// Zero times (PinnedAt) are allowed — the application sets them on capture.
func (p RendererPin) IsComplete() bool {
	return strings.TrimSpace(p.RendererVersion) != "" &&
		strings.TrimSpace(p.LayoutIRHash) != "" &&
		strings.TrimSpace(p.TemplateKey) != "" &&
		p.TemplateVersion > 0
}

// Validate returns an error if any required field is missing or malformed.
func (p RendererPin) Validate() error {
	if strings.TrimSpace(p.RendererVersion) == "" {
		return errors.New("renderer pin: rendererVersion is required")
	}
	if strings.TrimSpace(p.LayoutIRHash) == "" {
		return errors.New("renderer pin: layoutIRHash is required")
	}
	if strings.TrimSpace(p.TemplateKey) == "" {
		return errors.New("renderer pin: templateKey is required")
	}
	if p.TemplateVersion <= 0 {
		return errors.New("renderer pin: templateVersion must be positive")
	}
	return nil
}
