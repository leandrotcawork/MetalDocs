ALTER TABLE metaldocs.document_profile_schema_versions
  ADD COLUMN IF NOT EXISTS content_schema_json JSONB NOT NULL DEFAULT '{}'::jsonb;

UPDATE metaldocs.document_profile_schema_versions
SET content_schema_json = $$
{
  "profile": "po",
  "sections": [
    {
      "key": "identification",
      "title": "Identificacao",
      "description": "Objetivo, escopo e responsavel.",
      "fields": [
        { "key": "objetivo", "label": "Objetivo", "type": "textarea", "required": true },
        { "key": "escopo", "label": "Escopo", "type": "textarea" },
        { "key": "responsavel", "label": "Responsavel", "type": "text" },
        { "key": "participantes", "label": "Participantes", "type": "array", "itemType": "text" },
        { "key": "canal", "label": "Canal", "type": "select", "options": ["Balcao", "WhatsApp", "Externo", "E-commerce"] }
      ]
    },
    {
      "key": "io",
      "title": "Entradas e saidas",
      "description": "Entradas, saidas e sistemas.",
      "fields": [
        { "key": "entradas", "label": "Entradas", "type": "textarea" },
        { "key": "saidas", "label": "Saidas", "type": "textarea" },
        { "key": "documentos", "label": "Documentos", "type": "array", "itemType": "text" },
        { "key": "sistemas", "label": "Sistemas", "type": "array", "itemType": "text" }
      ]
    },
    {
      "key": "process",
      "title": "Processo",
      "description": "Etapas e pontos de controle.",
      "fields": [
        {
          "key": "etapas",
          "label": "Etapas",
          "type": "table",
          "columns": [
            { "key": "num", "label": "#", "type": "number" },
            { "key": "etapa", "label": "Etapa", "type": "text" },
            { "key": "responsavel", "label": "Responsavel", "type": "text" },
            { "key": "prazo", "label": "Prazo", "type": "text" },
            { "key": "observacao", "label": "Observacao", "type": "text" }
          ]
        },
        { "key": "pontos_controle", "label": "Pontos de controle", "type": "textarea" },
        { "key": "excecoes", "label": "Excecoes", "type": "textarea" }
      ]
    },
    {
      "key": "kpis",
      "title": "Indicadores",
      "description": "Indicadores e metas.",
      "fields": [
        {
          "key": "kpis",
          "label": "Indicadores",
          "type": "table",
          "columns": [
            { "key": "indicador", "label": "Indicador", "type": "text" },
            { "key": "meta", "label": "Meta", "type": "text" },
            { "key": "frequencia", "label": "Frequencia", "type": "select", "options": ["Diario", "Semanal", "Mensal"] }
          ]
        }
      ]
    }
  ]
}
$$::jsonb
WHERE profile_code = 'po' AND version = 1;

UPDATE metaldocs.document_profile_schema_versions
SET content_schema_json = $$
{
  "profile": "it",
  "sections": [
    {
      "key": "context",
      "title": "Contexto",
      "description": "Quando e como executar.",
      "fields": [
        { "key": "cargo_executor", "label": "Cargo executor", "type": "text" },
        { "key": "quando_executar", "label": "Quando executar", "type": "textarea" },
        { "key": "tempo_estimado", "label": "Tempo estimado (min)", "type": "number" },
        { "key": "materiais", "label": "Materiais", "type": "array", "itemType": "text" },
        { "key": "resultado_esperado", "label": "Resultado esperado", "type": "textarea" }
      ]
    },
    {
      "key": "steps",
      "title": "Passos",
      "description": "Sequencia operacional.",
      "fields": [
        {
          "key": "passos",
          "label": "Passos",
          "type": "table",
          "columns": [
            { "key": "num", "label": "#", "type": "number" },
            { "key": "acao", "label": "Acao", "type": "text" },
            { "key": "alerta", "label": "Alerta", "type": "text" }
          ]
        },
        { "key": "pontos_atencao", "label": "Pontos de atencao", "type": "textarea" },
        { "key": "se_der_errado", "label": "Se der errado", "type": "textarea" }
      ]
    },
    {
      "key": "verification",
      "title": "Verificacao",
      "description": "Checklist e registro.",
      "fields": [
        { "key": "checklist", "label": "Checklist", "type": "array", "itemType": "text" },
        { "key": "registro_gerado", "label": "Registro gerado", "type": "text" }
      ]
    },
    {
      "key": "media",
      "title": "Midia",
      "description": "Imagens, video e anexos.",
      "fields": [
        { "key": "imagens", "label": "Imagens", "type": "array", "itemType": "text" },
        { "key": "video", "label": "Video", "type": "text" },
        { "key": "anexos", "label": "Anexos", "type": "array", "itemType": "text" }
      ]
    }
  ]
}
$$::jsonb
WHERE profile_code = 'it' AND version = 1;

UPDATE metaldocs.document_profile_schema_versions
SET content_schema_json = $$
{
  "profile": "rg",
  "sections": [
    {
      "key": "event",
      "title": "Evento",
      "description": "Dados basicos do registro.",
      "fields": [
        { "key": "canal", "label": "Canal", "type": "select", "options": ["Balcao", "WhatsApp", "E-commerce"] }
      ]
    },
    {
      "key": "content",
      "title": "Conteudo",
      "description": "Informacoes do registro.",
      "fields": [
        { "key": "observacoes", "label": "Observacoes", "type": "textarea" }
      ]
    },
    {
      "key": "closure",
      "title": "Encerramento",
      "description": "Status e encerramento.",
      "fields": [
        { "key": "status", "label": "Status", "type": "select", "options": ["aberto", "concluido", "gerou_pa"] },
        { "key": "pa_vinculado", "label": "PA vinculado", "type": "text" },
        { "key": "data_encerramento", "label": "Data de encerramento", "type": "text" }
      ]
    }
  ]
}
$$::jsonb
WHERE profile_code = 'rg' AND version = 1;
