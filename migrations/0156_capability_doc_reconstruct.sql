BEGIN;

INSERT INTO metaldocs.role_capabilities (capability, role)
VALUES
    ('doc.reconstruct', 'qms_admin')
ON CONFLICT DO NOTHING;

COMMIT;
