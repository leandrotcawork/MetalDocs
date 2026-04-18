-- 0112_docx_v2_schema_migrations_ledger.sql
-- Introduces a minimal schema_migrations table so destructive migrations
-- (0113 onwards) can be idempotent under re-run, and operators can query
-- applied-state without re-executing. Idempotent: IF NOT EXISTS guards
-- every CREATE; INSERT is ON CONFLICT DO NOTHING.

BEGIN;

CREATE TABLE IF NOT EXISTS public.schema_migrations (
  version     TEXT PRIMARY KEY,
  applied_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  description TEXT
);

-- Record this migration itself.
INSERT INTO public.schema_migrations (version, description)
VALUES ('0112', 'bootstrap schema_migrations ledger (docx-v2/w5)')
ON CONFLICT (version) DO NOTHING;

COMMIT;
