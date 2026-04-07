package mddm

import (
	"encoding/json"
	"fmt"
)

// LockViolationError is returned when a locked-block check fails.
type LockViolationError struct {
	BlockID string
	Code    string
	Message string
}

func (e *LockViolationError) Error() string {
	return fmt.Sprintf("[%s] %s (block=%s)", e.Code, e.Message, e.BlockID)
}

// structuralBlockTypes are blocks that may have template_block_id and are subject to lock checks.
var structuralBlockTypes = map[string]bool{
	"section":    true,
	"fieldGroup": true,
	"field":      true,
	"repeatable": true,
	"dataTable":  true,
	"richBlock":  true,
}

// EnforceLockedBlocks walks the template tree and verifies that every templated structural
// block exists in the document tree (matched by template_block_id), with unchanged props
// when locked: true, and matching position among templated siblings.
func EnforceLockedBlocks(templateBlocks, docBlocks []any) error {
	templateIndex := indexByTemplateBlockID(templateBlocks, nil)
	docIndex := indexByTemplateBlockID(docBlocks, nil)

	for tbID, tNode := range templateIndex {
		dNode, ok := docIndex[tbID]
		if !ok {
			return &LockViolationError{
				BlockID: tbID,
				Code:    "LOCKED_BLOCK_DELETED",
				Message: "templated block missing from document",
			}
		}
		if isLocked(tNode.block) {
			if !propsEqual(tNode.block, dNode.block) {
				id, _ := dNode.block["id"].(string)
				return &LockViolationError{
					BlockID: id,
					Code:    "LOCKED_BLOCK_PROP_MUTATED",
					Message: "props of locked block were modified",
				}
			}
		}
		// Position check: ensure parent template_block_id matches
		if tNode.parentTBID != dNode.parentTBID {
			id, _ := dNode.block["id"].(string)
			return &LockViolationError{
				BlockID: id,
				Code:    "LOCKED_BLOCK_REPARENTED",
				Message: "templated block moved to different parent",
			}
		}
	}

	return nil
}

type indexedNode struct {
	block      map[string]any
	parentTBID string
}

func indexByTemplateBlockID(blocks []any, parentTBID *string) map[string]indexedNode {
	out := map[string]indexedNode{}
	var parentID string
	if parentTBID != nil {
		parentID = *parentTBID
	}
	for _, b := range blocks {
		bm, ok := b.(map[string]any)
		if !ok {
			continue
		}
		blockType, _ := bm["type"].(string)
		tbID, hasTB := bm["template_block_id"].(string)
		if hasTB && structuralBlockTypes[blockType] {
			out[tbID] = indexedNode{block: bm, parentTBID: parentID}
		}
		if children, ok := bm["children"].([]any); ok {
			var nextParent *string
			if hasTB {
				p := tbID
				nextParent = &p
			} else {
				nextParent = parentTBID
			}
			for k, v := range indexByTemplateBlockID(children, nextParent) {
				out[k] = v
			}
		}
	}
	return out
}

func isLocked(block map[string]any) bool {
	props, ok := block["props"].(map[string]any)
	if !ok {
		return false
	}
	locked, _ := props["locked"].(bool)
	return locked
}

func propsEqual(a, b map[string]any) bool {
	pa, _ := json.Marshal(a["props"])
	pb, _ := json.Marshal(b["props"])
	return string(pa) == string(pb)
}
