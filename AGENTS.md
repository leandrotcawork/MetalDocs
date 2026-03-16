# MetalDocs AGENTS.md

## Project Mission
MetalDocs centraliza documentos com versionamento imutavel, workflow e auditoria confiavel.

## Scope Freeze (v1)
- Backend-first com UI minima.
- Auth v1: RBAC local com login interno.
- Deploy v1: Docker Compose single-node em `192.168.0.3`.
- Sem IA/LLM no produto (feature/runtime) na v1.

## Token-Safe Workflow
1. Comece com busca direcionada: `rg -n "symbol|text" .`.
2. Abra no maximo 3 arquivos inicialmente.
3. Nao escaneie pastas inteiras sem necessidade.
4. Expanda leitura apenas depois de localizar alvos exatos.
5. Menor diff seguro vence.

## Non-Negotiable Architecture
- Arquitetura: modular monolith vertical slice.
- Fluxo de request: `delivery -> application -> domain -> infrastructure`.
- Frontend nunca executa regra de negocio.
- `domain` nao depende de `infrastructure`.
- Modulos nao importam internals de outros modulos diretamente.
- Comunicacao cross-module via interface explicita ou evento interno.

## Immutable Contracts
- OpenAPI source of truth: `api/openapi/v1/openapi.yaml`.
- Versionamento de documento e imutavel (sem update de versao existente).
- Auditoria e append-only (sem update/delete de eventos).
- Permissao sempre validada no backend.
- Nunca sobrescrever valor populado com `null`, vazio ou default acidental.

## Decision Policy (ADR vs RFC vs Direct)
### ADR obrigatoria quando houver:
- Mudanca de arquitetura, boundary ou contrato publico.
- Mudanca de seguranca, autenticacao ou autorizacao.
- Mudanca de schema de dados, migracao destrutiva ou retencao.
- Mudanca de estrategia de deploy/rollback/observabilidade.
- Inclusao de dependencia relevante.

### RFC curta (`docs/rfc/`) quando houver:
- Mudanca relevante sem alterar contrato publico.
- Troca de implementacao com tradeoff claro.

### Sem cerimonia adicional quando houver:
- Bugfix local sem impacto de contrato.
- Ajuste de texto/docs sem impacto de governanca.

## Definition of Ready (DoR)
### API change
- Endpoint e payload definidos em OpenAPI.
- Erros mapeados em catalogo padrao.
- Regras de permissao definidas.

### Domain change
- Invariantes e eventos definidos.
- Regras de idempotencia e auditoria definidas.
- Criterios de aceite testaveis definidos.

### Infra change
- Impacto operacional documentado.
- Rollback definido.
- Risco e mitigacao registrados.

### UI change
- Contrato de API existente/definido.
- Estados de loading/error/empty definidos.
- Permissao e visibilidade por role definidas.

## Definition of Done (DoD)
### API change
- OpenAPI atualizado.
- Contract tests passando.
- Breaking change apenas em nova versao.

### Domain change
- Unit + integration tests da regra nova.
- Auditoria e eventos validados.
- Sem violacao de boundary.

### Infra change
- Runbook atualizado.
- Smoke test executado.
- Plano de rollback validado.

### UI change
- Fluxo principal + estado de erro cobertos.
- Sem logica de negocio fora do backend.

## PR Gates (mandatory)
- Checklist de PR preenchido.
- Evidencias de teste anexadas.
- OpenAPI atualizado quando API mudar.
- ADR/RFC anexado quando exigido pela policy.
- Nao misturar refactor amplo com feature.

## Forbidden Patterns
- Hardcode de segredo/credencial.
- Bypass de autorizacao no backend.
- Update/delete em log de auditoria.
- Regras de negocio em handler HTTP ou frontend.
- Refactor estrutural fora de escopo.

## Delivery Format (Required)
Sempre incluir:
- Summary of what changed.
- File list touched.
- Commands executed for validation.
- Risks and follow-up notes.
