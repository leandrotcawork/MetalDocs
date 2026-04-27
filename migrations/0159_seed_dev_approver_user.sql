-- migrations/0159_seed_dev_approver_user.sql
-- Seed a second admin-role user for local dev so template approval can be
-- performed by a different account than the author (domain enforces actorID !=
-- authorID, so two distinct admin users satisfy segregation of duties).
--
-- Credentials: identifier=approver  password=ApproverMetalDocs123!
-- Idempotent.

BEGIN;

INSERT INTO metaldocs.auth_identities (user_id, username, display_name, password_hash, password_algo)
VALUES (
  'approver',
  'approver',
  'Approver Dev',
  crypt('ApproverMetalDocs123!', gen_salt('bf', 12)),
  'bcrypt'
)
ON CONFLICT (user_id) DO NOTHING;

INSERT INTO metaldocs.iam_users (user_id, display_name)
VALUES ('approver', 'Approver Dev')
ON CONFLICT (user_id) DO NOTHING;

INSERT INTO metaldocs.iam_user_roles (user_id, role_code)
VALUES ('approver', 'admin')
ON CONFLICT DO NOTHING;

COMMIT;
