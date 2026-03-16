# MetalDocs

MetalDocs e uma plataforma interna para centralizacao, versionamento, workflow e auditoria de documentos.

## Objetivo
- Ter controle total de documentos e historico de mudancas.
- Garantir rastreabilidade (quem fez, o que mudou, quando mudou).
- Escalar sem reescrita estrutural.

## Escopo v1
- Cadastro de documentos e metadados.
- Versionamento imutavel.
- Workflow de aprovacao.
- Busca e consulta.
- Auditoria append-only.
- RBAC no backend.

## Fora de escopo v1
- Funcionalidades de IA generativa, NLP ou agentes.
- Sugestoes automaticas baseadas em IA.
- Qualquer processamento que dependa de modelos de linguagem.

## Principios
- Arquitetura modular monolith com boundaries explicitos.
- API-first com OpenAPI versionado.
- Observabilidade desde o dia 1.
- Sem mudancas destrutivas sem ADR.

## Estrutura
- `apps/`: processos executaveis (API e worker).
- `internal/platform/`: infraestrutura compartilhada.
- `internal/modules/`: dominios de negocio (vertical slice).
- `api/openapi/`: contrato de API.
- `docs/`: arquitetura, ADRs e runbooks.
- `tests/`: estrategia de testes por tipo.

## Ambiente inicial
- Host local: `192.168.0.3`.
- Deploy inicial via Docker Compose.
