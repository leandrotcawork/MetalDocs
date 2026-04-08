package mddm

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestDiff_DetectsAddedBlock(t *testing.T) {
	prev := []any{
		map[string]any{
			"id": "doc-1", "type": "paragraph", "props": map[string]any{},
			"children": []any{map[string]any{"text": "x"}},
		},
	}
	curr := []any{
		map[string]any{
			"id": "doc-1", "type": "paragraph", "props": map[string]any{},
			"children": []any{map[string]any{"text": "x"}},
		},
		map[string]any{
			"id": "doc-2", "type": "paragraph", "props": map[string]any{},
			"children": []any{map[string]any{"text": "new"}},
		},
	}
	diff := ComputeDiff(prev, curr)
	if len(diff.Added) != 1 || diff.Added[0].ID != "doc-2" {
		t.Errorf("expected 1 added block doc-2, got %+v", diff.Added)
	}
	if len(diff.Removed) != 0 || len(diff.Modified) != 0 {
		t.Errorf("expected only added, got %+v", diff)
	}
}

func TestDiff_DetectsRemovedBlock(t *testing.T) {
	prev := []any{
		map[string]any{"id": "doc-1", "type": "paragraph", "props": map[string]any{}, "children": []any{map[string]any{"text": "x"}}},
		map[string]any{"id": "doc-2", "type": "paragraph", "props": map[string]any{}, "children": []any{map[string]any{"text": "y"}}},
	}
	curr := []any{
		map[string]any{"id": "doc-1", "type": "paragraph", "props": map[string]any{}, "children": []any{map[string]any{"text": "x"}}},
	}
	diff := ComputeDiff(prev, curr)
	if len(diff.Removed) != 1 || diff.Removed[0].ID != "doc-2" {
		t.Errorf("expected 1 removed block doc-2, got %+v", diff.Removed)
	}
}

func TestDiff_DetectsModifiedProps(t *testing.T) {
	prev := []any{
		map[string]any{"id": "doc-1", "type": "section", "props": map[string]any{"title": "Old", "color": "#000000", "locked": true}, "children": []any{}},
	}
	curr := []any{
		map[string]any{"id": "doc-1", "type": "section", "props": map[string]any{"title": "New", "color": "#000000", "locked": true}, "children": []any{}},
	}
	diff := ComputeDiff(prev, curr)
	if len(diff.Modified) != 1 || diff.Modified[0].ID != "doc-1" {
		t.Errorf("expected 1 modified block doc-1, got %+v", diff.Modified)
	}
}

func TestDiff_RoundTripsAsJSON(t *testing.T) {
	diff := Diff{
		Added:    []DiffEntry{{ID: "a", Type: "paragraph"}},
		Removed:  []DiffEntry{{ID: "r"}},
		Modified: []DiffEntry{{ID: "m"}},
	}
	bytes, err := json.Marshal(diff)
	if err != nil {
		t.Fatal(err)
	}
	var back Diff
	if err := json.Unmarshal(bytes, &back); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(diff, back) {
		t.Error("round-trip mismatch")
	}
}
