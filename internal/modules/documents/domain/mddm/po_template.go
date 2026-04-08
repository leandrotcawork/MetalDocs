package mddm

// POTemplateMDDM returns the canonical PO template as an MDDM envelope (Go map form).
// Block IDs are stable; they become template_block_ids on instantiation.
func POTemplateMDDM() map[string]any {
	return map[string]any{
		"mddm_version": 1,
		"template_ref": nil,
		"blocks": []map[string]any{
			sectionBlock("a0000001-0000-0000-0000-000000000000", "Identificação do Processo", []map[string]any{
				fieldGroupBlock("a0000002-0000-0000-0000-000000000000", 1, []map[string]any{
					fieldBlock("a0000003-0000-0000-0000-000000000000", "Objetivo", "multiParagraph"),
					fieldBlock("a0000004-0000-0000-0000-000000000000", "Escopo", "multiParagraph"),
					fieldBlock("a0000005-0000-0000-0000-000000000000", "Cargo responsável", "inline"),
					fieldBlock("a0000006-0000-0000-0000-000000000000", "Canal / Contexto", "inline"),
					fieldBlock("a0000007-0000-0000-0000-000000000000", "Participantes", "multiParagraph"),
				}),
			}),
			sectionBlock("a0000010-0000-0000-0000-000000000000", "Entradas e Saídas", []map[string]any{
				fieldGroupBlock("a0000011-0000-0000-0000-000000000000", 2, []map[string]any{
					fieldBlock("a0000012-0000-0000-0000-000000000000", "Entradas", "multiParagraph"),
					fieldBlock("a0000013-0000-0000-0000-000000000000", "Saídas", "multiParagraph"),
					fieldBlock("a0000014-0000-0000-0000-000000000000", "Documentos relacionados", "multiParagraph"),
					fieldBlock("a0000015-0000-0000-0000-000000000000", "Sistemas utilizados", "multiParagraph"),
				}),
			}),
			sectionBlock("a0000020-0000-0000-0000-000000000000", "Visão Geral do Processo", []map[string]any{
				richBlockBlock("a0000021-0000-0000-0000-000000000000", "Descrição do processo"),
				richBlockBlock("a0000022-0000-0000-0000-000000000000", "Diagrama"),
			}),
			sectionBlock("a0000030-0000-0000-0000-000000000000", "Detalhamento das Etapas", []map[string]any{
				repeatableBlock("a0000031-0000-0000-0000-000000000000", "Etapas", "Etapa", 1, 100),
			}),
			sectionBlock("a0000040-0000-0000-0000-000000000000", "Indicadores de Desempenho", []map[string]any{
				dataTableBlock("a0000041-0000-0000-0000-000000000000", "KPIs", []map[string]any{
					{"key": "indicator", "label": "Indicador / KPI", "type": "text", "required": false},
					{"key": "target", "label": "Meta", "type": "text", "required": false},
					{"key": "frequency", "label": "Frequência", "type": "text", "required": false},
				}),
			}),
		},
	}
}

func sectionBlock(id, title string, children []map[string]any) map[string]any {
	return map[string]any{
		"id":                id,
		"template_block_id": id,
		"type":              "section",
		"props": map[string]any{
			"title":  title,
			"color":  "#6b1f2a",
			"locked": true,
		},
		"children": toAnySlice(children),
	}
}

func fieldGroupBlock(id string, columns int, children []map[string]any) map[string]any {
	return map[string]any{
		"id":                id,
		"template_block_id": id,
		"type":              "fieldGroup",
		"props": map[string]any{
			"columns": columns,
			"locked":  true,
		},
		"children": toAnySlice(children),
	}
}

func fieldBlock(id, label, valueMode string) map[string]any {
	return map[string]any{
		"id":                id,
		"template_block_id": id,
		"type":              "field",
		"props": map[string]any{
			"label":     label,
			"valueMode": valueMode,
			"locked":    true,
		},
		"children": []any{},
	}
}

func repeatableBlock(id, label, itemPrefix string, minItems, maxItems int) map[string]any {
	return map[string]any{
		"id":                id,
		"template_block_id": id,
		"type":              "repeatable",
		"props": map[string]any{
			"label":      label,
			"itemPrefix": itemPrefix,
			"locked":     true,
			"minItems":   minItems,
			"maxItems":   maxItems,
		},
		"children": []any{},
	}
}

func dataTableBlock(id, label string, columns []map[string]any) map[string]any {
	return map[string]any{
		"id":                id,
		"template_block_id": id,
		"type":              "dataTable",
		"props": map[string]any{
			"label":   label,
			"columns": toAnySlice(columns),
			"locked":  true,
			"minRows": 0,
			"maxRows": 500,
		},
		"children": []any{},
	}
}

func richBlockBlock(id, label string) map[string]any {
	return map[string]any{
		"id":                id,
		"template_block_id": id,
		"type":              "richBlock",
		"props": map[string]any{
			"label":  label,
			"locked": true,
		},
		"children": []any{},
	}
}

func toAnySlice(in []map[string]any) []any {
	out := make([]any, len(in))
	for i, v := range in {
		out[i] = v
	}
	return out
}
