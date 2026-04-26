-- migrations/0158_fix_process_area_role_constraint.sql
--
-- The user_process_areas_role_check constraint (introduced in 0125) only
-- allowed the four IAM roles (viewer/editor/reviewer/approver), but
-- role_capabilities (0142b) defines process-area roles (author/signer/
-- area_admin/qms_admin) that the authz JOIN resolves.  Without this fix
-- any authz.Require call for an approval capability returns 403 even for
-- privileged users.
--
-- Fix:
--   1. Widen the CHECK constraint to include all role_capabilities roles.
--   2. Add doc.submit to the reviewer role (small-team convenience).
--   3. Seed the default dev admin user with qms_admin in the general area
--      so the full Draft→Submit→Signoff→Publish flow works out of the box.
--
-- Idempotent throughout.

BEGIN;

-- 1. Widen allowlist.
ALTER TABLE public.user_process_areas
  DROP CONSTRAINT IF EXISTS user_process_areas_role_check,
  ADD CONSTRAINT user_process_areas_role_check
    CHECK (role IN (
      'viewer', 'editor', 'reviewer', 'approver',
      'author', 'signer', 'area_admin', 'qms_admin'
    ));

-- 2. Fill capability gaps: qms_admin should be able to submit, reviewer too.
INSERT INTO metaldocs.role_capabilities (role, capability, description) VALUES
  ('qms_admin', 'doc.submit', 'Submit document for approval'),
  ('reviewer',  'doc.submit', 'Submit document for approval')
ON CONFLICT (role, capability) DO NOTHING;

-- 3. Dev-tenant seed: give the default admin user qms_admin in general area.
--    Revoke any conflicting active row first (trigger allows effective_to update).
UPDATE public.user_process_areas
   SET effective_to = now(),
       revoked_by   = 'admin'
 WHERE user_id      = 'admin'
   AND tenant_id    = 'ffffffff-ffff-ffff-ffff-ffffffffffff'
   AND area_code    = 'general'
   AND role        != 'qms_admin'
   AND effective_to IS NULL;

INSERT INTO public.user_process_areas
  (user_id, tenant_id, area_code, role, effective_from, granted_by)
VALUES
  ('admin', 'ffffffff-ffff-ffff-ffff-ffffffffffff', 'general', 'qms_admin', now(), 'admin')
ON CONFLICT DO NOTHING;

COMMIT;
