-- migrations/0141_governance_events_dedupe_signoff_uniqueness.sql

BEGIN;

-- 1. Add dedupe_key column to governance_events (idempotent)
ALTER TABLE governance_events ADD COLUMN IF NOT EXISTS dedupe_key TEXT;

-- 2. Add correlation_id column to governance_events (idempotent)
ALTER TABLE governance_events ADD COLUMN IF NOT EXISTS correlation_id TEXT;

-- 3. Partial unique index on governance_events for deduplication (idempotent via IF NOT EXISTS)
CREATE UNIQUE INDEX IF NOT EXISTS ux_gov_events_dedupe
    ON governance_events (event_type, dedupe_key)
    WHERE dedupe_key IS NOT NULL;

-- 4. Ensure a named unique constraint exists on approval_signoffs (stage_instance_id, actor_user_id).
--    Migration 0135 may have created an inline UNIQUE auto-named
--    approval_signoffs_stage_instance_id_actor_user_id_key.
--    Guard by known constraint names to avoid duplicate_object error on re-run.
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.approval_signoffs'::regclass
          AND conname IN (
              'ux_signoff_stage_actor',
              'approval_signoffs_stage_instance_id_actor_user_id_key'
          )
    ) THEN
        ALTER TABLE approval_signoffs
            ADD CONSTRAINT ux_signoff_stage_actor
            UNIQUE (stage_instance_id, actor_user_id);
    END IF;
END;
$$;

COMMIT;
