-- published_by holds auth identity user_id strings (not IAM UUIDs)
ALTER TABLE template_versions ALTER COLUMN published_by TYPE TEXT;
