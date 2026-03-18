# Next Product Execution Plan

## Objective
Definir a ordem correta para evoluir MetalDocs apos o fechamento de auth/runtime/hardening, com foco agora em taxonomia documental multiempresa e configuravel sem drift arquitetural.

## Strategic Decision
O proximo ciclo oficial deve comecar por **documents configuraveis**, nao por mais UI ou por features operacionais isoladas.

O foco imediato passa a ser:
- tornar `documents` governado por familias canonicas + perfis documentais configuraveis
- suportar nomenclaturas por empresa sem hardcode na plataforma
- separar claramente:
  - natureza documental
  - processo/assunto
  - governanca

Somente depois desse ciclo entram:
- audit timeline HTTP completa
- tela operacional de notificacoes
- gestao administrativa de usuarios mais completa
- reset administrativo de senha / lifecycle de usuario
- readiness de producao e observabilidade mais profunda

## Why this order
Sem um modelo profissional para:
- familias documentais canonicas
- perfis documentais por empresa
- schemas versionados por perfil
- processo/assunto como taxonomia separada
- governanca por perfil

o risco e construir:
- tipos documentais hardcoded por cliente
- UI acoplada a uma empresa
- APIs que confundem tipo documental com assunto
- crescimento desordenado de metadata
- governanca/regra de workflow espalhada no codigo

## Market-Proven Direction
Arquiteturas maduras de DMS/ECM seguem o mesmo principio:
- **SharePoint**: `content types` customizaveis por organizacao
- **Alfresco**: `types + aspects + properties + constraints`
- **M-Files**: modelo explicitamente metadata-driven

Conclusao arquitetural para o MetalDocs:
- nao modelar por pasta
- nao modelar por assunto como tipo base
- usar tipo/familia canonica + perfil configuravel + metadata governado

## Official Modeling Direction

### 1. Canonical document families
Famílias pequenas, estaveis e da plataforma:
- `policy`
- `procedure`
- `work_instruction`
- `record`
- `form`
- `report`
- `manual`
- `contract`
- `external_document`

### 2. Company-specific document profiles
Perfis configuraveis por empresa apontando para uma familia canonica.

Exemplo Metal Nobre:
- `PO` -> `procedure`
- `IT` -> `work_instruction`
- `RG` -> `record`

Regra:
o documento referencia o **profile**, nao apenas a familia bruta.

### 3. Process/subject taxonomy
Assunto/processo nao deve virar tipo base.

Exemplos:
- `marketplaces`
- `quality`
- `commercial`
- `purchasing`
- `logistics`

Exemplo correto:
- `PO-MKT-001`
  - family: `procedure`
  - profile: `PO`
  - process_area: `marketplaces`

### 4. Versioned schema and governance
Cada profile deve poder definir:
- metadata obrigatorio
- metadata opcional
- prefixo/codigo
- workflow profile
- review cadence
- retention/validity
- access defaults

## Execution Order

### Step 1. Introduce configurable document profile registry
Status: `done`

Entregaveis:
- families canonicas da plataforma
- profiles documentais por empresa
- relacao `profile -> family`
- endpoints HTTP para families/profiles
- compatibilidade com `documentType` como alias de transicao

Saida esperada:
- nomenclaturas como `PO`, `IT`, `RG` deixam de ser hardcoded no frontend/backend

### Step 2. Add process area and subject taxonomy
Status: `done`

Entregaveis:
- `process_area`
- opcionalmente `subject/domain`
- vinculo com profiles e documentos

Saida esperada:
- `marketplaces` e outros assuntos deixam de competir com tipo documental

### Step 3. Add versioned schema and governance profile
Status: `done`

Entregaveis:
- schema versionado por profile
- regras de validacao backend por profile
- workflow/revisao/retencao por profile

Saida esperada:
- governanca documental configuravel por organizacao
- `documents` persistindo `profileSchemaVersion` para rastreabilidade

### Step 4. Evolve API and UI to operate by profile
Entregaveis:
- OpenAPI refletindo `document_profile`
- listagem de registry/perfis
- UI criando documento por profile
- exibicao de family + profile + process area

Saida esperada:
- experiencia operacional pronta para empresas diferentes sem fork de codigo

### Step 5. Only then expand operational product surface
Depois de estabilizar documents configuraveis:
- audit timeline HTTP completa
- notificacoes operacionais
- IAM administrativo mais rico
- lifecycle administrativo de usuario
- readiness/observabilidade mais profunda

## Recommended Metal Nobre Initial Profiles
- `PO` -> `procedure`
- `IT` -> `work_instruction`
- `RG` -> `record`

Status: `implemented` via seed inicial do registry

## Recommended Metal Nobre Initial Process Areas
- `quality`
- `marketplaces`
- `commercial`
- `purchasing`
- `logistics`
- `finance`

Status: `implemented` via seed inicial do registry

## Recommended Data Modeling

### Platform-owned stable concepts
- `document_family`
- `document_profile`
- `document_profile_schema_version`
- `document_governance_profile`

### Existing structured metadata that remains important
- `business_unit`
- `department`
- `owner_id`
- `classification`
- `tags`
- `effective_at`
- `expiry_at`

### Search and filtering
Busca deve continuar baseada em:
- texto livre
- family
- profile
- process_area
- unidade
- departamento
- classificacao
- status
- owner

## What not to do next
- nao transformar cada assunto em um novo tipo documental
- nao hardcodar perfis da Metal Nobre diretamente no frontend
- nao deixar schema por empresa dentro de `if/else` no codigo
- nao criar modulo separado de taxonomy cedo demais; manter dentro de `documents` enquanto o bounded context ainda for o dono natural
- nao voltar para organizacao por pasta como source of truth

## Exit Criteria for the next cycle
- registry documental configuravel implementado
- Metal Nobre operando com `PO`, `IT`, `RG`
- `marketplaces` tratado como taxonomia/processo, nao como tipo base acidental
- validacao backend baseada em profile
- OpenAPI e UI alinhadas ao profile registry
- backlog operacional seguinte congelado
