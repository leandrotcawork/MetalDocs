-- migrations/0145_route_config_immutable_trigger.sql
-- Spec 2 Phase 6 (A6). DB-enforced immutability for approval_routes once
-- referenced by an approval_instance. Service-level check is the fast path;
-- this trigger is the race-proof backstop.
-- Idempotent.

BEGIN;

-- -------------------------------------------------------------------------
-- Trigger function: block UPDATE/DELETE on approval_routes if any
-- approval_instance references that route.
-- -------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION enforce_route_immutable()
  RETURNS trigger
  LANGUAGE plpgsql
  SET search_path = pg_catalog, pg_temp
AS $$
BEGIN
  IF EXISTS (
    SELECT 1
      FROM public.approval_instances
     WHERE route_id = OLD.id
  ) THEN
    RAISE EXCEPTION 'ErrRouteInUse: route % is referenced by one or more approval instances and cannot be modified', OLD.id
      USING ERRCODE = 'P0001';
  END IF;
  RETURN NEW; -- for UPDATE; DELETE triggers ignore return value
END;
$$;

DROP TRIGGER IF EXISTS trg_route_config_immutable_upd ON approval_routes;
CREATE TRIGGER trg_route_config_immutable_upd
  BEFORE UPDATE ON approval_routes
  FOR EACH ROW EXECUTE FUNCTION enforce_route_immutable();

DROP TRIGGER IF EXISTS trg_route_config_immutable_del ON approval_routes;
CREATE TRIGGER trg_route_config_immutable_del
  BEFORE DELETE ON approval_routes
  FOR EACH ROW EXECUTE FUNCTION enforce_route_immutable();

COMMIT;
