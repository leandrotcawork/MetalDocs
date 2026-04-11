package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"metaldocs/internal/modules/documents/domain"
)

// RendererPinRepo is the minimal repository surface the capturer needs.
// It is satisfied by the postgres Repository via SetVersionRendererPin.
type RendererPinRepo interface {
	SetVersionRendererPin(ctx context.Context, documentID string, versionNumber int, pin *domain.RendererPin) error
}

type RendererPinCapturerConfig struct {
	CurrentRendererVersion string
	CurrentLayoutIRHash    string
	Repo                   RendererPinRepo
	Clock                  func() time.Time
}

// RendererPinCapturer writes a RendererPin when a version transitions from
// DRAFT to RELEASED. It's a tiny domain service, not a generic hook, so the
// transition site can call OnRelease explicitly with the version record.
type RendererPinCapturer struct {
	cfg RendererPinCapturerConfig
}

func NewRendererPinCapturer(cfg RendererPinCapturerConfig) *RendererPinCapturer {
	if cfg.Clock == nil {
		cfg.Clock = time.Now
	}
	return &RendererPinCapturer{cfg: cfg}
}

// OnRelease captures a pin for the given version if the version's content
// source uses the MDDM engine. Non-MDDM content sources (native, docx_upload)
// are skipped because they don't go through the MDDM renderer.
func (c *RendererPinCapturer) OnRelease(ctx context.Context, version domain.Version) error {
	if version.ContentSource != domain.ContentSourceBrowserEditor {
		return nil
	}
	if strings.TrimSpace(version.TemplateKey) == "" || version.TemplateVersion <= 0 {
		return fmt.Errorf("browser editor version %s/%d missing template ref", version.DocumentID, version.Number)
	}

	pin := &domain.RendererPin{
		RendererVersion: c.cfg.CurrentRendererVersion,
		LayoutIRHash:    c.cfg.CurrentLayoutIRHash,
		TemplateKey:     version.TemplateKey,
		TemplateVersion: version.TemplateVersion,
		PinnedAt:        c.cfg.Clock().UTC(),
	}
	if err := pin.Validate(); err != nil {
		return fmt.Errorf("build renderer pin: %w", err)
	}

	return c.cfg.Repo.SetVersionRendererPin(ctx, version.DocumentID, version.Number, pin)
}

// Rollback clears a previously captured pin. It is called as compensation when
// the release transition fails after OnRelease already wrote the pin, so draft
// rows never retain a stale pin. Mirrors OnRelease's content-source guard.
func (c *RendererPinCapturer) Rollback(ctx context.Context, version domain.Version) error {
	if version.ContentSource != domain.ContentSourceBrowserEditor {
		return nil
	}
	return c.cfg.Repo.SetVersionRendererPin(ctx, version.DocumentID, version.Number, nil)
}
