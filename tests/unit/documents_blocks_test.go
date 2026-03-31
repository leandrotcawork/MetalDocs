package unit

import (
	"encoding/json"
	"testing"

	"metaldocs/internal/modules/documents/domain"
)

func TestValidateBlocksAcceptsParagraphAndImage(t *testing.T) {
	raw := []byte(`[
	  {"type":"paragraph","content":[{"type":"text","text":"Step A","bold":true}]},
	  {"type":"image","base64":"data","mimeType":"image/png","width":320,"caption":"shot"}
	]`)

	var blocks []domain.Block
	if err := json.Unmarshal(raw, &blocks); err != nil {
		t.Fatalf("unmarshal blocks: %v", err)
	}
	if err := domain.ValidateBlocks(blocks); err != nil {
		t.Fatalf("expected blocks to validate, got %v", err)
	}
}
