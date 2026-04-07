package mddm

import (
	"strings"
	"testing"
)

func TestIDContinuity_RejectsTemplateBlockIDRewrite(t *testing.T) {
	prev := []any{
		map[string]any{
			"id":                "doc-1",
			"template_block_id": "tpl-A",
			"type":              "section",
			"props":             map[string]any{"title": "T", "color": "#000000", "locked": true},
			"children":          []any{},
		},
	}
	curr := []any{
		map[string]any{
			"id":                "doc-NEW", // changed!
			"template_block_id": "tpl-A",
			"type":              "section",
			"props":             map[string]any{"title": "T", "color": "#000000", "locked": true},
			"children":          []any{},
		},
	}
	err := CheckBlockIDContinuity(prev, curr)
	if err == nil || !strings.Contains(err.Error(), "BLOCK_ID_REWRITE_FORBIDDEN") {
		t.Errorf("expected BLOCK_ID_REWRITE_FORBIDDEN, got %v", err)
	}
}

func TestIDContinuity_AcceptsUnchangedIDs(t *testing.T) {
	prev := []any{
		map[string]any{
			"id":                "doc-1",
			"template_block_id": "tpl-A",
			"type":              "section",
			"props":             map[string]any{"title": "T", "color": "#000000", "locked": true},
			"children":          []any{},
		},
	}
	curr := []any{
		map[string]any{
			"id":                "doc-1",
			"template_block_id": "tpl-A",
			"type":              "section",
			"props":             map[string]any{"title": "T2", "color": "#000000", "locked": true},
			"children":          []any{},
		},
	}
	if err := CheckBlockIDContinuity(prev, curr); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}
