# Document Workspace UX Architecture

## Status
- Task: `034`
- Status: `planned`
- Source: alignment between product direction, Task 033 authoring model, and design exploration

## Objective
Traduzir o modelo de authoring e operacao documental em uma experiencia de frontend de alto nivel sem quebrar o dominio ja congelado.

Este documento existe para separar claramente:
- o que entra na `Task 034`
- o que fica explicitamente fora da `Task 034`
- quais capacidades futuras precisam de tasks dedicadas

## UX Direction

O frontend documental deve ser organizado em camadas claras, evitando uma tela unica monolitica e evitando misturar:
- authoring
- consulta
- operacao
- administracao de registry

## Proposed Screen Map

### Layer 1. App Shell
Responsabilidade:
- navegacao principal
- contexto global
- identidade visual do produto
- entrada para operacao e authoring

Telas/paineis:
- app shell
- top navigation / workspace navigation

### Layer 2. Operations Center
Responsabilidade:
- oferecer visao executiva e operacional do estado documental
- destacar trabalho pendente e sinais de atencao

Telas/paineis:
- operations center dashboard
- approvals / pending reviews snapshot
- notifications snapshot
- expirations / review cadence snapshot

Observacao:
- esta camada deve nascer preparada para atualizacao frequente
- mas a `Task 034` nao depende de SSE/WebSocket para ficar correta

### Layer 3. Document Workspace
Responsabilidade:
- navegação do acervo
- consulta detalhada
- leitura operacional completa

Telas/paineis:
- document catalog
- document detail
- review pane

Areas dentro do detalhe:
- metadata summary
- governance summary
- versions
- diff
- approvals
- attachments
- policies
- audit timeline

### Layer 4. Authoring
Responsabilidade:
- criar e evoluir documento de forma profile-first

Tela/painel central:
- document editor wizard

Etapas do wizard:
1. `Profile`
2. `Metadata`
3. `Content`
4. `Review`

Regras:
- profile define family, schema e governance
- metadata e dinamica por schema
- conteudo principal e tratado separadamente da metadata
- review deve consolidar tudo antes de salvar/criar versao

### Layer 5. Registry and Administration
Responsabilidade:
- expor o modelo configuravel que sustenta a experiencia profile-first

Telas/paineis:
- registry explorer
- profile detail
- schema detail
- governance detail
- process areas
- subjects

Regra importante:
- na `Task 034`, essa camada nasce como **explorer/consulta forte**
- CRUD completo de registry fica para task posterior

## Official Scope for Task 034

### In scope
- app shell documental coerente
- operations center documental
- document catalog
- document detail / review
- document authoring wizard
- registry explorer em modo leitura
- metadata dinamica por schema
- governanca sempre visivel
- layout preparado para crescer sem refazer a arquitetura

### Explicitly out of scope
- SSE como requisito obrigatorio
- WebSocket como requisito obrigatorio
- colaboracao multiusuario em tempo real
- presence indicator em tempo real
- locking colaborativo de edicao
- CRUD completo de profiles/schema/governance/process areas/subjects
- editor rico colaborativo em tempo real

## Realtime Policy

## Why realtime was raised
O Operations Center e os paineis operacionais realmente se beneficiam de atualizacao ao vivo.

Exemplos:
- novas notificacoes
- approvals pendentes
- status de workflow mudando
- atividade recente em documentos

## Current architectural decision
Para a `Task 034`, o frontend deve nascer com **compatibilidade arquitetural para realtime**, mas nao com dependencia obrigatoria de stream.

Padrao oficial:
1. primeiro adapter: `refresh/polling`
2. futura extensao: `SSE`
3. WebSocket apenas se houver necessidade real de bidirecionalidade ou colaboracao forte

Regra:
- nao bloquear a UX base por falta de infra realtime
- nao enfiar uma nova superficie de backend no meio da Task 034 sem task propria

## Recommended future direction
Se quisermos real-time operacional profissional:
- priorizar `SSE` antes de `WebSocket`

Motivo:
- o caso atual e majoritariamente server-to-client
- menor complexidade operacional
- suficiente para feed operacional, notificacoes e status

WebSocket so deve entrar se surgirem requisitos como:
- co-authoring
- presence bidirecional
- locks colaborativos
- sinais interativos em tempo real entre usuarios

## Registry Administration Policy

## Current state
Hoje a plataforma suporta fortemente:
- listagem de families
- listagem de profiles
- listagem de schema por profile
- consulta de governance por profile
- listagem de process areas
- listagem de subjects

Ou seja:
- o dominio e o backend ja estao bons para leitura e exploracao
- mas ainda nao temos CRUD administrativo completo do registry

## Decision for Task 034
Na `Task 034`, o frontend deve tratar registry como:
- explorer
- painel de referencia
- painel de consulta/admin operacional

Nao como:
- painel full de criacao/edicao de schema/governance/profile

Isso evita:
- UI mentirosa
- promessa de capability que o backend ainda nao entrega
- acoplamento prematuro entre UX e mutacoes administrativas ainda nao modeladas

## Screen Intent by Area

### Operations Center
Objetivo:
- responder rapidamente “o que precisa de atencao agora?”

Widgets esperados:
- documentos recentes
- approvals pendentes
- notificacoes nao lidas
- documentos proximos de revisao/validade
- atalho para criar documento por profile

### Document Catalog
Objetivo:
- navegar o acervo de forma estruturada

Filtros esperados:
- texto
- family
- profile
- process area
- business unit
- department
- classification
- status
- owner

### Document Detail / Review
Objetivo:
- consultar o documento com contexto operacional completo

A experiencia deve mostrar:
- header documental
- metadata
- governance
- status atual
- versao atual
- timeline de versoes
- diff
- approvals
- attachments
- policies
- audit

### Document Authoring Wizard
Objetivo:
- criar documento novo ou iniciar fluxo de evolucao controlada

Passos esperados:
1. escolher profile
2. preencher metadata
3. preencher conteudo
4. revisar antes de confirmar

### Registry Explorer
Objetivo:
- tornar visivel o “motor” profile-first da plataforma

Deve exibir:
- profiles
- family associada
- schema ativo
- governance
- process areas
- subjects

## Technical Guidance for Frontend

### Data loading
- workspace e catalogo: query-based
- authoring: loader por `documentProfile`
- registry explorer: query-based
- operacao: polling/refresh inicialmente

### Component boundaries
A `Task 034` deve continuar a decomposicao estrutural iniciada na `Task 030`, sem voltar para monolito.

Slices esperados:
- `app-shell`
- `operations-center`
- `document-catalog`
- `document-detail`
- `document-authoring-wizard`
- `registry-explorer`
- `notifications-center`

### UX rules
- profile sempre no centro do fluxo
- family sempre derivada e visivel
- governanca nunca escondida
- metadata renderizada dinamicamente a partir do schema
- historico operacional acessivel sem trocar de contexto demais

## Future Tasks to Open Explicitly

### Task 036 - Registry administration CRUD
Objetivo:
- permitir criar/editar profiles, schema versions, governance, process areas e subjects pelo produto

### Task 037 - Realtime event stream for operations center
Objetivo:
- introduzir stream server-to-client para notificacoes, approvals e sinais operacionais

Direcao recomendada:
- SSE primeiro
- WebSocket apenas se SSE ficar insuficiente

### Task 038 - Collaborative editing and presence
Objetivo:
- presence
- locking/logica de concorrencia
- co-authoring e sinais colaborativos em tempo real

## Exit Criteria for Task 034
- workspace documental com mapa de telas coerente
- authoring wizard profile-first implementado
- registry explorer entregue
- operations center funcional sem depender de realtime obrigatorio
- backlog futuro de realtime e registry CRUD explicitamente separado
