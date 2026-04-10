package mddm

func CheckBlockIDContinuity(prev, curr []any) error {
	prevIdx := indexTemplateBlocks(prev)
	currIdx := indexTemplateBlocks(curr)

	for tbID, currNode := range currIdx {
		prevNode, ok := prevIdx[tbID]
		if !ok {
			continue // new templated block (rare; only on template changes)
		}
		if prevNode.id != currNode.id {
			return &RuleViolation{
				Code:    "BLOCK_ID_REWRITE_FORBIDDEN",
				BlockID: currNode.id,
				Message: "templated block id changed across save (was " + prevNode.id + ")",
			}
		}
	}
	return nil
}

type templateNodeRef struct {
	id string
}

func indexTemplateBlocks(blocks []any) map[string]templateNodeRef {
	out := map[string]templateNodeRef{}
	var walk func([]any)
	walk = func(bs []any) {
		for _, b := range bs {
			bm, ok := b.(map[string]any)
			if !ok {
				continue
			}
			tbID, hasTB := bm["template_block_id"].(string)
			id, _ := bm["id"].(string)
			if hasTB {
				out[tbID] = templateNodeRef{id: id}
			}
			if children, ok := bm["children"].([]any); ok {
				walk(children)
			}
		}
	}
	walk(blocks)
	return out
}
