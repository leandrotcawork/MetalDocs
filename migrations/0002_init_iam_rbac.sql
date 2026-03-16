-- MetalDocs IAM/RBAC foundational schema

CREATE TABLE IF NOT EXISTS metaldocs.iam_users (
  user_id TEXT PRIMARY KEY,
  display_name TEXT NOT NULL,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS metaldocs.iam_user_roles (
  user_id TEXT NOT NULL REFERENCES metaldocs.iam_users(user_id) ON DELETE CASCADE,
  role_code TEXT NOT NULL,
  assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  assigned_by TEXT,
  PRIMARY KEY (user_id, role_code)
);

CREATE INDEX IF NOT EXISTS idx_iam_user_roles_role_code ON metaldocs.iam_user_roles(role_code);
