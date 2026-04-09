UPDATE metaldocs.document_profile_template_defaults
SET
  template_key     = 'po-mddm-canvas',
  template_version = 1,
  assigned_at      = NOW()
WHERE profile_code = 'po';
