-- Execute como usuario admin/superuser no PostgreSQL.
-- Ajuste senha e database conforme ambiente.

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'metaldocs_backup') THEN
    CREATE ROLE metaldocs_backup LOGIN PASSWORD 'CHANGE_ME_BACKUP_PASSWORD';
  END IF;
END$$;

GRANT CONNECT ON DATABASE metaldocs TO metaldocs_backup;
GRANT USAGE ON SCHEMA metaldocs TO metaldocs_backup;
GRANT SELECT ON ALL TABLES IN SCHEMA metaldocs TO metaldocs_backup;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA metaldocs TO metaldocs_backup;

ALTER DEFAULT PRIVILEGES IN SCHEMA metaldocs
  GRANT SELECT ON TABLES TO metaldocs_backup;

ALTER DEFAULT PRIVILEGES IN SCHEMA metaldocs
  GRANT USAGE, SELECT ON SEQUENCES TO metaldocs_backup;

