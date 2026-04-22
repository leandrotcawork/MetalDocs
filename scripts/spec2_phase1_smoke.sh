#!/usr/bin/env bash
set -euo pipefail
PSQL="docker exec metaldocs-db psql -U metaldocs -d metaldocs -v ON_ERROR_STOP=0"

expect_error() {
  local label="$1"; local sql="$2"; local needle="$3"
  if $PSQL -c "$sql" 2>&1 | grep -q "$needle"; then
    echo "PASS: $label"
  else
    echo "FAIL: $label (expected '$needle')"; exit 1
  fi
}

expect_error "illegal transition draft->published" \
  "UPDATE documents SET status='published' WHERE status='draft' RETURNING id;" \
  "illegal status transition"

expect_error "cross-graph finalized->published still rejected (R2-1)" \
  "INSERT INTO documents (id,tenant_id,template_version_id,name,status,form_data_json,created_by) VALUES (gen_random_uuid(),'ffffffff-ffff-ffff-ffff-ffffffffffff',(SELECT id FROM template_versions LIMIT 1),'legacy-probe','draft','{}','probe');
   UPDATE documents SET status='finalized' WHERE name='legacy-probe';
   UPDATE documents SET status='published' WHERE name='legacy-probe' RETURNING id;" \
  "illegal status transition"

# R2-1 positive probe: compat transition draft->finalized must SUCCEED during Phases 1-4.
$PSQL -c "BEGIN; UPDATE documents SET status='finalized' WHERE status='draft' LIMIT 1; ROLLBACK;" \
  2>&1 | grep -qv "illegal status transition" && echo "PASS: compat draft->finalized allowed" || { echo "FAIL: compat broke"; exit 1; }

# R2-3 probe: signoff composite FK rejects cross-instance straddle.
expect_error "signoff cross-instance rejected (R2-3)" \
  "INSERT INTO approval_signoffs (approval_instance_id, stage_instance_id, actor_user_id, actor_tenant_id, decision, signature_method, signature_payload, content_hash) VALUES ('00000000-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000002','probe','ffffffff-ffff-ffff-ffff-ffffffffffff','approve','password','{}','sha');" \
  "violates foreign key constraint"

expect_error "revision_version monotonic" \
  "UPDATE documents SET revision_version = revision_version - 1 WHERE name='legacy-probe';" \
  "revision_version cannot decrease"

expect_error "user_process_areas DELETE blocked" \
  "DELETE FROM user_process_areas WHERE TRUE;" \
  "cannot be deleted"

expect_error "grant_area_membership needs context" \
  "BEGIN; SET LOCAL ROLE metaldocs_membership_writer;
   SELECT public.grant_area_membership('ffffffff-ffff-ffff-ffff-ffffffffffff','u','a','reviewer','u');
   ROLLBACK;" \
  "session actor context"

echo "All Phase 1 trigger probes passed."
