-- 0070: telemetry table for shadow-mode DOCX export comparison.
-- During Plan 4 Phase 1, the frontend runs both the docgen and new
-- client-side paths in parallel on every browser_editor export and
-- posts a hash + diff summary here. Engineers aggregate these rows
-- off-line to decide when Phase 2 (canary) is safe to enable.

CREATE TABLE IF NOT EXISTS metaldocs.mddm_shadow_diff_events (
    id                 BIGSERIAL PRIMARY KEY,
    document_id        VARCHAR(64)   NOT NULL,
    version_number     INTEGER       NOT NULL,
    user_id_hash       VARCHAR(64)   NOT NULL,
    current_xml_hash   VARCHAR(64)   NOT NULL,
    shadow_xml_hash    VARCHAR(64)   NOT NULL,
    diff_summary       JSONB         NOT NULL DEFAULT '{}'::jsonb,
    current_duration_ms INTEGER      NOT NULL DEFAULT 0,
    shadow_duration_ms  INTEGER      NOT NULL DEFAULT 0,
    shadow_error       TEXT,
    recorded_at        TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    trace_id           VARCHAR(64)
);

CREATE INDEX IF NOT EXISTS mddm_shadow_diff_events_recorded_at_idx
    ON metaldocs.mddm_shadow_diff_events (recorded_at DESC);

COMMENT ON TABLE metaldocs.mddm_shadow_diff_events IS
    'Phase 1 shadow-test telemetry: compares docgen DOCX against the new client-side DOCX for browser_editor documents. Rows are append-only. user_id_hash is a salted SHA-256 so individual users cannot be identified from the raw table.';
