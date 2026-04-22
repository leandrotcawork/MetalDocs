-- migrations/0148_job_leases.sql
-- Phase 8: scheduler lease infrastructure with fencing tokens (O1, O5, O9).
-- Idempotent.

BEGIN;

CREATE TABLE IF NOT EXISTS metaldocs.job_leases (
    job_name    TEXT        PRIMARY KEY,
    leader_id   TEXT        NOT NULL,
    lease_epoch BIGINT      NOT NULL DEFAULT 0,
    acquired_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    heartbeat_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at  TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_idempotency_keys_completed_expires
    ON metaldocs.idempotency_keys (expires_at)
    WHERE status = 'completed';

CREATE OR REPLACE FUNCTION metaldocs.acquire_lease(_job text, _leader text, _ttl interval)
RETURNS TABLE(acquired bool, epoch bigint)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = metaldocs, pg_temp
AS $$
DECLARE
    v_now TIMESTAMPTZ := now();
    v_lease metaldocs.job_leases%ROWTYPE;
BEGIN
    SELECT *
    INTO v_lease
    FROM metaldocs.job_leases
    WHERE job_name = _job
    FOR UPDATE SKIP LOCKED;

    IF NOT FOUND THEN
        IF EXISTS (
            SELECT 1
            FROM metaldocs.job_leases
            WHERE job_name = _job
        ) THEN
            RETURN QUERY SELECT false, -1::bigint;
            RETURN;
        END IF;

        BEGIN
            INSERT INTO metaldocs.job_leases (
                job_name,
                leader_id,
                lease_epoch,
                acquired_at,
                heartbeat_at,
                expires_at
            )
            VALUES (_job, _leader, 1, v_now, v_now, v_now + _ttl);

            RETURN QUERY SELECT true, 1::bigint;
            RETURN;
        EXCEPTION
            WHEN unique_violation THEN
                RETURN QUERY SELECT false, -1::bigint;
                RETURN;
        END;
    END IF;

    IF v_lease.expires_at < v_now THEN
        UPDATE metaldocs.job_leases
        SET
            leader_id = _leader,
            lease_epoch = v_lease.lease_epoch + 1,
            acquired_at = v_now,
            heartbeat_at = v_now,
            expires_at = v_now + _ttl
        WHERE job_name = _job;

        RETURN QUERY SELECT true, (v_lease.lease_epoch + 1)::bigint;
        RETURN;
    END IF;

    IF v_lease.leader_id = _leader THEN
        UPDATE metaldocs.job_leases
        SET
            heartbeat_at = v_now,
            expires_at = v_now + _ttl
        WHERE job_name = _job;

        RETURN QUERY SELECT true, v_lease.lease_epoch;
        RETURN;
    END IF;

    RETURN QUERY SELECT false, -1::bigint;
END;
$$;

ALTER FUNCTION metaldocs.acquire_lease(text, text, interval) OWNER TO metaldocs_admin;
REVOKE EXECUTE ON FUNCTION metaldocs.acquire_lease(text, text, interval) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION metaldocs.acquire_lease(text, text, interval) TO metaldocs_writer;

CREATE OR REPLACE FUNCTION metaldocs.heartbeat_lease(_job text, _leader text, _epoch bigint)
RETURNS bool
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = metaldocs, pg_temp
AS $$
BEGIN
    UPDATE metaldocs.job_leases
    SET
        heartbeat_at = now(),
        expires_at = now() + interval '5 minutes'
    WHERE job_name = _job
      AND leader_id = _leader
      AND lease_epoch = _epoch;

    RETURN FOUND;
END;
$$;

ALTER FUNCTION metaldocs.heartbeat_lease(text, text, bigint) OWNER TO metaldocs_admin;
REVOKE EXECUTE ON FUNCTION metaldocs.heartbeat_lease(text, text, bigint) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION metaldocs.heartbeat_lease(text, text, bigint) TO metaldocs_writer;

CREATE OR REPLACE FUNCTION metaldocs.release_lease(_job text, _leader text, _epoch bigint)
RETURNS void
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = metaldocs, pg_temp
AS $$
BEGIN
    DELETE FROM metaldocs.job_leases
    WHERE job_name = _job
      AND leader_id = _leader
      AND lease_epoch = _epoch;
END;
$$;

ALTER FUNCTION metaldocs.release_lease(text, text, bigint) OWNER TO metaldocs_admin;
REVOKE EXECUTE ON FUNCTION metaldocs.release_lease(text, text, bigint) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION metaldocs.release_lease(text, text, bigint) TO metaldocs_writer;

CREATE OR REPLACE FUNCTION metaldocs.assert_lease_epoch(_job text, _epoch bigint)
RETURNS void
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = metaldocs, pg_temp
AS $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM metaldocs.job_leases
        WHERE job_name = _job
          AND lease_epoch = _epoch
    ) THEN
        RAISE EXCEPTION 'ErrLeaseEpochStale: job % epoch % no longer current', _job, _epoch
            USING ERRCODE = 'P0001';
    END IF;
END;
$$;

ALTER FUNCTION metaldocs.assert_lease_epoch(text, bigint) OWNER TO metaldocs_admin;
REVOKE EXECUTE ON FUNCTION metaldocs.assert_lease_epoch(text, bigint) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION metaldocs.assert_lease_epoch(text, bigint) TO metaldocs_writer;

GRANT SELECT, INSERT, UPDATE, DELETE ON metaldocs.job_leases TO metaldocs_writer;

COMMIT;