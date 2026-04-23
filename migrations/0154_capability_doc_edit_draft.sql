BEGIN;

INSERT INTO metaldocs.role_capabilities (capability, role)
VALUES
    ('doc.edit_draft', 'author'),
    ('doc.edit_draft', 'qms_admin')
ON CONFLICT DO NOTHING;

COMMIT;
