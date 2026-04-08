-- 0064_clean_slate_old_documents.sql
-- Clean-slate: delete all existing PO documents, versions, and old templates.
-- Per the MDDM design spec, existing PO docs are test data.

BEGIN;

-- Delete all document versions for PO documents (cascades to revisions, etc.)
DELETE FROM metaldocs.document_versions_mddm
WHERE document_id IN (
  SELECT id FROM metaldocs.documents WHERE id LIKE 'PO-%'
);

DELETE FROM metaldocs.document_versions
WHERE document_id IN (
  SELECT id FROM metaldocs.documents WHERE id LIKE 'PO-%'
);

DELETE FROM metaldocs.documents WHERE id LIKE 'PO-%';

-- Delete old browser template versions (replaced by MDDM template)
DELETE FROM metaldocs.document_template_versions
WHERE template_key = 'po-default-browser';

DELETE FROM metaldocs.document_profile_template_defaults
WHERE template_key = 'po-default-browser';

-- Clean up orphan images that are no longer referenced
DELETE FROM metaldocs.document_images
WHERE NOT EXISTS (
  SELECT 1 FROM metaldocs.document_version_images WHERE image_id = document_images.id
);

COMMIT;
