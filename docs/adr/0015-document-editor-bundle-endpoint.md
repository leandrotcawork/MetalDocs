# ADR-0015: Document Editor Bundle Endpoint

## Status
Accepted

## Context
Abrir o editor hoje exige multiplas chamadas sequenciais (documento, versoes, schema, governanca, presenca e lock). Isso aumenta a latencia percebida e cria loading excessivo ao transitar do create para o editor.

## Decision
Adicionar um endpoint agregado:
`GET /documents/{documentId}/editor-bundle`

O payload inclui:
- `document` (dados principais)
- `versions` (lista de versoes)
- `schema` (schema ativo do profile)
- `governance` (governanca ativa do profile)
- `presence` (presencas ativas)
- `editLock` (lock atual, se existir)

As demais informacoes do editor (approvals, audit, attachments, policies, diff) permanecem fora do bundle.

## Consequences
- Positive:
  - Reduz round-trips no primeiro render do editor.
  - Menos bloqueio de UI ao navegar para a tela de edicao.
  - Mantem compatibilidade com endpoints existentes.
- Negative:
  - Payload maior e necessidade de manter contrato agregado sincronizado.

## Alternatives Considered
- Option A: Manter chamadas individuais e apenas cache no frontend.
- Option B: Carregar tudo no `loadWorkspace` antes de abrir o editor.
