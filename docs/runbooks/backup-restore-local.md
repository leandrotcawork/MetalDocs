# Runbook: Backup and Restore (Local PostgreSQL)

## Objetivo
Executar backup consistente e validar restore do banco `metaldocs` em ambiente local.

## Pre-requisitos
- PostgreSQL tools instaladas (`pg_dump`, `pg_restore`, `psql`).
- Variaveis de ambiente definidas:
  - `PGHOST`
  - `PGPORT`
  - `PGDATABASE`
  - `PGUSER` e `PGPASSWORD` (ou passar `-PgUser` e `-PgPassword` nos scripts)
- Usuario de backup com permissao de leitura no schema `metaldocs`.

## Setup do usuario de backup (uma vez por ambiente)
1. Execute como admin:
   - `psql -h <host> -p <port> -U <admin_user> -d postgres -f scripts/sql/create-backup-role.sql`
2. Ajuste a senha no script SQL antes de executar.

## Backup
1. Rodar script:
   - `powershell -ExecutionPolicy Bypass -File scripts/backup-postgres.ps1 -PgUser <backup_user> -PgPassword <backup_password>`
2. Confirmar arquivo gerado em `backups/` com extensao `.dump`.
3. Validar dump:
   - `powershell -ExecutionPolicy Bypass -File scripts/validate-backup.ps1 -BackupFile backups/<arquivo>.dump -PgRestorePath "C:\Program Files\PostgreSQL\16\bin\pg_restore.exe"`

## Restore (ambiente de validacao)
Recomendacao: restaurar em database de validacao, nao na principal.

1. Criar database de validacao com usuario admin:
   - `CREATE DATABASE metaldocs_restore_validation;`
2. Rodar restore:
   - `powershell -ExecutionPolicy Bypass -File scripts/restore-postgres.ps1 -BackupFile backups/<arquivo>.dump -TargetDatabase metaldocs_restore_validation -PgUser <restore_user> -PgPassword <restore_password>`
3. Validar objetos principais:
   - `\dt metaldocs.*`
   - `SELECT COUNT(*) FROM metaldocs.documents;`
   - `SELECT COUNT(*) FROM metaldocs.audit_events;`
   - `SELECT COUNT(*) FROM metaldocs.outbox_events;`

## Evidencia minima para gate Phase 2
- Nome do arquivo de backup.
- Timestamp do backup.
- Comando de restore executado.
- Resultado das queries de validacao.

## Cuidados
- O script de restore usa `--clean --if-exists`.
- Nao rode restore direto no banco de producao sem janela e aprovacao.
- O usuario da aplicacao (`metaldocs_app`) pode nao ter `SELECT` em tabelas sensiveis (ex.: `audit_events`, `outbox_events`) por politica de least privilege. Use usuario dedicado de backup.
