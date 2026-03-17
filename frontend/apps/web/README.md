# MetalDocs Web

UI operacional minima para cadastro, busca, detalhe, permissoes, anexos e workflow.

## Run local
1. Copie `.env.example` para `.env`.
2. Instale dependencias com `npm install`.
3. Rode `npm run dev`.

## Expected backend
- API em `http://127.0.0.1:8080`
- Frontend usa `/api/v1` por padrao
- Em desenvolvimento, o Vite faz proxy local para a API
- Se a web for servida por origem separada fora do proxy, habilite CORS na API com allowlist explicita

## Scope
- formulario de documento
- listagem com filtros
- detalhe do documento
- controle de permissao por recurso
- anexos
- timeline operacional de versoes, aprovacoes e anexos

## Notes
- Audit append-only ainda nao possui endpoint HTTP dedicado, entao a timeline operacional usa contratos reais ja disponiveis no backend.
