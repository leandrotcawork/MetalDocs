package mddm

import "testing"

func TestInstantiate_AssignsNewIDs(t *testing.T) {
	template := []any{
		map[string]any{
			"id":                "tpl-A",
			"template_block_id": "tpl-A",
			"type":              "section",
			"props": map[string]any{
				"locked": true,
			},
			"children": []any{},
		},
	}

	instantiated := InstantiateTemplate(template)

	if len(instantiated) != 1 {
		t.Fatalf("expected 1 block, got %d", len(instantiated))
	}

	block, ok := instantiated[0].(map[string]any)
	if !ok {
		t.Fatalf("expected map block, got %T", instantiated[0])
	}

	if got := block["id"]; got == "tpl-A" {
		t.Fatalf("expected regenerated id, got %v", got)
	}

	if got := block["template_block_id"]; got != "tpl-A" {
		t.Fatalf("expected template_block_id tpl-A, got %v", got)
	}
}

func TestInstantiate_ContentSlotChildrenLoseTemplateBlockID(t *testing.T) {
	template := []any{
		map[string]any{
			"id":                "tpl-field",
			"template_block_id": "tpl-field",
			"type":              "field",
			"props": map[string]any{
				"locked": true,
			},
			"children": []any{
				map[string]any{
					"id":                "tpl-child",
					"template_block_id": "tpl-child",
					"type":              "paragraph",
					"props":             map[string]any{},
					"children":          []any{},
				},
			},
		},
	}

	instantiated := InstantiateTemplate(template)

	block, ok := instantiated[0].(map[string]any)
	if !ok {
		t.Fatalf("expected map block, got %T", instantiated[0])
	}

	children, ok := block["children"].([]any)
	if !ok {
		t.Fatalf("expected children slice, got %T", block["children"])
	}
	if len(children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(children))
	}

	child, ok := children[0].(map[string]any)
	if !ok {
		t.Fatalf("expected child map, got %T", children[0])
	}

	if _, ok := child["template_block_id"]; ok {
		t.Fatalf("expected child template_block_id to be removed, got %v", child["template_block_id"])
	}
}
