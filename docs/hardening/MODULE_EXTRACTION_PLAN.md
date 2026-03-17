# Module Extraction Plan (Future-Ready)

## Objective
Definir estrategia de extracao progressiva de modulos para servicos independentes, sem quebrar contratos atuais.

## Extraction Principles
- Extrair apenas quando houver necessidade clara de escala/autonomia.
- Preservar contrato de API e eventos durante migracao.
- Sempre usar strangler pattern: novo caminho coexistindo com o antigo.
- Nunca extrair sem observabilidade e rollback definidos.

## Candidate Order (if extraction is required)
1. `search`
- Motivo: leitura intensiva e menor acoplamento transacional.
- Contrato: manter `/api/v1/search/documents` com facade no monolith.

2. `workflow`
- Motivo: regras de estado independentes e potencial de throughput proprio.
- Contrato: manter transicoes e audit/outbox com idempotencia forte.

3. `documents + versions`
- Motivo: dominio central; extracao somente com justificativa forte.
- Pre-condicao: estrategia de consistencia e ownership de dados definida.

## Required Preconditions
- [ ] SLOs por modulo definidos.
- [ ] Contratos de eventos versionados e documentados.
- [ ] Testes de contrato e e2e cobrindo caminho antigo e novo.
- [ ] Observabilidade por modulo (logs, metrics, tracing) validada.
- [ ] Plano de rollback por fase de extracao.

## Migration Blueprint (per module)
1. Publicar ADR da extracao.
2. Introduzir facade interna no monolith.
3. Duplicar escrita com outbox/event bridge (quando necessario).
4. Habilitar leitura no novo modulo por feature flag.
5. Fazer canary de trafego.
6. Promover para 100% e desativar caminho antigo.
7. Executar postmortem de migracao e consolidar runbook.

## Anti-Patterns to Avoid
- Extrair modulo central sem contratos maduros.
- Extrair por hype sem ganho operacional mensuravel.
- Quebrar API publica para acomodar design interno.
