-- W4: document exports append-only ledger.
-- composite_hash is a 32-byte SHA-256 keyed on content + render options.
CREATE TABLE IF NOT EXISTS document_exports (
    id              uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id     uuid        NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    revision_id     uuid        NOT NULL,
    composite_hash  bytea       NOT NULL CHECK (octet_length(composite_hash) = 32),
    storage_key     text        NOT NULL,
    size_bytes      bigint      NOT NULL CHECK (size_bytes > 0),
    paper_size      text        NOT NULL DEFAULT 'A4',
    landscape       boolean     NOT NULL DEFAULT FALSE,
    docgen_v2_ver   text        NOT NULL,
    created_at      timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS document_exports_doc_hash_uidx
    ON document_exports (document_id, composite_hash);

CREATE INDEX IF NOT EXISTS document_exports_document_id_idx
    ON document_exports (document_id);
