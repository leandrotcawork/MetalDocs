CREATE TABLE IF NOT EXISTS metaldocs.auth_identities (
  user_id TEXT PRIMARY KEY REFERENCES metaldocs.iam_users(user_id) ON DELETE CASCADE,
  username TEXT NOT NULL,
  email TEXT,
  password_hash TEXT NOT NULL,
  password_algo TEXT NOT NULL,
  must_change_password BOOLEAN NOT NULL DEFAULT FALSE,
  last_login_at TIMESTAMPTZ,
  failed_login_attempts INT NOT NULL DEFAULT 0,
  locked_until TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_auth_identities_username_ci ON metaldocs.auth_identities (LOWER(username));
CREATE UNIQUE INDEX IF NOT EXISTS uq_auth_identities_email_ci ON metaldocs.auth_identities (LOWER(email)) WHERE email IS NOT NULL;

CREATE TABLE IF NOT EXISTS metaldocs.auth_sessions (
  session_id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL REFERENCES metaldocs.iam_users(user_id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  expires_at TIMESTAMPTZ NOT NULL,
  revoked_at TIMESTAMPTZ,
  ip_address TEXT,
  user_agent TEXT,
  last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_auth_sessions_user_id ON metaldocs.auth_sessions (user_id);
CREATE INDEX IF NOT EXISTS idx_auth_sessions_active ON metaldocs.auth_sessions (user_id, expires_at DESC) WHERE revoked_at IS NULL;
