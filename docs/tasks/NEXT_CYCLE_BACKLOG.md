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
Status: `todo`

Objetivo:
Subir a maturidade operacional da plataforma apos o ciclo de taxonomia documental e superficies operacionais.

Escopo:
- observabilidade mais profunda
- readiness de producao
- indicadores de auth/session/worker
- evidencias de operacao segura

Saida:
- baseline mais proxima de release-grade

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
