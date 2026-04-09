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

func TestPOTemplateMDDM_MatchesApprovedTemplateV2Layout(t *testing.T) {
	tpl := POTemplateMDDM()
	blocks := templateSections(t, tpl)

	if got := len(blocks); got != 10 {
		t.Fatalf("section count = %d, want 10", got)
	}

	wantTitles := []string{
		"Identificação",
		"Identificação do Processo",
		"Entradas e Saídas",
		"Visão Geral do Processo",
		"Detalhamento das Etapas",
		"Controle e Exceções",
		"Indicadores de Desempenho",
		"Documentos e Referências",
		"Glossário",
		"Histórico de Revisões",
	}

	wantOptional := []bool{false, false, false, false, false, false, true, true, true, true}

	for i, section := range blocks {
		props := blockProps(t, section, "section")
		if got := props["title"].(string); got != wantTitles[i] {
			t.Fatalf("section %d title = %q, want %q", i+1, got, wantTitles[i])
		}

		optional, hasOptional := props["optional"].(bool)
		if wantOptional[i] {
			if !hasOptional || !optional {
				t.Fatalf("section %d optional = %v (present=%v), want true", i+1, optional, hasOptional)
			}
		} else if hasOptional {
			t.Fatalf("section %d unexpectedly marked optional", i+1)
		}
	}

	repeatables := map[string]map[string]any{}
	for _, section := range blocks {
		for _, child := range templateChildren(t, section) {
			if blockType(child) == "repeatable" {
				props := blockProps(t, child, "repeatable")
				repeatables[props["label"].(string)] = props
			}
		}
	}

	if got := int(repeatables["Etapas"]["minItems"].(int)); got != 1 {
		t.Fatalf("Etapas minItems = %d, want 1", got)
	}
	if got := len(templateChildrenByLabel(t, blocks, "Etapas")); got != 1 {
		t.Fatalf("Etapas child count = %d, want 1 repeatable item", got)
	}

	for _, label := range []string{"KPIs", "Referências", "Glossário", "Revisões"} {
		props, ok := repeatables[label]
		if !ok {
			t.Fatalf("repeatable %q not found", label)
		}
		if got := props["minItems"].(int); got != 0 {
			t.Fatalf("%s minItems = %d, want 0", label, got)
		}
	}

	stageItem := templateRepeatableItemByLabel(t, blocks, "Etapas")
	stageRichBlock := templateChildByType(t, stageItem, "richBlock")
	richTypes := childTypes(templateChildren(t, stageRichBlock))
	for _, want := range []string{"paragraph", "bulletListItem", "numberedListItem", "dataTable"} {
		if !containsString(richTypes, want) {
			t.Fatalf("etapa rich area missing %q child; got %v", want, richTypes)
		}
	}

	overviewRichBlock := templateChildByType(t, blocks[3], "richBlock")
	overviewRichTypes := childTypes(templateChildren(t, overviewRichBlock))
	for _, want := range []string{"paragraph", "bulletListItem"} {
		if !containsString(overviewRichTypes, want) {
			t.Fatalf("overview rich area missing %q child; got %v", want, overviewRichTypes)
		}
	}
}

func templateSections(t *testing.T, tpl map[string]any) []map[string]any {
	t.Helper()
	blocks, ok := tpl["blocks"].([]map[string]any)
	if !ok {
		t.Fatalf("blocks type = %T, want []map[string]any", tpl["blocks"])
	}
	return blocks
}

func templateChildren(t *testing.T, block map[string]any) []map[string]any {
	t.Helper()
	children, ok := block["children"].([]any)
	if !ok {
		t.Fatalf("block %q children type = %T, want []any", blockType(block), block["children"])
	}
	out := make([]map[string]any, 0, len(children))
	for _, child := range children {
		childMap, ok := child.(map[string]any)
		if !ok {
			t.Fatalf("child type = %T, want map[string]any", child)
		}
		out = append(out, childMap)
	}
	return out
}

func templateChildrenByLabel(t *testing.T, blocks []map[string]any, label string) []map[string]any {
	t.Helper()
	stageItem := templateRepeatableItemByLabel(t, blocks, label)
	return templateChildren(t, stageItem)
}

func templateRepeatableItemByLabel(t *testing.T, blocks []map[string]any, label string) map[string]any {
	t.Helper()
	for _, section := range blocks {
		for _, child := range templateChildren(t, section) {
			if blockType(child) != "repeatable" {
				continue
			}
			props := blockProps(t, child, "repeatable")
			if props["label"].(string) != label {
				continue
			}
			items := templateChildren(t, child)
			if len(items) == 0 {
				t.Fatalf("repeatable %q has no items", label)
			}
			return items[0]
		}
	}
	t.Fatalf("repeatable %q not found", label)
	return nil
}

func templateChildByType(t *testing.T, block map[string]any, wantType string) map[string]any {
	t.Helper()
	for _, child := range templateChildren(t, block) {
		if blockType(child) == wantType {
			return child
		}
	}
	t.Fatalf("block %q has no child of type %q", blockType(block), wantType)
	return nil
}

func blockProps(t *testing.T, block map[string]any, wantType string) map[string]any {
	t.Helper()
	if got := blockType(block); got != wantType {
		t.Fatalf("block type = %q, want %q", got, wantType)
	}
	props, ok := block["props"].(map[string]any)
	if !ok {
		t.Fatalf("block %q props type = %T, want map[string]any", wantType, block["props"])
	}
	return props
}

func blockType(block map[string]any) string {
	t, _ := block["type"].(string)
	return t
}

func childTypes(children []map[string]any) []string {
	out := make([]string, 0, len(children))
	for _, child := range children {
		out = append(out, blockType(child))
	}
	return out
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func countType(values []string, want string) int {
	count := 0
	for _, value := range values {
		if value == want {
			count++
		}
	}
	return count
}
