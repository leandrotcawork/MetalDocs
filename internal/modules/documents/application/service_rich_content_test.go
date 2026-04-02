package application

import (
	"errors"
	"testing"

	"metaldocs/internal/modules/documents/domain"
)

func TestValidateContentSchemaRequiresRichEnvelope(t *testing.T) {
	schema := map[string]any{
		"sections": []any{
			map[string]any{
				"key": "operacao",
				"fields": []any{
					map[string]any{
						"key":  "descricao",
						"type": "rich",
					},
				},
			},
		},
	}

	valid := map[string]any{
		"operacao": map[string]any{
			"descricao": richEnvelopeFixture(map[string]any{
				"type":    "doc",
				"content": []any{},
			}),
		},
	}
	if err := validateContentSchema(schema, valid); err != nil {
		t.Fatalf("validateContentSchema(valid) = %v, want nil", err)
	}

	invalid := map[string]any{
		"operacao": map[string]any{
			"descricao": "<p>Legacy HTML</p>",
		},
	}
	err := validateContentSchema(schema, invalid)
	if !errors.Is(err, domain.ErrInvalidNativeContent) {
		t.Fatalf("validateContentSchema(invalid) = %v, want ErrInvalidNativeContent", err)
	}
}

func TestProjectDocumentValuesForDocgenConvertsRichEnvelopes(t *testing.T) {
	svc := &Service{}
	schema := map[string]any{
		"sections": []any{
			map[string]any{
				"key": "operacao",
				"fields": []any{
					map[string]any{
						"key":  "descricao",
						"type": "rich",
					},
					map[string]any{
						"key":  "etapas",
						"type": "repeat",
						"itemFields": []any{
							map[string]any{
								"key":  "observacoes",
								"type": "rich",
							},
						},
					},
				},
			},
		},
	}

	values := map[string]any{
		"operacao": map[string]any{
			"descricao": richEnvelopeFixture(map[string]any{
				"type": "doc",
				"content": []any{
					map[string]any{
						"type": "paragraph",
						"content": []any{
							map[string]any{
								"type": "text",
								"text": "Teste rich",
								"marks": []any{
									map[string]any{"type": "bold"},
								},
							},
						},
					},
					map[string]any{
						"type": "bulletList",
						"content": []any{
							map[string]any{
								"type": "listItem",
								"content": []any{
									map[string]any{
										"type": "paragraph",
										"content": []any{
											map[string]any{
												"type": "text",
												"text": "Item 1",
											},
										},
									},
								},
							},
						},
					},
				},
			}),
			"etapas": []any{
				map[string]any{
					"titulo": "Preparar",
					"observacoes": richEnvelopeFixture(map[string]any{
						"type": "doc",
						"content": []any{
							map[string]any{
								"type": "paragraph",
								"content": []any{
									map[string]any{
										"type": "text",
										"text": "Observacao inicial",
									},
								},
							},
						},
					}),
				},
			},
		},
	}

	projected, err := svc.projectDocumentValuesForDocgen(schema, values)
	if err != nil {
		t.Fatalf("projectDocumentValuesForDocgen() = %v", err)
	}

	operacao, ok := projected["operacao"].(map[string]any)
	if !ok {
		t.Fatalf("projected operacao = %T, want map[string]any", projected["operacao"])
	}

	descricao, ok := operacao["descricao"].([]projectedRichBlock)
	if !ok {
		t.Fatalf("projected descricao = %T, want []projectedRichBlock", operacao["descricao"])
	}
	if len(descricao) != 2 {
		t.Fatalf("projected descricao len = %d, want 2", len(descricao))
	}
	if descricao[0].Type != "text" {
		t.Fatalf("projected descricao[0].Type = %q, want text", descricao[0].Type)
	}
	if len(descricao[0].Runs) != 1 || !descricao[0].Runs[0].Bold || descricao[0].Runs[0].Text != "Teste rich" {
		t.Fatalf("projected descricao[0].Runs = %+v, want bold text run", descricao[0].Runs)
	}
	if descricao[1].Type != "list" || len(descricao[1].Items) != 1 || descricao[1].Items[0] != "Item 1" {
		t.Fatalf("projected descricao[1] = %+v, want list with Item 1", descricao[1])
	}

	etapas, ok := operacao["etapas"].([]any)
	if !ok || len(etapas) != 1 {
		t.Fatalf("projected etapas = %#v, want single item slice", operacao["etapas"])
	}
	item, ok := etapas[0].(map[string]any)
	if !ok {
		t.Fatalf("projected etapas[0] = %T, want map[string]any", etapas[0])
	}
	observacoes, ok := item["observacoes"].([]projectedRichBlock)
	if !ok {
		t.Fatalf("projected observacoes = %T, want []projectedRichBlock", item["observacoes"])
	}
	if len(observacoes) != 1 || observacoes[0].Type != "text" {
		t.Fatalf("projected observacoes = %+v, want single text block", observacoes)
	}

	original := values["operacao"].(map[string]any)["descricao"].(map[string]any)
	if original["format"] != domain.RichEnvelopeFormatTipTap {
		t.Fatalf("original rich envelope mutated: %+v", original)
	}
}

func richEnvelopeFixture(content map[string]any) map[string]any {
	return map[string]any{
		"format":  domain.RichEnvelopeFormatTipTap,
		"version": domain.RichEnvelopeVersionV1,
		"content": content,
	}
}
