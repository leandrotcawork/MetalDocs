-- migrations/0142b_role_capabilities_v2_enforce.sql
-- Spec 2 Phase 6 — Enforcement step.
-- Applied AFTER all binary instances are upgraded to v2 (0142a must precede this).
--
-- This migration:
--   1. Strips legacy capability names from role_capabilities
--   2. Adds CHECK constraints preventing legacy caps from being re-inserted
--   3. Installs a tripwire trigger that reads metaldocs.asserted_caps GUC and
--      verifies the required {cap} tuple is present before state-changing DML
--      on approval_instances and approval_signoffs.
--
-- Idempotent: DROP CONSTRAINT IF EXISTS before ADD; DROP TRIGGER IF EXISTS before CREATE.
-- Trigger function uses SECURITY DEFINER + SET search_path = pg_catalog, pg_temp
-- matching Phase 1 hardening pattern (0137).

BEGIN;

-- ---------------------------------------------------------------------------
-- 1. Strip legacy capabilities (explicit count notice for review gate).
--    'document.finalize' and 'document.archive' were v1 names superseded by
--    'doc.publish' and 'doc.obsolete' in 0142a.
-- ---------------------------------------------------------------------------
DO $$ DECLARE deleted_count INT; BEGIN
  DELETE FROM metaldocs.role_capabilities
  WHERE capability IN ('document.finalize', 'document.archive');
  GET DIAGNOSTICS deleted_count = ROW_COUNT;
  RAISE NOTICE 'Deleted % legacy capability rows', deleted_count;
END $$;

-- ---------------------------------------------------------------------------
-- 2. CHECK constraints on role_capabilities.
--    Idempotent: drop first, then add.
-- ---------------------------------------------------------------------------
ALTER TABLE metaldocs.role_capabilities
  DROP CONSTRAINT IF EXISTS ck_cap_not_legacy;

ALTER TABLE metaldocs.role_capabilities
  ADD CONSTRAINT ck_cap_not_legacy
    CHECK (capability NOT IN ('document.finalize', 'document.archive'));

ALTER TABLE metaldocs.role_capabilities
  DROP CONSTRAINT IF EXISTS ck_cap_format;

ALTER TABLE metaldocs.role_capabilities
  ADD CONSTRAINT ck_cap_format
    CHECK (capability ~ '^[a-z][a-z0-9._]*[a-z0-9]$');

-- ---------------------------------------------------------------------------
-- 3. Tripwire trigger function.
--
--    Reads session-local GUCs set by application middleware:
--      metaldocs.asserted_caps — JSONB array of {"cap": "...", "area": "..."}
--      metaldocs.bypass_authz  — opaque bypass token (e.g. 'scheduler')
--
--    Per-table required capability (keyed on TG_TABLE_NAME):
--      approval_instances  INSERT → 'doc.submit'
--      approval_signoffs   INSERT → 'doc.signoff'
--      anything else              → skip check (conservative — don't break unknown tables)
--
--    Bypass path: if bypass_authz = 'scheduler', logs an authz.bypass_used event
--    to governance_events using the instance/signoff tenant_id as context, then
--    allows the DML through. All other bypass tokens are rejected.
--
--    Area enforcement deferred to Phase 7 HTTP middleware — this version checks
--    cap presence only, not {cap, area} tuple.
-- ---------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION public.enforce_capability_asserted()
  RETURNS trigger
  LANGUAGE plpgsql
  SECURITY DEFINER
  SET search_path = pg_catalog, pg_temp
AS $$
DECLARE
  v_bypass       TEXT;
  v_asserted_raw TEXT;
  v_asserted     JSONB;
  v_required_cap TEXT;
  v_tenant_id    UUID;
  v_cap_found    BOOLEAN := FALSE;
  v_element      JSONB;
BEGIN
  -- Determine required capability for this table/operation.
  -- Only INSERT is guarded; UPDATE/DELETE on these tables go through other paths.
  IF TG_TABLE_NAME = 'approval_instances' AND TG_OP = 'INSERT' THEN
    v_required_cap := 'doc.submit';
    v_tenant_id    := NEW.tenant_id;
  ELSIF TG_TABLE_NAME = 'approval_signoffs' AND TG_OP = 'INSERT' THEN
    v_required_cap := 'doc.signoff';
    -- approval_signoffs has actor_tenant_id; use that as the logging context.
    v_tenant_id    := NEW.actor_tenant_id;
  ELSE
    -- Unknown table or operation — conservative pass-through.
    RETURN NEW;
  END IF;

  -- Read bypass token.
  v_bypass := pg_catalog.current_setting('metaldocs.bypass_authz', true);

  IF v_bypass IS NOT NULL AND v_bypass <> '' THEN
    IF v_bypass = 'scheduler' THEN
      -- Audit the bypass and allow through.
      BEGIN
        INSERT INTO public.governance_events
          (tenant_id, event_type, actor_user_id, resource_type, resource_id, reason, payload_json)
        VALUES
          (
            v_tenant_id,
            'authz.bypass_used',
            'system:scheduler',
            TG_TABLE_NAME,
            -- resource_id: use NEW.id if it already exists (gen_random_uuid default
            -- means it is set at row construction time before trigger fires).
            COALESCE(NEW.id::TEXT, 'unknown'),
            'scheduler bypass for ' || v_required_cap,
            pg_catalog.to_jsonb(
              jsonb_build_object(
                'required_cap', v_required_cap,
                'bypass_token',  v_bypass,
                'table',         TG_TABLE_NAME,
                'op',            TG_OP
              )
            )
          );
      EXCEPTION WHEN others THEN
        -- Governance insert failure must not block the bypass itself; log as notice.
        RAISE NOTICE 'enforce_capability_asserted: governance_events insert failed: %', SQLERRM;
      END;
      RETURN NEW;
    ELSE
      -- Unrecognised bypass token — treat as no bypass (fall through to cap check).
      -- Do not silently accept unknown tokens.
      RAISE EXCEPTION 'ErrCapabilityNotAsserted: unrecognised bypass token; cap % required on %',
                      v_required_cap, TG_TABLE_NAME
        USING ERRCODE = 'P0001';
    END IF;
  END IF;

  -- Read asserted_caps GUC.
  v_asserted_raw := pg_catalog.current_setting('metaldocs.asserted_caps', true);

  IF v_asserted_raw IS NULL OR v_asserted_raw = '' THEN
    RAISE EXCEPTION 'ErrCapabilityNotAsserted: cap % required but metaldocs.asserted_caps is not set',
                    v_required_cap
      USING ERRCODE = 'P0001';
  END IF;

  -- Parse as JSONB array; handle malformed input safely.
  BEGIN
    v_asserted := v_asserted_raw::JSONB;
  EXCEPTION WHEN invalid_text_representation OR others THEN
    RAISE EXCEPTION 'ErrCapabilityNotAsserted: metaldocs.asserted_caps is not valid JSONB (cap % required)',
                    v_required_cap
      USING ERRCODE = 'P0001';
  END;

  IF jsonb_typeof(v_asserted) <> 'array' THEN
    RAISE EXCEPTION 'ErrCapabilityNotAsserted: metaldocs.asserted_caps must be a JSONB array (cap % required)',
                    v_required_cap
      USING ERRCODE = 'P0001';
  END IF;

  -- Scan the array for the required cap.
  -- Phase 7 will add area matching; this version checks cap name only.
  FOR v_element IN SELECT * FROM jsonb_array_elements(v_asserted) LOOP
    IF (v_element->>'cap') = v_required_cap THEN
      v_cap_found := TRUE;
      EXIT;
    END IF;
  END LOOP;

  IF NOT v_cap_found THEN
    RAISE EXCEPTION 'ErrCapabilityNotAsserted: cap % required but not present in asserted_caps',
                    v_required_cap
      USING ERRCODE = 'P0001';
  END IF;

  RETURN NEW;
END;
$$;

-- Ownership: assign to security owner so the SECURITY DEFINER context is
-- the hardened role, not the migration executor (matches Phase 1 pattern).
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'metaldocs_security_owner') THEN
    EXECUTE 'ALTER FUNCTION public.enforce_capability_asserted() OWNER TO metaldocs_security_owner';
  END IF;
END $$;

-- Revoke public EXECUTE — only internal trigger mechanism invokes this.
REVOKE EXECUTE ON FUNCTION public.enforce_capability_asserted() FROM PUBLIC;

-- ---------------------------------------------------------------------------
-- 4. Install tripwire triggers.
--    BEFORE INSERT so the function can RAISE EXCEPTION and abort the DML.
--    Idempotent: DROP IF EXISTS before CREATE.
-- ---------------------------------------------------------------------------

-- approval_instances
DROP TRIGGER IF EXISTS trg_require_cap_asserted_instances ON public.approval_instances;
CREATE TRIGGER trg_require_cap_asserted_instances
  BEFORE INSERT ON public.approval_instances
  FOR EACH ROW EXECUTE FUNCTION public.enforce_capability_asserted();

-- approval_signoffs
DROP TRIGGER IF EXISTS trg_require_cap_asserted_signoffs ON public.approval_signoffs;
CREATE TRIGGER trg_require_cap_asserted_signoffs
  BEFORE INSERT ON public.approval_signoffs
  FOR EACH ROW EXECUTE FUNCTION public.enforce_capability_asserted();

-- Note: documents UPDATE is NOT guarded here — too broad for a DB trigger;
-- deferred to Phase 7 HTTP layer capability middleware.

COMMIT;
