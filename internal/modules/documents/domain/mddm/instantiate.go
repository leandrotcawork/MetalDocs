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
		return cloneAny(b)
	}

	blockType, _ := block["type"].(string)
	if blockType == "" {
		return cloneAny(block)
	}

	cloned := cloneMap(block)
	originalID, _ := block["id"].(string)
	cloned["id"] = uuid.NewString()

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

func cloneAny(v any) any {
	switch value := v.(type) {
	case map[string]any:
		return cloneMap(value)
	case []any:
		cloned := make([]any, len(value))
		for i, item := range value {
			cloned[i] = cloneAny(item)
		}
		return cloned
	default:
		return v
	}
}

func cloneMap(src map[string]any) map[string]any {
	cloned := make(map[string]any, len(src))
	for k, v := range src {
		cloned[k] = cloneAny(v)
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
