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
Status: `in_progress`

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

## Task 037 - Realtime event stream for operations center
Status: `pending`

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

## Task 038 - Collaborative editing and presence
Status: `pending`

Objetivo:
Evoluir a plataforma para colaboracao documental em tempo real quando isso realmente virar necessidade de produto.

Escopo:
- presence
- locking/conflito de edicao
- sinais colaborativos
- co-authoring

Saida:
- base para experiencia colaborativa mais profunda sem improviso arquitetural

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
