BEGIN;

INSERT INTO metaldocs.role_capabilities (capability, role)
VALUES
    ('doc.view_published', 'reader'),
    ('doc.view_published', 'author'),
    ('doc.view_published', 'reviewer'),
    ('doc.view_published', 'signer'),
    ('doc.view_published', 'area_admin'),
    ('doc.view_published', 'qms_admin')
ON CONFLICT DO NOTHING;

COMMIT;
