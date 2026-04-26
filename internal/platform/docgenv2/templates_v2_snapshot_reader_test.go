package docgenv2

import (
	"testing"

	"metaldocs/internal/modules/documents_v2/application"
)

func TestNewTemplatesV2SnapshotReader(t *testing.T) {
	var _ application.SnapshotTemplateReader = (*TemplatesV2SnapshotReader)(nil)

	r := NewTemplatesV2SnapshotReader(nil)
	if r == nil {
		t.Fatal("expected reader")
	}
}
