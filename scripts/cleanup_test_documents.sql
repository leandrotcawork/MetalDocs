-- scripts/cleanup_test_documents.sql
-- Run manually to delete test documents. NOT an auto-migration.
-- Usage: psql -U metaldocs -d metaldocs -f scripts/cleanup_test_documents.sql

BEGIN;

DELETE FROM metaldocs.document_collaboration_presence;
DELETE FROM metaldocs.document_edit_locks;
DELETE FROM metaldocs.document_attachments;
DELETE FROM metaldocs.document_versions;
DELETE FROM metaldocs.documents;
DELETE FROM metaldocs.outbox_events WHERE aggregate_type = 'document';

COMMIT;
