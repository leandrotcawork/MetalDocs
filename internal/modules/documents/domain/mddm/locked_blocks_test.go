package mddm

import (
	"errors"
	"testing"
)

func assertLockViolationCode(t *testing.T, err error, wantCode string) *LockViolationError {
	t.Helper()

	if err == nil {
		t.Fatalf("expected %s error, got nil", wantCode)
	}

	var lockErr *LockViolationError
	if !errors.As(err, &lockErr) {
		t.Fatalf("expected *LockViolationError, got %T (%v)", err, err)
	}

	if lockErr.Code != wantCode {
		t.Fatalf("expected lock violation code %s, got %s (%v)", wantCode, lockErr.Code, err)
	}

	return lockErr
}

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
	assertLockViolationCode(t, err, "LOCKED_BLOCK_PROP_MUTATED")
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
	assertLockViolationCode(t, err, "LOCKED_BLOCK_DELETED")
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
		assertLockViolationCode(t, err, "LOCKED_BLOCK_DELETED")
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

func TestLockedBlocks_RejectsDeletedTemplatedDescendantInOptionalSection(t *testing.T) {
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

	doc := map[string]any{
		"id":                "doc-root",
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
				"id":                "doc-group",
				"template_block_id": "tpl-group",
				"type":              "fieldGroup",
				"props": map[string]any{
					"locked": true,
				},
				"children": []any{},
			},
		},
	}

	err := EnforceLockedBlocks([]any{template}, []any{doc})
	assertLockViolationCode(t, err, "LOCKED_BLOCK_DELETED")
}

func TestLockedBlocks_RejectsReparentedTemplatedDescendantAfterOptionalSectionDeletion(t *testing.T) {
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

	doc := map[string]any{
		"id":                "doc-host",
		"template_block_id": "tpl-host",
		"type":              "section",
		"props": map[string]any{
			"title":  "Host",
			"color":  "#0f172a",
			"locked": true,
		},
		"children": []any{
			map[string]any{
				"id":                "doc-field",
				"template_block_id": "tpl-field",
				"type":              "field",
				"props": map[string]any{
					"locked": true,
				},
				"children": []any{},
			},
		},
	}

	err := EnforceLockedBlocks([]any{template}, []any{doc})
	assertLockViolationCode(t, err, "LOCKED_BLOCK_REPARENTED")
}
