-- migrations/0152_placeholder_fillin_columns.sql
-- Spec 3 Phase 2: placeholder fill-in snapshot columns + value tables + snapshot enforcement trigger.
BEGIN;

ALTER TABLE documents
    ADD COLUMN placeholder_schema_snapshot     JSONB,
    ADD COLUMN placeholder_schema_hash         BYTEA,
    ADD COLUMN composition_config_snapshot     JSONB,
    ADD COLUMN composition_config_hash         BYTEA,
    ADD COLUMN editable_zones_schema_snapshot  JSONB,
    ADD COLUMN body_docx_snapshot_s3_key       TEXT,
    ADD COLUMN body_docx_hash                  BYTEA,
    ADD COLUMN values_frozen_at                TIMESTAMPTZ,
    ADD COLUMN values_hash                     BYTEA,
    ADD COLUMN final_docx_s3_key               TEXT,
    ADD COLUMN final_pdf_s3_key                TEXT,
    ADD COLUMN pdf_hash                        BYTEA,
    ADD COLUMN pdf_generated_at                TIMESTAMPTZ,
    ADD COLUMN reconstruction_attempts         JSONB NOT NULL DEFAULT '[]'::jsonb;

CREATE OR REPLACE FUNCTION enforce_snapshot_on_submit() RETURNS trigger AS $$
BEGIN
    IF NEW.status IN ('under_review','approved','scheduled','published')
       AND (NEW.placeholder_schema_snapshot IS NULL
         OR NEW.placeholder_schema_hash IS NULL
         OR NEW.composition_config_snapshot IS NULL
         OR NEW.composition_config_hash IS NULL
         OR NEW.body_docx_snapshot_s3_key IS NULL
         OR NEW.body_docx_hash IS NULL) THEN
        RAISE EXCEPTION 'documents.% snapshot columns required for status=%',
            NEW.id, NEW.status
            USING ERRCODE = 'check_violation';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS enforce_snapshot_on_submit_trg ON documents;
CREATE TRIGGER enforce_snapshot_on_submit_trg
    BEFORE INSERT OR UPDATE ON documents
    FOR EACH ROW EXECUTE FUNCTION enforce_snapshot_on_submit();

CREATE TABLE document_placeholder_values (
    tenant_id        UUID        NOT NULL,
    revision_id      UUID        NOT NULL,
    placeholder_id   TEXT        NOT NULL,
    value_text       TEXT,
    value_typed      JSONB,
    source           TEXT        NOT NULL CHECK (source IN ('user','computed','default')),
    computed_from    TEXT,
    resolver_version INT,
    inputs_hash      BYTEA,
    validated_at     TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, revision_id, placeholder_id),
    FOREIGN KEY (revision_id) REFERENCES documents(id) ON DELETE CASCADE
);

CREATE INDEX idx_dpv_revision ON document_placeholder_values(revision_id);

CREATE TABLE document_editable_zone_content (
    tenant_id     UUID  NOT NULL,
    revision_id   UUID  NOT NULL,
    zone_id       TEXT  NOT NULL,
    content_ooxml TEXT  NOT NULL,
    content_hash  BYTEA NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, revision_id, zone_id),
    FOREIGN KEY (revision_id) REFERENCES documents(id) ON DELETE CASCADE
);

CREATE INDEX idx_dezc_revision ON document_editable_zone_content(revision_id);

COMMIT;
