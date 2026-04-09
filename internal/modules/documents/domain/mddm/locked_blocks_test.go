package mddm

import (
	"strings"
	"testing"
)

func TestLockedBlocks_AcceptsUnchangedTemplate(t *testing.T) {
	template := map[string]any{
		"id":                "tpl-aaa",
		"template_block_id": "tpl-aaa",
		"type":              "section",
		"props": map[string]any{
			"title":  "Identification",
			"color":  "#6b1f2a",
			"locked": true,
		},
		"children": []any{},
	}

	doc := map[string]any{
		"id":                "doc-aaa",
		"template_block_id": "tpl-aaa",
		"type":              "section",
		"props": map[string]any{
			"title":  "Identification",
			"color":  "#6b1f2a",
			"locked": true,
		},
		"children": []any{},
	}

	err := EnforceLockedBlocks([]any{template}, []any{doc})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestLockedBlocks_RejectsLockedPropChange(t *testing.T) {
	template := map[string]any{
		"id":                "tpl-aaa",
		"template_block_id": "tpl-aaa",
		"type":              "section",
		"props": map[string]any{
			"title":  "Identification",
			"color":  "#6b1f2a",
			"locked": true,
		},
		"children": []any{},
	}

	doc := map[string]any{
		"id":                "doc-aaa",
		"template_block_id": "tpl-aaa",
		"type":              "section",
		"props": map[string]any{
			"title":  "Modified Title",
			"color":  "#6b1f2a",
			"locked": true,
		},
		"children": []any{},
	}

	err := EnforceLockedBlocks([]any{template}, []any{doc})
	if err == nil {
		t.Error("expected lock violation error")
	}
}

func TestLockedBlocks_RejectsDeletedTemplatedBlock(t *testing.T) {
	template := map[string]any{
		"id":                "tpl-aaa",
		"template_block_id": "tpl-aaa",
		"type":              "section",
		"props":             map[string]any{"title": "X", "color": "#000000", "locked": true},
		"children":          []any{},
	}

	err := EnforceLockedBlocks([]any{template}, []any{})
	if err == nil {
		t.Error("expected LOCKED_BLOCK_DELETED error")
	}
}

func TestLockedBlocks_AllowsDeletingOptionalSection(t *testing.T) {
	t.Run("rejects deleting non-optional section", func(t *testing.T) {
		template := map[string]any{
			"id":                "tpl-aaa",
			"template_block_id": "tpl-aaa",
			"type":              "section",
			"props": map[string]any{
				"title":  "Indicadores",
				"color":  "#6b1f2a",
				"locked": true,
			},
			"children": []any{},
		}

		err := EnforceLockedBlocks([]any{template}, []any{})
		if err == nil {
			t.Fatal("expected LOCKED_BLOCK_DELETED error for non-optional section")
		}
		if !strings.Contains(err.Error(), "LOCKED_BLOCK_DELETED") {
			t.Fatalf("EnforceLockedBlocks() error = %v, want LOCKED_BLOCK_DELETED", err)
		}
	})

	t.Run("allows deleting optional section", func(t *testing.T) {
		template := map[string]any{
			"id":                "tpl-aaa",
			"template_block_id": "tpl-aaa",
			"type":              "section",
			"props": map[string]any{
				"title":    "Indicadores",
				"color":    "#6b1f2a",
				"locked":   true,
				"optional": true,
			},
			"children": []any{},
		}

		err := EnforceLockedBlocks([]any{template}, []any{})
		if err != nil {
			t.Fatalf("EnforceLockedBlocks() error = %v, want nil for optional section deletion", err)
		}
	})
}

func TestLockedBlocks_AllowsDeletingOptionalSectionWithNestedChildren(t *testing.T) {
	template := map[string]any{
		"id":                "tpl-root",
		"template_block_id": "tpl-root",
		"type":              "section",
		"props": map[string]any{
			"title":    "Optional cover",
			"color":    "#6b1f2a",
			"locked":   true,
			"optional": true,
		},
		"children": []any{
			map[string]any{
				"id":                "tpl-group",
				"template_block_id": "tpl-group",
				"type":              "fieldGroup",
				"props": map[string]any{
					"locked": true,
				},
				"children": []any{
					map[string]any{
						"id":                "tpl-field",
						"template_block_id": "tpl-field",
						"type":              "field",
						"props": map[string]any{
							"locked": true,
						},
						"children": []any{},
					},
				},
			},
		},
	}

	err := EnforceLockedBlocks([]any{template}, []any{})
	if err != nil {
		t.Fatalf("EnforceLockedBlocks() error = %v, want nil for optional subtree deletion", err)
	}
}
