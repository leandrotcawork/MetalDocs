# ADR-0013: Version Content Storage, Files, and Full-Text Search

## Status
Proposed

## Context
Hoje `document_versions` armazena apenas `content` (texto) e `content_hash`. Com autoria estruturada e fluxo DOCX, uma versao precisa carregar:
- conteudo estruturado (JSON) no modo nativo,
- referencia ao DOCX original (upload) quando aplicavel,
- referencia ao PDF renderizado (ambos os modos),
- texto plano extraido para busca full-text.

Precisamos fazer isso mantendo:
- versionamento imutavel (append-only),
- migrations additive-first (ADR-0007),
- contratos explicitos via OpenAPI.

## Decision
- Evoluir `metaldocs.document_versions` com colunas adicionais (additive):
  - `content_source` (native|docx_upload)
  - `native_content` JSONB (conteudo estruturado)
  - `docx_storage_key` TEXT (nullable)
  - `pdf_storage_key` TEXT (nullable)
  - `text_content` TEXT (nullable)
  - `file_size_bytes` BIGINT (nullable)
  - `original_filename` TEXT (nullable)
  - `page_count` INTEGER (nullable)
  - `search_vector` TSVECTOR gerado a partir de `text_content` (portuguese) + indice GIN
- `content` (texto legado) permanece por compatibilidade no curto prazo; no MVP novo fluxo passa a preencher `text_content` como fonte preferida para busca.

## Consequences
- Leitura de versoes deve mapear os novos campos para o dominio/DTO.
- Search module deve usar `search_vector` quando presente.
- Storage keys devem seguir convencao de namespace (ex: `documents/{documentId}/versions/{n}/...`).

