CREATE TABLE templates_v2_template (
  id                    uuid PRIMARY KEY,
  tenant_id             text NOT NULL,
  doc_type_code         text NOT NULL,
  key                   text NOT NULL,
  name                  text NOT NULL,
  description           text NOT NULL DEFAULT '',
  areas                 text[] NOT NULL DEFAULT '{}',
  visibility            text NOT NULL,
  specific_areas        text[] NOT NULL DEFAULT '{}',
  latest_version        int NOT NULL DEFAULT 0,
  published_version_id  uuid NULL,
  created_by            text NOT NULL,
  created_at            timestamptz NOT NULL DEFAULT now(),
  archived_at           timestamptz NULL,
  UNIQUE (tenant_id, key)
);

CREATE TABLE templates_v2_template_version (
  id                  uuid PRIMARY KEY,
  template_id         uuid NOT NULL REFERENCES templates_v2_template(id),
  version_number      int  NOT NULL,
  status              text NOT NULL,
  docx_storage_key    text NOT NULL,
  content_hash        text NOT NULL,
  metadata_schema     jsonb NOT NULL,
  placeholder_schema  jsonb NOT NULL,
  editable_zones      jsonb NOT NULL,
  author_id               text NOT NULL,
  pending_reviewer_role   text NULL,
  pending_approver_role   text NOT NULL DEFAULT '',
  reviewer_id             text NULL,
  approver_id             text NULL,
  submitted_at        timestamptz NULL,
  reviewed_at         timestamptz NULL,
  approved_at         timestamptz NULL,
  published_at        timestamptz NULL,
  obsoleted_at        timestamptz NULL,
  created_at          timestamptz NOT NULL DEFAULT now(),
  UNIQUE (template_id, version_number)
);

ALTER TABLE templates_v2_template
  ADD CONSTRAINT fk_templates_v2_published_version
  FOREIGN KEY (published_version_id) REFERENCES templates_v2_template_version(id);

CREATE TABLE templates_v2_approval_config (
  template_id     uuid PRIMARY KEY REFERENCES templates_v2_template(id),
  reviewer_role   text NULL,
  approver_role   text NOT NULL
);

CREATE TABLE templates_v2_audit_log (
  id            bigserial PRIMARY KEY,
  tenant_id     text NOT NULL,
  template_id   uuid NOT NULL,
  version_id    uuid NULL,
  actor_id      text NOT NULL,
  action        text NOT NULL,
  details       jsonb NOT NULL DEFAULT '{}',
  occurred_at   timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_templates_v2_template_tenant_doctype ON templates_v2_template (tenant_id, doc_type_code);
CREATE INDEX idx_templates_v2_version_template_status ON templates_v2_template_version (template_id, status);
CREATE INDEX idx_templates_v2_audit_template_time ON templates_v2_audit_log (template_id, occurred_at DESC);
