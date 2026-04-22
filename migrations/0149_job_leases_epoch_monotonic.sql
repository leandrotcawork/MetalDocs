-- migrations/0149_job_leases_epoch_monotonic.sql
-- Fix: release_lease must preserve epoch monotonicity by expiring the row
-- instead of deleting it. Deleting + reinserting resets epoch to 1, which
-- allows stale fencing tokens to become valid again after release+reacquire.
-- Idempotent (CREATE OR REPLACE).

BEGIN;

CREATE OR REPLACE FUNCTION metaldocs.release_lease(_job text, _leader text, _epoch bigint)
RETURNS void
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = metaldocs, pg_temp
AS $$
BEGIN
    -- Mark the lease as immediately expired rather than deleting the row.
    -- The next acquire_lease call will hit the "expired" branch and increment
    -- the epoch, ensuring lease_epoch is strictly monotonic for the lifetime
    -- of the job_name key.
    UPDATE metaldocs.job_leases
    SET expires_at = now() - interval '1 second'
    WHERE job_name  = _job
      AND leader_id = _leader
      AND lease_epoch = _epoch;
END;
$$;

ALTER FUNCTION metaldocs.release_lease(text, text, bigint) OWNER TO metaldocs_admin;
REVOKE EXECUTE ON FUNCTION metaldocs.release_lease(text, text, bigint) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION metaldocs.release_lease(text, text, bigint) TO metaldocs_writer;

COMMIT;
