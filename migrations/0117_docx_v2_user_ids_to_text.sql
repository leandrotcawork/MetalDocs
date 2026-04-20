-- user-facing ID columns hold auth identity strings (not IAM UUIDs)
ALTER TABLE editor_sessions ALTER COLUMN user_id TYPE TEXT;
ALTER TABLE document_checkpoints ALTER COLUMN created_by TYPE TEXT;
ALTER TABLE template_audit_log ALTER COLUMN actor_user_id TYPE TEXT;
