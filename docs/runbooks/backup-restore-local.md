# Runbook: Backup/Restore Gate (Local PostgreSQL)

## Objetivo
Executar e comprovar o gate da Fase 2 com fluxo oficial:
`backup -> validate -> restore`.

O gate so e aprovado quando o restore em banco isolado e o smoke SQL passam 100%.

## Pre-requisitos
- PostgreSQL tools instaladas (`pg_dump`, `pg_restore`, `psql`).
- Variaveis de ambiente definidas:
  - `PGHOST`
  - `PGPORT`
  - `PGDATABASE` (fonte, ex.: `metaldocs`)
- Usuario dedicado de backup (nao `metaldocs_app`).
- Usuario de restore com permissao para criar DB e restaurar.

## Setup do usuario de backup (uma vez por ambiente)
1. Ajustar senha no arquivo `scripts/sql/create-backup-role.sql`.
2. Executar como admin:
   - `psql -h <host> -p <port> -U <admin_user> -d postgres -f scripts/sql/create-backup-role.sql`

## Contrato operacional (scripts)
- Entrada minima:
  - Backup: host/port/database/user/password.
  - Validate: arquivo de dump.
  - Restore: arquivo de dump + banco alvo + user/password.
- Saida minima:
  - `status`, timestamps de inicio/fim, `duration_seconds`.
  - `backup_file`.
  - `checksum_sha256` (backup).
  - `validation_passed` (validate).

## Execucao oficial do gate (1 comando)
Rodar o orquestrador:

```powershell
powershell -ExecutionPolicy Bypass -File scripts/run-backup-restore-gate.ps1 -EnvironmentName local -BackupUser metaldocs_backup -BackupPassword "<backup_password>" -RestoreUser "<restore_user>" -RestorePassword "<restore_password>" -RestoreValidationDatabase "metaldocs_restore_test"
```

## Opcional: execucao em 3 comandos explicitos
1. Backup:
   - `powershell -ExecutionPolicy Bypass -File scripts/backup-postgres.ps1 -EnvironmentName local -PgUser metaldocs_backup -PgPassword "<backup_password>"`
2. Validate:
   - `powershell -ExecutionPolicy Bypass -File scripts/validate-backup.ps1 -BackupFile backups/<arquivo>.dump -EnvironmentName local`
3. Restore:
   - `powershell -ExecutionPolicy Bypass -File scripts/restore-postgres.ps1 -BackupFile backups/<arquivo>.dump -TargetDatabase metaldocs_restore_test -EnvironmentName local -PgUser "<restore_user>" -PgPassword "<restore_password>"`

## Evidence Required (obrigatorio)
Anexar o JSON em `backups/evidence/backup_restore_gate_<env>_<db>_<timestamp>.json` contendo:
- Status final do gate (`approved` ou `rejected`).
- Operador.
- Caminho do dump.
- Hash SHA-256 do dump.
- Resultado da validacao.
- Resultado do restore.
- Resultado do smoke SQL com contagem de:
  - `metaldocs.documents`
  - `metaldocs.document_versions`
  - `metaldocs.iam_users`
  - `metaldocs.iam_user_roles`
  - `metaldocs.audit_events`
  - `metaldocs.outbox_events`

## Criterios de aceite (gate aprovado)
- Backup concluido sem erro.
- Dump valido no `validate-backup`.
- Restore concluido em banco isolado (`metaldocs_restore_test`).
- Smoke SQL 100% aprovado.
- Evidence JSON gerado com status `approved`.

## Criterios de reprovacao
- Falha em qualquer etapa do fluxo.
- Credencial invalida.
- Dump invalido/corrompido.
- Restore incompleto.
- Falha em qualquer consulta de smoke.

## Cuidados
- Nao executar restore no banco principal sem janela aprovada.
- `metaldocs_app` permanece least-privilege e nao deve ser usado para backup.
