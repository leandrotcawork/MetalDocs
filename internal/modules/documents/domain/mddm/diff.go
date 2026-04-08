package mddm

import "encoding/json"

type DiffEntry struct {
	ID       string `json:"id"`
	Type     string `json:"type,omitempty"`
	ParentID string `json:"parent_id,omitempty"`
}

type Diff struct {
	Added    []DiffEntry `json:"added"`
	Removed  []DiffEntry `json:"removed"`
	Modified []DiffEntry `json:"modified"`
}

// ComputeDiff produces a structured diff between two block trees keyed by stable block IDs.
// Both inputs MUST be canonicalized before calling.
func ComputeDiff(prev, curr []any) Diff {
	prevIdx := flatIndex(prev, "")
	currIdx := flatIndex(curr, "")

	diff := Diff{Added: []DiffEntry{}, Removed: []DiffEntry{}, Modified: []DiffEntry{}}

	// Added: in curr but not in prev
	for id, node := range currIdx {
		if _, exists := prevIdx[id]; !exists {
			diff.Added = append(diff.Added, DiffEntry{ID: id, Type: node.blockType, ParentID: node.parentID})
		}
	}

	// Removed: in prev but not in curr
	for id, node := range prevIdx {
		if _, exists := currIdx[id]; !exists {
			diff.Removed = append(diff.Removed, DiffEntry{ID: id, Type: node.blockType, ParentID: node.parentID})
		}
	}

	// Modified: in both, but props differ
	for id, currNode := range currIdx {
		prevNode, exists := prevIdx[id]
		if !exists {
			continue
		}
		if !propsEqualBlocks(prevNode.block, currNode.block) {
			diff.Modified = append(diff.Modified, DiffEntry{ID: id, Type: currNode.blockType})
		}
	}

	return diff
}

type flatNode struct {
	block     map[string]any
	blockType string
	parentID  string
}

func flatIndex(blocks []any, parentID string) map[string]flatNode {
	out := map[string]flatNode{}
	for _, b := range blocks {
		bm, ok := b.(map[string]any)
		if !ok {
			continue
		}
		id, _ := bm["id"].(string)
		t, _ := bm["type"].(string)
		out[id] = flatNode{block: bm, blockType: t, parentID: parentID}
		if children, ok := bm["children"].([]any); ok {
			for k, v := range flatIndex(children, id) {
				out[k] = v
			}
		}
	}
	return out
}

func propsEqualBlocks(a, b map[string]any) bool {
	pa, _ := json.Marshal(a["props"])
	pb, _ := json.Marshal(b["props"])
	return string(pa) == string(pb)
}
