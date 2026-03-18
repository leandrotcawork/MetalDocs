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
- Storage oficial de runtime real e object storage S3-compatible (MinIO no deploy local).
- Filesystem local apenas para dev controlado e testes fora da stack oficial.

## Event Reliability
- Toda mutacao relevante publica evento de dominio.
- Publicacao usa tabela outbox transacional.
- Consumidores devem ser idempotentes por `idempotency_key`.
- Worker deve suportar retry deterministico com backoff, erro persistido e DLQ.

## Data and Migration Policy
- Migrations additive-first.
- Migracao destrutiva requer ADR + janela de manutencao.
- Rollback documentado antes do merge.

## Security Guardrails
- Segredos so por env var/secret manager.
- Tokens de auth com expiracao curta.
- Auth oficial web = sessao por cookie HTTP-only.
- Runtime oficial nao usa `X-User-Id`; header legado fica apenas para testes tecnicos controlados.
- Requests mutaveis autenticados por cookie exigem verificacao de origem (`Origin`/`Referer`) ou mecanismo equivalente.
- Nenhum endpoint mutavel sem verificacao de permissao.
- Nenhum segredo sensivel com fallback hardcoded em runtime oficial.

## Observability Guardrails
- Logs estruturados com `trace_id`.
- Metricas por endpoint e por caso de uso.
- Tracing habilitado no caminho critico.

## Anti-patterns (strictly prohibited)
- Business rule no frontend ou no handler.
- Shared utils virando "god package".
- Dependencia circular entre modulos.
- Ajuste direto em producao sem runbook.
