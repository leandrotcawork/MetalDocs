-- created_by columns hold auth identity user_id strings (not IAM UUIDs)
ALTER TABLE templates ALTER COLUMN created_by TYPE TEXT;
ALTER TABLE template_versions ALTER COLUMN created_by TYPE TEXT;
