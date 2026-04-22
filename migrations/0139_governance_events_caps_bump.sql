-- migrations/0139_governance_events_caps_bump.sql

BEGIN;

CREATE UNIQUE INDEX IF NOT EXISTS ux_governance_events_caps_bump_spec_version
  ON governance_events (event_type, (payload_json->>'spec'), (payload_json->>'to'))
  WHERE event_type = 'role_capabilities_version_bump';

INSERT INTO governance_events
  (tenant_id, event_type, actor_user_id, resource_type, resource_id, reason, payload_json)
VALUES
  ('ffffffff-ffff-ffff-ffff-ffffffffffff',
   'role_capabilities_version_bump',
   'system:spec2-migration',
   'role_capabilities',
   'global',
   'Spec 2 workflow.* capabilities added',
   jsonb_build_object('from', 1, 'to', 2, 'spec', 'spec-2-approval'))
ON CONFLICT (event_type, (payload_json->>'spec'), (payload_json->>'to'))
  WHERE event_type = 'role_capabilities_version_bump'
  DO NOTHING;

COMMIT;
