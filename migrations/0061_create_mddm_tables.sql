-- 0061_create_mddm_tables.sql
-- Creates the foundational MDDM tables: document_versions (replaces old layout),
-- document_images, document_version_images, and document_template_versions.

-- New status enum for document versions
DO $$ BEGIN
  CREATE TYPE metaldocs.mddm_version_status AS ENUM ('draft', 'pending_approval', 'released', 'archived');
EXCEPTION
  WHEN duplicate_object THEN NULL;
END $$;

-- Image storage (deduplicated by content hash)
CREATE TABLE IF NOT EXISTS metaldocs.document_images (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  sha256      TEXT NOT NULL UNIQUE,
  mime_type   TEXT NOT NULL,
  byte_size   INTEGER NOT NULL CHECK (byte_size > 0),
  bytes       BYTEA NOT NULL,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_document_images_sha256 ON metaldocs.document_images(sha256);

-- Templates (independently versioned, immutable when published)
CREATE TABLE IF NOT EXISTS metaldocs.document_template_versions_mddm (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  template_id     UUID NOT NULL,
  version         INTEGER NOT NULL CHECK (version >= 1),
  mddm_version    INTEGER NOT NULL CHECK (mddm_version >= 1),
  content_blocks  JSONB NOT NULL,
  content_hash    TEXT NOT NULL,
  is_published    BOOLEAN NOT NULL DEFAULT false,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (template_id, version)
);

-- Document versions (drafts, released, archived)
CREATE TABLE IF NOT EXISTS metaldocs.document_versions_mddm (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  document_id     TEXT NOT NULL REFERENCES metaldocs.documents(id) ON DELETE CASCADE,
  version_number  INTEGER NOT NULL CHECK (version_number >= 1),
  revision_label  TEXT NOT NULL,
  status          metaldocs.mddm_version_status NOT NULL,
  content_blocks  JSONB,
  docx_bytes      BYTEA,
  template_ref    JSONB,
  content_hash    TEXT,
  revision_diff   JSONB,
  change_summary  TEXT,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  created_by      TEXT NOT NULL,
  approved_at     TIMESTAMPTZ,
  approved_by     TEXT,
  UNIQUE (document_id, version_number)
);

-- Cardinality enforcement
CREATE UNIQUE INDEX IF NOT EXISTS idx_one_released_per_doc
  ON metaldocs.document_versions_mddm(document_id)
  WHERE status = 'released';

CREATE UNIQUE INDEX IF NOT EXISTS idx_one_active_draft_per_doc
  ON metaldocs.document_versions_mddm(document_id)
  WHERE status IN ('draft', 'pending_approval');

-- M:N image references
CREATE TABLE IF NOT EXISTS metaldocs.document_version_images (
  document_version_id UUID NOT NULL REFERENCES metaldocs.document_versions_mddm(id) ON DELETE CASCADE,
  image_id            UUID NOT NULL REFERENCES metaldocs.document_images(id),
  PRIMARY KEY (document_version_id, image_id)
);

CREATE INDEX IF NOT EXISTS idx_dvi_image ON metaldocs.document_version_images(image_id);
