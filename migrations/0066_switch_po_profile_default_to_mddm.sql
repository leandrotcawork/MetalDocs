-- Switch the PO profile default template to po-mddm-canvas (MDDM).
-- Idempotent: safe to re-run; creates the row if missing, otherwise updates it.

INSERT INTO metaldocs.document_profile_template_defaults (
  profile_code,
  template_key,
  template_version,
  assigned_at
)
VALUES ('po', 'po-mddm-canvas', 1, NOW())
ON CONFLICT (profile_code) DO UPDATE
SET
  template_key     = EXCLUDED.template_key,
  template_version = EXCLUDED.template_version,
  assigned_at      = NOW();
