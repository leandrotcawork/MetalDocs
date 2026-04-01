package domain

import (
	"encoding/json"
	"strings"
)

type DocumentValues = map[string]any
type RichBlock = json.RawMessage

var allowedFieldTypes = map[string]struct{}{
	"text":     {},
	"textarea": {},
	"number":   {},
	"date":     {},
	"select":   {},
	"checkbox": {},
	"table":    {},
	"rich":     {},
	"repeat":   {},
}

type DocumentTypeSchema struct {
	Sections []SectionDef `json:"sections"`
}

type SectionDef struct {
	Key    string     `json:"key"`
	Num    string     `json:"num"`
	Title  string     `json:"title"`
	Color  string     `json:"color,omitempty"`
	Fields []FieldDef `json:"fields"`
}

type FieldDef struct {
	Key        string     `json:"key"`
	Label      string     `json:"label"`
	Type       string     `json:"type"`
	Options    []string   `json:"options,omitempty"`
	Columns    []FieldDef `json:"columns,omitempty"`
	ItemFields []FieldDef `json:"itemFields,omitempty"`
}

func ValidateDocumentTypeSchema(schema DocumentTypeSchema) error {
	if len(schema.Sections) == 0 {
		return ErrDocumentSchemaInvalid
	}
	for _, section := range schema.Sections {
		if err := validateSectionDef(section); err != nil {
			return err
		}
	}
	return nil
}

func validateSectionDef(section SectionDef) error {
	if strings.TrimSpace(section.Key) == "" || strings.TrimSpace(section.Num) == "" || strings.TrimSpace(section.Title) == "" {
		return ErrDocumentSchemaInvalidSection
	}
	if len(section.Fields) == 0 {
		return ErrDocumentSchemaInvalidSection
	}
	for _, field := range section.Fields {
		if err := validateFieldDef(field); err != nil {
			return err
		}
	}
	return nil
}

func validateFieldDef(field FieldDef) error {
	if strings.TrimSpace(field.Key) == "" || strings.TrimSpace(field.Label) == "" {
		return ErrDocumentSchemaInvalidField
	}

	fieldType := strings.ToLower(strings.TrimSpace(field.Type))
	if fieldType == "" {
		return ErrDocumentSchemaInvalidField
	}
	if _, ok := allowedFieldTypes[fieldType]; !ok {
		return ErrDocumentSchemaInvalidField
	}

	switch fieldType {
	case "table":
		if len(field.Columns) == 0 {
			return ErrDocumentSchemaInvalidField
		}
		for _, column := range field.Columns {
			if err := validateFieldDef(column); err != nil {
				return err
			}
		}
	case "repeat":
		if len(field.ItemFields) == 0 {
			return ErrDocumentSchemaInvalidField
		}
		for _, itemField := range field.ItemFields {
			if err := validateFieldDef(itemField); err != nil {
				return err
			}
		}
	}

	return nil
}

func DefaultDocumentTypeDefinitions() []DocumentTypeDefinition {
	return []DocumentTypeDefinition{
		{
			Key:           "po",
			Name:          "Procedimento Operacional",
			ActiveVersion: 2,
			Schema: DocumentTypeSchema{
				Sections: []SectionDef{
					{
						Key:   "identificacao",
						Num:   "1",
						Title: "Identificação",
						Color: "#0F6E56",
						Fields: []FieldDef{
							{Key: "elaboradoPor", Label: "Elaborado por", Type: "text"},
							{Key: "aprovadoPor", Label: "Aprovado por", Type: "text"},
							{Key: "createdAt", Label: "Data de criação", Type: "date"},
							{Key: "approvedAt", Label: "Data de aprovação", Type: "date"},
						},
					},
					{
						Key:   "identificacaoProcesso",
						Num:   "2",
						Title: "Identificação do Processo",
						Color: "#0F6E56",
						Fields: []FieldDef{
							{Key: "objetivo", Label: "Objetivo", Type: "textarea"},
							{Key: "escopo", Label: "Escopo", Type: "textarea"},
							{Key: "responsavel", Label: "Cargo responsável", Type: "text"},
							{Key: "canal", Label: "Canal / Contexto", Type: "text"},
							{Key: "participantes", Label: "Participantes", Type: "textarea"},
						},
					},
					{
						Key:   "entradasSaidas",
						Num:   "3",
						Title: "Entradas e Saídas",
						Color: "#0F6E56",
						Fields: []FieldDef{
							{Key: "entradas", Label: "Entradas", Type: "textarea"},
							{Key: "saidas", Label: "Saídas", Type: "textarea"},
							{Key: "documentos", Label: "Documentos relacionados", Type: "textarea"},
							{Key: "sistemas", Label: "Sistemas utilizados", Type: "textarea"},
						},
					},
					{
						Key:   "visaoGeral",
						Num:   "4",
						Title: "Visão Geral do Processo",
						Color: "#BA7517",
						Fields: []FieldDef{
							{Key: "descricaoProcesso", Label: "Descrição do processo", Type: "textarea"},
							{Key: "fluxogramaFerramenta", Label: "Ferramenta do fluxograma", Type: "text"},
							{Key: "fluxogramaUrl", Label: "Link do fluxograma", Type: "text"},
						},
					},
					{
						Key:   "etapas",
						Num:   "5",
						Title: "Detalhamento das Etapas",
						Color: "#993C1D",
						Fields: []FieldDef{
							{
								Key:   "etapas",
								Label: "Etapas",
								Type:  "repeat",
								ItemFields: []FieldDef{
									{Key: "num", Label: "Número", Type: "text"},
									{Key: "titulo", Label: "Título", Type: "text"},
									{Key: "responsavel", Label: "Responsável", Type: "text"},
									{Key: "prazo", Label: "Prazo / SLA", Type: "text"},
									{Key: "descricao", Label: "Descrição", Type: "rich"},
									{Key: "observacao", Label: "Observações", Type: "textarea"},
									{Key: "alerta", Label: "Alertas / Desvios", Type: "textarea"},
								},
							},
						},
					},
					{
						Key:   "controle",
						Num:   "6",
						Title: "Controle e Exceções",
						Color: "#0F6E56",
						Fields: []FieldDef{
							{Key: "pontosControle", Label: "Pontos de controle", Type: "textarea"},
							{Key: "excecoes", Label: "Exceções e desvios", Type: "textarea"},
						},
					},
					{
						Key:   "kpis",
						Num:   "7",
						Title: "Indicadores de Desempenho",
						Color: "#0F6E56",
						Fields: []FieldDef{
							{
								Key:   "kpis",
								Label: "KPIs",
								Type:  "table",
								Columns: []FieldDef{
									{Key: "indicador", Label: "Indicador / KPI", Type: "text"},
									{Key: "meta", Label: "Meta", Type: "text"},
									{Key: "frequencia", Label: "Frequência", Type: "text"},
								},
							},
						},
					},
					{
						Key:   "referencias",
						Num:   "8",
						Title: "Documentos e Referências",
						Color: "#185FA5",
						Fields: []FieldDef{
							{
								Key:   "referencias",
								Label: "Referências",
								Type:  "table",
								Columns: []FieldDef{
									{Key: "codigo", Label: "Código", Type: "text"},
									{Key: "titulo", Label: "Título / Descrição", Type: "text"},
									{Key: "url", Label: "Link", Type: "text"},
								},
							},
						},
					},
					{
						Key:   "glossario",
						Num:   "9",
						Title: "Glossário",
						Color: "#185FA5",
						Fields: []FieldDef{
							{
								Key:   "glossario",
								Label: "Glossário",
								Type:  "table",
								Columns: []FieldDef{
									{Key: "termo", Label: "Termo", Type: "text"},
									{Key: "definicao", Label: "Definição", Type: "text"},
								},
							},
						},
					},
					{
						Key:   "historico",
						Num:   "10",
						Title: "Histórico de Revisões",
						Color: "#444441",
						Fields: []FieldDef{
							{
								Key:   "revisoes",
								Label: "Revisões",
								Type:  "table",
								Columns: []FieldDef{
									{Key: "versao", Label: "Versão", Type: "text"},
									{Key: "data", Label: "Data", Type: "date"},
									{Key: "descricao", Label: "O que foi alterado", Type: "text"},
									{Key: "por", Label: "Por", Type: "text"},
								},
							},
						},
					},
				},
			},
		},
		{
			Key:           "it",
			Name:          "Instrucao de Trabalho",
			ActiveVersion: 1,
			Schema: DocumentTypeSchema{
				Sections: []SectionDef{
					{Key: "contexto", Num: "1", Title: "Contexto"},
				},
			},
		},
		{
			Key:           "rg",
			Name:          "Registro",
			ActiveVersion: 1,
			Schema: DocumentTypeSchema{
				Sections: []SectionDef{
					{Key: "evento", Num: "1", Title: "Evento"},
				},
			},
		},
	}
}
