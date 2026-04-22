-- migrations/0143_area_membership_fns.sql
-- Phase 6: canonical SECURITY DEFINER functions for area-membership writes.
-- After this migration, direct INSERT/UPDATE/DELETE on user_process_areas by
-- metaldocs_membership_writer and metaldocs_app is revoked — all writes go
-- through metaldocs.grant_area_membership / metaldocs.revoke_area_membership.
--
-- Schema note: user_process_areas has no surrogate id column (PK is composite).
--   grant_area_membership returns gen_random_uuid() as a correlation handle,
--   consistent with the existing public.grant_area_membership in 0137.
-- Column note: soft-delete uses effective_to (not revoked_at); see 0136.
-- Role values: must match the CHECK constraint defined in 0125.

BEGIN;

-- ─── 1. GRANT USAGE on metaldocs schema to security owner (idempotent) ────────
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_namespace WHERE nspname = 'metaldocs') THEN
    EXECUTE 'GRANT USAGE ON SCHEMA metaldocs TO metaldocs_security_owner';
  END IF;
END $$;

-- ─── 2. Ensure security owner can write to the tables it needs ────────────────
DO $$
BEGIN
  -- user_process_areas lives in public schema (0125 created it there).
  EXECUTE 'GRANT SELECT, INSERT, UPDATE ON public.user_process_areas TO metaldocs_security_owner';
  IF EXISTS (SELECT 1 FROM pg_tables WHERE schemaname = 'metaldocs' AND tablename = 'governance_events') THEN
    EXECUTE 'GRANT INSERT ON metaldocs.governance_events TO metaldocs_security_owner';
  ELSIF EXISTS (SELECT 1 FROM pg_tables WHERE schemaname = 'public' AND tablename = 'governance_events') THEN
    EXECUTE 'GRANT INSERT ON public.governance_events TO metaldocs_security_owner';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_tables WHERE schemaname = 'metaldocs' AND tablename = 'iam_users') THEN
    EXECUTE 'GRANT SELECT ON metaldocs.iam_users TO metaldocs_security_owner';
  END IF;
END $$;

-- ─── 3. grant_area_membership ─────────────────────────────────────────────────
CREATE OR REPLACE FUNCTION metaldocs.grant_area_membership(
  _tenant_id   UUID,
  _user_id     TEXT,
  _area_code   TEXT,
  _role        TEXT,
  _granted_by  TEXT
) RETURNS UUID AS $$
DECLARE
  _existing_from  TIMESTAMPTZ;
  _now            TIMESTAMPTZ := pg_catalog.clock_timestamp();
  _correlation_id UUID        := pg_catalog.gen_random_uuid();
BEGIN
  -- ── Input validation (before any query) ──────────────────────────────────
  IF _user_id !~ '^[a-z0-9_.@-]+$' THEN
    RAISE EXCEPTION 'invalid user_id: %', _user_id USING ERRCODE = '22023';
  END IF;
  IF _area_code !~ '^[A-Z0-9_]+$' THEN
    RAISE EXCEPTION 'invalid area_code: %', _area_code USING ERRCODE = '22023';
  END IF;
  -- Role values must match the CHECK constraint on user_process_areas (0125).
  IF _role NOT IN ('viewer', 'editor', 'reviewer', 'approver') THEN
    RAISE EXCEPTION 'invalid role: % (allowed: viewer, editor, reviewer, approver)', _role
      USING ERRCODE = '22023';
  END IF;
  IF _granted_by !~ '^[a-z0-9_.@-]+$' THEN
    RAISE EXCEPTION 'invalid granted_by: %', _granted_by USING ERRCODE = '22023';
  END IF;

  -- ── Idempotency check ────────────────────────────────────────────────────
  SELECT effective_from
    INTO _existing_from
    FROM public.user_process_areas
   WHERE tenant_id    = _tenant_id
     AND user_id      = _user_id
     AND area_code    = _area_code
     AND role         = _role
     AND effective_to IS NULL
   LIMIT 1;

  IF FOUND THEN
    -- Already active — return correlation id without side-effects.
    RETURN _correlation_id;
  END IF;

  -- ── Insert membership row ────────────────────────────────────────────────
  INSERT INTO public.user_process_areas
    (user_id, tenant_id, area_code, role, effective_from, effective_to, granted_by, revoked_by)
  VALUES
    (_user_id, _tenant_id, _area_code, _role, _now, NULL, _granted_by, NULL);

  -- ── Emit governance event ────────────────────────────────────────────────
  INSERT INTO metaldocs.governance_events
    (tenant_id, event_type, actor_user_id, resource_type, resource_id, payload_json)
  VALUES
    (_tenant_id,
     'membership.granted',
     _granted_by,
     'user_process_area',
     _user_id || ':' || _area_code || ':' || _role,
     pg_catalog.to_jsonb(
       pg_catalog.json_build_object(
         'user_id',        _user_id,
         'area_code',      _area_code,
         'role',           _role,
         'granted_by',     _granted_by,
         'effective_from', _now
       )
     )
    );

  RETURN _correlation_id;
END;
$$ LANGUAGE plpgsql
   SECURITY DEFINER
   SET search_path = metaldocs, pg_temp;

-- ─── 4. revoke_area_membership ────────────────────────────────────────────────
CREATE OR REPLACE FUNCTION metaldocs.revoke_area_membership(
  _tenant_id   UUID,
  _user_id     TEXT,
  _area_code   TEXT,
  _role        TEXT,
  _revoked_by  TEXT
) RETURNS void AS $$
DECLARE
  _rows_affected  INT;
  _now            TIMESTAMPTZ := pg_catalog.clock_timestamp();
BEGIN
  -- ── Input validation (before any query) ──────────────────────────────────
  IF _user_id !~ '^[a-z0-9_.@-]+$' THEN
    RAISE EXCEPTION 'invalid user_id: %', _user_id USING ERRCODE = '22023';
  END IF;
  IF _area_code !~ '^[A-Z0-9_]+$' THEN
    RAISE EXCEPTION 'invalid area_code: %', _area_code USING ERRCODE = '22023';
  END IF;
  IF _role NOT IN ('viewer', 'editor', 'reviewer', 'approver') THEN
    RAISE EXCEPTION 'invalid role: % (allowed: viewer, editor, reviewer, approver)', _role
      USING ERRCODE = '22023';
  END IF;
  IF _revoked_by !~ '^[a-z0-9_.@-]+$' THEN
    RAISE EXCEPTION 'invalid revoked_by: %', _revoked_by USING ERRCODE = '22023';
  END IF;

  -- ── Soft-delete: set effective_to + revoked_by ───────────────────────────
  UPDATE public.user_process_areas
     SET effective_to = _now,
         revoked_by   = _revoked_by
   WHERE tenant_id    = _tenant_id
     AND user_id      = _user_id
     AND area_code    = _area_code
     AND role         = _role
     AND effective_to IS NULL;

  GET DIAGNOSTICS _rows_affected = ROW_COUNT;

  IF _rows_affected = 0 THEN
    RAISE EXCEPTION 'membership not found'
      USING ERRCODE = 'P0002';
  END IF;

  -- ── Emit governance event ────────────────────────────────────────────────
  INSERT INTO metaldocs.governance_events
    (tenant_id, event_type, actor_user_id, resource_type, resource_id, payload_json)
  VALUES
    (_tenant_id,
     'membership.revoked',
     _revoked_by,
     'user_process_area',
     _user_id || ':' || _area_code || ':' || _role,
     pg_catalog.to_jsonb(
       pg_catalog.json_build_object(
         'user_id',      _user_id,
         'area_code',    _area_code,
         'role',         _role,
         'revoked_by',   _revoked_by,
         'effective_to', _now
       )
     )
    );
END;
$$ LANGUAGE plpgsql
   SECURITY DEFINER
   SET search_path = metaldocs, pg_temp;

-- ─── 5. Ownership ────────────────────────────────────────────────────────────
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'metaldocs_security_owner') THEN
    EXECUTE 'ALTER FUNCTION metaldocs.grant_area_membership(UUID, TEXT, TEXT, TEXT, TEXT) OWNER TO metaldocs_security_owner';
    EXECUTE 'ALTER FUNCTION metaldocs.revoke_area_membership(UUID, TEXT, TEXT, TEXT, TEXT) OWNER TO metaldocs_security_owner';
  END IF;
END $$;

-- ─── 6. Revoke EXECUTE from PUBLIC ───────────────────────────────────────────
REVOKE EXECUTE ON FUNCTION metaldocs.grant_area_membership(UUID, TEXT, TEXT, TEXT, TEXT)
  FROM PUBLIC;
REVOKE EXECUTE ON FUNCTION metaldocs.revoke_area_membership(UUID, TEXT, TEXT, TEXT, TEXT)
  FROM PUBLIC;

-- ─── 7. Grant EXECUTE to writer role ─────────────────────────────────────────
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'metaldocs_membership_writer') THEN
    EXECUTE 'GRANT EXECUTE ON FUNCTION metaldocs.grant_area_membership(UUID, TEXT, TEXT, TEXT, TEXT) TO metaldocs_membership_writer';
    EXECUTE 'GRANT EXECUTE ON FUNCTION metaldocs.revoke_area_membership(UUID, TEXT, TEXT, TEXT, TEXT) TO metaldocs_membership_writer';
  END IF;
END $$;

-- ─── 8. Revoke direct DML — all writes must go through SECURITY DEFINER fns ──
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'metaldocs_membership_writer') THEN
    EXECUTE 'REVOKE INSERT, UPDATE, DELETE ON public.user_process_areas FROM metaldocs_membership_writer';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'metaldocs_app') THEN
    EXECUTE 'REVOKE INSERT, UPDATE, DELETE ON public.user_process_areas FROM metaldocs_app';
  END IF;
END $$;

COMMIT;
