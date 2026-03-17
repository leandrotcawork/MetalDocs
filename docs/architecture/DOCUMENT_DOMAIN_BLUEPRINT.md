# Document Domain Blueprint

## Purpose
Definir como o MetalDocs organiza documentos de forma profissional, escalavel e consistente com a arquitetura modular ja estabelecida.

## Core Principle
No MetalDocs, o documento nao e organizado primariamente por "pasta fisica". O source of truth e o conjunto de metadados estruturados.

Pastas podem existir no futuro como visao de navegacao, mas nao como modelo primario de negocio.

## Canonical Domain Model

### 1. Document
Documento logico que representa o registro principal.

Campos base obrigatorios em v1:
- `document_id`
- `title`
- `document_type`
- `owner_id`
- `classification`
- `status`
- `business_unit`
- `department`
- `tags[]`
- `current_version`
- `created_at`
- `updated_at`

### 2. Document Version
Representa uma versao imutavel do documento.

Campos base:
- `document_id`
- `version_number`
- `storage_key`
- `content_hash`
- `content_mime_type`
- `created_by`
- `created_at`
- `change_summary`

### 3. Document Type
Tipo documental governado, versionado e definido pela plataforma.

Exemplos iniciais:
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

Cada tipo documental deve definir:
- nome de exibicao
- descricao
- workflow padrao
- metadados obrigatorios
- metadados opcionais
- regras de validade/expiracao
- politicas de acesso padrao

### 4. Metadata Schema
Conjunto de campos adicionais exigidos por tipo documental.

Exemplos:
- `contract`: `counterparty`, `start_date`, `end_date`, `contract_number`
- `certificate`: `issuer`, `issue_date`, `expiry_date`
- `technical_drawing`: `drawing_code`, `revision_code`, `plant`

## Information Architecture

### Primary Organization Dimensions
Estas dimensoes devem sustentar busca, listagem, dashboards e navegacao:
- `document_type`
- `business_unit`
- `department`
- `owner_id`
- `classification`
- `status`
- `tags`
- `created_at`
- `effective_at`
- `expiry_at`

### Rule: Metadata over folders
Nao modelar a plataforma com dependencia forte de arvores de pasta como:
`/finance/contracts/2026/vendor-x/...`

Isso pode existir apenas como:
- visao derivada
- atalho de navegacao
- exportacao

Nunca como unica forma de localizar ou autorizar acesso a um documento.

### Rule: Tags are secondary
Tags ajudam descoberta, mas nao substituem:
- tipo documental
- unidade de negocio
- departamento
- classificacao

## Lifecycle Model

## Lifecycle Layers
Existem duas camadas distintas:

### 1. Workflow status
Estado operacional do documento:
- `DRAFT`
- `IN_REVIEW`
- `APPROVED`
- `PUBLISHED`
- `ARCHIVED`

### 2. Validity lifecycle
Estado temporal/regulatorio do conteudo:
- `effective_at`
- `expiry_at`
- `retention_until`

Regra importante:
workflow nao substitui validade.

Exemplo:
um documento pode estar `PUBLISHED` e ainda assim estar expirado.

## Ownership and Responsibility

### Required actors per document
- `owner_id`: dono funcional do documento
- `created_by`
- `last_updated_by`

### Future-ready actors
Preparar o modelo para:
- `approver_id`
- `reviewer_group`
- `custodian_id`

## Access and Security Model

Permissao de acesso deve considerar combinacao de:
- role global RBAC
- classificacao do documento
- ownership
- escopo organizacional (`business_unit`, `department`)
- policy especifica do documento
- policy por tipo documental
- policy por area

## Access Control Layers

### 1. Global role
Define capacidade macro do usuario no sistema.

Exemplos:
- `admin`
- `editor`
- `reviewer`
- `viewer`

### 2. Area scope
Define em quais areas o usuario pode operar.

Exemplos:
- pode ver documentos da area `quality`
- pode editar documentos da area `engineering`
- nao pode acessar documentos da area `finance`

### 3. Document type scope
Define em quais tipos documentais o usuario pode operar.

Exemplos:
- pode visualizar `policy` e `procedure`
- pode editar `work_instruction`
- nao pode editar `contract`

### 4. Document-level overrides
Permite override por documento especifico quando necessario.

Exemplos:
- usuario pode apenas visualizar um documento especifico
- grupo pode editar um documento especifico
- usuario pode anexar arquivo mas nao alterar metadados

## Capability Model

O modelo de permissao deve nascer orientado a capacidades e nao apenas a roles.

Capacidades minimas para o proximo ciclo:
- `document.view`
- `document.edit`
- `document.upload_attachment`
- `document.change_workflow`
- `document.manage_permissions`

Regra importante:
na tela de criar ou editar documento deve ser possivel definir:
- quem pode apenas visualizar
- quem pode visualizar e editar
- quem pode anexar/subir arquivo
- em qual tipo documental isso vale
- em qual area isso vale

Recomendacao de modelagem:
guardar policies de forma estruturada e avaliavel no backend, nunca apenas no frontend.

Estrutura conceitual inicial:
- `subject_type` (`user`, `role`, `group`)
- `subject_id`
- `resource_scope` (`document`, `document_type`, `area`)
- `resource_id`
- `capability`
- `effect` (`allow`, `deny`)

Regra:
RBAC puro por role global e suficiente para v1 inicial, mas o modelo de dados ja deve nascer preparado para policy por recurso.

## Search Model

Busca profissional de documentos precisa separar:

### Structured filters
- tipo
- status
- classificacao
- owner
- unidade
- departamento
- data
- validade

### Text search
- titulo
- resumo
- identificadores externos
- metadados textuais

Regra:
nenhuma tela importante deve depender apenas de busca textual livre.

## V1 Scope Freeze for Document Modeling

Implementar no proximo ciclo:
- tipo documental obrigatorio
- unidade de negocio obrigatoria
- departamento obrigatorio
- tags opcionais
- metadados estruturados por tipo documental
- campos opcionais de vigencia (`effective_at`, `expiry_at`)

Fica fora do primeiro incremento:
- arvore hierarquica arbitraria de pastas
- taxonomia livre administravel por UI complexa
- retention policy automatica
- OCR/indexacao pesada

## Architectural Placement

### Module ownership
- `documents`: aggregate principal, metadata base, versions
- `workflow`: transicoes de estado
- `iam`: autorizacao
- `search`: leitura/projecao e filtros
- `audit`: trilha append-only

### Recommendation
`document_type` e `metadata schema` devem nascer dentro do dominio `documents` no inicio.

Nao criar modulo separado de taxonomy no primeiro momento.

Motivo:
- reduz over-engineering
- mantem ownership claro
- facilita evolucao para registry dedicado no futuro

## Non-Negotiable Rules
- Documento sempre tem `document_type`.
- Documento sempre pertence a um contexto organizacional minimo.
- Versao nunca e alterada em lugar.
- Workflow e validade sao conceitos separados.
- Busca principal baseada em metadados estruturados.
- Pasta, se existir, e visao derivada e nao identidade do documento.
