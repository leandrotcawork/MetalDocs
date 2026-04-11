package domain

import "time"

// ShadowDiffEvent is a single telemetry row captured by the frontend during
// Phase 1 shadow testing. It is append-only; engineers aggregate over the
// table off-line to decide when Phase 2 (canary) is safe.
type ShadowDiffEvent struct {
	DocumentID        string
	VersionNumber     int
	UserIDHash        string
	CurrentXMLHash    string
	ShadowXMLHash     string
	DiffSummary       map[string]any
	CurrentDurationMs int
	ShadowDurationMs  int
	ShadowError       string
	RecordedAt        time.Time
	TraceID           string
}
