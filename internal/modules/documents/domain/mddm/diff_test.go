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

func TestDiff_IgnoresInlineTextChildrenWithoutIDs(t *testing.T) {
	prev := []any{
		map[string]any{
			"id": "doc-1",
			"type": "paragraph",
			"props": map[string]any{},
			"children": []any{
				map[string]any{"text": "a"},
				map[string]any{"text": "b"},
			},
		},
	}
	curr := []any{
		map[string]any{
			"id": "doc-1",
			"type": "paragraph",
			"props": map[string]any{},
			"children": []any{
				map[string]any{"text": "a"},
			},
		},
	}

	diff := ComputeDiff(prev, curr)
	if len(diff.Added) != 0 || len(diff.Removed) != 0 || len(diff.Modified) != 0 {
		t.Fatalf("expected no diff entries for inline-text child change, got %+v", diff)
	}
	for _, entry := range append(append([]DiffEntry{}, diff.Added...), append(diff.Removed, diff.Modified...)...) {
		if entry.ID == "" {
			t.Fatalf("expected no empty-id diff entries, got %+v", diff)
		}
	}
}

func TestDiff_ProducesDeterministicOrdering(t *testing.T) {
	prev := []any{
		map[string]any{"id": "c", "type": "paragraph", "props": map[string]any{"text": "keep"}, "children": []any{}},
		map[string]any{"id": "b", "type": "paragraph", "props": map[string]any{"text": "old-b"}, "children": []any{}},
		map[string]any{"id": "a", "type": "paragraph", "props": map[string]any{"text": "old-a"}, "children": []any{}},
	}
	curr := []any{
		map[string]any{"id": "b", "type": "paragraph", "props": map[string]any{"text": "new-b"}, "children": []any{}},
		map[string]any{"id": "a", "type": "paragraph", "props": map[string]any{"text": "new-a"}, "children": []any{}},
		map[string]any{"id": "d", "type": "paragraph", "props": map[string]any{"text": "new-d"}, "children": []any{}},
	}

	first := ComputeDiff(prev, curr)
	for i := 0; i < 10; i++ {
		next := ComputeDiff(prev, curr)
		if !reflect.DeepEqual(first, next) {
			t.Fatalf("expected deterministic diff ordering, first=%+v next=%+v", first, next)
		}
	}

	if got := diffIDs(first.Added); !reflect.DeepEqual(got, []string{"d"}) {
		t.Fatalf("unexpected added order: %v", got)
	}
	if got := diffIDs(first.Removed); !reflect.DeepEqual(got, []string{"c"}) {
		t.Fatalf("unexpected removed order: %v", got)
	}
	if got := diffIDs(first.Modified); !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Fatalf("unexpected modified order: %v", got)
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

func diffIDs(entries []DiffEntry) []string {
	ids := make([]string, 0, len(entries))
	for _, entry := range entries {
		ids = append(ids, entry.ID)
	}
	return ids
}
