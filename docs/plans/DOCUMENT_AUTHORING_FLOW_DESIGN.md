# Document Authoring Flow Design

## Status
- Task: `033`
- Status: `done`
- Phase: `document authoring and UX preparation`

## Objective
Congelar o modelo oficial de criacao, edicao e consulta documental antes da fase forte de UI/UX.

Este documento define:
- o fluxo profile-first de authoring
- a separacao entre estrutura, metadata, governanca e versionamento
- os modos de tela esperados
- o contrato mental que o frontend deve seguir

## Design Principles

### 1. Profile-first authoring
Toda experiencia de authoring comeca por `documentProfile`, nao por `documentType` hardcoded.

Regra:
- `documentType` permanece apenas como alias de transicao
- a UI deve operar por `documentProfile`
- `documentFamily` e derivada do profile

### 2. Metadata-driven, not folder-driven
Documento e identificado pelo que ele e, pelo contexto onde opera e pelos metadados validados.

Regra:
- pasta pode existir futuramente como visao derivada
- nunca como source of truth do documento

### 3. Immutable versioning
Conteudo e historico nao devem ser sobrescritos silenciosamente.

Regra:
- criacao inicial pode gerar `version 1`
- edicao de conteudo gera nova versao imutavel
- consulta deve sempre poder mostrar historico e diff

### 4. Governance is visible, not hidden
A governanca do profile deve ficar clara para o usuario durante authoring.

Regra:
- workflow
- review cadence
- approval requirement
- retention
- validity
devem aparecer como contexto operacional na UI

### 5. Structure and metadata are different concerns
Frontend nao deve misturar corpo do documento com campos classificatorios.

Regra:
- conteudo principal vive em area de authoring
- metadata vive em painel estruturado
- governanca vive em painel de contexto
- versoes vivem em timeline/historico

## Core Authoring Model

O authoring do documento deve ser organizado em quatro blocos principais.

### A. Document Structure
Representa o documento em si.

Campos/foco:
- `title`
- `initialContent` no create
- `content` no add-version/edit flow

Responsabilidade:
- definir corpo principal do documento
- permitir leitura/edicao clara do texto base

### B. Document Metadata
Representa classificacao e contexto.

Campos:
- `documentProfile`
- `documentFamily` (readonly derivado)
- `processArea`
- `subject`
- `ownerId`
- `businessUnit`
- `department`
- `classification`
- `tags`
- `effectiveAt`
- `expiryAt`
- `metadata` validada por schema do profile

Responsabilidade:
- classificar o documento
- torná-lo pesquisavel
- garantir compatibilidade com o schema ativo

### C. Governance Context
Representa regras operacionais do profile.

Campos/indicadores:
- `workflowProfile`
- `reviewIntervalDays`
- `approvalRequired`
- `retentionDays`
- `validityDays`
- `profileSchemaVersion`

Responsabilidade:
- informar o usuario
- contextualizar o documento
- orientar o fluxo, sem ficar escondido em tela secundaria

### D. Versioning and Operational History
Representa evolucao e prova operacional.

Superficies:
- lista de versoes
- diff entre versoes
- approvals
- attachments
- audit timeline
- access policies

Responsabilidade:
- mostrar como o documento evoluiu
- tornar revisao e auditoria acessiveis

## Official Authoring Modes

### 1. Create mode
Objetivo:
- criar um documento novo e produzir a versao inicial

Fluxo:
1. selecionar `documentProfile`
2. carregar `documentFamily`, schema ativo e governance
3. preencher metadata base
4. preencher conteudo inicial
5. criar documento com `initialContent`

Regras:
- `documentProfile` e obrigatorio
- metadata obrigatoria deve ser validada pelo schema ativo
- `initialContent` gera a primeira versao

### 2. Edit metadata mode
Objetivo:
- ajustar classificacao e contexto do documento sem necessariamente criar uma nova versao de conteudo

Escopo:
- `processArea`
- `subject`
- `ownerId`
- `businessUnit`
- `department`
- `classification`
- `tags`
- `effectiveAt`
- `expiryAt`
- metadata do schema

Regra:
- esse modo nao deve ser confundido com alteracao do corpo principal

### 3. Edit content mode
Objetivo:
- criar nova versao do documento com novo conteudo

Fluxo:
1. abrir documento existente
2. editar conteudo
3. informar `changeSummary`
4. criar nova versao via `/documents/{documentId}/versions`

Regra:
- edicao de conteudo nao sobrescreve versao existente
- sempre gera nova versao imutavel

### 4. Review mode
Objetivo:
- apoiar revisao operacional e workflow

Superficies:
- versoes
- diff
- approvals
- audit
- anexos

Regra:
- revisar deve ser experiencia de comparacao e decisao, nao de edicao livre

### 5. Read mode
Objetivo:
- consultar documento com contexto completo

Superficies:
- resumo estrutural
- metadata
- governanca
- versao atual
- historico operacional

## Recommended Screen Structure

## A. Authoring shell
Area principal do fluxo documental.

Blocos visuais:
- header do documento
- perfil/gov summary
- editor de conteudo
- formulario de metadata
- barra de acoes

## B. Sidebar or context rail
Painel lateral persistente com:
- family
- profile
- process area
- schema version
- workflow profile
- review cadence
- retention/validity

## C. Bottom or secondary tabs
Abas/paineis secundarios para:
- versions
- diff
- approvals
- attachments
- policies
- audit timeline

## Field Matrix by Experience

### Create
Campos principais:
- `title`
- `documentProfile`
- `processArea`
- `subject`
- `ownerId`
- `businessUnit`
- `department`
- `classification`
- `tags`
- `effectiveAt`
- `expiryAt`
- metadata dinamica por schema
- `initialContent`

Read-only derivados:
- `documentFamily`
- `workflowProfile`
- `reviewIntervalDays`
- `approvalRequired`
- `retentionDays`
- `validityDays`
- `activeSchemaVersion`

### Edit metadata
Editaveis:
- `title`
- `processArea`
- `subject`
- `ownerId`
- `businessUnit`
- `department`
- `classification`
- `tags`
- `effectiveAt`
- `expiryAt`
- metadata dinamica

Travados/derivados:
- `documentProfile`
- `documentFamily`
- governance do profile
- `profileSchemaVersion` do snapshot ja persistido

### Edit content
Editaveis:
- conteudo textual
- `changeSummary`

Contexto visivel:
- versao atual
- profile/family
- governanca
- diff potencial

### Consult
Visivel:
- identificacao do documento
- metadata
- governanca
- versao atual
- timeline de versoes
- approvals
- anexos
- access policies
- audit timeline

## API Mapping (Current Platform Capabilities)

### Registry/context
- `GET /api/v1/document-families`
- `GET /api/v1/document-profiles`
- `GET /api/v1/document-profiles/{profileCode}/schema`
- `GET /api/v1/document-profiles/{profileCode}/governance`
- `GET /api/v1/process-areas`
- `GET /api/v1/document-subjects`

### Documents
- `POST /api/v1/documents`
- `GET /api/v1/documents`
- `GET /api/v1/documents/{documentId}`
- `GET /api/v1/search/documents`

### Versioning
- `POST /api/v1/documents/{documentId}/versions`
- `GET /api/v1/documents/{documentId}/versions`
- `GET /api/v1/documents/{documentId}/versions/diff`

### Operational side surfaces
- `GET /api/v1/workflow/documents/{documentId}/approvals`
- `POST /api/v1/workflow/documents/{documentId}/transitions`
- `GET /api/v1/documents/{documentId}/attachments`
- `POST /api/v1/documents/{documentId}/attachments`
- `GET /api/v1/access-policies`
- `PUT /api/v1/access-policies`
- `GET /api/v1/audit/events`

## What the Frontend Must Not Do
- nao hardcodar `PO`, `IT`, `RG` como logica fixa de plataforma
- nao misturar `processArea` com `documentFamily`
- nao tratar metadata JSON como textarea final da experiencia madura
- nao esconder governanca em modal secundaria
- nao sobrescrever conteudo existente sem criar nova versao
- nao usar pasta/hierarquia como identidade principal do documento

## Guidance for Next Tasks

### Task 034 - Document Workspace UX
Deve transformar este modelo em experiencia visual clara:
- profile-first
- metadata dinamica
- governanca sempre visivel
- historico operacional navegavel

### Task 035 - Metal Nobre Applied Experience
Deve aplicar este modelo para:
- `po`
- `it`
- `rg`
- `marketplaces`
- `quality`
- `commercial`
- `purchasing`
- `logistics`
- `finance`

## Exit Criteria
- modelo de authoring congelado por documento
- separacao clara entre estrutura, metadata, governanca e versionamento
- frontend design pode evoluir sem redefinir o dominio
- proxima fase de UX parte de um contrato conceitual estavel
