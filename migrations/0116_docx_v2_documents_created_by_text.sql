-- documents.created_by holds auth identity user_id strings (not IAM UUIDs)
ALTER TABLE documents ALTER COLUMN created_by TYPE TEXT;
