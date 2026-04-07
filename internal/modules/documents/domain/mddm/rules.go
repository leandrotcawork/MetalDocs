package mddm

import (
	"context"
	"fmt"
)

const (
	MaxBlocksPerDocument = 5000
	MaxNestingDepth      = 20
	MaxChildrenPerBlock  = 1000
	MaxDataTableRows     = 500
	MaxRepeatableItems   = 200
	MaxPayloadBytes      = 5 * 1024 * 1024
	MaxInlineTextLength  = 10000
)

// RulesContext carries dependencies needed by Layer 2 validators (DB, auth, etc.).
// Concrete fields are injected by the application layer at save time.
type RulesContext struct {
	Ctx              context.Context
	DocumentID       string
	UserID           string
	TemplateBlocks   []any // canonicalized template blocks (already hash-verified by caller)
	PreviousBlocks   []any // canonicalized previous version blocks (for ID continuity)
	ImageStorage     ImageStorage
	DocumentLookup   DocumentLookup
	ImageAuthChecker ImageAuthChecker
}

// DocumentLookup checks if a document_id exists and the user can read it.
type DocumentLookup interface {
	Exists(ctx context.Context, documentID string) (bool, error)
	UserCanRead(ctx context.Context, userID, documentID string) (bool, error)
}

// ImageAuthChecker checks if an image_id is reachable for a given user.
type ImageAuthChecker interface {
	UserCanReadImage(ctx context.Context, userID, imageID string) (bool, error)
}

type RuleViolation struct {
	Code    string
	BlockID string
	Message string
}

func (e *RuleViolation) Error() string {
	return fmt.Sprintf("[%s] %s (block=%s)", e.Code, e.Message, e.BlockID)
}

// EnforceLayer2 runs all business-rule validators in order.
// Each validator is small and named after the rule it enforces.
func EnforceLayer2(rctx RulesContext, envelope map[string]any) error {
	blocks, _ := envelope["blocks"].([]any)

	if err := checkSizeLimits(blocks); err != nil {
		return err
	}
	if err := checkIDUniqueness(blocks); err != nil {
		return err
	}
	if err := checkParentChildGrammar(blocks); err != nil {
		return err
	}
	// Other validators are added by subsequent tasks: minItems/maxItems,
	// DataTable consistency, image existence, cross-doc references,
	// block ID continuity. Each is wired here after its task lands.
	return nil
}

func checkSizeLimits(blocks []any) error {
	count := 0
	var maxDepth int
	var walk func([]any, int)
	walk = func(bs []any, depth int) {
		if depth > maxDepth {
			maxDepth = depth
		}
		for _, b := range bs {
			count++
			bm, ok := b.(map[string]any)
			if !ok {
				continue
			}
			if children, ok := bm["children"].([]any); ok {
				walk(children, depth+1)
			}
		}
	}
	walk(blocks, 1)

	if count > MaxBlocksPerDocument {
		return &RuleViolation{Code: "MAX_BLOCKS_EXCEEDED", Message: fmt.Sprintf("blocks=%d > %d", count, MaxBlocksPerDocument)}
	}
	if maxDepth > MaxNestingDepth {
		return &RuleViolation{Code: "MAX_DEPTH_EXCEEDED", Message: fmt.Sprintf("depth=%d > %d", maxDepth, MaxNestingDepth)}
	}
	return nil
}

func checkIDUniqueness(blocks []any) error {
	seen := map[string]bool{}
	var walk func([]any) error
	walk = func(bs []any) error {
		for _, b := range bs {
			bm, ok := b.(map[string]any)
			if !ok {
				continue
			}
			id, _ := bm["id"].(string)
			if seen[id] {
				return &RuleViolation{Code: "ID_NOT_UNIQUE", BlockID: id, Message: "duplicate block id"}
			}
			seen[id] = true
			if children, ok := bm["children"].([]any); ok {
				if err := walk(children); err != nil {
					return err
				}
			}
		}
		return nil
	}
	return walk(blocks)
}

// allowedChildren returns the set of allowed child block types for each parent type.
var allowedChildren = map[string]map[string]bool{
	"section":        mapSet("fieldGroup", "field", "richBlock", "repeatable", "dataTable", "paragraph", "heading", "bulletListItem", "numberedListItem", "image", "quote", "code", "divider"),
	"fieldGroup":     mapSet("field"),
	"repeatable":     mapSet("repeatableItem"),
	"repeatableItem": mapSet("paragraph", "heading", "bulletListItem", "numberedListItem", "image", "quote", "code", "divider", "richBlock"),
	"dataTable":      mapSet("dataTableRow"),
	"dataTableRow":   mapSet("dataTableCell"),
	"richBlock":      mapSet("paragraph", "heading", "bulletListItem", "numberedListItem", "image", "quote", "code", "divider"),
	"quote":          mapSet("paragraph"),
}

func mapSet(items ...string) map[string]bool {
	out := map[string]bool{}
	for _, i := range items {
		out[i] = true
	}
	return out
}

func checkParentChildGrammar(blocks []any) error {
	var walk func([]any, string) error
	walk = func(bs []any, parentType string) error {
		for _, b := range bs {
			bm, ok := b.(map[string]any)
			if !ok {
				continue
			}
			blockType, _ := bm["type"].(string)
			if allowed, has := allowedChildren[parentType]; has && !allowed[blockType] {
				id, _ := bm["id"].(string)
				return &RuleViolation{Code: "GRAMMAR_VIOLATION", BlockID: id, Message: fmt.Sprintf("%s not allowed inside %s", blockType, parentType)}
			}
			if children, ok := bm["children"].([]any); ok {
				if err := walk(children, blockType); err != nil {
					return err
				}
			}
		}
		return nil
	}
	return walk(blocks, "section") // top-level treated like a section's allowed children
}
