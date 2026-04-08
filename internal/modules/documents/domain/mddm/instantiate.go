package mddm

import "github.com/google/uuid"

// InstantiateTemplate clones template blocks into document blocks with fresh ids.
func InstantiateTemplate(template []any) []any {
	out := make([]any, len(template))
	for i, block := range template {
		out[i] = instantiateBlock(block, false)
	}
	return out
}

func instantiateBlock(b any, insideContentSlot bool) any {
	block, ok := b.(map[string]any)
	if !ok {
		return b
	}

	cloned := make(map[string]any, len(block))
	for k, v := range block {
		cloned[k] = v
	}

	originalID, _ := block["id"].(string)
	cloned["id"] = uuid.NewString()

	blockType, _ := block["type"].(string)
	if structuralBlockTypes[blockType] && !insideContentSlot {
		cloned["template_block_id"] = originalID
	} else {
		delete(cloned, "template_block_id")
	}

	if children, ok := block["children"].([]any); ok {
		nextInsideContentSlot := insideContentSlot || isContentSlotParent(blockType)
		instantiatedChildren := make([]any, len(children))
		for i, child := range children {
			instantiatedChildren[i] = instantiateBlock(child, nextInsideContentSlot)
		}
		cloned["children"] = instantiatedChildren
	}

	return cloned
}

func isContentSlotParent(t string) bool {
	switch t {
	case "field", "repeatableItem", "richBlock", "dataTableRow", "dataTableCell":
		return true
	default:
		return false
	}
}
