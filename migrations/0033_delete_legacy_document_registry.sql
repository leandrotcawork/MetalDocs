-- NOTE:
-- This migration is intentionally a no-op in the official chain.
-- A previous destructive cleanup (including deletes on audit-linked data)
-- was moved to dev-only tooling to preserve additive-first policy and
-- append-only audit guarantees.
--
-- For local/dev reset workflows only, use:
--   scripts/sql/dev_reset_legacy_document_registry.sql
-- and follow:
--   docs/runbooks/dev-legacy-registry-reset.md

SELECT 1;
