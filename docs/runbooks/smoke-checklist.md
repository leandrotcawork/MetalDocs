# Runbook: Smoke Checklist (Local)

## Objetivo
Validar rapidamente o fluxo principal do MetalDocs antes de refactors e releases locais.

## Pre-requisitos
- API e banco rodando via Docker Compose.
- Frontend buildado localmente.

## Passos (manual)
1. Login como admin.
2. Criar usuario editor.
3. Fazer logout.
4. Login como editor (troca de senha).
5. Criar documento.
6. Abrir editor de conteudo.
7. Salvar conteudo e gerar PDF.
8. Voltar ao acervo e validar o documento.

## Comandos sugeridos
- `cd frontend/apps/web; npm.cmd run build`
- `docker compose up -d`
