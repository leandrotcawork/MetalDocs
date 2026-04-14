-- 0075_create_template_drafts_and_audit.sql
-- Creates template_drafts (admin editing scratch-pad) and template_audit_log
-- (append-only event ledger) for the Template Admin Phase 2 feature.
-- No application code reads these tables yet; they are created here ahead of
-- the feature implementation so schema migrations stay ahead of code.

CREATE TABLE IF NOT EXISTS metaldocs.template_drafts (
    template_key         TEXT PRIMARY KEY,
    profile_code         TEXT NOT NULL REFERENCES metaldocs.document_profiles(code),
    base_version         INT NOT NULL DEFAULT 0,
    name                 TEXT NOT NULL,
    theme_json           JSONB NOT NULL DEFAULT '{}'::jsonb,
    meta_json            JSONB NOT NULL DEFAULT '{}'::jsonb,
    blocks_json          JSONB NOT NULL,
    lock_version         INT NOT NULL DEFAULT 1,
    has_stripped_fields  BOOLEAN NOT NULL DEFAULT false,
    stripped_fields_json JSONB,
    created_by           TEXT NOT NULL,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_template_drafts_profile ON metaldocs.template_drafts(profile_code);

CREATE TABLE IF NOT EXISTS metaldocs.template_audit_log (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    template_key TEXT NOT NULL,
    version      INT,
    action       TEXT NOT NULL,
    actor_id     TEXT NOT NULL,
    diff_summary TEXT,
    trace_id     TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_template_audit_log_key   ON metaldocs.template_audit_log(template_key);
CREATE INDEX IF NOT EXISTS idx_template_audit_log_actor ON metaldocs.template_audit_log(actor_id);
