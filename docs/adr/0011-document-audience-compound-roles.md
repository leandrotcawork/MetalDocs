# ADR-0011: Compound Audience Roles for Restricted Access

## Status
Accepted

## Context
`audience.mode = AREAS` hoje gera duas roles simples (`dept:<code>` e `area:<code>`). O modelo de policy e avaliado por uniao (OR), portanto qualquer usuario com uma das roles ganha acesso. Para `RESTRICTED`, precisamos exigir a combinacao departamento + area (AND) sem mover regra para o frontend nem alterar o schema de policies.

## Decision
- `audience.mode = AREAS` passa a gerar uma **role composta** no formato:
  - `dept:<departmentCode>:area:<processAreaCode>`
- Para `RESTRICTED`, o backend **exige** `audience.mode = AREAS` e ambos os codigos (department + process area).
- Para `CONFIDENTIAL`, o fluxo recomendado permanece `DEPARTMENT` (acesso por departamento completo).
- Nao gerar roles simples (`dept:<code>` ou `area:<code>`) quando o modo for `AREAS`.

## Consequences
- Sem migracao de schema: apenas mudanca de regra de geracao de policy.
- Admin/ops devem atribuir roles compostas aos usuarios que podem acessar combinacoes especificas.
- UI continua enviando `audience` e pode restringir visualmente o modo para `RESTRICTED`.

## Migration Notes
Nenhuma alteracao em banco. A mudanca so afeta novos documentos RESTRICTED/AREAS criados apos o deploy.
