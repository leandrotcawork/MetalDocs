-- migrations/0150_authz_user_process_areas_view.sql
-- authz.Require queries metaldocs.user_process_areas and metaldocs.role_capabilities.
-- role_capabilities is a native metaldocs-schema table (created in 0142a).
-- user_process_areas lives in public schema (0125/0136); expose it through a
-- metaldocs-schema view so authz.go can use a single qualified path.

BEGIN;

CREATE OR REPLACE VIEW metaldocs.user_process_areas AS
    SELECT user_id,
           tenant_id,
           area_code,
           role,
           effective_from,
           effective_to,
           granted_by
      FROM public.user_process_areas;

COMMIT;
