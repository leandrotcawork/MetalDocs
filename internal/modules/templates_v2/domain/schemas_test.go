package domain

import (
	"encoding/json"
	"testing"
)

func TestPlaceholderType_AllConstants(t *testing.T) {
	types := []PlaceholderType{PHText, PHDate, PHNumber, PHSelect, PHUser, PHPicture, PHComputed}
	wants := []string{"text", "date", "number", "select", "user", "picture", "computed"}
	for i, pt := range types {
		if string(pt) != wants[i] {
			t.Fatalf("PlaceholderType[%d] = %q, want %q", i, pt, wants[i])
		}
	}
}

func TestPlaceholder_JSONRoundTrip_AllFields(t *testing.T) {
	regex := "^[A-Z]{3}-\\d{4}$"
	mn, mx := 0.0, 100.0
	maxLen := 120
	rkey := "doc_code"
	ph := Placeholder{
		ID: "p1", Label: "Doc Code", Type: PHText, Required: true,
		Regex: &regex, MaxLength: &maxLen, MinNumber: &mn, MaxNumber: &mx,
		VisibleIf: &VisibilityCondition{PlaceholderID: "p0", Op: "eq", Value: "x"},
		Computed:  true, ResolverKey: &rkey,
	}
	b, err := json.Marshal(ph)
	if err != nil {
		t.Fatal(err)
	}
	var back Placeholder
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	if back.ID != "p1" || !back.Computed || back.ResolverKey == nil || *back.ResolverKey != "doc_code" {
		t.Fatalf("round-trip mismatch: %+v", back)
	}
	if back.VisibleIf == nil || back.VisibleIf.Op != "eq" {
		t.Fatalf("visible_if lost: %+v", back.VisibleIf)
	}
}

func TestCompositionConfig_RoundTrip(t *testing.T) {
	c := CompositionConfig{
		HeaderSubBlocks: []string{"doc_header_standard"},
		FooterSubBlocks: []string{"footer_page_numbers", "footer_controlled_copy_notice"},
		SubBlockParams: map[string]map[string]any{
			"doc_header_standard": {"show_logo": true},
		},
	}
	b, _ := json.Marshal(c)
	var back CompositionConfig
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	if len(back.FooterSubBlocks) != 2 {
		t.Fatalf("footer: %+v", back.FooterSubBlocks)
	}
	if back.SubBlockParams["doc_header_standard"]["show_logo"] != true {
		t.Fatalf("params lost: %+v", back.SubBlockParams)
	}
}
