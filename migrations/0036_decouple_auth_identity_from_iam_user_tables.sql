ALTER TABLE metaldocs.auth_identities
  ADD COLUMN IF NOT EXISTS display_name TEXT,
  ADD COLUMN IF NOT EXISTS is_active BOOLEAN;

UPDATE metaldocs.auth_identities ai
SET display_name = COALESCE(ai.display_name, iu.display_name, ai.username),
    is_active = COALESCE(ai.is_active, iu.is_active, TRUE)
FROM metaldocs.iam_users iu
WHERE iu.user_id = ai.user_id;

UPDATE metaldocs.auth_identities
SET display_name = COALESCE(display_name, username),
    is_active = COALESCE(is_active, TRUE)
WHERE display_name IS NULL OR is_active IS NULL;

ALTER TABLE metaldocs.auth_identities
  ALTER COLUMN display_name SET NOT NULL,
  ALTER COLUMN is_active SET NOT NULL,
  ALTER COLUMN is_active SET DEFAULT TRUE;

DO $$
DECLARE
  constraint_name TEXT;
BEGIN
  SELECT con.conname
  INTO constraint_name
  FROM pg_constraint con
  JOIN pg_class rel ON rel.oid = con.conrelid
  JOIN pg_namespace ns ON ns.oid = rel.relnamespace
  JOIN pg_class ref ON ref.oid = con.confrelid
  JOIN pg_namespace refns ON refns.oid = ref.relnamespace
  WHERE ns.nspname = 'metaldocs'
    AND rel.relname = 'auth_identities'
    AND refns.nspname = 'metaldocs'
    AND ref.relname = 'iam_users'
    AND con.contype = 'f'
  LIMIT 1;

  IF constraint_name IS NOT NULL THEN
    EXECUTE format('ALTER TABLE metaldocs.auth_identities DROP CONSTRAINT %I', constraint_name);
  END IF;
END $$;

DO $$
DECLARE
  constraint_name TEXT;
BEGIN
  SELECT con.conname
  INTO constraint_name
  FROM pg_constraint con
  JOIN pg_class rel ON rel.oid = con.conrelid
  JOIN pg_namespace ns ON ns.oid = rel.relnamespace
  JOIN pg_class ref ON ref.oid = con.confrelid
  JOIN pg_namespace refns ON refns.oid = ref.relnamespace
  WHERE ns.nspname = 'metaldocs'
    AND rel.relname = 'auth_sessions'
    AND refns.nspname = 'metaldocs'
    AND ref.relname = 'iam_users'
    AND con.contype = 'f'
  LIMIT 1;

  IF constraint_name IS NOT NULL THEN
    EXECUTE format('ALTER TABLE metaldocs.auth_sessions DROP CONSTRAINT %I', constraint_name);
  END IF;
END $$;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint con
    JOIN pg_class rel ON rel.oid = con.conrelid
    JOIN pg_namespace ns ON ns.oid = rel.relnamespace
    JOIN pg_class ref ON ref.oid = con.confrelid
    JOIN pg_namespace refns ON refns.oid = ref.relnamespace
    WHERE ns.nspname = 'metaldocs'
      AND rel.relname = 'auth_sessions'
      AND refns.nspname = 'metaldocs'
      AND ref.relname = 'auth_identities'
      AND con.contype = 'f'
  ) THEN
    ALTER TABLE metaldocs.auth_sessions
      ADD CONSTRAINT fk_auth_sessions_identity
      FOREIGN KEY (user_id)
      REFERENCES metaldocs.auth_identities(user_id)
      ON DELETE CASCADE;
  END IF;
END $$;
