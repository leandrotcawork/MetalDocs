-- 0069: add renderer_pin JSONB column to document_versions.
-- Captures the renderer version + Layout IR hash + template ref at
-- DRAFT→RELEASED transition so released documents always re-render
-- with the engine that approved them.

ALTER TABLE metaldocs.document_versions
    ADD COLUMN IF NOT EXISTS renderer_pin JSONB;

-- Optional GIN index if we ever query by renderer version; off by default
-- to keep the write path fast. Add later if needed.

COMMENT ON COLUMN metaldocs.document_versions.renderer_pin IS
    'Frozen renderer inputs captured at release time: {renderer_version, layout_ir_hash, template_key, template_version, pinned_at}. NULL for drafts.';
