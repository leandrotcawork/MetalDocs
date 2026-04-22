-- migrations/0138_grants_approval_tables.sql

BEGIN;

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'metaldocs_app') THEN
    EXECUTE 'GRANT SELECT, INSERT, UPDATE ON approval_routes, approval_route_stages, approval_instances, approval_stage_instances TO metaldocs_app';
    EXECUTE 'GRANT SELECT, INSERT ON approval_signoffs TO metaldocs_app';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'metaldocs_readonly') THEN
    EXECUTE 'GRANT SELECT ON approval_routes, approval_route_stages, approval_instances, approval_stage_instances, approval_signoffs TO metaldocs_readonly';
  END IF;
END $$;

COMMIT;
