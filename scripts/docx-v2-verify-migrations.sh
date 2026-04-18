#!/usr/bin/env bash
set -euo pipefail

# Verifies all docx-v2 migrations applied and all tables+indexes present.

SERVICE="${SERVICE:-postgres}"
COMPOSE_FILE="${COMPOSE_FILE:-deploy/compose/docker-compose.yml}"
DB_USER="${PGUSER:-metaldocs}"
DB_NAME="${PGDATABASE:-metaldocs}"

expected_tables=(
  templates
  template_versions
  documents_v2
  editor_sessions
  document_revisions
  autosave_pending_uploads
  document_checkpoints
  template_audit_log
)

expected_indexes=(
  idx_one_draft_per_template
  idx_one_active_session_per_doc
  idx_pending_expired
  idx_documents_v2_form_data_gin
)

psql_exec() {
  docker compose -f "$COMPOSE_FILE" exec -T "$SERVICE" \
    psql -U "$DB_USER" -d "$DB_NAME" -tAc "$1"
}

fail=0
for t in "${expected_tables[@]}"; do
  got=$(psql_exec "SELECT to_regclass('public.$t') IS NOT NULL")
  if [[ "$got" != "t" ]]; then
    echo "MISSING table: $t"
    fail=1
  fi
done

for idx in "${expected_indexes[@]}"; do
  got=$(psql_exec "SELECT COUNT(*) FROM pg_indexes WHERE indexname='$idx'")
  if [[ "$got" == "0" ]]; then
    echo "MISSING index: $idx"
    fail=1
  fi
done

if [[ "$fail" == "1" ]]; then
  echo "FAIL"
  exit 1
fi

echo "OK: all 8 tables + 4 critical indexes present"
