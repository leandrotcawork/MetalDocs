package mddm

// POTemplateMDDM returns the canonical PO template as an MDDM envelope (Go map form).
// Block IDs are stable; they become template_block_ids on instantiation.
func POTemplateMDDM() map[string]any {
	return map[string]any{
		"mddm_version": 1,
		"template_ref": nil,
		"blocks": []map[string]any{
			sectionBlock("a0000001-0000-0000-0000-000000000001", "Identificação", true, false, []map[string]any{
				fieldGroupBlock("a0000002-0000-0000-0000-000000000002", 2, []map[string]any{
					fieldBlock("a0000003-0000-0000-0000-000000000003", "Elaborado por", "inline"),
					fieldBlock("a0000004-0000-0000-0000-000000000004", "Aprovado por", "inline"),
					fieldBlock("a0000005-0000-0000-0000-000000000005", "Data de criação", "inline"),
					fieldBlock("a0000006-0000-0000-0000-000000000006", "Data de aprovação", "inline"),
				}),
			}),
			sectionBlock("a0000010-0000-0000-0000-000000000010", "Identificação do Processo", true, false, []map[string]any{
				fieldGroupBlock("a0000011-0000-0000-0000-000000000011", 1, []map[string]any{
					fieldBlock("a0000012-0000-0000-0000-000000000012", "Objetivo", "multiParagraph"),
					fieldBlock("a0000013-0000-0000-0000-000000000013", "Escopo", "multiParagraph"),
					fieldBlock("a0000014-0000-0000-0000-000000000014", "Cargo responsável", "inline"),
					fieldBlock("a0000015-0000-0000-0000-000000000015", "Canal / Contexto", "inline"),
					fieldBlock("a0000016-0000-0000-0000-000000000016", "Participantes", "multiParagraph"),
				}),
			}),
			sectionBlock("a0000020-0000-0000-0000-000000000020", "Entradas e Saídas", true, false, []map[string]any{
				fieldGroupBlock("a0000021-0000-0000-0000-000000000021", 2, []map[string]any{
					stackFieldBlock("a0000022-0000-0000-0000-000000000022", "Entradas", "multiParagraph"),
					stackFieldBlock("a0000023-0000-0000-0000-000000000023", "Saídas", "multiParagraph"),
					stackFieldBlock("a0000024-0000-0000-0000-000000000024", "Documentos relacionados", "multiParagraph"),
					stackFieldBlock("a0000025-0000-0000-0000-000000000025", "Sistemas utilizados", "multiParagraph"),
				}),
			}),
			sectionBlock("a0000030-0000-0000-0000-000000000030", "Visão Geral do Processo", true, false, []map[string]any{
				richBlockBlock("a0000031-0000-0000-0000-000000000031", "Descrição do processo", true, []map[string]any{
					paragraphBlock("a0000032-0000-0000-0000-000000000032", "Descreva o fluxo e adicione imagens do fluxograma quando fizer sentido."),
					bulletListItemBlock("a0000033-0000-0000-0000-000000000033", "Inclua os principais pontos de decisão do fluxo"),
				}),
				richBlockBlock("a0000034-0000-0000-0000-000000000034", "Diagrama", true, []map[string]any{
					paragraphBlock("a0000035-0000-0000-0000-000000000035", "Adicione aqui o diagrama do processo quando estiver disponivel."),
				}),
			}),
			sectionBlock("a0000040-0000-0000-0000-000000000040", "Detalhamento das Etapas", true, false, []map[string]any{
				repeatableBlock("a0000041-0000-0000-0000-000000000041", "Etapas", "Etapa", 1, 100, false, []map[string]any{
					repeatableItemBlock("a0000042-0000-0000-0000-000000000042", "Etapa 1", []map[string]any{
						richBlockBlock("a0000043-0000-0000-0000-000000000043", "Detalhes da etapa", true, []map[string]any{
							paragraphBlock("a0000044-0000-0000-0000-000000000044", "Descreva o objetivo e o contexto desta etapa."),
							bulletListItemBlock("a0000045-0000-0000-0000-000000000045", "Liste as entradas necessarias para iniciar a etapa"),
							numberedListItemBlock("a0000046-0000-0000-0000-000000000046", "Descreva a sequencia principal de execucao"),
							dataTableBlock(
								"a0000047-0000-0000-0000-000000000047",
								"Entradas e saidas da etapa",
								[]string{"campo", "descricao"},
								[][]string{},
							),
						}),
					}),
				}),
			}),
			sectionBlock("a0000055-0000-0000-0000-000000000055", "Controle e Exceções", true, false, []map[string]any{
				fieldGroupBlock("a0000056-0000-0000-0000-000000000056", 2, []map[string]any{
					fieldBlock("a0000057-0000-0000-0000-000000000057", "Pontos de controle", "multiParagraph"),
					fieldBlock("a0000058-0000-0000-0000-000000000058", "Exceções e desvios", "multiParagraph"),
				}),
			}),
			sectionBlock("a0000060-0000-0000-0000-000000000060", "Indicadores de Desempenho", true, true, []map[string]any{
				repeatableBlock("a0000061-0000-0000-0000-000000000061", "KPIs", "KPI", 0, 100, false, []map[string]any{}),
			}),
			sectionBlock("a0000070-0000-0000-0000-000000000070", "Documentos e Referências", true, true, []map[string]any{
				repeatableBlock("a0000071-0000-0000-0000-000000000071", "Referências", "Referência", 0, 100, false, []map[string]any{}),
			}),
			sectionBlock("a0000080-0000-0000-0000-000000000080", "Glossário", true, true, []map[string]any{
				repeatableBlock("a0000081-0000-0000-0000-000000000081", "Glossário", "Termo", 0, 100, false, []map[string]any{}),
			}),
			sectionBlock("a0000090-0000-0000-0000-000000000090", "Histórico de Revisões", true, true, []map[string]any{
				repeatableBlock("a0000091-0000-0000-0000-000000000091", "Revisões", "Revisão", 0, 100, false, []map[string]any{}),
			}),
		},
	}
}

func sectionBlock(id, title string, locked, optional bool, children []map[string]any) map[string]any {
	props := map[string]any{
		"title":  title,
		"color":  "#6b1f2a",
		"locked": locked,
	}
	if optional {
		props["optional"] = true
	}
	return map[string]any{
		"id":                id,
		"template_block_id": id,
		"type":              "section",
		"props":             props,
		"children":          toAnySlice(children),
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

func stackFieldBlock(id, label, valueMode string) map[string]any {
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

func repeatableBlock(id, label, itemPrefix string, minItems, maxItems int, locked bool, children []map[string]any) map[string]any {
	return map[string]any{
		"id":                id,
		"template_block_id": id,
		"type":              "repeatable",
		"props": map[string]any{
			"label":      label,
			"itemPrefix": itemPrefix,
			"locked":     locked,
			"minItems":   minItems,
			"maxItems":   maxItems,
		},
		"children": toAnySlice(children),
	}
}

func repeatableItemBlock(id, title string, children []map[string]any) map[string]any {
	return map[string]any{
		"id":   id,
		"type": "repeatableItem",
		"props": map[string]any{
			"title": title,
		},
		"children": toAnySlice(children),
	}
}

func richBlockBlock(id, label string, locked bool, children []map[string]any) map[string]any {
	return map[string]any{
		"id":                id,
		"template_block_id": id,
		"type":              "richBlock",
		"props": map[string]any{
			"label":  label,
			"locked": locked,
		},
		"children": toAnySlice(children),
	}
}

func contentRichBlockBlock(id, label string, locked bool, children []map[string]any) map[string]any {
	return map[string]any{
		"id":   id,
		"type": "richBlock",
		"props": map[string]any{
			"label":  label,
			"locked": locked,
		},
		"children": toAnySlice(children),
	}
}

func paragraphBlock(id, text string) map[string]any {
	return map[string]any{
		"id":    id,
		"type":  "paragraph",
		"props": map[string]any{},
		"children": []any{
			textRun(text),
		},
	}
}

func bulletListItemBlock(id, text string) map[string]any {
	return map[string]any{
		"id":   id,
		"type": "bulletListItem",
		"props": map[string]any{
			"level": 0,
		},
		"children": []any{
			textRun(text),
		},
	}
}

func numberedListItemBlock(id, text string) map[string]any {
	return map[string]any{
		"id":   id,
		"type": "numberedListItem",
		"props": map[string]any{
			"level": 0,
		},
		"children": []any{
			textRun(text),
		},
	}
}

func imageBlock(id, src, alt, caption string) map[string]any {
	return map[string]any{
		"id":   id,
		"type": "image",
		"props": map[string]any{
			"src":     src,
			"alt":     alt,
			"caption": caption,
		},
	}
}

// dataTableBlock creates a schema-compatible DataTable block.
// columnLabels: column keys/labels (e.g., []string{"item", "quantidade"})
// rowTexts: data rows, each row is a slice of cell texts (one per column)
func dataTableBlock(id, label string, columnLabels []string, rowTexts [][]string) map[string]any {
	columns := make([]any, 0, len(columnLabels))
	for _, col := range columnLabels {
		columns = append(columns, map[string]any{
			"key":      col,
			"label":    col,
			"type":     "text",
			"required": false,
		})
	}

	rows := make([]any, 0, len(rowTexts))
	for i, row := range rowTexts {
		cells := make([]any, 0, len(columnLabels))
		for j, col := range columnLabels {
			text := ""
			if j < len(row) {
				text = row[j]
			}
			cells = append(cells, map[string]any{
				"id":   id + "-cell-" + string(rune('a'+i)) + string(rune('a'+j)),
				"type": "dataTableCell",
				"props": map[string]any{
					"columnKey": col,
				},
				"children": []any{textRun(text)},
			})
		}
		rows = append(rows, map[string]any{
			"id":       id + "-row-" + string(rune('a'+i)),
			"type":     "dataTableRow",
			"props":    map[string]any{},
			"children": cells,
		})
	}

	return map[string]any{
		"id":   id,
		"type": "dataTable",
		"props": map[string]any{
			"label":   label,
			"columns": columns,
			"locked":  false,
			"minRows": 0,
			"maxRows": 100,
		},
		"children": rows,
	}
}

func textRun(text string) map[string]any {
	return map[string]any{
		"text": text,
	}
}

func toAnySlice(in []map[string]any) []any {
	out := make([]any, len(in))
	for i, v := range in {
		out[i] = v
	}
	return out
}
