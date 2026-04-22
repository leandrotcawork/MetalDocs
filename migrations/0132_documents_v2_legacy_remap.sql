-- migrations/0132_documents_v2_legacy_remap.sql
-- Spec 2 Phase 1. Runs against superset CHECK (0131).

BEGIN;

WITH remap AS (
  UPDATE documents
     SET status         = 'published',
         effective_from = COALESCE(effective_from, finalized_at, updated_at)
   WHERE status = 'finalized'
   RETURNING id, tenant_id, created_by
)
INSERT INTO governance_events
  (tenant_id, event_type, actor_user_id, resource_type, resource_id, reason, payload_json)
SELECT tenant_id,
       'legacy_status_remap',
       COALESCE(created_by, 'system:spec2-migration'),
       'document_v2',
       id::TEXT,
       'Spec 2 legacy collapse: finalized -> published',
       jsonb_build_object('from','finalized','to','published','remapped_at',now())
  FROM remap
 WHERE NOT EXISTS (
   SELECT 1 FROM governance_events ge
    WHERE ge.resource_type = 'document_v2'
      AND ge.resource_id   = remap.id::TEXT
      AND ge.event_type    = 'legacy_status_remap'
      AND ge.payload_json->>'from' = 'finalized'
 );

WITH remap AS (
  UPDATE documents
     SET status       = 'obsolete',
         effective_to = COALESCE(effective_to, archived_at, updated_at)
   WHERE status = 'archived'
   RETURNING id, tenant_id, created_by
)
INSERT INTO governance_events
  (tenant_id, event_type, actor_user_id, resource_type, resource_id, reason, payload_json)
SELECT tenant_id,
       'legacy_status_remap',
       COALESCE(created_by, 'system:spec2-migration'),
       'document_v2',
       id::TEXT,
       'Spec 2 legacy collapse: archived -> obsolete',
       jsonb_build_object('from','archived','to','obsolete','remapped_at',now())
  FROM remap
 WHERE NOT EXISTS (
   SELECT 1 FROM governance_events ge
    WHERE ge.resource_type = 'document_v2'
      AND ge.resource_id   = remap.id::TEXT
      AND ge.event_type    = 'legacy_status_remap'
      AND ge.payload_json->>'from' = 'archived'
 );

COMMIT;
