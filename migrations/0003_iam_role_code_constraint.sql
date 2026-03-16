ALTER TABLE metaldocs.iam_user_roles
  DROP CONSTRAINT IF EXISTS chk_iam_user_roles_role_code;

ALTER TABLE metaldocs.iam_user_roles
  ADD CONSTRAINT chk_iam_user_roles_role_code
  CHECK (role_code IN ('admin', 'editor', 'reviewer', 'viewer'));
