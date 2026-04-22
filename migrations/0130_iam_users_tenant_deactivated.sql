-- migrations/0130_iam_users_tenant_deactivated.sql
-- Spec 2 Phase 1 (Codex-revised). Global user_id PK preserved.
-- tenant_id is a lookup attribute (NOT part of identity namespace).
-- DEFAULT sentinel is backfill-only; Phase 5 migration drops DEFAULT.

BEGIN;

ALTER TABLE metaldocs.iam_users
  ADD COLUMN IF NOT EXISTS tenant_id UUID NOT NULL
    DEFAULT 'ffffffff-ffff-ffff-ffff-ffffffffffff',
  ADD COLUMN IF NOT EXISTS deactivated_at TIMESTAMPTZ;

ALTER TABLE metaldocs.iam_users
  DROP CONSTRAINT IF EXISTS iam_users_deactivated_after_created,
  ADD  CONSTRAINT iam_users_deactivated_after_created
    CHECK (deactivated_at IS NULL OR deactivated_at >= created_at);

CREATE UNIQUE INDEX IF NOT EXISTS ux_iam_users_tenant_user
  ON metaldocs.iam_users (tenant_id, user_id);

CREATE UNIQUE INDEX IF NOT EXISTS ux_iam_users_tenant_user_active
  ON metaldocs.iam_users (tenant_id, user_id)
  WHERE deactivated_at IS NULL;

COMMIT;
