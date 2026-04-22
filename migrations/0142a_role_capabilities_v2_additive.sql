-- migrations/0142a_role_capabilities_v2_additive.sql
-- Spec 2 Phase 6. Additive step: new capabilities + tripwire GUC support.
-- No legacy strip, no enforcement trigger. Old binaries unaffected.
--
-- role_capabilities schema v2

BEGIN;

-- ---------------------------------------------------------------------------
-- 1. Ensure role_capabilities table exists (idempotent).
--    PK is (role, capability) — the conflict target used by all INSERTs below.
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS metaldocs.role_capabilities (
  role         TEXT        NOT NULL,
  capability   TEXT        NOT NULL,
  description  TEXT        NOT NULL DEFAULT '',
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (role, capability)
);

-- ---------------------------------------------------------------------------
-- 2. Schema version bump (soft, additive).
--    Uses governance_events as the version tracker (no separate schema_versions
--    table exists in this codebase; 0139 established the convention).
--    If a schema_versions table is added later, the block below will activate.
-- ---------------------------------------------------------------------------
DO $$ BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.tables
     WHERE table_schema = 'metaldocs'
       AND table_name   = 'schema_versions'
  ) THEN
    INSERT INTO metaldocs.schema_versions (key, value)
      VALUES ('role_capabilities_version', '2')
    ON CONFLICT (key) DO UPDATE SET value = '2';
  END IF;
END $$;

-- ---------------------------------------------------------------------------
-- 3. Insert v2 capability rows.
--    All INSERTs use ON CONFLICT (role, capability) DO NOTHING — fully
--    idempotent; safe to re-run; old binaries reading v1 caps are unaffected.
-- ---------------------------------------------------------------------------
INSERT INTO metaldocs.role_capabilities (role, capability, description) VALUES
  -- doc.submit
  ('author',     'doc.submit',              'Submit document for approval'),
  ('area_admin', 'doc.submit',              'Submit document for approval'),

  -- doc.signoff
  ('signer',     'doc.signoff',             'Sign off on document approval stage'),
  ('reviewer',   'doc.signoff',             'Sign off on document approval stage'),
  ('area_admin', 'doc.signoff',             'Sign off on document approval stage'),
  ('qms_admin',  'doc.signoff',             'Sign off on document approval stage'),

  -- doc.publish
  ('area_admin', 'doc.publish',             'Publish approved document'),
  ('qms_admin',  'doc.publish',             'Publish approved document'),

  -- doc.supersede
  ('area_admin', 'doc.supersede',           'Supersede published document'),
  ('qms_admin',  'doc.supersede',           'Supersede published document'),

  -- doc.obsolete
  ('qms_admin',  'doc.obsolete',            'Mark document as obsolete'),

  -- workflow.instance.cancel
  ('qms_admin',  'workflow.instance.cancel','Cancel in-flight approval workflow'),

  -- route.admin
  ('qms_admin',  'route.admin',             'Administer approval route configurations'),

  -- membership.grant
  ('area_admin', 'membership.grant',        'Grant area membership'),
  ('qms_admin',  'membership.grant',        'Grant area membership'),

  -- membership.revoke
  ('area_admin', 'membership.revoke',       'Revoke area membership'),
  ('qms_admin',  'membership.revoke',       'Revoke area membership')

ON CONFLICT (role, capability) DO NOTHING;

-- ---------------------------------------------------------------------------
-- 4. Tripwire GUC convention (0142b enables enforcement trigger).
--
--    Application code gates capability checks via session-local GUCs:
--
--      SET LOCAL metaldocs.asserted_caps = '[{"cap":"doc.submit","area":"PROD"}]';
--      SET LOCAL metaldocs.bypass_authz  = 'scheduler';
--
--    These are session-local GUCs; no DDL registration is needed in PostgreSQL
--    (custom GUCs are implicitly accepted via current_setting(..., true)).
--    The enforcement trigger introduced in 0142b will read these GUCs to
--    validate that the application has asserted the required capability tuple
--    before any state-mutating DML is allowed through.
--
--    Convention:
--      metaldocs.asserted_caps — JSONB array of {cap, area} objects, set per-tx
--                                by application middleware BEFORE the first DML.
--      metaldocs.bypass_authz  — opaque token for scheduler/system callers that
--                                bypasses capability assertion (audited separately).
--    Both GUCs are session-local only; setting them at session level is a
--    misconfiguration caught by Probe I (see ops/DEPLOY.md).
-- ---------------------------------------------------------------------------

COMMIT;
