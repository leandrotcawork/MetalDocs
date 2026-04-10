package mddm

import (
	"bytes"
	"encoding/json"
	"sort"

	"golang.org/x/text/unicode/norm"
)

var markOrder = map[string]int{
	"bold":      0,
	"code":      1,
	"italic":    2,
	"strike":    3,
	"underline": 4,
}

func nfc(s string) string {
	return norm.NFC.String(s)
}

// CanonicalizeMDDM produces a canonical form of an MDDM envelope.
func CanonicalizeMDDM(envelope map[string]any) (map[string]any, error) {
	out := map[string]any{
		"mddm_version": envelope["mddm_version"],
		"template_ref": envelope["template_ref"],
		"blocks":       []any{},
	}
	if blocks, ok := envelope["blocks"].([]any); ok {
		canonicalBlocks := make([]any, 0, len(blocks))
		for _, b := range blocks {
			canonicalBlocks = append(canonicalBlocks, canonicalizeBlock(b.(map[string]any)))
		}
		out["blocks"] = canonicalBlocks
	}
	return out, nil
}

func canonicalizeBlock(block map[string]any) map[string]any {
	result := make(map[string]any, len(block))
	for k, v := range block {
		result[k] = v
	}

	blockType, _ := block["type"].(string)

	inlineParents := map[string]bool{
		"paragraph":        true,
		"heading":          true,
		"bulletListItem":   true,
		"numberedListItem": true,
		"dataTableCell":    true,
		"field":            true,
	}

	if children, ok := block["children"].([]any); ok {
		if inlineParents[blockType] {
			result["children"] = canonicalizeInlineContent(children)
		} else {
			canonicalChildren := make([]any, 0, len(children))
			for _, c := range children {
				if cm, ok := c.(map[string]any); ok {
					canonicalChildren = append(canonicalChildren, canonicalizeBlock(cm))
				} else {
					canonicalChildren = append(canonicalChildren, c)
				}
			}
			result["children"] = canonicalChildren
		}
	}

	// NFC normalize string props except for Code blocks
	if blockType != "code" {
		if props, ok := result["props"].(map[string]any); ok {
			normalizedProps := make(map[string]any, len(props))
			for k, v := range props {
				if s, ok := v.(string); ok && (k == "title" || k == "label") {
					normalizedProps[k] = nfc(s)
				} else {
					normalizedProps[k] = v
				}
			}
			result["props"] = normalizedProps
		}
	}

	return result
}

func canonicalizeInlineContent(runs []any) []any {
	// Step 1: NFC + sort marks within each run
	prepared := make([]map[string]any, 0, len(runs))
	for _, r := range runs {
		runMap, ok := r.(map[string]any)
		if !ok {
			continue
		}
		newRun := make(map[string]any, len(runMap))
		for k, v := range runMap {
			newRun[k] = v
		}
		if text, ok := newRun["text"].(string); ok {
			newRun["text"] = nfc(text)
		}
		if marks, ok := newRun["marks"].([]any); ok {
			sortedMarks := make([]any, len(marks))
			copy(sortedMarks, marks)
			sort.SliceStable(sortedMarks, func(i, j int) bool {
				aType := sortedMarks[i].(map[string]any)["type"].(string)
				bType := sortedMarks[j].(map[string]any)["type"].(string)
				aIdx, aKnown := markOrder[aType]
				bIdx, bKnown := markOrder[bType]
				if !aKnown && !bKnown {
					return aType < bType
				}
				if !aKnown {
					return false
				}
				if !bKnown {
					return true
				}
				return aIdx < bIdx
			})
			newRun["marks"] = sortedMarks
		}
		prepared = append(prepared, newRun)
	}

	// Step 2: Merge adjacent runs with identical marks/link/document_ref
	merged := make([]map[string]any, 0, len(prepared))
	for _, run := range prepared {
		if len(merged) > 0 && runsEquivalent(merged[len(merged)-1], run) {
			last := merged[len(merged)-1]
			last["text"] = last["text"].(string) + run["text"].(string)
		} else {
			merged = append(merged, run)
		}
	}

	out := make([]any, 0, len(merged))
	for _, m := range merged {
		out = append(out, m)
	}
	return out
}

func runsEquivalent(a, b map[string]any) bool {
	aMarks, _ := json.Marshal(a["marks"])
	bMarks, _ := json.Marshal(b["marks"])
	if !bytes.Equal(aMarks, bMarks) {
		return false
	}
	aLink, _ := json.Marshal(a["link"])
	bLink, _ := json.Marshal(b["link"])
	if !bytes.Equal(aLink, bLink) {
		return false
	}
	aRef, _ := json.Marshal(a["document_ref"])
	bRef, _ := json.Marshal(b["document_ref"])
	return bytes.Equal(aRef, bRef)
}

// MarshalCanonical marshals a map to JSON with sorted keys (deterministic).
func MarshalCanonical(v any) ([]byte, error) {
	return marshalSortedKeys(v)
}

func marshalSortedKeys(v any) ([]byte, error) {
	switch val := v.(type) {
	case map[string]any:
		var buf bytes.Buffer
		buf.WriteByte('{')
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for i, k := range keys {
			if i > 0 {
				buf.WriteByte(',')
			}
			kb, _ := json.Marshal(k)
			buf.Write(kb)
			buf.WriteByte(':')
			vb, err := marshalSortedKeys(val[k])
			if err != nil {
				return nil, err
			}
			buf.Write(vb)
		}
		buf.WriteByte('}')
		return buf.Bytes(), nil
	case []any:
		var buf bytes.Buffer
		buf.WriteByte('[')
		for i, item := range val {
			if i > 0 {
				buf.WriteByte(',')
			}
			ib, err := marshalSortedKeys(item)
			if err != nil {
				return nil, err
			}
			buf.Write(ib)
		}
		buf.WriteByte(']')
		return buf.Bytes(), nil
	default:
		return json.Marshal(v)
	}
}
