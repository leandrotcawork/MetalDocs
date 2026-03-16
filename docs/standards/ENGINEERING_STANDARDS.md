# Engineering Standards

## 1. Naming and Layout
- Pacotes em lowercase sem underscore.
- Arquivos por responsabilidade (`service.go`, `repository.go`, `errors.go`).
- DTO HTTP separado de entidades de dominio.

## 2. API and Contract Standards
- OpenAPI-first (`api/openapi/v1/openapi.yaml`).
- Sem endpoint fora do spec.
- Breaking change apenas em `/api/v2`.
- Toda mudanca de endpoint exige update de OpenAPI no mesmo PR.

## 3. Error Standard
API error payload padrao:
```json
{
  "error": {
    "code": "STRING_CODE",
    "message": "Human readable message",
    "details": {},
    "trace_id": "TRACE_ID"
  }
}
```
- `code` estavel e versionado.
- Nao retornar stack trace para cliente.

## 4. Logging, Metrics, Tracing
- Logs estruturados JSON em runtime nao-dev.
- Campos minimos: `trace_id`, `user_id`, `module`, `action`, `result`, `duration_ms`.
- RED metrics por endpoint (rate/errors/duration).
- Span por caso de uso principal.

## 5. Timeout, Retry, Idempotency
- Timeout explicito em chamadas externas.
- Retry apenas em operacao idempotente.
- Evento interno sempre com `idempotency_key`.

## 6. Data and Migrations
- Migration numerada sequencialmente.
- Additive-first por padrao.
- Migracao destrutiva exige ADR + plano de rollback.

## 7. Test Policy (minimum per feature)
- Unit: invariantes de dominio e regras de caso de uso.
- Contract: aderencia API <-> OpenAPI.
- Integration: fluxo com DB e adapters principais.
- E2E: fluxo critico de negocio.

Minimum gate por PR de feature:
- 1 teste unit novo ou atualizado.
- 1 teste integration/contract quando mudar API/dominio.

## 8. PR Policy
- PR pequeno e focado.
- Nao misturar feature + refactor amplo.
- Checklist preenchido e evidencias anexadas.
- ADR/RFC anexado quando exigido.

## 9. Review Policy
- Review prioriza: contratos, seguranca, regressao, dados, observabilidade.
- Falta de testes obrigatorios bloqueia merge.
- Divergencia de boundary bloqueia merge.

## 10. Dependency Management (Go)
- Source of truth de dependencias: `go.mod`.
- Lock e integridade: `go.sum`.
- Nao usar `requirements.txt` para backend Go.
- Sempre commitar `go.mod` e `go.sum` juntos quando houver mudanca de dependencia.
- Atualizacao de dependencia deve ser minima e justificada por feature/fix.
