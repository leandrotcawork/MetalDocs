# ADR-0009: Document Audience vs Classification

## Status
Accepted

## Context
`classification` existe hoje no documento como nivel de sensibilidade, mas nao controla acesso diretamente. O acesso real e decidido por `access_policies` (capability-based). Precisamos de um modelo claro e escalavel para definir "quem pode ver" sem empurrar regra para o frontend, mantendo compatibilidade com o contrato atual.

## Decision
- `classification` permanece como label de sensibilidade (PUBLIC, INTERNAL, CONFIDENTIAL, RESTRICTED) e **nao** eh ACL.
- Um novo bloco opcional `audience` define a politica de visibilidade e **gera** policies no backend.
- O backend aplica defaults seguros quando `audience` nao e informado:
  - PUBLIC/INTERNAL: sem policies extras (comportamento atual).
  - CONFIDENTIAL: default `audience.mode = DEPARTMENT` usando o `department` do documento.
  - RESTRICTED: default `audience.mode = DEPARTMENT` (mais fechado) e permite `EXPLICIT` quando necessario.

## Audience v1 (RBAC-first)
Campos no contrato:
- `mode`: INTERNAL | DEPARTMENT | AREAS | EXPLICIT
- `departmentCodes`: lista de departamentos (DEPARTMENT/AREAS)
- `processAreaCodes`: lista de areas (AREAS)
- `roleCodes`: roles explicitas (EXPLICIT)
- `userIds`: usuarios explicitos (EXPLICIT)

Mapeamento para policies (v1):
- Sempre garantir `owner` com `document.view` e `document.edit`.
- Role `admin` sempre com todas capabilities.
- DEPARTMENT: allow `document.view` para roles `dept:<code>`.
- AREAS: allow `document.view` para roles `dept:<code>` e `area:<code>`.
- EXPLICIT: allow `document.view` para `roleCodes` e `userIds`.

## Consequences
- OpenAPI ganha `audience` em `CreateDocumentRequest`.
- Backend passa a gerar policies por documento quando `audience` exigir.
- Frontend exibe seletor de audiencia apenas para CONFIDENTIAL/RESTRICTED, enviando o bloco para o backend.

## Migration Notes
Sem migracao de schema neste passo. O enforcement vem no passo seguinte com uso da tabela `document_access_policies` existente.
