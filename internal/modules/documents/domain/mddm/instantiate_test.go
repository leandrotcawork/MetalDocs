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

func TestInstantiate_InlineRunKeepsOriginalShape(t *testing.T) {
	template := []any{
		map[string]any{
			"id":                "tpl-field",
			"template_block_id": "tpl-field",
			"type":              "field",
			"props": map[string]any{
				"locked": true,
			},
			"children": []any{
				map[string]any{"text": "x"},
			},
		},
	}

	instantiated := InstantiateTemplate(template)

	block := instantiated[0].(map[string]any)
	children := block["children"].([]any)
	inline := children[0].(map[string]any)

	if _, ok := inline["id"]; ok {
		t.Fatalf("expected inline run to keep no synthetic id, got %v", inline["id"])
	}
	if _, ok := inline["template_block_id"]; ok {
		t.Fatalf("expected inline run to keep no template_block_id, got %v", inline["template_block_id"])
	}
	if got := inline["text"]; got != "x" {
		t.Fatalf("expected inline text x, got %v", got)
	}
}

func TestInstantiate_OutputMutationDoesNotMutateSourceProps(t *testing.T) {
	template := []any{
		map[string]any{
			"id":                "tpl-A",
			"template_block_id": "tpl-A",
			"type":              "section",
			"props": map[string]any{
				"locked": true,
				"meta": map[string]any{
					"title": "Original",
				},
			},
			"children": []any{},
		},
	}

	instantiated := InstantiateTemplate(template)
	block := instantiated[0].(map[string]any)
	props := block["props"].(map[string]any)
	props["locked"] = false
	props["meta"].(map[string]any)["title"] = "Changed"

	sourceProps := template[0].(map[string]any)["props"].(map[string]any)
	if got := sourceProps["locked"]; got != true {
		t.Fatalf("expected source locked to remain true, got %v", got)
	}
	if got := sourceProps["meta"].(map[string]any)["title"]; got != "Original" {
		t.Fatalf("expected source nested title to remain Original, got %v", got)
	}
}
