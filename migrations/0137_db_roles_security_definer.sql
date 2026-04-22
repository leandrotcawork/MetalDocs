-- migrations/0137_db_roles_security_definer.sql
-- Codex Round 1 fixes: DO $$ guards, pg_temp search_path, schema-qualified refs.
-- DML revoke on user_process_areas deferred to Phase 6 (co-release with IAM cutover).

BEGIN;

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'metaldocs_security_owner') THEN
    CREATE ROLE metaldocs_security_owner NOLOGIN;
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'metaldocs_membership_writer') THEN
    CREATE ROLE metaldocs_membership_writer NOLOGIN NOINHERIT;
  END IF;
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'metaldocs_app') THEN
    EXECUTE 'ALTER ROLE metaldocs_app NOINHERIT';
  END IF;
END $$;

-- Codex Round 2 fix #5: explicit USAGE on metaldocs schema for SECURITY DEFINER owner
-- and the membership_writer role. In hardened envs PUBLIC may lack USAGE here, in which
-- case table-level GRANTs alone are not enough.
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_namespace WHERE nspname='metaldocs') THEN
    EXECUTE 'GRANT USAGE ON SCHEMA metaldocs TO metaldocs_security_owner';
    EXECUTE 'GRANT USAGE ON SCHEMA metaldocs TO metaldocs_membership_writer';
  END IF;
END $$;

-- Owner needs read on iam_users and write on user_process_areas (function body).
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_tables WHERE schemaname='metaldocs' AND tablename='iam_users') THEN
    EXECUTE 'GRANT SELECT ON metaldocs.iam_users TO metaldocs_security_owner';
  END IF;
  EXECUTE 'GRANT SELECT, INSERT, UPDATE ON public.user_process_areas TO metaldocs_security_owner';
END $$;

-- Schema public CREATE lockdown (safe to repeat).
REVOKE CREATE ON SCHEMA public FROM PUBLIC;
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'metaldocs_app') THEN
    EXECUTE 'REVOKE CREATE ON SCHEMA public FROM metaldocs_app';
    EXECUTE 'GRANT  USAGE  ON SCHEMA public TO metaldocs_app';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'metaldocs_readonly') THEN
    EXECUTE 'REVOKE CREATE ON SCHEMA public FROM metaldocs_readonly';
    EXECUTE 'GRANT  USAGE  ON SCHEMA public TO metaldocs_readonly';
  END IF;
  EXECUTE 'REVOKE CREATE ON SCHEMA public FROM metaldocs_membership_writer';
  EXECUTE 'GRANT  USAGE  ON SCHEMA public TO metaldocs_membership_writer';
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'metaldocs_migrator') THEN
    EXECUTE 'GRANT  CREATE ON SCHEMA public TO metaldocs_migrator';
  END IF;
  EXECUTE 'GRANT  CREATE ON SCHEMA public TO metaldocs_security_owner';
END $$;

CREATE OR REPLACE FUNCTION public.grant_area_membership(
  _tenant_id   UUID,
  _user_id     TEXT,
  _area_code   TEXT,
  _role        TEXT,
  _granted_by  TEXT
) RETURNS UUID AS $$
DECLARE
  session_actor  TEXT := pg_catalog.current_setting('metaldocs.actor_id', true);
  session_cap    TEXT := pg_catalog.current_setting('metaldocs.verified_capability', true);
  actor_tenant   UUID;
BEGIN
  IF session_actor IS NULL OR session_actor = '' OR session_actor IS DISTINCT FROM _granted_by THEN
    RAISE EXCEPTION 'session actor context missing or mismatched'
      USING ERRCODE = 'insufficient_privilege';
  END IF;
  IF session_cap IS NULL OR session_cap <> 'workflow.route.edit' THEN
    RAISE EXCEPTION 'session capability context missing or wrong'
      USING ERRCODE = 'insufficient_privilege';
  END IF;
  SELECT tenant_id INTO actor_tenant
    FROM metaldocs.iam_users WHERE user_id = _granted_by;
  IF actor_tenant IS DISTINCT FROM _tenant_id THEN
    RAISE EXCEPTION 'granted_by must belong to same tenant'
      USING ERRCODE = 'check_violation';
  END IF;
  IF NOT EXISTS (
    SELECT 1 FROM metaldocs.iam_users
     WHERE user_id = _granted_by AND deactivated_at IS NULL
  ) THEN
    RAISE EXCEPTION 'granted_by must be active user'
      USING ERRCODE = 'check_violation';
  END IF;
  INSERT INTO public.user_process_areas
    (user_id, tenant_id, area_code, role, effective_from, effective_to, granted_by, revoked_by)
    VALUES (_user_id, _tenant_id, _area_code, _role,
            pg_catalog.clock_timestamp(), NULL, _granted_by, NULL);
  RETURN pg_catalog.gen_random_uuid();
END;
$$ LANGUAGE plpgsql
   SECURITY DEFINER
   SET search_path = pg_catalog, pg_temp;

CREATE OR REPLACE FUNCTION public.revoke_area_membership(
  _tenant_id   UUID,
  _user_id     TEXT,
  _area_code   TEXT,
  _role        TEXT,
  _revoked_by  TEXT
) RETURNS UUID AS $$
DECLARE
  session_actor  TEXT := pg_catalog.current_setting('metaldocs.actor_id', true);
  session_cap    TEXT := pg_catalog.current_setting('metaldocs.verified_capability', true);
  actor_tenant   UUID;
  rows_affected  INT;
BEGIN
  IF session_actor IS NULL OR session_actor = '' OR session_actor IS DISTINCT FROM _revoked_by THEN
    RAISE EXCEPTION 'session actor context missing or mismatched'
      USING ERRCODE = 'insufficient_privilege';
  END IF;
  IF session_cap IS NULL OR session_cap <> 'workflow.route.edit' THEN
    RAISE EXCEPTION 'session capability context missing or wrong'
      USING ERRCODE = 'insufficient_privilege';
  END IF;
  SELECT tenant_id INTO actor_tenant
    FROM metaldocs.iam_users WHERE user_id = _revoked_by;
  IF actor_tenant IS DISTINCT FROM _tenant_id THEN
    RAISE EXCEPTION 'revoked_by must belong to same tenant'
      USING ERRCODE = 'check_violation';
  END IF;
  IF NOT EXISTS (
    SELECT 1 FROM metaldocs.iam_users
     WHERE user_id = _revoked_by AND deactivated_at IS NULL
  ) THEN
    RAISE EXCEPTION 'revoked_by must be active user'
      USING ERRCODE = 'check_violation';
  END IF;
  UPDATE public.user_process_areas
     SET effective_to = pg_catalog.clock_timestamp(),
         revoked_by   = _revoked_by
   WHERE tenant_id    = _tenant_id
     AND user_id      = _user_id
     AND area_code    = _area_code
     AND role         = _role
     AND effective_to IS NULL;
  GET DIAGNOSTICS rows_affected = ROW_COUNT;
  IF rows_affected = 0 THEN
    RAISE EXCEPTION 'no active membership to revoke'
      USING ERRCODE = 'no_data_found';
  END IF;
  RETURN pg_catalog.gen_random_uuid();
END;
$$ LANGUAGE plpgsql
   SECURITY DEFINER
   SET search_path = pg_catalog, pg_temp;

ALTER FUNCTION public.grant_area_membership(UUID, TEXT, TEXT, TEXT, TEXT)
  OWNER TO metaldocs_security_owner;
ALTER FUNCTION public.revoke_area_membership(UUID, TEXT, TEXT, TEXT, TEXT)
  OWNER TO metaldocs_security_owner;

REVOKE EXECUTE ON FUNCTION public.grant_area_membership(UUID, TEXT, TEXT, TEXT, TEXT),
                        public.revoke_area_membership(UUID, TEXT, TEXT, TEXT, TEXT)
  FROM PUBLIC;

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'metaldocs_app') THEN
    EXECUTE 'REVOKE EXECUTE ON FUNCTION public.grant_area_membership(UUID,TEXT,TEXT,TEXT,TEXT), public.revoke_area_membership(UUID,TEXT,TEXT,TEXT,TEXT) FROM metaldocs_app';
  END IF;
END $$;

GRANT EXECUTE ON FUNCTION public.grant_area_membership(UUID, TEXT, TEXT, TEXT, TEXT),
                       public.revoke_area_membership(UUID, TEXT, TEXT, TEXT, TEXT)
  TO metaldocs_membership_writer;

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'metaldocs_app') THEN
    EXECUTE 'GRANT metaldocs_membership_writer TO metaldocs_app';
  END IF;
END $$;

-- Phase 6 TODO: REVOKE INSERT, UPDATE, DELETE ON user_process_areas FROM metaldocs_app;
-- Left intentionally here -- Phase 6 migration co-releases with IAM service switching to
-- SECURITY DEFINER call path. Phase 1 alone must not strand the running IAM service.

COMMIT;
