package mddm

import (
	"encoding/json"
	"testing"
)

func TestPOTemplateMDDM_Validates(t *testing.T) {
	body, err := json.Marshal(POTemplateMDDM())
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateMDDMBytes(body); err != nil {
		t.Errorf("PO template fails MDDM schema: %v", err)
	}
}

func TestPOTemplateMDDM_HasExpectedSections(t *testing.T) {
	tpl := POTemplateMDDM()
	blocks := tpl["blocks"].([]map[string]any)
	if len(blocks) < 5 {
		t.Errorf("expected at least 5 sections, got %d", len(blocks))
	}
}
