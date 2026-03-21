# Next Cycle Backlog

## Objective
Registrar as tasks do proximo ciclo de execucao do MetalDocs com foco em dominio documental, permissao por recurso e implementacao incremental.

## Rules
- Cada task deve gerar commit proprio quando possivel.
- Nenhuma task de UI deve adiantar contrato que nao exista no backend.
- Toda task que mudar contrato deve atualizar OpenAPI e testes.
- Toda task que mudar autorizacao deve explicitar impacto em `iam` e `documents`.

## Status Legend
- `todo`
- `doing`
- `done`
- `blocked`

## Task 001 - Freeze document type registry
Status: `done`

Objetivo:
Definir os tipos documentais iniciais da plataforma sem acoplar ao negocio de uma empresa especifica.

Escopo:
- lista v1 de `document_type`
- campos base de cada tipo
- regras de validade/revisao por tipo

Saida:
- contrato documentado
- base para schema e validacoes

## Task 002 - Model document metadata base
Status: `done`

Objetivo:
Evoluir `documents` para suportar metadados estruturados obrigatorios.

Escopo:
- `document_type`
- `business_unit`
- `department`
- `tags`
- `effective_at`
- `expiry_at`
- `metadata_json`

Saida:
- modelo de dominio atualizado
- OpenAPI atualizada

## Task 003 - Define permission matrix by resource
Status: `done`

Objetivo:
Definir o modelo de permissao que permita controlar quem pode ver, editar, anexar e alterar workflow.

Escopo:
- permissoes por `area`
- permissoes por `document_type`
- override por `document`
- capacidades:
  - `document.view`
  - `document.edit`
  - `document.upload_attachment`
  - `document.change_workflow`
  - `document.manage_permissions`

Saida:
- policy model documentado
- estrategia de avaliacao backend definida

## Task 004 - Add access policy schema and persistence
Status: `done`

Objetivo:
Criar a persistencia do modelo de acesso por recurso.

Escopo:
- migrations additive-first
- estrutura de policy por sujeito/recurso/capacidade
- adaptadores Postgres

Saida:
- schema pronto para enforcement

## Task 005 - Enforce authorization in documents flows
Status: `done`

Objetivo:
Aplicar o novo modelo de permissao nas operacoes de documentos.

Escopo:
- criar documento
- editar documento
- listar documento
- visualizar documento
- anexar arquivo
- alterar permissoes

Saida:
- autorizacao orientada a capacidade no backend

## Task 006 - Evolve versions inside documents aggregate
Status: `done`

Objetivo:
Fortalecer o modelo de versao sem quebrar o ownership do aggregate `documents`.

Escopo:
- versionamento imutavel
- resumo de mudanca
- suporte a diff no dominio/aplicacao

Saida:
- contrato de versao ampliado
- testes de regressao

## Task 007 - Add type-aware metadata validation
Status: `done`

Objetivo:
Validar `metadata_json` conforme `document_type`.

Escopo:
- schema por tipo
- obrigatoriedade por campo
- validacao de formato/tipo

Saida:
- erro de dominio consistente para metadata invalida

## Task 008 - Extend search with structured filters
Status: `done`

Objetivo:
Permitir busca profissional baseada em filtros estruturados e autorizacao.

Escopo:
- filtro por tipo
- filtro por area
- filtro por departamento
- filtro por owner
- filtro por validade
- filtro respeitando policy de visualizacao

Saida:
- busca pronta para UI operacional

## Task 009 - Add attachments module
Status: `done`

Objetivo:
Criar o fluxo de anexos com storage abstraido e permissao separada.

Escopo:
- upload
- validacao de tipo e tamanho
- registro no banco
- download com URL assinada/temporaria
- enforcement de `document.upload_attachment`

Saida:
- modulo operacional para anexos

## Task 010 - Extend workflow with approval ownership
Status: `done`

Objetivo:
Evoluir workflow para aprovacoes mais proximas do negocio documental.

Escopo:
- aprovador responsavel
- motivo da aprovacao/rejeicao
- trilha de aprovacao
- integracao com notificacoes

Saida:
- workflow pronto para cenarios de aprovacao reais

## Task 011 - Build worker for notifications and review reminders
Status: `done`

Objetivo:
Consumir outbox e executar jobs assincronos do dominio.

Escopo:
- eventos de workflow
- solicitacao de aprovacao
- documento aprovado
- revisao prestes a vencer

Saida:
- worker production-ready para notificacoes

## Task 012 - Build operational UI
Status: `done`

Objetivo:
Criar a UI minima em cima do contrato real do backend.

Escopo:
- formulario de documento com tipo e metadata
- tela de permissoes
- listagem com filtros
- detalhe do documento
- anexos
- timeline de workflow/audit

Saida:
- interface minima usavel para operacao real

## Task 013 - Promote MinIO to official runtime storage
Status: `done`

Objetivo:
Alinhar o runtime real de anexos com a arquitetura oficial de object storage.

Escopo:
- provider explicito de storage
- adapter MinIO/S3-compatible
- bootstrap de bucket
- compose alinhado ao provider oficial

Saida:
- stack Docker usando MinIO de verdade para blobs

## Task 014 - Harden outbox worker with retry and DLQ
Status: `done`

Objetivo:
Tornar o processamento assincrono resiliente a falhas temporarias e eventos envenenados.

Escopo:
- retry com backoff deterministico
- persistencia de erro e tentativa
- DLQ operacional
- logs operacionais do worker

Saida:
- worker pronto para operacao mais robusta

## Task 015 - Harden auth, web session security and local dev runtime
Status: `done`

Objetivo:
Fechar auth v1, seguranca web e modo dev local sem ambiguidade antes do proximo ciclo.

Escopo:
- sessao por cookie HTTP-only como runtime oficial
- first login com troca obrigatoria sem recarregar workspace antes da hora
- protecao de origem para requests mutaveis autenticados por cookie
- `X-User-Id` apenas como modo tecnico controlado
- Postgres Docker como unica fonte de verdade local
- scripts e runbooks do modo dev rapido

Saida:
- auth v1 estavel para web
- runtime local previsivel
- backlog liberado para a proxima camada de produto

## Task 016 - Introduce document family and profile registry
Status: `done`

Objetivo:
Separar familias documentais canonicas da plataforma de perfis documentais configuraveis por empresa.

Escopo:
- familias canonicas fixas da plataforma
- perfis documentais configuraveis
- relacao `profile -> family`
- `documents` passa a operar por profile sem perder a family

Saida:
- base multiempresa sem hardcode de tipos documentais por cliente
- compatibilidade preservada com `documentType` como alias transitório de `documentProfile`

## Task 017 - Add process area and subject taxonomy
Status: `done`

Objetivo:
Separar assunto/processo da natureza documental para evitar crescimento caotico de tipos.

Escopo:
- `process_area`
- opcionalmente `subject/domain`
- vinculo entre documentos, profiles e taxonomia
- busca/filtro por processo

Saida:
- `marketplaces` e outros contextos deixam de competir com tipo documental
- `documents` e `search` passam a carregar `processArea` e `subject` como taxonomia separada

## Task 018 - Add versioned schema and governance by profile
Status: `done`

Objetivo:
Permitir que cada profile defina metadata e governanca proprios de forma versionada e auditavel.

Escopo:
- schema versionado por profile
- campos obrigatorios/opcionais por profile
- prefixo/codigo por profile
- workflow/revisao/retencao por profile

Saida:
- validacao e governanca documental configuraveis por empresa
- `documents` valida metadata pelo schema ativo e persiste `profileSchemaVersion`

## Task 019 - Seed Metal Nobre document registry
Status: `done`

Objetivo:
Materializar o primeiro caso real da plataforma com taxonomia alinhada a ISO-9001 e operacao da Metal Nobre.

Escopo:
- profiles iniciais:
  - `PO`
  - `IT`
  - `RG`
- process areas iniciais:
  - `quality`
  - `marketplaces`
  - `commercial`
  - `purchasing`
  - `logistics`
  - `finance`

Saida:
- registry real pronto para organizar os documentos da Metal Nobre
- profiles `po`, `it`, `rg` e process areas iniciais sem hardcode espalhado na aplicacao

## Task 020 - Evolve API and UI to create documents by profile
Status: `done`

Objetivo:
Fazer a plataforma operar pelo registry configuravel em vez de lista fixa de tipos.

Escopo:
- OpenAPI refletindo family + profile
- endpoints para listar registry/perfis
- UI criando documentos por profile
- exibicao de process area e governanca do profile

Saida:
- experiencia operacional multiempresa sobre a arquitetura correta
- UI criando documentos por `documentProfile` com schema/governanca carregados do registry
- exibicao de `documentFamily`, `documentProfile`, `processArea` e `profileSchemaVersion` na experiencia operacional

## Task 021 - Add audit timeline HTTP surface
Status: `done`

Objetivo:
Fechar a timeline operacional do produto com trilha HTTP de auditoria real.

Escopo:
- endpoint de timeline/audit
- consulta por documento
- ordenacao por tempo
- payload alinhado com eventos append-only existentes

Saida:
- timeline operacional completa via backend
- endpoint `/api/v1/audit/events` com filtro por recurso, ordenacao decrescente por tempo e payload alinhado aos eventos append-only

## Task 022 - Build operational notifications experience
Status: `done`

Objetivo:
Expor no produto a camada de notificacoes que hoje ja existe no worker/backend.

Escopo:
- endpoint/listagem de notificacoes
- tela operacional de notificacoes
- marcacao de leitura/estado operacional

Saida:
- notificacoes visiveis e utilizaveis na web app
- endpoint `/api/v1/notifications` para listagem operacional
- endpoint `/api/v1/notifications/{notificationId}/read` para marcacao de leitura
- painel web de notificacoes com estado operacional

## Task 023 - Expand administrative user management
Status: `done`

Objetivo:
Evoluir IAM/Auth administrativo para um ciclo de vida de usuario mais profissional.

Escopo:
- gestao administrativa de usuarios mais completa
- ativacao/inativacao
- visualizacao de estado de auth
- atribuicao de roles mais rica

Saida:
- superficie administrativa mais proxima de produto real
- ativacao/inativacao de usuario via produto
- visualizacao de estado de auth (lock, falhas, ultimo login, troca obrigatoria)
- reconciliacao completa de roles por usuario

## Task 024 - Add administrative password reset and user lifecycle actions
Status: `done`

Objetivo:
Fechar o lifecycle operacional de usuarios internos sem depender de SQL manual.

Escopo:
- reset administrativo de senha
- obrigar troca de senha no proximo login
- bloqueio/desbloqueio operacional
- trilha auditavel dessas acoes

Saida:
- lifecycle de usuario operacionalizado
- reset administrativo de senha com troca obrigatoria no proximo login
- desbloqueio operacional com limpeza de lock e tentativas falhas
- revogacao de sessoes ativas apos reset administrativo
- trilha auditavel para update, reset e unlock

## Task 025 - Deepen production readiness and observability
Status: `done`

Objetivo:
Subir a maturidade operacional da plataforma apos o ciclo de taxonomia documental e superficies operacionais.

Escopo:
- observabilidade mais profunda
- readiness de producao
- indicadores de auth/session/worker
- evidencias de operacao segura

Saida:
- baseline mais proxima de release-grade
- `/api/v1/health/ready` com checks estruturados e status degradado quando dependencias caem
- `/api/v1/metrics` enriquecido com indicadores de auth, sessions e outbox worker
- runtime local e Postgres com a mesma superficie de observabilidade

## Recommended Commit Order
1. Task 001 + Task 002
2. Task 003 + Task 004
3. Task 005
4. Task 006 + Task 007
5. Task 008
6. Task 009
7. Task 010
8. Task 011
9. Task 012
10. Task 013
11. Task 014
12. Task 015
13. Task 016 + Task 017
14. Task 018 + Task 019
15. Task 020
16. Task 021 + Task 022
17. Task 023 + Task 024
18. Task 025

## Task 026 - Restrict operational metrics surface
Status: `done`

Objetivo:
Fechar a exposicao publica indevida da superficie de metricas operacionais.

Escopo:
- `/api/v1/metrics` deixa de ser publico no runtime oficial
- metrics passa a exigir auth + permissao administrativa
- `/metrics` sai do bypass de auth/rate-limit
- OpenAPI, smoke e runbooks alinhados a politica nova

Saida:
- `health/live` e `health/ready` continuam superficies operacionais controladas
- `metrics` deixa de ser canal anonimo de telemetria operacional

## Task 027 - Remove legacy auth header from official runtime
Status: `done`

Objetivo:
Eliminar `X-User-Id` como caminho oficial de runtime.

Escopo:
- runtime oficial ignora auth por header legado
- docs e env examples deixam explicito que o header e apenas tecnico/teste
- smoke e middleware passam a tratar cookie-session como auth oficial

Saida:
- auth oficial unificada em sessao por cookie
- menor risco de impersonation por flag/env drift

## Task 028 - Remove hardcoded secrets and sensitive fallbacks
Status: `done`

Objetivo:
Remover segredos embutidos e atalhos sensiveis de runtime.

Escopo:
- fim do fallback hardcoded para `METALDOCS_ATTACHMENTS_SIGNING_SECRET`
- attachment downloads dependem apenas de config/env valido
- testes e runbooks atualizados para explicitar o segredo obrigatorio

Saida:
- nenhum segredo sensivel padrao embutido no caminho normal de runtime
- falha explicita quando configuracao obrigatoria estiver ausente

## Task 029 - Move health/readiness ownership to platform
Status: `done`

Objetivo:
Corrigir boundary arquitetural das rotas operacionais.

Escopo:
- `/health/live` e `/health/ready` saem do modulo `documents`
- ownership passa para `internal/platform/observability`
- bootstrap da API registra health/readiness fora dos modulos de negocio

Saida:
- platform serve health/readiness
- modules consomem capability, nao sao donos da superficie global

## Task 030 - Refactor web app into operational slices
Status: `done`

Objetivo:
Reduzir risco estrutural do frontend antes da fase forte de authoring/UX.

Escopo:
- `App.tsx` quebrado em slices operacionais
- auth shell, app shell header, documents workspace, IAM admin panel e notifications panel extraidos
- bootstrap e regras de negocio preservados sem redesign visual

Saida:
- frontend mais seguro para evolucao
- menor acoplamento de render entre auth, documentos, IAM e notificacoes

## Task 031 - Runtime/contract consistency cleanup
Status: `done`

Objetivo:
Eliminar drift restante entre docs, contrato e runtime.

Escopo:
- OpenAPI sem server local hardcoded
- runbooks alinhados com `localhost`/runtime oficial local
- contrato baseline e observability docs atualizados para metrics autenticadas
- env example limpo de hosts antigos nao-oficiais

Saida:
- docs e contratos representando a verdade do sistema com menos ambiguidade operacional

## Task 032 - Security/architecture release review gate
Status: `done`

Objetivo:
Fechar a fase de hardening com gate formal antes de authoring/UX.

Escopo:
- checklist final de auth, metrics, secrets, observability e frontend boundaries
- registro dos residual risks aceitos conscientemente
- criterio explicito de go/no-go para abrir a fase documental/UX

Saida:
- fase atual fechada com evidencias tecnicas
- backlog liberado para `Task 033+`

## Recommended Commit Order
1. Task 001 + Task 002
2. Task 003 + Task 004
3. Task 005
4. Task 006 + Task 007
5. Task 008
6. Task 009
7. Task 010
8. Task 011
9. Task 012
10. Task 013
11. Task 014
12. Task 015
13. Task 016 + Task 017
14. Task 018 + Task 019
15. Task 020
16. Task 021 + Task 022
17. Task 023 + Task 024
18. Task 025
19. Task 026 + Task 027 + Task 028 + Task 029
20. Task 030 + Task 031 + Task 032

## Task 033 - Document authoring flow design
Status: `done`

Objetivo:
Congelar o modelo oficial de criacao, edicao e consulta documental antes da fase forte de UX.

Escopo:
- definir authoring profile-first
- separar estrutura, metadata, governanca e versionamento
- congelar modos de tela: create, edit metadata, edit content, review e read
- registrar mapeamento das superficies HTTP atuais ao fluxo de authoring

Saida:
- documento formal de authoring publicado
- base pronta para `Task 034` e `Task 035` sem reabrir o modelo de dominio

## Task 034 - Document workspace UX
Status: `done`

Objetivo:
Traduzir o modelo de authoring em experiencia visual e navegacao operacional de alto nivel.

Escopo:
- app shell documental
- operations center documental
- workspace documental orientado por `documentProfile`
- catalogo documental
- detalhe/review documental
- authoring wizard em 4 etapas (`Profile -> Metadata -> Content -> Review`)
- registry explorer em modo leitura forte
- metadata dinamica por schema
- governanca sempre visivel
- area de versoes/diff/approvals/attachments/audit navegavel
- compatibilidade arquitetural com realtime futuro sem depender de SSE/WebSocket agora

Saida:
- UX documental consistente, profissional e preparada para empresas diferentes

Progresso fase 1:
- shell documental consolidado como ponto unico de composicao das views
- renderizacao de views em `App.tsx` unificada por roteamento interno (`renderWorkspaceView`)
- fallback explicito de view com placeholder para evitar estados vazios/improvisados

Progresso fase 2:
- views de `notifications` e `admin` migradas para padrao `catalog-shell` (header, grid e painel consistente)
- padrao visual unificado entre telas operacionais sem quebrar contratos de dados existentes

Progresso fase 3:
- componente compartilhado `WorkspaceViewFrame` criado para padronizar estrutura base (`kicker`, `title`, `description`, `actions`, `stats`)
- `operations`, `create`, `registry`, `notifications` e `admin` migrados para o frame comum

Progresso fase 4:
- componente compartilhado `WorkspaceDataState` criado para padrao de `loading/error/empty`
- `documents`, `notifications` e `admin` passaram a usar estado visual consistente com acao de retry

Progresso fase 5:
- `refreshWorkspace` centralizado em `App.tsx` (sem callbacks duplicados por view)
- shell ganhou acao global de refresh com estado de loading consistente

Progresso fase 6:
- frame estrutural + estados operacionais compartilhados consolidados para as principais views do workspace
- arquitetura de frontend pronta para avancar para customizacao Metal Nobre (`Task 035`) e evolucao de registry/admin (`Task 036`) sem refatoracao de base

## Task 035 - Metal Nobre applied experience
Status: `done`

Objetivo:
Aplicar a experiencia documental ao caso real da Metal Nobre.

Escopo:
- perfis `po`, `it`, `rg`
- process areas como `marketplaces`, `quality`, `commercial`, `purchasing`, `logistics` e `finance`
- nomenclaturas, hints e experiencia proximas do uso real ISO-inspired

Saida:
- experiencia documental aplicada ao caso de negocio da Metal Nobre sem hardcode de plataforma

Progresso fase 1:
- UX de `authoring` e `registry` alinhada ao contexto Metal Nobre com nomenclatura operacional e hints ISO-inspired
- adapter de experiencia (`metalNobreExperience`) centralizado no frontend para evitar texto hardcoded espalhado em componentes
- sidebar `Por tipo` corrigida na raiz para usar nomes canonicos de `processAreas` no agrupamento (em vez de codigos crus)

Progresso fase 2:
- `Todos Documentos` passou a exibir prioridades por processo com hints operacionais da Metal Nobre
- `Centro Operacional` ganhou snapshot de processos com nomes canonicos de area (registry) em vez de codigos crus
- labels de perfil/processo no acervo e painel operacional foram alinhados para leitura executiva consistente

Progresso fase 3:
- `Centro Operacional` e `Registry` passaram a usar `WorkspaceDataState` com mensagens contextuais de loading/error/empty
- renderizacao operacional agora evita telas "meio prontas" quando o estado nao esta `ready`
- linguagem operacional foi alinhada (ex.: `Aprovacoes`) mantendo consistencia entre shell e paineis

Progresso fase 4:
- consolidacao de experiencia aplicada com acabamento de estados operacionais e leitura consistente em `operations`, `catalog` e `registry`
- base frontend pronta para abrir trilha de administracao do registry (`Task 036`) sem refatorar shell/workspace
- Task 035 encerrada com foco em UX aplicada ao dominio Metal Nobre mantendo contratos e boundaries

## Task 036 - Registry administration CRUD
Status: `completed`

Objetivo:
Permitir administrar o registry documental pelo produto, e nao apenas consulta-lo.

Escopo:
- CRUD de profiles
- CRUD/versionamento de schema por profile
- CRUD de governance por profile
- CRUD de process areas
- CRUD de subjects

Saida:
- painel administrativo que torna o modelo profile-first realmente configuravel por UI

Progresso fase 1:
- backend ganhou write-path admin para `process areas` e `subjects` (create/update/deactivate) com authz administrativa
- OpenAPI v1 atualizado para novos endpoints de administracao do taxonomy registry
- `RegistryExplorer` recebeu controles admin para operar CRUD de `process areas` e `subjects` consumindo os novos endpoints

Progresso fase 2:
- backend passou a suportar write-path admin para `document profiles` (create/update/deactivate) e update de `governance`
- OpenAPI v1 evoluiu para cobrir os novos contratos de profile/governance write mantendo compatibilidade additive
- `RegistryExplorer` ganhou controles admin para perfil e governanca, mantendo `registry` como superficie unica de administracao

Progresso fase 3:
- backend passou a suportar write-path admin para versionamento de `schema` por profile (upsert de versao + ativacao explicita)
- persistencia `memory` e `postgres` alinhada para garantir apenas uma versao ativa por profile durante ativacao
- OpenAPI v1 e frontend (`api client` + `RegistryExplorer`) atualizados para administrar versoes de schema sem hardcode local

Progresso fase 4:
- editor admin de schema no `RegistryExplorer` evoluiu de JSON livre para formulario estruturado de regras (`name`, `type`, `required`)
- UX de administracao de schema ficou mais segura para operacao diaria sem depender de edicao manual de payload
- Task 036 encerrada com CRUD funcional de profiles, governance, schema versions, process areas e subjects

## Task 037 - Realtime event stream for operations center
Status: `done`

Objetivo:
Adicionar atualizacao ao vivo para paineis operacionais sem acoplar isso prematuramente ao authoring base.

Escopo:
- feed operacional server-to-client
- notificacoes em tempo real
- approvals e sinais operacionais em tempo real
- adaptador frontend de stream

Direcao:
- priorizar `SSE`
- avaliar `WebSocket` apenas se houver necessidade real de bidirecionalidade

Saida:
- operations center com atualizacao ao vivo sem polling como mecanismo unico

Progresso fase 1:
- endpoint SSE inicial entregue em `GET /api/v1/operations/stream` com snapshot operacional periodico (notificacoes pendentes + documentos em revisao + total de documentos)
- authz integrado ao middleware IAM (`PermDocumentRead`) sem bypass
- frontend conectado ao stream com refresh resiliente e fallback preservado (sem acoplamento forte ao authoring)

Progresso fase 2:
- adapter de stream operacional centralizado em `frontend/apps/web/src/lib.api.ts` para evitar wiring SSE espalhado
- `App` passou a consumir stream via API client (boundary mais limpo e evolutivo para retry/observabilidade futura)

Progresso fase 3:
- snapshot operacional do SSE passou a incluir sinal de `pendingApprovals` (derive de notificacoes `workflow.approval.requested`)
- payload realtime ficou mais aderente ao Operations Center sem exigir polling para indicador de fila de aprovacoes

## Task 038 - Collaborative editing and presence
Status: `done`

Objetivo:
Evoluir a plataforma para colaboracao documental em tempo real quando isso realmente virar necessidade de produto.

Escopo:
- presence
- locking/conflito de edicao
- sinais colaborativos
- co-authoring

Saida:
- base para experiencia colaborativa mais profunda sem improviso arquitetural

Progresso fase 1:
- dominio `documents` ganhou modelos canonicos de colaboracao (`CollaborationPresence`, `DocumentEditLock`) e invariantes de normalizacao
- repositórios `memory` e `postgres` implementaram write/read path para presence e lock com regras de conflito
- service application adicionou use-cases autorizados para heartbeat/list de presence e acquire/get/release de lock
- delivery HTTP e OpenAPI v1 evoluiram com endpoints dedicados de colaboracao em `/documents/{id}/collaboration/*`
- frontend passou a consumir presence/lock no detalhe de documento com heartbeat periodico best-effort
- migrations `0038` e `0039` adicionaram schema e grants de runtime para colaboracao documental

## Task 039 - Fix OpenAPI v1 compatibility for DocumentProfile alias
Status: `done`

Objetivo:
Garantir evolucao compat no contrato v1 ao introduzir `alias` em `DocumentProfileItem` sem quebrar clientes estritos.

Contexto:
Follow-up documentado em `../hardening/DOCUMENTS_ARCH_FOLLOWUPS_20260318.md`.

Escopo:
- manter `alias` sempre presente no payload do backend
- ajustar OpenAPI v1 para `alias` nao ser `required` em response
- atualizar contract tests para validar presenca de `alias` sem exigir `required` em schema

Saida:
- OpenAPI v1 compativel (additive)
- smoke contract mantendo garantia de `alias` presente

Aceite:
- `go test ./tests/contract -count=1` verde
- `GET /api/v1/document-profiles` sempre retorna `alias` preenchido

## Task 040 - Remove destructive cleanup from official migrations (dev-only tooling)
Status: `done`

Objetivo:
Alinhar migrations com politica additive-first e invariavel append-only de auditoria, isolando reset/cleanup em tooling dev.

Contexto:
Follow-up documentado em `../hardening/DOCUMENTS_ARCH_FOLLOWUPS_20260318.md`.

Escopo:
- remover uso de `DELETE` em `audit_events` de qualquer migration oficial
- mover "reset do registry legado" para `scripts/` + runbook dedicado
- documentar janela/rollback e impacto operacional (quando aplicavel)

Saida:
- cadeia de migrations conforme ADR-0007
- reset de ambiente dev suportado por script/runbook, nao por migration de produto

Aceite:
- `docs/adr/0007-schema-migration-policy.md` obedecido
- nenhum `DELETE/UPDATE` em `audit_events` em migrations oficiais

## Task 041 - Decouple documents domain from workflow domain types
Status: `done`

Objetivo:
Restaurar disciplina de boundary: `documents/domain` nao deve depender de tipos do modulo `workflow`.

Contexto:
Follow-up documentado em `../hardening/DOCUMENTS_ARCH_FOLLOWUPS_20260318.md`.

Escopo:
- substituir `workflowdomain.Approval` em `internal/modules/documents/domain/port.go` por tipo/DTO do proprio modulo
- manter integracao via porta dedicada (application) ou evento interno (quando aplicavel)
- ajustar repositorios `memory/postgres` e testes afetados

Saida:
- dominio `documents` compilando sem import de `internal/modules/workflow/domain`

Aceite:
- `rg -n \"internal/modules/workflow/domain\" internal/modules/documents/domain` nao retorna resultados
- `go test ./...` verde

## Task 042 - Decouple documents application from iam domain context helpers
Status: `done`

Objetivo:
Evitar dependencia do dominio de IAM para extrair contexto de autenticacao/autorizacao no modulo `documents`.

Contexto:
Follow-up documentado em `../hardening/DOCUMENTS_ARCH_FOLLOWUPS_20260318.md`.

Escopo:
- criar/usar helper de plataforma (ex.: `internal/platform/authn`) para `UserIDFromContext` e roles
- atualizar `internal/modules/documents/application/service.go` para depender de plataforma, nao de `iam/domain`
- manter authz sempre validada no backend

Saida:
- `documents/application` sem import de `internal/modules/iam/domain`

Aceite:
- `rg -n \"internal/modules/iam/domain\" internal/modules/documents/application` nao retorna resultados
- `go test ./...` verde

## Task 043 - Frontend structure: UI primitives, feature slices, and catalog summary performance
Status: `done`

Objetivo:
Evoluir `apps/web` para estrutura escalavel: primitives reutilizaveis, feature slices e performance previsivel do shell.

Contexto:
Follow-up documentado em `../hardening/DOCUMENTS_ARCH_FOLLOWUPS_20260318.md`.

Escopo:
- separar `frontend/apps/web/src/components/ui/*` (primitives) de views/features
- mover `TopbarDropdown` e futuros primitives para `ui/`
- reduzir recomputacoes O(P*N) no shell (memoizacao e/ou adapter)
- definir contrato de "catalog summary" (frontend-only adapter ou endpoint backend em task propria)

Saida:
- organizacao de pastas clara por tipo de componente
- shell sem agregacao cara em render

Aceite:
- `frontend` build e typecheck verdes
- sem logica de negocio no frontend (apenas adaptacao/formatacao)

## Task 044 - Fix storage boundary: workflow approvals persistence
Status: `done`

Objetivo:
Remover acoplamento por storage onde `documents` persiste `workflow_approvals`.

Escopo:
- mover CRUD de approvals para `internal/modules/workflow/infrastructure/*`
- definir porta no `workflow/domain` para persistencia de approvals
- `workflow/application` usa seu proprio repo (nao `documents` repo) para approvals
- `documents` nao acessa `metaldocs.workflow_approvals` diretamente

Aceite:
- `rg -n "workflow_approvals" internal/modules/documents/infrastructure/postgres` nao retorna resultados
- `go test ./...` verde

## Task 045 - Clarify IAM/Auth module ownership and persistence boundaries
Status: `done`

Objetivo:
Eliminar acesso cruzado a tabelas `iam_*` via `auth` e tornar ownership claro (um modulo ou portas dedicadas).

Escopo:
- decidir: unificar `auth+iam` ou manter separado com ports
- remover `auth` escrevendo/consultando `iam_users` e `iam_user_roles` diretamente
- atualizar bootstrap, repos e testes conforme decisao

Progresso fase 1:
- `auth/application` passou a depender de `iam/domain.RoleAdminRepository` para bootstrap e atribuicao de roles
- `auth/infrastructure/postgres` nao grava mais em `metaldocs.iam_user_roles` (write path movido para IAM)
- composicao e testes atualizados para injetar repo administrativo de roles

Progresso fase 2:
- `auth/infrastructure/postgres` nao consulta mais `metaldocs.iam_user_roles` em `FindIdentity*` e `ListUsers`
- `auth/application.ListUsers` passou a resolver roles via `RoleProvider` (ownership de role-read centralizado em IAM)

Progresso fase 3:
- `auth/infrastructure/postgres` nao acessa mais `metaldocs.iam_users`; `display_name/is_active` passam a ser lidos/escritos em `auth_identities`
- migration `0036_decouple_auth_identity_from_iam_user_tables.sql` remove FKs de auth para `iam_users` e liga `auth_sessions` -> `auth_identities`
- bootstrap admin agora consulta existencia de role admin via `iam/domain.RoleAdminRepository.HasAnyRole`

Aceite:
- nenhum `JOIN/INSERT` em `metaldocs.iam_*` dentro de `internal/modules/auth/infrastructure/*`
- gates + `go test ./...` verdes

## Task 046 - Replace hardcoded route permission matching with declarative policy
Status: `done`

Objetivo:
Evitar autorizacao baseada em string match de path, reduzindo risco de drift de rota e bug de seguranca.

Escopo (faseado):
- fase 1 (feito): middleware usa `r.Context()` e suporta `PermissionResolver`
- fase 2 (feito): mapping movido para composition root/registro declarativo por rota
- fase 3 (feito): teste de resolver declarativo adicionado em `apps/api/cmd/metaldocs-api/permissions_test.go`

Aceite:
- `requiredPermission()` nao contem lista hardcoded de rotas (ou fica apenas como fallback dev)
- contract baseline verde

## Task 047 - Use platform auth context helpers across application modules
Status: `done`

Objetivo:
Padronizar extracao de `userId/roles` via `internal/platform/authn` (nao via `iam/domain`) nos casos de uso.

Escopo:
- atualizar `search/application` para `internal/platform/authn`
- manter policy enforcement no backend

Aceite:
- `rg -n "internal/modules/iam/domain" internal/modules/search/application` nao retorna resultados
- `go test ./...` verde

## Task 048 - Scale review reminder emission
Status: `done`

Objetivo:
Evitar varredura de todos documentos em reminder; suportar query incremental.

Escopo:
- repo query dedicada para docs expirando nos proximos X dias
- worker/reminder usa query (nao `ListDocuments`)
- benchmark simples ou evidencias no runbook

Aceite:
- reminders nao fazem O(N) sobre todo acervo por tick

Entrega:
- `documents.Repository` ganhou `ListDocumentsForReviewReminder(from,to)`
- `notifications.EmitReviewReminders` usa query incremental por janela
- migration `0037_add_documents_review_reminder_index.sql` adiciona indice parcial para `status + expiry_at`
- evidencia registrada em `docs/runbooks/performance-baseline.md`

## Task 049 - Frontend: remove external font dependency and prepare CSP-ready assets
Status: `done`

Objetivo:
Remover dependencia runtime em Google Fonts e preparar app para CSP mais restrito.

Escopo:
- self-host DM Sans/DM Mono no bundle ou via assets locais
- atualizar `styles.css` para importar fontes locais
- documentar em runbook do frontend

Aceite:
- sem request para `fonts.googleapis.com` em runtime

Entrega:
- `@fontsource/dm-sans` e `@fontsource/dm-mono` adicionados no app web
- `main.tsx` importa pesos necessarios localmente
- `styles.css` remove `@import` de Google Fonts
- runbook de dev atualizado com regra CSP/fonts

## Task 050 - Classification semantics + audience model (ADR + contract)
Status: `done`

Objetivo:
Congelar uma semantica profissional e escalavel para `classification` (sensibilidade) e `audience` (quem pode ver/editar), evitando acoplamento de regra no frontend e prevenindo drift de contrato.

Contexto:
Hoje `classification` existe no documento, mas nao muda permissao. Permissao real e decidida por `access_policies`.

Decisao (a ser registrada em ADR):
- `classification` = label de sensibilidade (PUBLIC/INTERNAL/CONFIDENTIAL/RESTRICTED). Nao e ACL.
- `audience` = policy explicita que gera/enforce policies no backend.
- Modelo inicial v1 baseado em RBAC (roles), sem grupos complexos.

Entregaveis:
- ADR novo: `docs/adr/0009-document-audience-and-classification.md`
- OpenAPI v1 atualizado com novo bloco opcional `audience` no create.

Contrato (OpenAPI):
Adicionar em `CreateDocumentRequest`:
- `audience` (opcional):
  - `mode` enum: `INTERNAL` | `DEPARTMENT` | `AREAS` | `EXPLICIT`
  - `departmentCodes` array string (opcional; usado em `DEPARTMENT/AREAS`)
  - `processAreaCodes` array string (opcional; usado em `AREAS`)
  - `roleCodes` array string (opcional; usado em `EXPLICIT`)
  - `userIds` array string (opcional; usado em `EXPLICIT` para RESTRICTED)

Regras de negocio (descricao, nao implementacao aqui):
- `PUBLIC`: sem restricao adicional (policies podem ser vazias).
- `INTERNAL`: sem restricao adicional (padrao atual).
- `CONFIDENTIAL`:
  - default `audience.mode = DEPARTMENT` usando `department` selecionado no documento.
- `RESTRICTED`:
  - default `audience.mode = DEPARTMENT` (mais fechado por padrao) e permite EXPLICIT mais tarde.

Aceite:
- ADR aceito descrevendo semantica, defaults e tradeoffs.
- OpenAPI compila e contract tests passam.
- Frontend nao hardcoda regra de permissao; apenas envia `audience` quando selecionado.

## Task 051 - Persist and enforce audience policies for documents
Status: `done`

Objetivo:
Implementar enforcement real de acesso baseado em `audience` gerando `access_policies` por documento, mantendo o modelo atual (capability-based) como fonte de verdade.

Escopo backend:
- Domain:
  - novo tipo `domain.DocumentAudience` (ou equivalente) apenas como comando de aplicacao (nao como regra no frontend).
- Application:
  - `CreateDocument` passa a criar policies por `document` scope quando `audience` exigir (CONFIDENTIAL/RESTRICTED).
  - policies minimas por documento para capabilities:
    - `document.view`
    - `document.edit` (opcional v1: apenas owner/admin)
    - `document.upload_attachment` (seguir `edit`)
- Infrastructure:
  - reuse `metaldocs.document_access_policies` (nao criar tabela nova).
  - inserir policies no mesmo fluxo atomico da criacao (ideal: transacao/AtomicCreateRepository).

Modelo RBAC recomendado (sem grupos):
- Mapear audiencias para `subject_type=role`:
  - dept role: `dept:<departmentCode>`
  - area role: `area:<processAreaCode>`
  - explicit roles: conforme `roleCodes`
  - explicit users: `subject_type=user` com `userId`

Defaults de policy:
- Sempre garantir `owner` tem `document.view` e `document.edit`.
- `admin` role sempre tem todas capabilities.
- Para `audience.mode=DEPARTMENT`: adicionar allow de `document.view` para `dept:<departmentCode>`.
- Para `audience.mode=AREAS`: allow `document.view` para `dept:<departmentCode>` e `area:<processAreaCode>` selecionados.
- Para `audience.mode=EXPLICIT`: allow `document.view` para `roleCodes/userIds`.

Aceite:
- Criar documento `CONFIDENTIAL/RESTRICTED` resulta em policies persistidas por `ResourceScopeDocument`.
- Usuario sem role/allow nao consegue `GET /documents/{id}` (retorna `DOC_NOT_FOUND` por design atual).
- Usuario com role apropriado consegue visualizar.
- Testes:
  - unit: policy building (audience -> policies)
  - integration: enforcement na rota de `GetDocument` e `ListDocuments` (ao menos smoke).

Entrega:
- `CreateDocument` gera policies por documento quando `audience` exige.
- Repositorio Postgres/Memoria suportam create atomico com policies.

## Task 052 - Departments registry + role conventions for access
Status: `done`

Objetivo:
Parar de tratar `department` como string livre e criar uma registry canonica para:
1) padronizar UI (dropdown)
2) padronizar roles `dept:<code>` e defaults de audience.

Escopo:
- DB migration:
  - tabela `metaldocs.document_departments`:
    - `code` TEXT PK
    - `name` TEXT NOT NULL
    - `description` TEXT NOT NULL DEFAULT ''
    - `is_active` BOOLEAN NOT NULL DEFAULT TRUE
    - timestamps
  - seed inicial Metal Nobre (exemplos): `quality`, `operations`, `commercial`, `finance`, `logistics`
- Backend:
  - endpoints CRUD admin-only (pode ser faseado: list/read primeiro).
  - validação: code lowercase, name trimmed.
- IAM/Admin:
  - orientar que usuarios recebam roles `dept:<code>` (via tela de roles existente) para viabilizar enforcement.

Aceite:
- Create document usa dropdown de departamentos.
- Roles `dept:<code>` reconhecidas pelo enforcement de `Task 051`.

Entrega:
- registry `document_departments` com endpoints admin.
- dropdown de departamentos na criacao de documentos.

## Task 053 - UI: Access selector tied to classification (Create Document)
Status: `done`

Objetivo:
Evoluir a UX de classificacao para um padrao profissional:
1) classificacao com explicacao rica
2) quando CONFIDENTIAL/RESTRICTED, exibir seletor de audiencia
3) defaults seguros e feedback claro.

Escopo frontend:
- `DocumentCreateContentStep`:
  - manter chips (PUBLIC/INTERNAL/CONFIDENTIAL/RESTRICTED) com copy melhor.
  - ao escolher CONFIDENTIAL/RESTRICTED, mostrar bloco "Quem pode ver":
    - modo: Departamento / Areas / Explicito (pode iniciar apenas Departamento)
    - selecao de departamento(s) (multi-select) e/ou areas (process areas)
  - nao aplicar regra no frontend: apenas montar payload `audience` do contrato.

Escopo backend (suporte):
- `CreateDocumentRequest` aceita `audience`.
- backend calcula defaults quando `audience` omitido (para manter compatibilidade).

Aceite:
- Criar doc CONFIDENTIAL sem mexer em nada gera audience default seguro (departamento).
- UI deixa explicito que permissao e aplicada pelo backend (sem promessa falsa).
- Nenhum hardcode de "departamentos/areas" fora da API.

Entrega:
- seletor de audiencia condicionado a CONFIDENTIAL/RESTRICTED.
- payload `audience` enviado no create.

## Task 054 - Enforce dept AND area access (compound policies)
Status: `done`

Objetivo:
Permitir regra AND real entre departamento e area (`dept ∧ area`) para classificacao Restrito, evitando a permissao por OR do modelo atual.

Contexto:
Hoje as policies sao avaliadas por uniao (OR). Se gravarmos `dept:<d>` e `area:<a>`, qualquer usuario com um dos dois entra. Precisamos de um conceito composto para exigir a combinacao.

Escopo:
- ADR nova descrevendo abordagem:
  - opcao A: role composta `dept:<d>:area:<a>`
  - opcao B: "group" com membership explicitando dept+area
  - opcao C: policy condition (ABAC) com atributos `department` e `process_area` (mais longo prazo)
- Contrato:
  - manter `audience.mode = AREAS` mas gerar `compoundRoleCodes` ou `groupIds` no backend.
- Backend:
  - gerar policies usando o conceito composto (nao gerar `dept:<d>` para Restrito).
  - garantir compatibilidade com `decidePolicies` atual.
- Admin/ops:
  - fluxo para atribuir roles compostas ou membership de grupos para usuarios.
- UI:
  - manter "Areas do departamento" mas sinalizar que apenas quem esta no grupo/role composto tera acesso.

Aceite:
- Documento Restrito com dept+area permite acesso somente para usuarios que possuem a combinacao.
- Usuario que tem apenas dept OU apenas area nao acessa.
- Tests cobrindo a avaliacao AND no fluxo de `GetDocument`.

## Task 055 - Content authoring ADRs (modes + storage)
Status: `done`

Objetivo:
Congelar decisoes de produto e arquitetura para autoria de conteudo (native + docx) e persistencia em `document_versions`, antes de alterar schema/contratos.

Entregaveis:
- ADR: `docs/adr/0012-content-authoring-modes-and-carbone.md`
- ADR: `docs/adr/0013-document-version-content-storage-and-search.md`

Aceite:
- ADRs revisadas e aceitas (Status: Accepted).
- Fluxo nao viola versionamento imutavel e policy de migrations (ADR-0007).

## Task 056 - Infra: Carbone service in compose + config
Status: `done`

Escopo:
- Adicionar servico `carbone` em `deploy/compose/docker-compose.yml` (porta interna 4000).
- Criar pastas versionadas para templates (ex: `carbone/templates/`) e renders (gitignored).
- Adicionar env `METALDOCS_CARBONE_API_URL=http://carbone:4000` no `api`.
- Runbook dev: como subir e validar o health do Carbone.

Aceite:
- `docker compose up` sobe `carbone` junto do stack sem regressao.
- API enxerga `METALDOCS_CARBONE_API_URL` (smoke log/healthcheck dedicado).

## Task 057 - Schema: extend document_versions for content + pdf/docx + FTS
Status: `done`

Escopo:
- Migration additive em `migrations/` adicionando colunas:
  - `content_source`, `native_content`, `docx_storage_key`, `pdf_storage_key`, `text_content`, `file_size_bytes`, `original_filename`, `page_count`.
  - `search_vector` gerado + indice GIN.
- Backfill seguro:
  - versoes existentes: `content_source='native'`, `text_content=content` quando aplicavel.

Aceite:
- Migrations rodam em ambiente novo e existente.
- Nenhuma coluna existente e removida.

## Task 058 - Contract: OpenAPI content endpoints + schemas
Status: `done`

Escopo:
- Atualizar `api/openapi/v1/openapi.yaml` com endpoints:
  - Native: `GET /documents/{id}/content/native`, `POST /documents/{id}/content/native` (cria nova versao).
  - Render: `POST /documents/{id}/content/render-pdf`, `GET /documents/{id}/content/pdf`.
  - DOCX: `GET /documents/{id}/template/docx`, `POST /documents/{id}/content/upload`, `GET /documents/{id}/content/docx`.
  - Profile blank template: `GET /document-profiles/{profileCode}/template/docx` (opcional MVP).
- Definir erros padrao: 400 invalid, 401/403 authz, 404 not found, 413 file too large, 415 unsupported media type.

Aceite:
- Contract tests passam.
- Sem breaking change: novos campos como opcionais quando necessario.

## Task 059 - Backend: Carbone client + template registry bootstrap
Status: `done`

Escopo:
- Criar client `CarboneService` (wrapper REST) com:
  - register template (multipart)
  - render template (data + convertTo)
  - download render
  - timeouts e logs com traceId
- Bootstrap: no start do API, registrar templates do repo (idempotente) e manter map `profileCode -> templateId`.

Aceite:
- Unit tests do client com server fake.
- Bootstrap nao falha o start se Carbone estiver indisponivel: falha controlada + health indicando degradacao.

## Task 060 - Backend: content flows (native + docx upload) creating versions
Status: `done`

Escopo:
- Native authoring:
  - Receber `native_content` (JSON) -> criar nova `document_version` (append-only).
  - Extrair `text_content` (string) para busca.
  - Renderizar PDF via Carbone e persistir `pdf_storage_key`.
- DOCX upload:
  - Validar magic bytes DOCX (ZIP `PK`), size limit.
  - Persistir DOCX (`docx_storage_key`) e converter para PDF via Carbone (`pdf_storage_key`).
  - Extrair texto do `word/document.xml` para `text_content`.
- Enforcements:
  - view endpoints exigem `document.view`
  - write/upload exigem `document.edit` (ou capability dedicada, se existir)

Aceite:
- Criar versao via native/docx resulta em PDF acessivel por URL assinada.
- `document_versions` permanece append-only.

## Task 061 - Frontend: Content mode selector + DOCX upload + PDF viewer
Status: `done`

Escopo MVP:
- Novo bloco "Conteudo" (step 5) com:
  - seletor de modo (native vs docx upload)
  - upload DOCX (drag/drop) + status + erro
  - viewer PDF (react-pdf) para PDF final gerado pelo backend
- Native editor MVP:
  - Comecar com um editor simples por profile (textarea/fields basicos) apenas para salvar JSON e gerar PDF.
  - Campos tipados por secoes vira Task futura (nao bloquear MVP).

Aceite:
- Usuario consegue gerar e visualizar PDF final no app em ambos os modos.
- UI nao hardcoda regra de negocio: apenas monta payload do contrato.

## Task 062 - Search: use version text_content/search_vector
Status: `todo`

Escopo:
- Ajustar search reader/query para usar `search_vector` quando presente.
- Definir se busca usa "ultima versao" apenas (MVP) ou todas (v2).

Aceite:
- Busca retorna documentos por texto do conteudo (modo native/docx).

## Task 063 - Frontend: Align Step 5 "Conteudo" UX to reference (mode cards + DOCX stepper)
Status: `todo`

Objetivo:
Alinhar a experiencia do Step 5 ("Conteudo do documento") ao mockup de referencia, mantendo o shell atual e sem mover regra de negocio para o frontend.

Escopo:
- Ajustar Step 5 para ter:
  - "Conteudo do documento" com seletor de modo em 2 cards (Native vs Word/.docx).
  - Fluxo DOCX com stepper vertical (3 passos) e estados: idle, downloading, ready, uploading, processing, preview.
  - Preview do PDF dentro do Step 5 para o modo DOCX (quando existir `pdfUrl`).
- Para modo Native:
  - Remover textarea "Conteudo (MVP)" do create flow.
  - Mostrar CTA claro: "Abrir editor de conteudo" (habilitado somente depois de criar o documento) e explicar que o editor e uma tela dedicada.
- Refatorar o Step 5 em componentes menores e reaproveitaveis (widget local de feature, sem promover para lib global antes de 3 usos).
- Garantir layout e microcopy consistentes com a linguagem MetalDocs (sem dependencias externas de fonte/CDN).

Aceite:
- Step 5 visualmente proximo do HTML de referencia: cards, stepper, dropzone, preview.
- Fluxo DOCX funciona end-to-end usando o contrato existente (download template, upload, preview PDF).
- Modo Native nao "parece um campo solto": direciona para o editor dedicado.

## Task 064 - Frontend: ContentBuilderView (Native) with split editor/preview layout
Status: `todo`

Objetivo:
Criar uma tela dedicada de autoria nativa (Modo A) no padrao do mockup: header com contexto do documento + editor por secoes + rodape fixo + preview de PDF em painel lateral recolhivel.

Escopo:
- Nova rota/view (ex: `DocumentContentBuilderView`) acessivel a partir do documento criado.
- Layout:
  - Header: breadcrumb, codigo/titulo/status, badges de versao/ultima geracao.
  - Corpo split: Editor (esquerda) | Preview PDF (direita) com recolher/expandir.
  - Footer fixo com acoes: "Salvar" (cria nova versao) e "Gerar PDF" (re-render do ultimo conteudo salvo).
- Estado:
  - state machine via `useReducer` para loading/dirty/saving/rendering/error.
  - UI lida com expiracao de URL assinada (recarrega ao abrir preview quando expirado).
- Integracao:
  - Carregar conteudo atual via `GET /documents/{id}/content/native`.
  - Salvar via `POST /documents/{id}/content/native` (append-only).
  - Preview server-side via `GET /documents/{id}/content/pdf`.
  - Rerender via `POST /documents/{id}/content/render-pdf` quando usuario solicitar.

Aceite:
- Editor nativo abre como "pagina oficial" (nao como card dentro do create view).
- Salvar cria versao nova e o preview mostra o PDF correspondente.
- Painel de preview pode ser recolhido sem quebrar layout.

## Task 065 - Backend: Freeze render semantics for "Salvar" vs "Gerar PDF" (contracts + invariants)
Status: `todo`

Objetivo:
Congelar o comportamento exato de renderizacao para suportar a UX do builder sem violar versionamento imutavel.

Escopo:
- Definir e implementar (conforme ADRs 0012/0013):
  - Quando `POST /content/native` deve (ou nao) renderizar automaticamente PDF.
  - O que `POST /content/render-pdf` faz: re-render da ultima versao (sem alterar o JSON) e como isso e persistido sem reescrever historico indevido.
- Garantir que o PDF mostrado por `GET /content/pdf` tenha vinculacao clara com uma versao (mesmo que seja artefato derivado).
- Ajustar OpenAPI (se necessario) para refletir a semantica final (ex: incluir `versionId`/`versionNumber` no response).
- Tests de invariantes:
  - Append-only: conteudo e versoes nao sao sobrescritos.
  - Render nao cria versao silenciosamente.

Aceite:
- Comportamento do builder (Salvar x Gerar PDF) e consistente, testado e documentado.
- Nenhuma operacao permite update de uma versao existente.

## Task 066 - Domain/Registry: Content schema per profile (server as source of truth)
Status: `todo`

Objetivo:
Tornar o editor nativo orientado a schema por profile sem hardcode de estrutura no frontend, mantendo o dominio (backend) como fonte de verdade.

Escopo:
- Introduzir um contrato de `content_schema` versionado por profile (PO/IT/RG/FM):
  - estrutura de secoes
  - campos tipados (text, textarea, enum, array, table, checklist)
  - labels/descriptions humanizadas (pt-BR)
  - defaults e obrigatoriedade quando aplicavel
- Persistencia: armazenar schema junto do registry (ex: tabela de schemas por profile) com seeds para Metal Nobre.
- API: expor schema no endpoint de profiles (ou endpoint dedicado) sem breaking change.
- Validacao backend: validar payload `native_content` contra o `content_schema` do profile ativo.

Aceite:
- Frontend consegue montar o editor sem mapa `profile -> fields` hardcoded.
- Payload invalido falha com erro de dominio consistente.

## Task 067 - Frontend: Schema-driven native editor widgets (sections, arrays, tables)
Status: `todo`

Objetivo:
Implementar um renderer de formulario baseado em schema (Task 066) com qualidade de UX semelhante ao mockup e sem duplicacao de componentes.

Escopo:
- Widgets de campo:
  - text/textarea/select
  - array de strings (add/remove)
  - tabela editavel (add/remove row)
  - checklist (boolean + label)
- Seções expansiveis (accordion) por `sectionKey`.
- Controles de erro/required baseado no schema e no retorno do backend.
- Output: produzir `native_content` estruturado exatamente no formato esperado pelo backend.

Aceite:
- PO/IT/RG/FM renderizam secoes e campos principais de forma editavel.
- Add/remove de itens funciona sem quebrar o JSON.
- Sem regra de negocio no frontend: apenas interpretacao de schema.

## Task 068 - Templates: DOCX master assets + runbook (Carbone)
Status: `todo`

Objetivo:
Padronizar e operacionalizar os templates master `.docx` (design assets) usados pelo Carbone para render e export, com governanca e reproducibilidade.

Escopo:
- Pastas canonicas no repo:
  - `carbone/templates/` (versionado) com templates master por profile.
  - `carbone/renders/` (gitignored) para saida local.
- Runbook:
  - como editar templates no Word/LibreOffice
  - como testar render localmente
  - convencoes de placeholders Carbone
- Bootstrap:
  - registrar templates na inicializacao do API (idempotente) e mapear `profileCode -> templateId`.

Aceite:
- Equipe consegue atualizar um template e validar render sem alterar codigo.
- Templates ficam versionados como ativos de design, nao como dados runtime.

## Task 069 - Repo Hygiene: Clean working tree + ignore artifacts
Status: `todo`

Objetivo:
Eliminar ruido operacional (artefatos locais) e garantir baseline limpo para refactors e hardening, evitando drift e commits acidentais.

Motivacao:
- Hoje existem artefatos locais frequentes (`.tmp/`, `frontend/apps/web/test-results/`) e mudancas pendentes no frontend que podem mascarar regressao.

Escopo:
- `.gitignore`:
  - ignorar `.tmp/`
  - ignorar `frontend/apps/web/test-results/`
  - (se aplicavel) ignorar `frontend/apps/web/playwright-report/` e `frontend/apps/web/.cache/`
- Resolver mudancas pendentes do frontend em commits pequenos e sem misturar refactor amplo:
  - `frontend/apps/web/src/App.tsx`: padronizar notificacoes (toast) sem banner que empurra layout.
  - `frontend/apps/web/tests/e2e/auth-smoke.spec.ts`: estabilizar ou reduzir escopo para nao flake.
- Documentar comando de smoke minimo no `docs/runbooks/` (1 pagina):
  - build frontend
  - subir compose
  - exercitar fluxo: login -> criar doc -> abrir editor -> salvar -> gerar PDF

Execucao (como fazer):
- Criar commit 1 apenas para `.gitignore` e runbook (sem codigo).
- Criar commit 2 apenas para `App.tsx` (UI/toast) com verificacao visual.
- Criar commit 3 apenas para `auth-smoke.spec.ts` (se necessario), ou reverter se nao for confiavel agora.

Aceite:
- `git status` limpo apos rodar testes/build local.
- Nenhum artefato local volta a aparecer como untracked.
- Notificacoes nao deslocam layout (toast overlay).

## Task 070 - Contracts/Domain: Dedicated error for invalid native content
Status: `todo`

Objetivo:
Trocar falhas de validacao de `native_content` de erro generico para um erro de dominio especifico, com codigo API estavel e observavel.

Escopo:
- Domain:
  - adicionar `ErrInvalidNativeContent` (ou `ErrInvalidContent`) em `internal/modules/documents/domain/errors.go`.
  - garantir que validacao de schema (Task 066) retorne esse erro (nao `ErrInvalidCommand`).
- Delivery:
  - mapear o erro para HTTP 400 com code dedicado (ex: `INVALID_NATIVE_CONTENT`) em `internal/modules/documents/delivery/http/handler.go`.
- Contracts:
  - atualizar `docs/contracts/INTERNAL_EVENTS_AND_ERRORS.md` adicionando `INVALID_NATIVE_CONTENT` ao catalogo.
  - se existir catalogo de erros por modulo, manter sincronizado.
- Observabilidade:
  - logar o `trace_id` e contexto minimo (documentId, profileCode, schemaVersion) sem vazar payload completo.

Execucao (como fazer):
- Alterar service para retornar erro especifico na validacao.
- Atualizar o mapeamento de erro no handler.
- Validar que o frontend recebe o `code` e exibe mensagem amigavel, sem banner persistente.

Aceite:
- Payload invalido gera 400 com `error.code=INVALID_NATIVE_CONTENT`.
- Payload valido continua criando nova versao e renderizando PDF.
- Sem breaking change em endpoints.

## Task 071 - Backend Maintainability: Split large documents files by sub-area
Status: `todo`

Objetivo:
Reduzir risco de regressao e custo de manutencao quebrando arquivos "god" em unidades menores, mantendo o mesmo desenho de arquitetura (vertical slice) e sem mudar contratos.

Escopo:
- `internal/modules/documents/application/service.go`:
  - mover funcoes/metodos para arquivos por sub-area no mesmo package `application`, por exemplo:
    - `service_content_native.go` (Get/Save native, render PDF)
    - `service_content_docx.go` (upload docx, extract text, convert PDF)
    - `service_registry.go` (profiles/schemas/governance/taxonomy)
    - `service_collaboration.go` (presence/locks)
    - `service_policies.go` (audience/policies helpers)
    - `service_helpers.go` (helpers puros)
- `internal/modules/documents/delivery/http/handler.go`:
  - manter `Handler` e `routes` estaveis, mas separar handlers e DTOs por area:
    - `handler_documents.go`, `handler_content.go`, `handler_profiles.go`, `handler_taxonomy.go`, `handler_collab.go`, `handler_attachments.go`
  - regra: sem mover logica de negocio para o handler; apenas parse/response.

Guardrails:
- Nao criar novos packages agora; apenas dividir em multiplos `.go` no mesmo package para diff pequeno.
- Sem mudanca de comportamento (refactor-only).
- Sem mudar nomes de rotas/paths.

Execucao (como fazer):
- Refactor em commits pequenos por area (content -> registry -> collab), com `go test ./...` a cada commit.

Aceite:
- `go test ./...` passa.
- Nao muda OpenAPI.
- Arquivos resultantes ficam < ~400-500 linhas por arquivo, com responsabilidades claras.

## Task 072 - Frontend Maintainability: Extract schema-driven widgets + reuse create widgets
  Status: `todo`

Objetivo:
Evitar repeticao e facilitar evolucao do editor nativo orientado a schema, com widgets reutilizaveis e padrao visual unico.

Escopo:
- Extrair renderer do schema em componentes menores:
  - `components/content-builder/ContentSchemaForm.tsx`
  - `components/content-builder/ContentSectionAccordion.tsx`
  - `components/content-builder/widgets/*` (TextField, TextAreaField, SelectField, ArrayField, TableField, ChecklistField)
- Reuso:
  - alinhar estilos e foco com widgets existentes em `frontend/apps/web/src/components/create/widgets/` quando fizer sentido (inputs, DateTimeField, etc.).
- Tipagem:
  - consolidar tipos de schema (Section/Field/Column) em um arquivo unico (ex: `contentSchemaTypes.ts`) e usar em toda a UI.
- UX:
  - required indicator consistente
  - estados de erro: ao receber `INVALID_NATIVE_CONTENT`, mostrar toast e destacar a secao com problema (sem logica de negocio).

Execucao (como fazer):
- Primeiro refactor-only (split do arquivo atual) sem mudar comportamento.
- Depois ajustes de reuso e tipagem.
- Commit por subpasso para reduzir risco.

  Aceite:
  - Editor continua funcionando para PO/IT/RG/FM.
  - Nenhum `fetch()` fora de `lib.api.ts`.
  - Componentes nao duplicam estilos/markup de inputs (uso de widgets padrao).

## Task 079 - Performance Diagnosis: Backend latency instrumentation
Status: `todo`

Objetivo:
Medir tempo real de resposta (TTFB/p95/p99) dos endpoints criticos ligados a criacao e edicao de documentos.

Escopo:
- Instrumentar logs/metrics em:
  - `listDocumentProfiles`
  - `listDocumentProfileSchemas`
  - `getDocumentContentNative`
  - `getDocumentContentPdf`
  - `createDocument`
  - `listProcessAreas`
  - `listSubjects`
- Registrar: `duration_ms`, `trace_id`, `route`, `status`, `user_id`, `document_id/profile_code` quando aplicavel.

Execucao (como fazer):
- Centralizar metricas no middleware HTTP (quando possivel).
- Adicionar logs detalhados apenas nos handlers criticos para evitar ruido.

Aceite:
- Logs mostram p50/p95/p99 com dados suficientes para identificar gargalos.

## Task 080 - Performance Diagnosis: Frontend request waterfall tracing
Status: `todo`

Objetivo:
Mapear quantas chamadas sao disparadas e em que ordem quando:
1) troca o tipo documental no create
2) abre o editor nativo

Escopo:
- Instrumentar `lib.api.ts` com logs `start/end` por request (apenas em dev).
- Salvar os tempos em uma tabela simples no console para analise.

Execucao (como fazer):
- Adicionar wrapper no `api` para medir `performance.now()`.
- Usar `console.groupCollapsed` para agrupar por acao do usuario.

Aceite:
- Relatorio mostrando cadeia de chamadas e tempo total por acao.

## Task 081 - Performance Diagnosis: Perceived latency markers
Status: `todo`

Objetivo:
Medir o tempo percebido do usuario desde a acao ate o primeiro render util.

Escopo:
- Criar marcadores `performance.now()` em:
  - clique em trocar profile
  - schema carregado
  - form renderizado
  - editor pronto

Execucao (como fazer):
- Adicionar logs condicionais em dev para cada marco.

Aceite:
- Relatorio com tempos reais de UX (ms).

## Task 082 - Performance Diagnosis: DB query analysis
Status: `todo`

Objetivo:
Identificar se o gargalo esta no banco (queries lentas).

Escopo:
- Executar `EXPLAIN ANALYZE` nas queries de schema/profile/metadata.
- Verificar indices em tabelas de profiles, schemas e versions.

Execucao (como fazer):
- Mapear SQL exato que o repo executa.
- Rodar explain com dados reais.

Aceite:
- Lista de queries lentas com sugestoes de indice.

## Task 083 - Performance Diagnosis: Concurrency test (light load)
Status: `todo`

Objetivo:
Ver como a latencia se comporta com 5-20 requests concorrentes.

Escopo:
- Script simples (curl loop ou k6) para endpoints criticos.
- Medir degradacao de p95/p99.

Execucao (como fazer):
- Rodar teste local em ambiente dev.

Aceite:
- Dados de latencia sob concorrencia leve.

## Task 084 - Performance Diagnosis: Editor flow isolation
Status: `todo`

Objetivo:
Separar custo de render do custo de fetch.

Escopo:
- Rodar editor com mock local (sem fetch) e comparar tempo.
- Medir render inicial e re-render.

Execucao (como fazer):
- Criar flag `DEV_MOCK_SCHEMA` (somente dev) para injetar schema local.

Aceite:
- Comparativo claro entre custo de render e custo de fetch.

## Task 085 - Performance Diagnosis: External dependencies
Status: `todo`

Objetivo:
Verificar se chamadas externas (Carbone/storage) afetam o load.

Escopo:
- Trace de chamadas para Carbone e storage durante editor/load.
- Medir tempos de cada chamada.

Execucao (como fazer):
- Log de duracao nas funcoes de client Carbone e storage.

Aceite:
- Relatorio de tempos por dependencia externa.

## Task 086 - Performance Diagnosis: Consolidated report
Status: `todo`

Objetivo:
Consolidar dados em um relatorio final com causas e prioridades.

Escopo:
- Documento com:
  - endpoints mais lentos
  - UX latency
  - waterfall
  - DB
  - dependencias externas

Aceite:
- Relatorio com ranking de gargalos e impacto.

## Task 087 - Performance Diagnosis: Optimization candidates shortlist
Status: `todo`

Objetivo:
Definir as 3-5 intervencoes mais impactantes antes de codar melhorias.

Escopo:
- Priorizar entre:
  - cache/swr frontend
  - prefetch
  - batching de endpoints
  - indices DB
  - ajuste de payloads

Aceite:
- Lista priorizada com justificativa de impacto.
