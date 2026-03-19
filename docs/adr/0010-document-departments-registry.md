# ADR-0010: Document Departments Registry

## Status
Proposed

## Context
`department` e hoje um campo livre em documentos. Para garantir consistencia, padronizar UX e sustentar regras de acesso por departamento, precisamos de um registry canonico similar ao de `process_areas` e `subjects`.

## Decision
- Introduzir `document_departments` como registry canonico.
- `department` em documentos continua como string, sem FK no v1, para nao quebrar dados legados.
- API expor `document-departments` para listagem e administracao (admin-only para write).

## Consequences
- Nova migration cria a tabela e permissao de leitura para `metaldocs_app`.
- Seeds incluem lista inicial Metal Nobre.
- Frontend passa a usar dropdown baseado no registry.
