-- migrations/0135_approval_instances.sql
-- Spec 2 Phase 1 Codex-revised. Denormalized approval_instance_id on signoffs
-- plus UNIQUE(approval_instance_id, actor_user_id) replaces race-prone SoD trigger
-- for cross-stage duplicate check. All composite FKs to iam_users NOT VALID;
-- Task 1.7 validates after explicit backfill verification.

BEGIN;

CREATE TABLE IF NOT EXISTS approval_instances (
  id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id               UUID NOT NULL,
  document_v2_id          UUID NOT NULL REFERENCES documents(id),
  route_id                UUID NOT NULL REFERENCES approval_routes(id),
  route_version_snapshot  INT  NOT NULL,
  status                  TEXT NOT NULL
    CHECK (status IN ('in_progress','approved','rejected','cancelled')),
  submitted_by            TEXT NOT NULL,
  submitted_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
  completed_at            TIMESTAMPTZ,
  content_hash_at_submit  TEXT NOT NULL,
  idempotency_key         TEXT NOT NULL,
  UNIQUE (document_v2_id, idempotency_key)
);

ALTER TABLE approval_instances
  ADD CONSTRAINT approval_instances_submitted_by_tenant_fkey
    FOREIGN KEY (tenant_id, submitted_by)
      REFERENCES metaldocs.iam_users (tenant_id, user_id)
    NOT VALID;

CREATE UNIQUE INDEX IF NOT EXISTS ux_approval_instances_active
  ON approval_instances (document_v2_id)
  WHERE status = 'in_progress';

CREATE INDEX IF NOT EXISTS ix_approval_instances_tenant_doc
  ON approval_instances (tenant_id, document_v2_id);

CREATE TABLE IF NOT EXISTS approval_stage_instances (
  id                             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  approval_instance_id           UUID NOT NULL REFERENCES approval_instances(id) ON DELETE CASCADE,
  stage_order                    INT  NOT NULL CHECK (stage_order >= 1),
  name_snapshot                  TEXT NOT NULL,
  required_role_snapshot         TEXT NOT NULL,
  required_capability_snapshot   TEXT NOT NULL,
  area_code_snapshot             TEXT NOT NULL,
  quorum_snapshot                TEXT NOT NULL
    CHECK (quorum_snapshot IN ('any_1_of','all_of','m_of_n')),
  quorum_m_snapshot              INT,
  on_eligibility_drift_snapshot  TEXT NOT NULL
    CHECK (on_eligibility_drift_snapshot IN ('reduce_quorum','fail_stage','keep_snapshot')),
  eligible_actor_ids             JSONB NOT NULL,
  effective_denominator          INT,
  status                         TEXT NOT NULL
    CHECK (status IN ('pending','active','completed','skipped','rejected_here')),
  opened_at                      TIMESTAMPTZ,
  completed_at                   TIMESTAMPTZ,
  UNIQUE (approval_instance_id, stage_order),
  -- Anchors composite FK from approval_signoffs(stage_instance_id, approval_instance_id)
  -- so a signoff cannot reference a stage from a different instance.
  UNIQUE (id, approval_instance_id)
);

CREATE INDEX IF NOT EXISTS ix_stage_instances_active
  ON approval_stage_instances (approval_instance_id, stage_order)
  WHERE status = 'active';

CREATE INDEX IF NOT EXISTS ix_stage_instances_eligible_actors
  ON approval_stage_instances USING GIN (eligible_actor_ids);

CREATE TABLE IF NOT EXISTS approval_signoffs (
  id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  approval_instance_id  UUID NOT NULL REFERENCES approval_instances(id),
  stage_instance_id     UUID NOT NULL REFERENCES approval_stage_instances(id),
  actor_user_id         TEXT NOT NULL,
  actor_tenant_id       UUID NOT NULL,
  decision              TEXT NOT NULL CHECK (decision IN ('approve','reject')),
  comment               TEXT,
  signed_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
  signature_method      TEXT NOT NULL,
  signature_payload     JSONB NOT NULL,
  content_hash          TEXT NOT NULL,
  UNIQUE (approval_instance_id, actor_user_id),
  UNIQUE (stage_instance_id, actor_user_id),
  -- Codex Round 2 fix #3: hard-bind stage to instance so signoff cannot straddle instances.
  CONSTRAINT approval_signoffs_stage_matches_instance
    FOREIGN KEY (stage_instance_id, approval_instance_id)
    REFERENCES approval_stage_instances (id, approval_instance_id)
);

ALTER TABLE approval_signoffs
  ADD CONSTRAINT approval_signoffs_actor_tenant_fkey
    FOREIGN KEY (actor_tenant_id, actor_user_id)
      REFERENCES metaldocs.iam_users (tenant_id, user_id)
    NOT VALID;

CREATE INDEX IF NOT EXISTS ix_signoffs_stage
  ON approval_signoffs (stage_instance_id);

-- Tenant-consistency trigger: actor_tenant_id must equal approval_instances.tenant_id.
CREATE OR REPLACE FUNCTION enforce_signoff_tenant_consistent()
  RETURNS trigger AS $$
DECLARE
  instance_tenant UUID;
BEGIN
  SELECT tenant_id INTO instance_tenant
    FROM public.approval_instances
   WHERE id = NEW.approval_instance_id;

  IF instance_tenant IS DISTINCT FROM NEW.actor_tenant_id THEN
    RAISE EXCEPTION 'cross-tenant signoff rejected (instance tenant %, actor tenant %)',
                    instance_tenant, NEW.actor_tenant_id
      USING ERRCODE = 'check_violation';
  END IF;

  RETURN NEW;
END;
$$ LANGUAGE plpgsql
   SET search_path = pg_catalog, pg_temp;

DROP TRIGGER IF EXISTS trg_signoff_tenant_consistent ON approval_signoffs;
CREATE TRIGGER trg_signoff_tenant_consistent
  BEFORE INSERT ON approval_signoffs
  FOR EACH ROW EXECUTE FUNCTION enforce_signoff_tenant_consistent();

-- SoD: author-self-sign block. Cross-stage duplicate is handled by
-- UNIQUE(approval_instance_id, actor_user_id) -- not this trigger.
CREATE OR REPLACE FUNCTION enforce_signoff_sod() RETURNS trigger AS $$
DECLARE
  author_id TEXT;
BEGIN
  SELECT d.created_by INTO author_id
    FROM public.approval_instances i
    JOIN public.documents d ON d.id = i.document_v2_id
   WHERE i.id = NEW.approval_instance_id;

  IF NEW.actor_user_id = author_id THEN
    RAISE EXCEPTION 'SoD: author cannot sign own revision'
      USING ERRCODE = 'check_violation';
  END IF;

  RETURN NEW;
END;
$$ LANGUAGE plpgsql
   SET search_path = pg_catalog, pg_temp;

DROP TRIGGER IF EXISTS trg_signoff_sod ON approval_signoffs;
CREATE TRIGGER trg_signoff_sod
  BEFORE INSERT ON approval_signoffs
  FOR EACH ROW EXECUTE FUNCTION enforce_signoff_sod();

-- Immutability.
CREATE OR REPLACE FUNCTION reject_signoff_update() RETURNS trigger AS $$
BEGIN
  RAISE EXCEPTION 'approval_signoffs rows are immutable'
    USING ERRCODE = 'check_violation';
END;
$$ LANGUAGE plpgsql
   SET search_path = pg_catalog, pg_temp;

DROP TRIGGER IF EXISTS trg_signoff_immutable ON approval_signoffs;
CREATE TRIGGER trg_signoff_immutable
  BEFORE UPDATE ON approval_signoffs
  FOR EACH ROW EXECUTE FUNCTION reject_signoff_update();

COMMIT;
