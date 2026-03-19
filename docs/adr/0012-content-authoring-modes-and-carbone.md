# ADR-0012: Content Authoring Modes + Carbone Rendering

## Status
Accepted

## Context
MetalDocs precisa suportar autoria de conteudo em dois modos:
1) editor nativo estruturado (conteudo como JSON por secoes), e
2) fluxo DOCX (exportar template -> editar offline -> reimportar).

O sistema ja possui versionamento imutavel e persistencia de anexos em storage (MinIO/local). Precisamos adicionar um renderer/conversor de documentos que seja consistente, operavel em Docker Compose single-node e que nao empurre regras para o frontend.

## Decision
- O renderer oficial de PDF/DOCX no servidor sera Carbone self-hosted (container no Compose).
- Ambos os modos produzem uma nova `document_version` (append-only):
  - `content_source = native` quando o conteudo veio do editor nativo.
  - `content_source = docx_upload` quando o conteudo veio de upload de DOCX.
- PDF final exibido no produto sempre e o PDF renderizado pelo backend via Carbone.
  - Preview instantaneo client-side (ex: `@react-pdf/renderer`) fica fora do MVP por duplicar templates (alto custo de manutencao) e risco de divergencia visual.
- Templates DOCX master sao assets versionados no repo (nao sao dados mutaveis em runtime).

## Consequences
- Docker Compose ganha um novo servico `carbone`.
- Backend precisa de um client Carbone com timeout, logs e tratamento de erro.
- OpenAPI ganha endpoints de conteudo (native + docx).
- Persistencia de `document_versions` evolui para armazenar conteudo estruturado e chaves de PDF/DOCX no storage.

## Notes
- O sistema continua RBAC/policy-first: permissao de view/edit/upload e sempre validada no backend.
- Renderizacao pode ser sincrona no MVP; async via worker/outbox vira evolucao.
