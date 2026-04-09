package mddm

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func parseEnvelope(t *testing.T, s string) map[string]any {
	t.Helper()
	var env map[string]any
	if err := json.Unmarshal([]byte(s), &env); err != nil {
		t.Fatal(err)
	}
	return env
}

func TestRules_RejectDuplicateBlockIDs(t *testing.T) {
	env := parseEnvelope(t, `{
		"mddm_version": 1,
		"template_ref": null,
		"blocks": [
			{"id":"11111111-1111-1111-1111-111111111111","type":"paragraph","props":{},"children":[{"text":"a"}]},
			{"id":"11111111-1111-1111-1111-111111111111","type":"paragraph","props":{},"children":[{"text":"b"}]}
		]
	}`)
	err := EnforceLayer2(RulesContext{}, env)
	if err == nil || !strings.Contains(err.Error(), "ID_NOT_UNIQUE") {
		t.Errorf("expected ID_NOT_UNIQUE error, got %v", err)
	}
}

func TestRules_RejectMaxBlocksExceeded(t *testing.T) {
	blocks := make([]any, 0, 5001)
	for i := 0; i < 5001; i++ {
		blocks = append(blocks, map[string]any{
			"id":       "11111111-1111-1111-1111-" + padHex(i, 12),
			"type":     "paragraph",
			"props":    map[string]any{},
			"children": []any{map[string]any{"text": "x"}},
		})
	}
	env := map[string]any{
		"mddm_version": float64(1),
		"template_ref": nil,
		"blocks":       blocks,
	}
	err := EnforceLayer2(RulesContext{}, env)
	if err == nil || !strings.Contains(err.Error(), "MAX_BLOCKS_EXCEEDED") {
		t.Errorf("expected MAX_BLOCKS_EXCEEDED error, got %v", err)
	}
}

func TestRules_RejectInvalidGrammar(t *testing.T) {
	// FieldGroup with a Paragraph child (not allowed)
	env := parseEnvelope(t, `{
		"mddm_version": 1,
		"template_ref": null,
		"blocks": [{
			"id":"11111111-1111-1111-1111-111111111111",
			"type":"fieldGroup",
			"props":{"columns":1,"locked":true},
			"children":[{"id":"22222222-2222-2222-2222-222222222222","type":"paragraph","props":{},"children":[{"text":"x"}]}]
		}]
	}`)
	err := EnforceLayer2(RulesContext{}, env)
	if err == nil || !strings.Contains(err.Error(), "GRAMMAR_VIOLATION") {
		t.Errorf("expected GRAMMAR_VIOLATION error, got %v", err)
	}
}

func padHex(n, width int) string {
	const hex = "0123456789abcdef"
	out := make([]byte, width)
	for i := width - 1; i >= 0; i-- {
		out[i] = hex[n&0xf]
		n >>= 4
	}
	return string(out)
}

func TestRules_RejectRepeatableBelowMinItems(t *testing.T) {
	env := parseEnvelope(t, `{
		"mddm_version": 1,
		"template_ref": null,
		"blocks": [{
			"id":"11111111-1111-1111-1111-111111111111",
			"type":"repeatable",
			"props":{"label":"E","itemPrefix":"Etapa","locked":true,"minItems":2,"maxItems":10},
			"children":[
				{"id":"22222222-2222-2222-2222-222222222222","type":"repeatableItem","props":{"title":"only one"},"children":[]}
			]
		}]
	}`)
	err := EnforceLayer2(RulesContext{}, env)
	if err == nil || !strings.Contains(err.Error(), "REPEATABLE_BELOW_MIN") {
		t.Errorf("expected REPEATABLE_BELOW_MIN error, got %v", err)
	}
}

func TestRules_RejectDataTableAboveMaxRows(t *testing.T) {
	rows := make([]any, 0, 6)
	for i := 0; i < 6; i++ {
		rows = append(rows, map[string]any{
			"id":       "33333333-3333-3333-3333-" + padHex(i, 12),
			"type":     "dataTableRow",
			"props":    map[string]any{},
			"children": []any{},
		})
	}
	env := map[string]any{
		"mddm_version": float64(1),
		"template_ref": nil,
		"blocks": []any{
			map[string]any{
				"id":   "11111111-1111-1111-1111-111111111111",
				"type": "dataTable",
				"props": map[string]any{
					"label": "T", "columns": []any{}, "locked": true,
					"minRows": float64(0), "maxRows": float64(5),
				},
				"children": rows,
			},
		},
	}
	err := EnforceLayer2(RulesContext{}, env)
	if err == nil || !strings.Contains(err.Error(), "DATATABLE_ABOVE_MAX") {
		t.Errorf("expected DATATABLE_ABOVE_MAX error, got %v", err)
	}
}

func TestRules_RejectDataTableCellMissingColumn(t *testing.T) {
	env := parseEnvelope(t, `{
		"mddm_version": 1, "template_ref": null,
		"blocks": [{
			"id":"11111111-1111-1111-1111-111111111111",
			"type":"dataTable",
			"props":{
				"label":"KPIs",
				"columns":[{"key":"a","label":"A","type":"text","required":false}],
				"locked":true,"minRows":0,"maxRows":500
			},
			"children":[{
				"id":"22222222-2222-2222-2222-222222222222",
				"type":"dataTableRow","props":{},
				"children":[{"id":"33333333-3333-3333-3333-333333333333","type":"dataTableCell","props":{"columnKey":"unknown"},"children":[{"text":"x"}]}]
			}]
		}]
	}`)
	err := EnforceLayer2(RulesContext{}, env)
	if err == nil || !strings.Contains(err.Error(), "DATATABLE_INVALID_COLUMN_KEY") {
		t.Errorf("expected DATATABLE_INVALID_COLUMN_KEY error, got %v", err)
	}
}

type fakeImageAuthChecker struct {
	allowed map[string]bool
}

func (f *fakeImageAuthChecker) UserCanReadImage(ctx context.Context, userID, imageID string) (bool, error) {
	return f.allowed[imageID], nil
}

func TestRules_RejectImageWithoutAuth(t *testing.T) {
	env := parseEnvelope(t, `{
		"mddm_version":1,"template_ref":null,
		"blocks":[{
			"id":"11111111-1111-1111-1111-111111111111",
			"type":"image",
			"props":{"src":"/api/images/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa","alt":"x","caption":""}
		}]
	}`)
	rctx := RulesContext{
		Ctx:              context.Background(),
		UserID:           "user-1",
		ImageAuthChecker: &fakeImageAuthChecker{allowed: map[string]bool{}},
	}
	err := EnforceLayer2(rctx, env)
	if err == nil || !strings.Contains(err.Error(), "IMAGE_FORBIDDEN") {
		t.Errorf("expected IMAGE_FORBIDDEN error, got %v", err)
	}
}

type fakeDocLookup struct {
	exists  map[string]bool
	canRead map[string]bool
}

func (f *fakeDocLookup) Exists(ctx context.Context, id string) (bool, error) {
	return f.exists[id], nil
}
func (f *fakeDocLookup) UserCanRead(ctx context.Context, userID, id string) (bool, error) {
	return f.canRead[id], nil
}

func TestRules_RejectCrossDocRefMissing(t *testing.T) {
	env := parseEnvelope(t, `{
		"mddm_version":1,"template_ref":null,
		"blocks":[{
			"id":"11111111-1111-1111-1111-111111111111",
			"type":"paragraph","props":{},
			"children":[{
				"text":"see PO-117",
				"document_ref":{"target_document_id":"PO-117"}
			}]
		}]
	}`)
	rctx := RulesContext{
		Ctx:            context.Background(),
		UserID:         "user-1",
		DocumentLookup: &fakeDocLookup{exists: map[string]bool{}, canRead: map[string]bool{}},
	}
	err := EnforceLayer2(rctx, env)
	if err == nil || !strings.Contains(err.Error(), "CROSS_DOC_REF_NOT_FOUND") {
		t.Errorf("expected CROSS_DOC_REF_NOT_FOUND error, got %v", err)
	}
}

func TestRules_AllowsDataTableInEtapaRichArea(t *testing.T) {
	t.Run("allows dataTable inside richBlock within repeatableItem", func(t *testing.T) {
		env := parseEnvelope(t, `{
			"mddm_version": 1,
			"template_ref": null,
			"blocks": [
				{
					"id": "11111111-1111-1111-1111-111111111111",
					"type": "repeatable",
					"props": {"label": "Etapas", "itemPrefix": "Etapa", "locked": true, "minItems": 1, "maxItems": 10},
					"children": [
						{
							"id": "22222222-2222-2222-2222-222222222222",
							"type": "repeatableItem",
							"props": {"title": "Etapa 1"},
							"children": [
								{
									"id": "33333333-3333-3333-3333-333333333333",
									"type": "richBlock",
									"props": {"label": "Conteudo livre", "locked": false},
									"children": [
										{
											"id": "44444444-4444-4444-4444-444444444444",
											"type": "dataTable",
											"props": {
												"label": "Checklist",
												"columns": [
													{"key": "item", "label": "Item", "type": "text", "required": false}
												],
												"locked": false,
												"minRows": 0,
												"maxRows": 10
											},
											"children": []
										}
									]
								}
							]
						}
					]
				}
			]
		}`)

		if err := EnforceLayer2(RulesContext{}, env); err != nil {
			t.Fatalf("EnforceLayer2() error = %v, want nil", err)
		}
	})

	t.Run("rejects dataTable directly inside repeatableItem", func(t *testing.T) {
		env := parseEnvelope(t, `{
			"mddm_version": 1,
			"template_ref": null,
			"blocks": [
				{
					"id": "11111111-1111-1111-1111-111111111111",
					"type": "repeatable",
					"props": {"label": "Etapas", "itemPrefix": "Etapa", "locked": true, "minItems": 1, "maxItems": 10},
					"children": [
						{
							"id": "22222222-2222-2222-2222-222222222222",
							"type": "repeatableItem",
							"props": {"title": "Etapa 1"},
							"children": [
								{
									"id": "44444444-4444-4444-4444-444444444444",
									"type": "dataTable",
									"props": {
										"label": "Checklist",
										"columns": [
											{"key": "item", "label": "Item", "type": "text", "required": false}
										],
										"locked": false,
										"minRows": 0,
										"maxRows": 10
									},
									"children": []
								}
							]
						}
					]
				}
			]
		}`)

		err := EnforceLayer2(RulesContext{}, env)
		if err == nil || !strings.Contains(err.Error(), "GRAMMAR_VIOLATION") {
			t.Fatalf("EnforceLayer2() error = %v, want GRAMMAR_VIOLATION", err)
		}
	})
}
