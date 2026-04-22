-- migrations/0146_approval_routes_active_column.sql
-- Adds an active flag to approval_routes (idempotent).

BEGIN;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
      FROM information_schema.columns
     WHERE table_schema = 'public'
       AND table_name = 'approval_routes'
       AND column_name = 'active'
  ) THEN
    ALTER TABLE public.approval_routes
      ADD COLUMN active BOOLEAN NOT NULL DEFAULT TRUE;
  END IF;
END;
$$;

COMMIT;
