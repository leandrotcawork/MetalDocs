# MetalDocs Web

UI operacional minima para cadastro, busca, detalhe, permissoes, anexos e workflow.

## Run local
1. Copie `.env.example` para `.env`.
2. Instale dependencias com `npm install`.
3. Rode `npm run dev`.

## Expected backend
- API em `http://192.168.0.3:8080/api/v1`
- Header `X-User-Id` configurado por `VITE_USER_ID`

## Scope
- formulario de documento
- listagem com filtros
- detalhe do documento
- controle de permissao por recurso
- anexos
- timeline operacional de versoes, aprovacoes e anexos

## Notes
- Audit append-only ainda nao possui endpoint HTTP dedicado, entao a timeline operacional usa contratos reais ja disponiveis no backend.
