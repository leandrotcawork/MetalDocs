# Runbook: Carbone Templates (DOCX)

## Objective
Padronizar a manutencao dos templates DOCX usados pelo Carbone para gerar PDF e exportar templates.

## Estrutura oficial
- `carbone/templates/`:
  - templates master versionados por profile:
    - `template-po.docx`
    - `template-it.docx`
    - `template-rg.docx`
    - `template-fm.docx` (ajuste quando o layout do formulario estiver definido)
- `carbone/renders/`:
  - saida local de testes (gitignored)

## Regras
- Templates master sao ativos de design e ficam versionados no repo.
- Nunca editar templates em runtime. Alteracoes sao feitas via commit.
- O bootstrap do API registra os templates automaticamente no start.

## Como editar um template
1. Abrir o `.docx` no Word/LibreOffice.
2. Atualizar textos, estilos e placeholders Carbone.
3. Salvar o arquivo mantendo o mesmo nome.
4. Commitar junto da tarefa correspondente.

## Placeholders Carbone (exemplos)
- Campos simples: `{d.title}`, `{d.owner}`
- Arrays: `{d.steps[i].acao}`
- Tabela (linha repetida): `{d.etapas[i].num}` | `{d.etapas[i].etapa}`

## Validacao local (dev)
1. Subir `carbone` no compose (Task 056).
2. Subir a API local.
3. Chamar qualquer endpoint de render (ex: salvar conteudo nativo) e verificar se o PDF foi gerado.

## Troubleshooting
- Template nao registrado:
  - verifique se o arquivo existe em `carbone/templates/`
  - reinicie a API para re-registrar
- PDF vazio:
  - placeholders nao batem com o payload `content`
  - revisar o JSON enviado pelo backend
