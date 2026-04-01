INSERT INTO metaldocs.document_type_schema_versions (type_key, version, schema_json, governance_json)
VALUES (
  'po',
  2,
  $$
  {
    "sections": [
      {
        "key": "identificacao",
        "num": "1",
        "title": "Identificação",
        "color": "#0F6E56",
        "fields": [
          { "key": "elaboradoPor", "label": "Elaborado por", "type": "text" },
          { "key": "aprovadoPor", "label": "Aprovado por", "type": "text" },
          { "key": "createdAt", "label": "Data de criação", "type": "date" },
          { "key": "approvedAt", "label": "Data de aprovação", "type": "date" }
        ]
      },
      {
        "key": "identificacaoProcesso",
        "num": "2",
        "title": "Identificação do Processo",
        "color": "#0F6E56",
        "fields": [
          { "key": "objetivo", "label": "Objetivo", "type": "textarea" },
          { "key": "escopo", "label": "Escopo", "type": "textarea" },
          { "key": "responsavel", "label": "Cargo responsável", "type": "text" },
          { "key": "canal", "label": "Canal / Contexto", "type": "text" },
          { "key": "participantes", "label": "Participantes", "type": "textarea" }
        ]
      },
      {
        "key": "entradasSaidas",
        "num": "3",
        "title": "Entradas e Saídas",
        "color": "#0F6E56",
        "fields": [
          { "key": "entradas", "label": "Entradas", "type": "textarea" },
          { "key": "saidas", "label": "Saídas", "type": "textarea" },
          { "key": "documentos", "label": "Documentos relacionados", "type": "textarea" },
          { "key": "sistemas", "label": "Sistemas utilizados", "type": "textarea" }
        ]
      },
      {
        "key": "visaoGeral",
        "num": "4",
        "title": "Visão Geral do Processo",
        "color": "#BA7517",
        "fields": [
          { "key": "descricaoProcesso", "label": "Descrição do processo", "type": "textarea" },
          { "key": "fluxogramaFerramenta", "label": "Ferramenta do fluxograma", "type": "text" },
          { "key": "fluxogramaUrl", "label": "Link do fluxograma", "type": "text" }
        ]
      },
      {
        "key": "etapas",
        "num": "5",
        "title": "Detalhamento das Etapas",
        "color": "#993C1D",
        "fields": [
          {
            "key": "etapas",
            "label": "Etapas",
            "type": "repeat",
            "itemFields": [
              { "key": "num", "label": "Número", "type": "text" },
              { "key": "titulo", "label": "Título", "type": "text" },
              { "key": "responsavel", "label": "Responsável", "type": "text" },
              { "key": "prazo", "label": "Prazo / SLA", "type": "text" },
              { "key": "descricao", "label": "Descrição", "type": "rich" },
              { "key": "observacao", "label": "Observações", "type": "textarea" },
              { "key": "alerta", "label": "Alertas / Desvios", "type": "textarea" }
            ]
          }
        ]
      },
      {
        "key": "controle",
        "num": "6",
        "title": "Controle e Exceções",
        "color": "#0F6E56",
        "fields": [
          { "key": "pontosControle", "label": "Pontos de controle", "type": "textarea" },
          { "key": "excecoes", "label": "Exceções e desvios", "type": "textarea" }
        ]
      },
      {
        "key": "kpis",
        "num": "7",
        "title": "Indicadores de Desempenho",
        "color": "#0F6E56",
        "fields": [
          {
            "key": "kpis",
            "label": "KPIs",
            "type": "table",
            "columns": [
              { "key": "indicador", "label": "Indicador / KPI", "type": "text" },
              { "key": "meta", "label": "Meta", "type": "text" },
              { "key": "frequencia", "label": "Frequência", "type": "text" }
            ]
          }
        ]
      },
      {
        "key": "referencias",
        "num": "8",
        "title": "Documentos e Referências",
        "color": "#185FA5",
        "fields": [
          {
            "key": "referencias",
            "label": "Referências",
            "type": "table",
            "columns": [
              { "key": "codigo", "label": "Código", "type": "text" },
              { "key": "titulo", "label": "Título / Descrição", "type": "text" },
              { "key": "url", "label": "Link", "type": "text" }
            ]
          }
        ]
      },
      {
        "key": "glossario",
        "num": "9",
        "title": "Glossário",
        "color": "#185FA5",
        "fields": [
          {
            "key": "glossario",
            "label": "Glossário",
            "type": "table",
            "columns": [
              { "key": "termo", "label": "Termo", "type": "text" },
              { "key": "definicao", "label": "Definição", "type": "text" }
            ]
          }
        ]
      },
      {
        "key": "historico",
        "num": "10",
        "title": "Histórico de Revisões",
        "color": "#444441",
        "fields": [
          {
            "key": "revisoes",
            "label": "Revisões",
            "type": "table",
            "columns": [
              { "key": "versao", "label": "Versão", "type": "text" },
              { "key": "data", "label": "Data", "type": "date" },
              { "key": "descricao", "label": "O que foi alterado", "type": "text" },
              { "key": "por", "label": "Por", "type": "text" }
            ]
          }
        ]
      }
    ]
  }
  $$::jsonb,
  '{}'::jsonb
)
ON CONFLICT (type_key, version) DO UPDATE
SET schema_json = EXCLUDED.schema_json;

UPDATE metaldocs.document_types
SET active_version = 2
WHERE type_key = 'po';
