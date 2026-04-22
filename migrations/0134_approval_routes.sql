BEGIN;

CREATE TABLE IF NOT EXISTS public.approval_routes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    profile_code TEXT NOT NULL,
    name TEXT NOT NULL,
    version INT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_by TEXT NOT NULL,
    CONSTRAINT approval_routes_tenant_profile_key UNIQUE (tenant_id, profile_code)
);

CREATE TABLE IF NOT EXISTS public.approval_route_stages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    route_id UUID NOT NULL REFERENCES public.approval_routes(id) ON DELETE CASCADE,
    stage_order INT NOT NULL CHECK (stage_order >= 1),
    name TEXT NOT NULL,
    required_role TEXT NOT NULL,
    required_capability TEXT NOT NULL,
    area_code TEXT NOT NULL,
    quorum TEXT NOT NULL CHECK (quorum IN ('any_1_of', 'all_of', 'm_of_n')),
    quorum_m INT,
    on_eligibility_drift TEXT NOT NULL CHECK (on_eligibility_drift IN ('reduce_quorum', 'fail_stage', 'keep_snapshot')),
    CONSTRAINT approval_route_stages_route_stage_order_key UNIQUE (route_id, stage_order),
    CONSTRAINT approval_route_stages_quorum_m_consistent CHECK (
        (
            quorum = 'm_of_n'
            AND quorum_m IS NOT NULL
            AND quorum_m >= 1
        )
        OR (
            quorum <> 'm_of_n'
            AND quorum_m IS NULL
        )
    )
);

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM pg_class c
        JOIN pg_namespace n ON n.oid = c.relnamespace
        WHERE n.nspname = 'metaldocs'
          AND c.relname = 'document_profiles'
    ) THEN
        IF NOT EXISTS (
            SELECT 1
            FROM pg_constraint
            WHERE conname = 'approval_routes_document_profile_fk'
              AND conrelid = 'public.approval_routes'::regclass
        ) THEN
            EXECUTE '
                ALTER TABLE public.approval_routes
                ADD CONSTRAINT approval_routes_document_profile_fk
                FOREIGN KEY (tenant_id, profile_code)
                REFERENCES metaldocs.document_profiles(tenant_id, code)
                NOT VALID
            ';
        END IF;

        IF EXISTS (
            SELECT 1
            FROM pg_constraint
            WHERE conname = 'approval_routes_document_profile_fk'
              AND conrelid = 'public.approval_routes'::regclass
              AND convalidated = false
        ) THEN
            EXECUTE '
                ALTER TABLE public.approval_routes
                VALIDATE CONSTRAINT approval_routes_document_profile_fk
            ';
        END IF;
    END IF;
END $$;

COMMIT;
