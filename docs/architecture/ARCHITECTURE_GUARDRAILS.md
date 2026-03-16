# Architecture Guardrails

## Target Architecture
- Estilo: modular monolith vertical slice.
- Padrao por modulo:
  - `domain/` (invariantes, entidades, value objects)
  - `application/` (casos de uso)
  - `infrastructure/` (adapters externos)
  - `delivery/http/` (handlers e DTO de entrada/saida)

## Request Flow (mandatory)
`delivery -> application -> domain -> infrastructure`

Rules:
- `delivery` nao contem regra de negocio.
- `application` orquestra casos de uso e transacao.
- `domain` define invariantes e nao depende de IO.
- `infrastructure` implementa ports.

## Module Boundaries
Modules in v1:
- `documents`
- `versions`
- `workflow`
- `iam`
- `audit`
- `search`

Allowed cross-module interaction:
- Via interfaces publicas do modulo.
- Via eventos internos padronizados.

Forbidden interaction:
- Import direto de internals privados de outro modulo.
- Acesso direto a tabela de outro modulo sem port/interface.

## Architectural Contracts (immutable)
- OpenAPI e source of truth para API publica.
- Versao de documento e imutavel.
- Auditoria e append-only.
- Autorizacao sempre no backend.
- Outbox + idempotency obrigatorios para publicacao de eventos.

## Event Reliability
- Toda mutacao relevante publica evento de dominio.
- Publicacao usa tabela outbox transacional.
- Consumidores devem ser idempotentes por `idempotency_key`.

## Data and Migration Policy
- Migrations additive-first.
- Migracao destrutiva requer ADR + janela de manutencao.
- Rollback documentado antes do merge.

## Security Guardrails
- Segredos so por env var/secret manager.
- Tokens de auth com expiracao curta.
- Nenhum endpoint mutavel sem verificacao de permissao.

## Observability Guardrails
- Logs estruturados com `trace_id`.
- Metricas por endpoint e por caso de uso.
- Tracing habilitado no caminho critico.

## Anti-patterns (strictly prohibited)
- Business rule no frontend ou no handler.
- Shared utils virando "god package".
- Dependencia circular entre modulos.
- Ajuste direto em producao sem runbook.
