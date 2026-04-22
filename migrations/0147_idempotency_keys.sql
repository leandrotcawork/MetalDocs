-- migrations/0147_idempotency_keys.sql
-- Phase 7 Q2: idempotency store for request replay protection and cached responses.

BEGIN;

CREATE TABLE IF NOT EXISTS metaldocs.idempotency_keys (
    tenant_id        UUID        NOT NULL,
    actor_user_id    TEXT        NOT NULL,
    route_template   TEXT        NOT NULL,  -- e.g. 'POST /api/v2/documents/{id}/submit'
    key              TEXT        NOT NULL,  -- client-supplied Idempotency-Key header value
    payload_hash     TEXT        NOT NULL,  -- SHA-256 hex of canonical request body
    response_status  INT         NOT NULL,
    response_body    JSONB       NOT NULL,
    status           TEXT        NOT NULL CHECK (status IN ('in_flight', 'completed', 'failed')),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at       TIMESTAMPTZ NOT NULL,  -- created_at + 24h
    PRIMARY KEY (tenant_id, actor_user_id, route_template, key)
);

-- Index for janitor sweep (Phase 8)
CREATE INDEX IF NOT EXISTS idx_idempotency_keys_expires_at
    ON metaldocs.idempotency_keys (expires_at);

-- Access grants (metaldocs_writer is the app role for mutations)
GRANT SELECT, INSERT, UPDATE ON metaldocs.idempotency_keys TO metaldocs_writer;

COMMIT;
