-- migrations/0151_seed_dev_tenant_approval_data.sql
-- Three fixes in one migration:
--
-- 1. Role alignment: user_process_areas_role_check only allowed viewer/editor/
--    reviewer/approver, but role_capabilities defines author/signer/area_admin/
--    qms_admin roles too. Extend the constraint so users can be assigned those
--    roles and the authz JOIN can resolve them.
--
-- 2. Capability addition: add doc.submit to reviewer role so that users with the
--    reviewer process-area role can also submit documents (small-team workflow).
--
-- 3. Dev seed: create local dev approval users with correct process-area roles
--    for the full Draft→Submit→Signoff→Publish flow.
--    Dev tenant: ffffffff-ffff-ffff-ffff-ffffffffffff
--    reviewer-1  : reviewer in quality  → doc.submit + doc.signoff
--    admin-local : qms_admin in quality → doc.publish + workflow.instance.cancel
--
-- Note: user_process_areas has a partial unique index on (user_id, tenant_id,
-- area_code) WHERE effective_to IS NULL — one active role per user per area.
-- The UPDATE below atomically shifts admin-local from reviewer → qms_admin
-- instead of inserting a second row.
--
-- Idempotent throughout.

BEGIN;

-- 1. Extend the role allowlist to include all role_capabilities roles.
ALTER TABLE public.user_process_areas
  DROP CONSTRAINT IF EXISTS user_process_areas_role_check,
  ADD  CONSTRAINT user_process_areas_role_check
    CHECK (role IN (
      'viewer', 'editor', 'reviewer', 'approver',
      'author', 'signer', 'area_admin', 'qms_admin'
    ));

-- 2. Add doc.submit capability to the reviewer role.
INSERT INTO metaldocs.role_capabilities (role, capability, description)
VALUES ('reviewer', 'doc.submit', 'Submit document for approval')
ON CONFLICT (role, capability) DO NOTHING;

-- 3. Ensure dev IAM users exist.
INSERT INTO metaldocs.iam_users (user_id, display_name) VALUES
  ('reviewer-1',  'Reviewer One'),
  ('admin-local', 'Administrator')
ON CONFLICT (user_id) DO NOTHING;

-- 4. Set reviewer-1 as reviewer in quality area (idempotent insert).
INSERT INTO public.user_process_areas (user_id, tenant_id, area_code, role, effective_from, granted_by)
VALUES ('reviewer-1', 'ffffffff-ffff-ffff-ffff-ffffffffffff', 'quality', 'reviewer', now(), 'system')
ON CONFLICT DO NOTHING;

-- 5. Set admin-local as qms_admin in quality area.
--    If they already have an active row (e.g. reviewer from manual dev setup),
--    update the role in-place. Otherwise insert.
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM public.user_process_areas
     WHERE user_id    = 'admin-local'
       AND tenant_id  = 'ffffffff-ffff-ffff-ffff-ffffffffffff'
       AND area_code  = 'quality'
       AND effective_to IS NULL
  ) THEN
    UPDATE public.user_process_areas
       SET role = 'qms_admin'
     WHERE user_id    = 'admin-local'
       AND tenant_id  = 'ffffffff-ffff-ffff-ffff-ffffffffffff'
       AND area_code  = 'quality'
       AND effective_to IS NULL;
  ELSE
    INSERT INTO public.user_process_areas (user_id, tenant_id, area_code, role, effective_from, granted_by)
    VALUES ('admin-local', 'ffffffff-ffff-ffff-ffff-ffffffffffff', 'quality', 'qms_admin', now(), 'system');
  END IF;
END $$;

COMMIT;
