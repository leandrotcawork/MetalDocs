-- 0059_add_template_export_config.sql
-- Adds export_config JSONB column to document_template_versions.
-- Stores per-template rendering configuration (margins, etc.) used by docgen.
-- NULL means "use docgen defaults" (backward compatible).

ALTER TABLE metaldocs.document_template_versions
  ADD COLUMN IF NOT EXISTS export_config JSONB;
