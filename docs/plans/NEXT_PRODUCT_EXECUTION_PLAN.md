# Next Product Execution Plan

## Objective
Definir a ordem correta para evoluir MetalDocs apos a fundacao arquitetural, evitando construir worker, UI ou APIs em cima de um modelo documental incompleto.

## Strategic Decision
O proximo ciclo nao deve comecar por UI.

O primeiro foco deve ser fechar o dominio funcional de documentos e somente depois evoluir:
- backend
- worker
- UI operacional

## Why this order
Sem definicao de:
- tipos documentais
- metadados obrigatorios
- organizacao dos documentos
- regras de validade
- criterios de busca
- regras de permissao por documento/tipo/area

o risco e construir:
- APIs genericas demais
- telas que precisarao ser refeitas
- worker processando eventos com payload pobre
- banco com schema insuficiente
- autorizacao incapaz de refletir a operacao real

## Execution Order

### Step 1. Freeze document information architecture
Entregaveis:
- blueprint de dominio documental
- taxonomia inicial de tipos
- metadados base obrigatorios
- separacao entre workflow e validade

Saida esperada:
- decisao clara sobre como um documento e classificado, encontrado e governado

### Step 2. Evolve domain and API contracts
Entregaveis:
- atualizar `documents` domain model
- atualizar OpenAPI para refletir `document_type`, contexto organizacional e metadados
- registrar regras de validacao por tipo documental
- registrar contrato de permissao por documento, tipo e area

Saida esperada:
- contratos publicos e internos alinhados com o produto real

### Step 3. Evolve database schema
Entregaveis:
- migrations additive-first
- colunas e tabelas de metadata base
- estrutura para metadados especificos por tipo
- estrutura para access policies por recurso

Saida esperada:
- persistencia preparada para crescer sem reestruturar tudo depois

### Step 4. Build worker production-ready
Entregaveis:
- consumidor do outbox
- retry/backoff
- idempotencia forte
- runbook de reprocessamento

Dependencia:
- payloads de evento ja precisam refletir o modelo documental correto

### Step 5. Build operational UI
Entregaveis:
- cadastro de documento com tipo e metadados
- listagem com filtros estruturados
- detalhe do documento
- timeline de workflow e auditoria

Dependencia:
- contratos de backend e busca ja estabilizados

## Recommended v1 Document Types
- `policy`
- `procedure`
- `work_instruction`
- `contract`
- `supplier_document`
- `technical_drawing`
- `certificate`
- `report`
- `form`
- `manual`

## Recommended v1 Core Metadata
- `document_type`
- `business_unit`
- `department`
- `owner_id`
- `classification`
- `tags`
- `effective_at`
- `expiry_at`

## Recommended v1 Access Control Scope
- permissao por `area`
- permissao por `document_type`
- override por `document`
- capacidades separadas de role:
  - `view`
  - `edit`
  - `upload_attachment`
  - `change_workflow`
  - `manage_permissions`

## Data Modeling Recommendation

### Base metadata
Campos estruturados em colunas dedicadas para filtros frequentes.

### Type-specific metadata
Persistir em estrutura flexivel controlada por schema, por exemplo:
- `metadata_json`

Regra:
nao permitir metadata arbitraria sem validacao por tipo documental.

## Search Recommendation
Busca do v1 deve suportar:
- texto livre
- filtros estruturados
- combinacao dos dois

Campos obrigatorios de filtro no proximo ciclo:
- tipo documental
- unidade de negocio
- departamento
- classificacao
- status
- owner

## What not to do next
- nao construir arvore de pasta complexa primeiro
- nao construir UI rica antes do contrato funcional
- nao criar modulo novo so para taxonomy agora
- nao empurrar regras de tipo documental para frontend
- nao modelar tudo como texto livre

## Exit Criteria for the next cycle
- blueprint documental aprovado
- OpenAPI atualizada
- schema evoluido
- testes de dominio cobrindo validacoes de tipo/metadado
- backlog do worker ajustado ao modelo novo

## Immediate Next Implementation Slice
O proximo slice concreto deve ser:

`documents v2 domain modeling`

Inclui:
- `document_type`
- `business_unit`
- `department`
- `tags`
- `effective_at`
- `expiry_at`
- `metadata_json` validado por tipo
- `access_policies` por documento/tipo/area
