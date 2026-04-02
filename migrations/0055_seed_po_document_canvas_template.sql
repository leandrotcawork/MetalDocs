INSERT INTO metaldocs.document_template_versions (
  template_key, version, profile_code, schema_version, name, definition_json
)
VALUES (
  'po-default-canvas',
  1,
  'po',
  3,
  'PO Governed Canvas v1',
  '{
    "type": "page",
    "id": "po-root",
    "children": [
      {
        "type": "section-frame",
        "id": "identificacao-processo",
        "title": "Identificacao do Processo",
        "children": [
          { "type": "label", "id": "lbl-objetivo", "text": "Objetivo" },
          { "type": "field-slot", "id": "slot-objetivo", "path": "identificacaoProcesso.objetivo", "fieldKind": "scalar" },
          { "type": "label", "id": "lbl-descricao", "text": "Descricao do processo" },
          { "type": "rich-slot", "id": "slot-descricao", "path": "visaoGeral.descricaoProcesso", "fieldKind": "rich" }
        ]
      }
    ]
  }'::jsonb
);

INSERT INTO metaldocs.document_profile_template_defaults (profile_code, template_key, template_version)
VALUES ('po', 'po-default-canvas', 1)
ON CONFLICT (profile_code) DO UPDATE
SET template_key = EXCLUDED.template_key,
    template_version = EXCLUDED.template_version,
    assigned_at = NOW();
