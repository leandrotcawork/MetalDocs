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
