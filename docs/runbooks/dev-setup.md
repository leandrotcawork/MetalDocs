# Runbook: Dev Setup

## Objective
Executar o MetalDocs em modo de desenvolvimento rapido sem perder a fonte unica de verdade local.

## Runtime oficial de dev local
- Docker:
  - `postgres`
  - `redis`
  - `minio`
- Local:
  - `api`
  - `web`

Regra:
- o banco oficial local e o Postgres do Docker
- a porta de host oficial do Postgres Docker e `5433`
- `api/web/gateway/worker` em Docker nao devem ficar rodando no dia a dia de dev rapido

## 1) Pre-requisitos
- Go instalado
- Node.js/NPM instalados
- Docker Desktop funcional
- `.env` criado a partir de `.env.example`

Validacao minima:
```powershell
go version
node --version
npm --version
docker version
```

## 2) Bootstrap inicial
Do root do repo:
```powershell
powershell -ExecutionPolicy Bypass -File scripts/dev-bootstrap.ps1
```

## 3) Subir o modo dev oficial
Do root do repo:
```powershell
powershell -ExecutionPolicy Bypass -File scripts/dev-local.ps1
```

Isso faz:
- para `api/web/gateway/worker` no Docker
- sobe `postgres/redis/minio` no Docker
- preserva o banco Docker como fonte unica de verdade

## 4) Rodar a API local
```powershell
powershell -ExecutionPolicy Bypass -File scripts/dev-api.ps1
```

O script:
- carrega `.env`
- valida vars obrigatorias
- verifica se `APP_PORT` esta livre
- sobe a API local ligada ao Postgres Docker

## 5) Rodar a web local
Em outro terminal:
```powershell
cd frontend/apps/web
npm run dev
```

Observacao:
- o Vite deriva o proxy da API a partir de `VITE_API_PROXY_TARGET`
- se esse override nao estiver definido, ele usa `APP_PORT` da `.env` do repo root
- assim, mudar `APP_PORT` nao deve mais exigir mexer em dois lugares

Abrir no navegador:
```text
http://127.0.0.1:4173
```

## 6) Regras de auth e browser
- auth oficial v1 = sessao por cookie HTTP-only
- `X-User-Id` nao e caminho oficial de runtime e deve ser tratado apenas como mecanismo tecnico de teste
- first login com senha temporaria:
  - autentica
  - entra em `mustChangePassword=true`
  - nao carrega workspace ate trocar a senha
- usar sempre a mesma origem no browser em dev:
  - preferencialmente `http://127.0.0.1:4173`

## 7) Validacoes do dia a dia
```powershell
# testes Go
powershell -ExecutionPolicy Bypass -File scripts/test.ps1

# build frontend
cd frontend/apps/web
npm run build
```

## 8) Smoke E2E oficial da fase
Do root do repo:
```powershell
powershell -ExecutionPolicy Bypass -File scripts/e2e-smoke.ps1
```

O smoke:
- garante Docker infra local
- sobe API local
- sobe web local
- semeia um admin tecnico idempotente para browser test
- valida:
  - login admin
  - criacao de usuario
  - first login
  - troca obrigatoria de senha
  - criacao de documento
  - logout e relogin
## 9) Dependency policy
- Nunca usar `go get -u ./...` cegamente.
- Adicionar/atualizar apenas o necessario para a feature em andamento.
- Commitar `go.mod` e `go.sum` juntos quando houver mudanca de dependencia.
- Dependencia critica nova exige ADR/RFC conforme `AGENTS.md`.
