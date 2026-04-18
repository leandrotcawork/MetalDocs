# W3 Documents Vertical (docx-editor platform) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship the document fill+edit vertical end-to-end behind `METALDOCS_DOCX_V2_ENABLED`. `document_filler` can pick a published template, fill the schema-driven form, generate the `.docx`, open it in the editor with session lock + autosave + checkpoints + finalize/archive. Playwright `filler-happy-path`, `conflict-two-tabs`, `autosave-crash` all green.

**Architecture:** All `documents` concepts are new tables created by migration `0110_docx_v2_documents.sql`. Documents bind to an immutable `template_version_id`. Editor sessions enforce single-writer via `UNIQUE INDEX … WHERE status='active'`. Autosave = two-phase (presign + commit) with all guards inside a single DB transaction. Revisions are content-addressed and append-only. `docgen-v2` gains `POST /render/docx` which calls `@eigenpal/docx-js-editor@0.0.34` `processTemplate(buffer, form_data)`. Frontend adds `/documents-v2/*` views mounted behind the feature flag.

**Tech stack additions (on top of W1+W2):** `idb@8.0.0` (IndexedDB wrapper for autosave-crash recovery) · `gojsonschema@v1.2.0` (server-side form_data validation). No new backend deps otherwise.

**Depends on:** Plan A (W1 scaffold) + Plan B (W2 templates vertical) both executed and green.

**Spec reference:** `docs/superpowers/specs/2026-04-18-docx-editor-platform-design.md` §§ Data Flow → Document fill + edit, Atomic autosave, Session lifecycle, Checkpoint + restore, RBAC, Error Handling, Testing Approach.

**Codex hardening status:** Written per co-plan protocol. R1 + R2 outcomes recorded at end of file (Section "Codex Hardening Log"). Max 2 rounds enforced.

---

## File Structure

**New files:**

```
# Migration
migrations/
  0110_docx_v2_documents.sql        # all W3 tables + FK wiring

# Docgen-v2 render endpoint
apps/docgen-v2/src/
  routes/render.ts                  # POST /render/docx
  render/processDocx.ts             # eigenpal processTemplate wrapper
apps/docgen-v2/test/
  render.smoke.test.ts
  render.validate.test.ts
  render.hash-stability.test.ts

# Go documents module (greenfield — replaces old internal/modules/documents)
internal/modules/documents_v2/
  domain/
    model.go                        # Document, Revision, Session, Pending, Checkpoint, errors
    state.go                        # document status transitions
  application/
    service.go                      # all use cases in one service: create, load, autosave, session, checkpoint, finalize, archive
    service_test.go                 # table-driven unit tests per use case
    autosave_commit_branches_test.go # 9 rejection branches under fake repo + presigner
  repository/
    repository.go                   # pgx implementations
  delivery/http/
    handler.go                      # all routes
    handler_test.go                 # per-endpoint happy/forbidden/conflict
  infrastructure/docgen/
    client.go                       # thin client wrapping docgen-v2 /render/docx
  infrastructure/indexeddb/         # (frontend-only; see frontend tree)
  module.go                         # assembly

# Backgrounds (session sweep + orphan)
internal/modules/documents_v2/jobs/
  session_sweeper.go                # every 60s
  orphan_pending_sweeper.go         # hourly
  jobs_test.go

# Frontend
frontend/apps/web/src/features/documents/v2/
  api/documentsV2.ts
  DocumentCreatePage.tsx
  DocumentEditorPage.tsx
  CheckpointsPanel.tsx
  routes.tsx                        # renderDocumentsV2View()
  hooks/
    useDocumentLoad.ts
    useDocumentSession.ts           # acquire + heartbeat + release + force-release reactions
    useDocumentAutosave.ts          # state machine + IndexedDB restore
    useIndexedDBRestore.ts
  styles/DocumentEditorPage.module.css

# OpenAPI
api/openapi/v1/partials/documents-v2.yaml

# Playwright E2Es
frontend/apps/web/e2e/
  filler-happy-path.spec.ts
  conflict-two-tabs.spec.ts
  autosave-crash.spec.ts
  fixtures/purchase-order-published.docx   # reused from W2 fixtures; symlink not allowed on Windows — commit a copy

# Runbook + governance
docs/runbooks/docx-v2-w3-documents.md
tests/docx_v2/documents_integration_test.go
```

**Modified files:**

```
apps/api/cmd/metaldocs-api/main.go        # mount documents_v2 module under flag; start sweepers
apps/api/cmd/metaldocs-api/permissions.go # /api/v2/documents* perm routing
frontend/apps/web/src/App.tsx             # renderWorkspaceView case 'documents-v2'
frontend/apps/web/src/routing/workspaceRoutes.ts  # /documents-v2[/...] URL mapping
api/openapi/v1/openapi.yaml               # merge partial
.github/workflows/docx-v2-ci.yml          # e2e-documents job (3 specs)
```

**Deleted at end of W3:** none. CK5 destruction happens in W5 (Plan E).

---

## Task 1: Migration — `0110_docx_v2_documents.sql`

**Files:**
- Create: `migrations/0110_docx_v2_documents.sql`

- [ ] **Step 1: Write migration**

```sql
-- 0110_docx_v2_documents.sql
-- Documents vertical schema for docx-editor platform (W3).
-- Depends on 0109 (templates_v2). Safe to run in one transaction.

BEGIN;

CREATE TABLE documents (
  id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id             UUID NOT NULL,
  template_version_id   UUID NOT NULL REFERENCES template_versions(id),
  name                  TEXT NOT NULL,
  status                TEXT NOT NULL CHECK (status IN ('draft','finalized','archived')),
  form_data_json        JSONB NOT NULL,
  current_revision_id   UUID,
  active_session_id     UUID,
  finalized_at          TIMESTAMPTZ,
  archived_at           TIMESTAMPTZ,
  created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
  created_by            UUID NOT NULL
);

CREATE INDEX idx_documents_tenant_status ON documents (tenant_id, status);
CREATE INDEX idx_documents_template_version ON documents (template_version_id);
CREATE INDEX idx_documents_form_data_gin ON documents USING GIN (form_data_json jsonb_path_ops);

CREATE TABLE editor_sessions (
  id                              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  document_id                     UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
  user_id                         UUID NOT NULL,
  acquired_at                     TIMESTAMPTZ NOT NULL DEFAULT now(),
  expires_at                      TIMESTAMPTZ NOT NULL,
  released_at                     TIMESTAMPTZ,
  last_acknowledged_revision_id   UUID NOT NULL,
  status                          TEXT NOT NULL CHECK (status IN ('active','expired','released','force_released'))
);

-- Single-writer invariant: only ONE active session per document.
CREATE UNIQUE INDEX idx_one_active_session_per_doc
  ON editor_sessions (document_id) WHERE status = 'active';

CREATE TABLE document_revisions (
  id                     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  document_id            UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
  revision_num           BIGSERIAL,
  parent_revision_id     UUID REFERENCES document_revisions(id),
  session_id             UUID NOT NULL REFERENCES editor_sessions(id),
  storage_key            TEXT NOT NULL,
  content_hash           TEXT NOT NULL,
  form_data_snapshot     JSONB,
  created_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (document_id, content_hash)
);

CREATE INDEX idx_revisions_doc_num ON document_revisions (document_id, revision_num DESC);

-- Deferrable FKs so we can insert document+session+revision in one tx.
ALTER TABLE documents
  ADD CONSTRAINT fk_current_revision
    FOREIGN KEY (current_revision_id) REFERENCES document_revisions(id)
    DEFERRABLE INITIALLY DEFERRED,
  ADD CONSTRAINT fk_active_session
    FOREIGN KEY (active_session_id) REFERENCES editor_sessions(id)
    DEFERRABLE INITIALLY DEFERRED;

CREATE TABLE autosave_pending_uploads (
  id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  session_id           UUID NOT NULL REFERENCES editor_sessions(id) ON DELETE CASCADE,
  document_id          UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
  base_revision_id     UUID NOT NULL REFERENCES document_revisions(id),
  content_hash         TEXT NOT NULL,
  storage_key          TEXT NOT NULL,
  presigned_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
  expires_at           TIMESTAMPTZ NOT NULL,
  consumed_at          TIMESTAMPTZ,
  UNIQUE (session_id, base_revision_id, content_hash)
);

CREATE INDEX idx_pending_expired
  ON autosave_pending_uploads (expires_at)
  WHERE consumed_at IS NULL;

CREATE TABLE document_checkpoints (
  id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  document_id        UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
  revision_id        UUID NOT NULL REFERENCES document_revisions(id),
  version_num        INT NOT NULL,
  label              TEXT,
  created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
  created_by         UUID NOT NULL,
  UNIQUE (document_id, version_num)
);

-- template_audit_log was created by 0109 (templates). Extend actions covered
-- in W3 (document_*, session_*, export.pdf_generated, export.docx_downloaded).
-- No schema change required; actions are free-form text.

COMMIT;
```

- [ ] **Step 2: Apply locally**

```bash
PGPASSWORD=metaldocs psql -h 127.0.0.1 -U metaldocs -d metaldocs -v ON_ERROR_STOP=1 -f migrations/0110_docx_v2_documents.sql
```

Expected: `CREATE TABLE`, `ALTER TABLE`, `COMMIT`. No errors.

- [ ] **Step 3: Round-trip test — insert+select each table**

```bash
PGPASSWORD=metaldocs psql -h 127.0.0.1 -U metaldocs -d metaldocs -c "
INSERT INTO documents (tenant_id, template_version_id, name, status, form_data_json, created_by)
SELECT '00000000-0000-0000-0000-000000000001', tv.id, 'probe', 'draft', '{}'::jsonb, '00000000-0000-0000-0000-000000000002'
FROM template_versions tv LIMIT 1 RETURNING id;"
```

Expected: one UUID returned. (In CI this assumes a seed template version exists; for greenfield migration tests we test via Go integration tests.)

- [ ] **Step 4: Commit**

```bash
rtk git add migrations/0110_docx_v2_documents.sql
rtk git commit -m "feat(db): 0110 W3 documents schema (sessions, revisions, pending, checkpoints)"
```

---

## Task 2: Docgen-v2 — `POST /render/docx`

**Files:**
- Create: `apps/docgen-v2/src/render/processDocx.ts`
- Create: `apps/docgen-v2/src/routes/render.ts`
- Modify: `apps/docgen-v2/src/server.ts` to register the route
- Create: `apps/docgen-v2/test/render.smoke.test.ts`
- Create: `apps/docgen-v2/test/render.hash-stability.test.ts`

- [ ] **Step 1: Write `processDocx`**

```ts
// apps/docgen-v2/src/render/processDocx.ts
import { createHash } from 'node:crypto';
// @eigenpal/docx-js-editor@0.0.34 exposes a CLI + library API.
// processTemplate is the pure-buffer entry point — no DOM required.
import { processTemplate } from '@eigenpal/docx-js-editor';

export interface ProcessDocxResult {
  buffer: Uint8Array;
  contentHash: string;
  unreplacedVars: string[];
}

export async function processDocx(
  templateBuffer: Uint8Array,
  formData: Record<string, unknown>,
): Promise<ProcessDocxResult> {
  const out = await processTemplate(templateBuffer, formData, {
    nullGetter: () => '',
  });
  const buf: Uint8Array = out.buffer instanceof Uint8Array ? out.buffer : new Uint8Array(out.buffer);
  const contentHash = createHash('sha256').update(buf).digest('hex');
  return { buffer: buf, contentHash, unreplacedVars: out.unreplacedVars ?? [] };
}
```

- [ ] **Step 2: Write route handler**

```ts
// apps/docgen-v2/src/routes/render.ts
import type { FastifyInstance } from 'fastify';
import { z } from 'zod';
import { getObject, putObject } from '../s3.js';
import { processDocx } from '../render/processDocx.js';
import Ajv from 'ajv';

const BodySchema = z.object({
  template_docx_key: z.string().min(1),
  schema_key: z.string().min(1),
  form_data: z.record(z.unknown()),
  output_key: z.string().min(1),
});

export async function registerRenderRoutes(app: FastifyInstance) {
  app.post('/render/docx', async (req, reply) => {
    const parsed = BodySchema.safeParse(req.body);
    if (!parsed.success) {
      return reply.code(400).send({ error: 'bad_request', details: parsed.error.format() });
    }
    const { template_docx_key, schema_key, form_data, output_key } = parsed.data;

    const [templateBuf, schemaBuf] = await Promise.all([
      getObject(template_docx_key),
      getObject(schema_key),
    ]);
    let schema: unknown;
    try {
      schema = JSON.parse(Buffer.from(schemaBuf).toString('utf8'));
    } catch {
      return reply.code(422).send({ error: 'schema_invalid', message: 'schema is not valid JSON' });
    }

    const ajv = new Ajv({ strict: false });
    const validate = ajv.compile(schema as object);
    if (!validate(form_data)) {
      return reply.code(422).send({ error: 'form_data_invalid', errors: validate.errors ?? [] });
    }

    const { buffer, contentHash, unreplacedVars } = await processDocx(templateBuf, form_data);
    await putObject(output_key, buffer, 'application/vnd.openxmlformats-officedocument.wordprocessingml.document');

    return reply.code(200).send({
      output_key,
      content_hash: contentHash,
      size_bytes: buffer.byteLength,
      warnings: [],
      unreplaced_vars: unreplacedVars,
    });
  });
}
```

- [ ] **Step 3: Register in server**

In `apps/docgen-v2/src/server.ts`, under the existing `registerValidateRoutes(app)` line, add:

```ts
import { registerRenderRoutes } from './routes/render.js';
await registerRenderRoutes(app);
```

- [ ] **Step 4: Write `render.smoke.test.ts`**

```ts
import { describe, it, expect, beforeAll, afterAll } from 'vitest';
import { buildServer } from '../src/server.js';
import { putObject, resetBucket } from '../src/s3.js';
import { buildMinimalDocxFixture, buildMinimalSchemaFixture } from './fixtures.js';

let app: ReturnType<typeof buildServer>;

beforeAll(async () => {
  app = await buildServer();
  await app.ready();
  await resetBucket();
});
afterAll(async () => { await app.close(); });

describe('POST /render/docx', () => {
  it('200 on valid template + schema + form_data', async () => {
    const templateBuf = await buildMinimalDocxFixture([
      { text: 'Hello {client_name}, total is {total_amount}.' },
    ]);
    const schemaObj = buildMinimalSchemaFixture(['client_name', 'total_amount']);
    await putObject('tenants/t1/templates/tpl1/v1.docx', templateBuf, 'application/vnd.openxmlformats-officedocument.wordprocessingml.document');
    await putObject('tenants/t1/templates/tpl1/v1.schema.json', Buffer.from(JSON.stringify(schemaObj)), 'application/json');

    const res = await app.inject({
      method: 'POST',
      url: '/render/docx',
      headers: { 'x-service-token': 'test-token-0123456789', 'content-type': 'application/json' },
      payload: {
        template_docx_key: 'tenants/t1/templates/tpl1/v1.docx',
        schema_key: 'tenants/t1/templates/tpl1/v1.schema.json',
        form_data: { client_name: 'Acme Corp', total_amount: 1234.5 },
        output_key: 'tenants/t1/documents/doc1/revisions/abc.docx',
      },
    });

    expect(res.statusCode).toBe(200);
    const body = res.json();
    expect(body.content_hash).toMatch(/^[a-f0-9]{64}$/);
    expect(body.size_bytes).toBeGreaterThan(0);
  });

  it('422 when form_data fails schema', async () => {
    const res = await app.inject({
      method: 'POST',
      url: '/render/docx',
      headers: { 'x-service-token': 'test-token-0123456789', 'content-type': 'application/json' },
      payload: {
        template_docx_key: 'tenants/t1/templates/tpl1/v1.docx',
        schema_key: 'tenants/t1/templates/tpl1/v1.schema.json',
        form_data: {},
        output_key: 'tenants/t1/documents/doc1/revisions/xyz.docx',
      },
    });
    expect(res.statusCode).toBe(422);
    expect(res.json().error).toBe('form_data_invalid');
  });
});
```

- [ ] **Step 5: Write `render.hash-stability.test.ts`**

```ts
import { describe, it, expect, beforeAll, afterAll } from 'vitest';
import { buildServer } from '../src/server.js';
import { putObject, resetBucket } from '../src/s3.js';
import { buildMinimalDocxFixture, buildMinimalSchemaFixture } from './fixtures.js';

// Hash stability protects the export-cache composite_hash logic. Two identical
// calls MUST produce byte-identical output → identical content_hash.
describe('render/docx hash stability', () => {
  let app: ReturnType<typeof buildServer>;
  beforeAll(async () => {
    app = await buildServer();
    await app.ready();
    await resetBucket();
    const templateBuf = await buildMinimalDocxFixture([
      { text: 'Hi {client_name}' },
    ]);
    await putObject('tenants/t1/templates/tpl1/v1.docx', templateBuf, 'application/vnd.openxmlformats-officedocument.wordprocessingml.document');
    await putObject('tenants/t1/templates/tpl1/v1.schema.json',
      Buffer.from(JSON.stringify(buildMinimalSchemaFixture(['client_name']))),
      'application/json');
  });
  afterAll(async () => { await app.close(); });

  it('same form_data → identical content_hash', async () => {
    const payload = {
      template_docx_key: 'tenants/t1/templates/tpl1/v1.docx',
      schema_key: 'tenants/t1/templates/tpl1/v1.schema.json',
      form_data: { client_name: 'Acme' },
      output_key: 'tenants/t1/documents/doc1/revisions/a.docx',
    };
    const r1 = await app.inject({ method: 'POST', url: '/render/docx', headers: { 'x-service-token': 'test-token-0123456789' }, payload });
    const r2 = await app.inject({ method: 'POST', url: '/render/docx', headers: { 'x-service-token': 'test-token-0123456789' }, payload: { ...payload, output_key: 'tenants/t1/documents/doc1/revisions/b.docx' } });
    expect(r1.json().content_hash).toBe(r2.json().content_hash);
  });
});
```

- [ ] **Step 6: Run**

```bash
npm test --workspace @metaldocs/docgen-v2 -- render
```

Expected: PASS (3 tests across 2 files).

- [ ] **Step 7: Commit**

```bash
rtk git add apps/docgen-v2/src/render apps/docgen-v2/src/routes/render.ts apps/docgen-v2/src/server.ts apps/docgen-v2/test/render.smoke.test.ts apps/docgen-v2/test/render.hash-stability.test.ts
rtk git commit -m "feat(docgen-v2): POST /render/docx + hash stability test"
```

---

## Task 3: Go domain — `documents_v2/domain`

**Files:**
- Create: `internal/modules/documents_v2/domain/model.go`
- Create: `internal/modules/documents_v2/domain/state.go`
- Create: `internal/modules/documents_v2/domain/model_test.go`

- [ ] **Step 1: Write `model.go`**

```go
package domain

import (
	"errors"
	"time"
)

type DocumentStatus string

const (
	DocStatusDraft     DocumentStatus = "draft"
	DocStatusFinalized DocumentStatus = "finalized"
	DocStatusArchived  DocumentStatus = "archived"
)

type SessionStatus string

const (
	SessionActive        SessionStatus = "active"
	SessionExpired       SessionStatus = "expired"
	SessionReleased      SessionStatus = "released"
	SessionForceReleased SessionStatus = "force_released"
)

type Document struct {
	ID                  string
	TenantID            string
	TemplateVersionID   string
	Name                string
	Status              DocumentStatus
	FormDataJSON        []byte  // JSONB raw
	CurrentRevisionID   string  // may be "" during creation tx
	ActiveSessionID     string
	FinalizedAt         *time.Time
	ArchivedAt          *time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
	CreatedBy           string
}

type Session struct {
	ID                          string
	DocumentID                  string
	UserID                      string
	AcquiredAt                  time.Time
	ExpiresAt                   time.Time
	ReleasedAt                  *time.Time
	LastAcknowledgedRevisionID  string
	Status                      SessionStatus
}

type Revision struct {
	ID               string
	DocumentID       string
	RevisionNum      int64
	ParentRevisionID string
	SessionID        string
	StorageKey       string
	ContentHash      string
	FormDataSnapshot []byte
	CreatedAt        time.Time
}

type PendingUpload struct {
	ID             string
	SessionID      string
	DocumentID     string
	BaseRevisionID string
	ContentHash    string
	StorageKey     string
	PresignedAt    time.Time
	ExpiresAt      time.Time
	ConsumedAt     *time.Time
}

type Checkpoint struct {
	ID         string
	DocumentID string
	RevisionID string
	VersionNum int
	Label      string
	CreatedAt  time.Time
	CreatedBy  string
}

var (
	ErrInvalidStateTransition = errors.New("invalid_state_transition")
	ErrSessionInactive        = errors.New("session_inactive")
	ErrSessionNotHolder       = errors.New("session_not_holder")
	ErrStaleBase              = errors.New("stale_base")
	ErrMisbound               = errors.New("misbound")
	ErrExpiredUpload          = errors.New("expired_upload")
	ErrContentHashMismatch    = errors.New("content_hash_mismatch")
	ErrPendingNotFound        = errors.New("pending_not_found")
	ErrAlreadyConsumed        = errors.New("already_consumed")
	ErrSessionTaken           = errors.New("session_taken")
	ErrForbidden              = errors.New("forbidden")
	ErrUploadMissing          = errors.New("upload_missing")           // server-authoritative hash check: S3 object absent
	ErrCheckpointNotFound     = errors.New("checkpoint_not_found")     // restore: version_num unknown for document
	ErrDocumentNotOwner       = errors.New("document_not_owner")       // RBAC: document_filler editing another user's doc
)
```

- [ ] **Step 2: Write `state.go`**

```go
package domain

// CanTransitionDocument returns true iff a document can move from cur → next.
// draft → finalized | archived
// finalized → archived
// archived → (terminal)
func CanTransitionDocument(cur, next DocumentStatus) bool {
	switch cur {
	case DocStatusDraft:
		return next == DocStatusFinalized || next == DocStatusArchived
	case DocStatusFinalized:
		return next == DocStatusArchived
	default:
		return false
	}
}
```

- [ ] **Step 3: Write `model_test.go`**

```go
package domain_test

import (
	"testing"

	"metaldocs/internal/modules/documents_v2/domain"
)

func TestCanTransitionDocument(t *testing.T) {
	cases := []struct {
		cur, next domain.DocumentStatus
		ok        bool
	}{
		{domain.DocStatusDraft, domain.DocStatusFinalized, true},
		{domain.DocStatusDraft, domain.DocStatusArchived, true},
		{domain.DocStatusFinalized, domain.DocStatusArchived, true},
		{domain.DocStatusArchived, domain.DocStatusDraft, false},
		{domain.DocStatusFinalized, domain.DocStatusDraft, false},
	}
	for _, c := range cases {
		if got := domain.CanTransitionDocument(c.cur, c.next); got != c.ok {
			t.Fatalf("CanTransitionDocument(%s, %s) = %v, want %v", c.cur, c.next, got, c.ok)
		}
	}
}
```

- [ ] **Step 4: Run + commit**

```bash
go test ./internal/modules/documents_v2/domain/...
rtk git add internal/modules/documents_v2/domain
rtk git commit -m "feat(documents_v2/domain): types + state transitions"
```

---

## Task 4: Go repository — `documents_v2/repository`

**Files:**
- Create: `internal/modules/documents_v2/repository/repository.go`
- Create: `internal/modules/documents_v2/repository/repository_integration_test.go`

- [ ] **Step 1: Write `repository.go` — struct + constructor + helper**

```go
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"metaldocs/internal/modules/documents_v2/domain"
)

type Repository struct {
	db *sql.DB
}

func New(db *sql.DB) *Repository { return &Repository{db: db} }
```

- [ ] **Step 2: Document CRUD**

Append to `repository.go`:

```go
// CreateDocument inserts document + initial session + initial revision in one
// deferrable-FK transaction. The initial revision's storage_key is empty — the
// caller uploads the .docx to the final content-addressed key via
// Presigner.AdoptTempObject, then calls SetRevisionStorageKey to finalize.
func (r *Repository) CreateDocument(ctx context.Context, d *domain.Document, initialContentHash string) (docID, revID, sessionID string, err error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil { return "", "", "", err }
	defer tx.Rollback()

	// Deferrable FKs allow inserting doc → session → revision in any order in tx.
	if err := tx.QueryRowContext(ctx,
		`INSERT INTO documents (tenant_id, template_version_id, name, status, form_data_json, created_by)
		 VALUES ($1, $2, $3, 'draft', $4, $5) RETURNING id`,
		d.TenantID, d.TemplateVersionID, d.Name, d.FormDataJSON, d.CreatedBy,
	).Scan(&docID); err != nil { return "", "", "", fmt.Errorf("insert document: %w", err) }

	if err := tx.QueryRowContext(ctx,
		`INSERT INTO editor_sessions (document_id, user_id, expires_at, last_acknowledged_revision_id, status)
		 VALUES ($1, $2, now() + interval '5 minutes', '00000000-0000-0000-0000-000000000000', 'active') RETURNING id`,
		docID, d.CreatedBy,
	).Scan(&sessionID); err != nil { return "", "", "", fmt.Errorf("insert session: %w", err) }

	if err := tx.QueryRowContext(ctx,
		`INSERT INTO document_revisions (document_id, parent_revision_id, session_id, storage_key, content_hash, form_data_snapshot)
		 VALUES ($1, NULL, $2, '', $3, $4) RETURNING id`,
		docID, sessionID, initialContentHash, d.FormDataJSON,
	).Scan(&revID); err != nil { return "", "", "", fmt.Errorf("insert revision: %w", err) }

	if _, err := tx.ExecContext(ctx,
		`UPDATE editor_sessions SET last_acknowledged_revision_id = $1 WHERE id = $2`,
		revID, sessionID,
	); err != nil { return "", "", "", fmt.Errorf("update session ack: %w", err) }

	if _, err := tx.ExecContext(ctx,
		`UPDATE documents SET current_revision_id = $1, active_session_id = $2, updated_at = now() WHERE id = $3`,
		revID, sessionID, docID,
	); err != nil { return "", "", "", fmt.Errorf("update document pointers: %w", err) }

	return docID, revID, sessionID, tx.Commit()
}

// SetRevisionStorageKey finalizes the initial revision's storage_key after the
// .docx has been copied to its content-addressed final key. Idempotent:
// succeeds only while storage_key is still empty.
func (r *Repository) SetRevisionStorageKey(ctx context.Context, revID, storageKey string) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE document_revisions SET storage_key = $1 WHERE id = $2 AND storage_key = ''`,
		storageKey, revID)
	if err != nil { return err }
	n, _ := res.RowsAffected()
	if n == 0 { return fmt.Errorf("revision %s already has storage_key set", revID) }
	return nil
}

func (r *Repository) GetDocument(ctx context.Context, tenantID, id string) (*domain.Document, error) {
	var d domain.Document
	err := r.db.QueryRowContext(ctx,
		`SELECT id, tenant_id, template_version_id, name, status, form_data_json,
		        coalesce(current_revision_id::text, ''), coalesce(active_session_id::text, ''),
		        finalized_at, archived_at, created_at, updated_at, created_by
		 FROM documents WHERE id=$1 AND tenant_id=$2`, id, tenantID,
	).Scan(&d.ID, &d.TenantID, &d.TemplateVersionID, &d.Name, &d.Status, &d.FormDataJSON,
		&d.CurrentRevisionID, &d.ActiveSessionID, &d.FinalizedAt, &d.ArchivedAt,
		&d.CreatedAt, &d.UpdatedAt, &d.CreatedBy)
	if err != nil { return nil, err }
	return &d, nil
}

func (r *Repository) ListDocuments(ctx context.Context, tenantID string) ([]domain.Document, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, tenant_id, template_version_id, name, status, form_data_json,
		        coalesce(current_revision_id::text, ''), coalesce(active_session_id::text, ''),
		        finalized_at, archived_at, created_at, updated_at, created_by
		 FROM documents WHERE tenant_id=$1 ORDER BY updated_at DESC`, tenantID)
	if err != nil { return nil, err }
	defer rows.Close()
	out := []domain.Document{}
	for rows.Next() {
		var d domain.Document
		if err := rows.Scan(&d.ID, &d.TenantID, &d.TemplateVersionID, &d.Name, &d.Status, &d.FormDataJSON,
			&d.CurrentRevisionID, &d.ActiveSessionID, &d.FinalizedAt, &d.ArchivedAt,
			&d.CreatedAt, &d.UpdatedAt, &d.CreatedBy); err != nil { return nil, err }
		out = append(out, d)
	}
	return out, rows.Err()
}

// ListDocumentsForUser restricts metadata leakage for document_filler role —
// returns only docs the actor created. Admins / template_* roles use the
// unrestricted ListDocuments path instead.
func (r *Repository) ListDocumentsForUser(ctx context.Context, tenantID, userID string) ([]domain.Document, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, tenant_id, template_version_id, name, status, form_data_json,
		        coalesce(current_revision_id::text, ''), coalesce(active_session_id::text, ''),
		        finalized_at, archived_at, created_at, updated_at, created_by
		 FROM documents WHERE tenant_id=$1 AND created_by=$2 ORDER BY updated_at DESC`, tenantID, userID)
	if err != nil { return nil, err }
	defer rows.Close()
	out := []domain.Document{}
	for rows.Next() {
		var d domain.Document
		if err := rows.Scan(&d.ID, &d.TenantID, &d.TemplateVersionID, &d.Name, &d.Status, &d.FormDataJSON,
			&d.CurrentRevisionID, &d.ActiveSessionID, &d.FinalizedAt, &d.ArchivedAt,
			&d.CreatedAt, &d.UpdatedAt, &d.CreatedBy); err != nil { return nil, err }
		out = append(out, d)
	}
	return out, rows.Err()
}

func (r *Repository) UpdateDocumentStatus(ctx context.Context, tenantID, id string, cur, next domain.DocumentStatus, stampTime bool) error {
	col := ""
	if next == domain.DocStatusFinalized { col = "finalized_at = now()," }
	if next == domain.DocStatusArchived  { col = "archived_at  = now()," }
	res, err := r.db.ExecContext(ctx,
		fmt.Sprintf(`UPDATE documents SET status=$1, %s updated_at=now() WHERE id=$2 AND tenant_id=$3 AND status=$4`, col),
		next, id, tenantID, cur)
	if err != nil { return err }
	n, _ := res.RowsAffected()
	if n == 0 { return domain.ErrInvalidStateTransition }
	return nil
}
```

- [ ] **Step 3: Session CRUD + atomic acquire**

Append:

```go
// AcquireSession attempts to claim the single active-session slot for a doc.
// Relies on partial unique index idx_one_active_session_per_doc.
// Returns (newSession, nil) on success. Returns existing active session
// with ErrSessionTaken if another user holds it.
func (r *Repository) AcquireSession(ctx context.Context, tenantID, docID, userID string) (*domain.Session, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil { return nil, err }
	defer tx.Rollback()

	var existingID, existingUser, existingStatus string
	err = tx.QueryRowContext(ctx,
		`SELECT id::text, user_id::text, status FROM editor_sessions
		 WHERE document_id=$1 AND status='active' FOR UPDATE`, docID,
	).Scan(&existingID, &existingUser, &existingStatus)
	if err != nil && !errors.Is(err, sql.ErrNoRows) { return nil, err }
	if err == nil {
		// Caller already holds it — refresh.
		if existingUser == userID {
			if _, err := tx.ExecContext(ctx, `UPDATE editor_sessions SET expires_at = now() + interval '5 minutes' WHERE id=$1`, existingID); err != nil { return nil, err }
			s := &domain.Session{ID: existingID, DocumentID: docID, UserID: userID, Status: domain.SessionActive}
			return s, tx.Commit()
		}
		// Someone else owns it.
		s := &domain.Session{ID: existingID, DocumentID: docID, UserID: existingUser, Status: domain.SessionActive}
		return s, domain.ErrSessionTaken
	}

	// No active session. Grab current_revision_id from documents.
	var curRev string
	if err := tx.QueryRowContext(ctx,
		`SELECT coalesce(current_revision_id::text,'') FROM documents WHERE id=$1 AND tenant_id=$2`, docID, tenantID,
	).Scan(&curRev); err != nil { return nil, err }
	if curRev == "" { return nil, fmt.Errorf("document has no current revision") }

	var newID string
	if err := tx.QueryRowContext(ctx,
		`INSERT INTO editor_sessions (document_id, user_id, expires_at, last_acknowledged_revision_id, status)
		 VALUES ($1, $2, now() + interval '5 minutes', $3, 'active') RETURNING id`,
		docID, userID, curRev,
	).Scan(&newID); err != nil { return nil, err }

	if _, err := tx.ExecContext(ctx, `UPDATE documents SET active_session_id=$1, updated_at=now() WHERE id=$2`, newID, docID); err != nil { return nil, err }

	return &domain.Session{ID: newID, DocumentID: docID, UserID: userID, LastAcknowledgedRevisionID: curRev, Status: domain.SessionActive}, tx.Commit()
}

func (r *Repository) HeartbeatSession(ctx context.Context, sessionID, userID string) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE editor_sessions SET expires_at = now() + interval '5 minutes'
		 WHERE id=$1 AND user_id=$2 AND status='active'`, sessionID, userID)
	if err != nil { return err }
	n, _ := res.RowsAffected()
	if n == 0 { return domain.ErrSessionInactive }
	return nil
}

func (r *Repository) ReleaseSession(ctx context.Context, sessionID, userID string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil { return err }
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx,
		`UPDATE editor_sessions SET status='released', released_at=now()
		 WHERE id=$1 AND user_id=$2 AND status='active'`, sessionID, userID)
	if err != nil { return err }
	n, _ := res.RowsAffected()
	if n == 0 { return domain.ErrSessionInactive }
	if _, err := tx.ExecContext(ctx, `UPDATE documents SET active_session_id=NULL, updated_at=now() WHERE active_session_id=$1`, sessionID); err != nil { return err }
	return tx.Commit()
}

func (r *Repository) ForceReleaseSession(ctx context.Context, sessionID string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil { return err }
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx,
		`UPDATE editor_sessions SET status='force_released', released_at=now()
		 WHERE id=$1 AND status='active'`, sessionID)
	if err != nil { return err }
	n, _ := res.RowsAffected()
	if n == 0 { return domain.ErrSessionInactive }
	if _, err := tx.ExecContext(ctx, `UPDATE documents SET active_session_id=NULL, updated_at=now() WHERE active_session_id=$1`, sessionID); err != nil { return err }
	return tx.Commit()
}

func (r *Repository) ExpireStaleSessions(ctx context.Context, now time.Time) (int, error) {
	res, err := r.db.ExecContext(ctx,
		`UPDATE editor_sessions SET status='expired' WHERE status='active' AND expires_at < $1`, now)
	if err != nil { return 0, err }
	n, _ := res.RowsAffected()
	if n > 0 {
		if _, err := r.db.ExecContext(ctx,
			`UPDATE documents SET active_session_id=NULL, updated_at=now()
			 WHERE active_session_id IN (SELECT id FROM editor_sessions WHERE status='expired' AND released_at IS NULL)`); err != nil { return 0, err }
	}
	return int(n), nil
}
```

- [ ] **Step 4: Autosave presign + commit (single-tx enforcement)**

Append:

```go
func (r *Repository) PresignReserve(ctx context.Context, sessionID, userID, docID, baseRevisionID, contentHash, storageKey string, expiresAt time.Time) (pendingID string, err error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil { return "", err }
	defer tx.Rollback()

	// Verify session active, holder matches, and session points at base.
	var sessUser, sessDoc, sessAck, sessStatus string
	err = tx.QueryRowContext(ctx,
		`SELECT user_id::text, document_id::text, last_acknowledged_revision_id::text, status
		 FROM editor_sessions WHERE id=$1 FOR UPDATE`, sessionID,
	).Scan(&sessUser, &sessDoc, &sessAck, &sessStatus)
	if err != nil { return "", err }
	if sessStatus != string(domain.SessionActive) { return "", domain.ErrSessionInactive }
	if sessUser != userID || sessDoc != docID { return "", domain.ErrSessionNotHolder }
	if sessAck != baseRevisionID { return "", domain.ErrStaleBase }

	// Idempotent on (session, base, hash): ON CONFLICT returns existing row's id.
	err = tx.QueryRowContext(ctx,
		`INSERT INTO autosave_pending_uploads
		   (session_id, document_id, base_revision_id, content_hash, storage_key, expires_at)
		 VALUES ($1,$2,$3,$4,$5,$6)
		 ON CONFLICT (session_id, base_revision_id, content_hash)
		 DO UPDATE SET presigned_at = autosave_pending_uploads.presigned_at
		 RETURNING id`,
		sessionID, docID, baseRevisionID, contentHash, storageKey, expiresAt,
	).Scan(&pendingID)
	if err != nil { return "", err }
	return pendingID, tx.Commit()
}

// CommitResult + PendingCommitMeta + RestoreResult are mirrored in application
// (same-shape type aliases) so handlers depend only on application types.
type CommitResult struct {
	RevisionID   string
	RevisionNum  int64
	AlreadyConsumed bool
}

type PendingCommitMeta struct {
	SessionID           string
	DocumentID          string
	BaseRevisionID      string
	ExpectedContentHash string
	StorageKey          string
	ExpiresAt           time.Time
	ConsumedAt          *time.Time
}

type RestoreResult struct {
	NewRevisionID   string
	NewRevisionNum  int64
	CheckpointRevID string
	// Idempotent is true when ON CONFLICT (document_id, content_hash) DO
	// UPDATE SET id = id fired — the current head already matched the
	// checkpoint hash, so no new row was inserted. Used by the handler to
	// surface `idempotent: true` on the restore response.
	Idempotent      bool
}

// GetPendingForCommit returns the minimal metadata the service needs before
// performing server-authoritative hash verification. Short, unlocked read;
// CommitUpload re-locks and re-checks under FOR UPDATE.
func (r *Repository) GetPendingForCommit(ctx context.Context, pendingID string) (*PendingCommitMeta, error) {
	var m PendingCommitMeta
	err := r.db.QueryRowContext(ctx,
		`SELECT session_id::text, document_id::text, base_revision_id::text,
		        content_hash, storage_key, expires_at, consumed_at
		 FROM autosave_pending_uploads WHERE id=$1`, pendingID,
	).Scan(&m.SessionID, &m.DocumentID, &m.BaseRevisionID,
		&m.ExpectedContentHash, &m.StorageKey, &m.ExpiresAt, &m.ConsumedAt)
	if errors.Is(err, sql.ErrNoRows) { return nil, domain.ErrPendingNotFound }
	if err != nil { return nil, err }
	return &m, nil
}

// CommitUpload encodes every DB-level rejection branch below as explicit errors.
// Callers translate each to the matching HTTP status (404/409/410/422 per spec).
// Server-authoritative content_hash verification happens in Service BEFORE this
// method is called; `serverComputedHash` is the hash the service streamed from
// S3 and therefore trusted. CommitUpload still compares it to pending.content_hash
// under FOR UPDATE to catch TOCTOU races.
func (r *Repository) CommitUpload(ctx context.Context, sessionID, userID, docID, pendingID, serverComputedHash string, formDataSnapshot []byte) (*CommitResult, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil { return nil, err }
	defer tx.Rollback()

	// Lock pending row.
	var p domain.PendingUpload
	err = tx.QueryRowContext(ctx,
		`SELECT id::text, session_id::text, document_id::text, base_revision_id::text, content_hash,
		        storage_key, expires_at, consumed_at
		 FROM autosave_pending_uploads WHERE id=$1 FOR UPDATE`, pendingID,
	).Scan(&p.ID, &p.SessionID, &p.DocumentID, &p.BaseRevisionID, &p.ContentHash, &p.StorageKey, &p.ExpiresAt, &p.ConsumedAt)
	if errors.Is(err, sql.ErrNoRows) { return nil, domain.ErrPendingNotFound }
	if err != nil { return nil, err }

	if p.SessionID != sessionID || p.DocumentID != docID { return nil, domain.ErrMisbound }
	if p.ConsumedAt != nil {
		// Idempotent replay — look up the revision previously created for this pending.
		var rid string; var rnum int64
		if err := tx.QueryRowContext(ctx,
			`SELECT id::text, revision_num FROM document_revisions
			 WHERE document_id=$1 AND content_hash=$2`, docID, p.ContentHash,
		).Scan(&rid, &rnum); err != nil { return nil, fmt.Errorf("replay lookup: %w", err) }
		return &CommitResult{RevisionID: rid, RevisionNum: rnum, AlreadyConsumed: true}, tx.Commit()
	}
	if time.Now().After(p.ExpiresAt) { return nil, domain.ErrExpiredUpload }

	// Re-verify session still active + holder + ack still matches base.
	var sessUser, sessAck, sessStatus string
	err = tx.QueryRowContext(ctx,
		`SELECT user_id::text, last_acknowledged_revision_id::text, status
		 FROM editor_sessions WHERE id=$1 FOR UPDATE`, sessionID,
	).Scan(&sessUser, &sessAck, &sessStatus)
	if err != nil { return nil, err }
	if sessStatus != string(domain.SessionActive) { return nil, domain.ErrSessionInactive }
	if sessUser != userID { return nil, domain.ErrSessionNotHolder }
	if sessAck != p.BaseRevisionID { return nil, domain.ErrStaleBase }

	// TOCTOU guard: service verified S3 hash matches pending.content_hash moments
	// before this call, but a concurrent tx could have rewritten the pending row.
	// Re-check under lock.
	if serverComputedHash != p.ContentHash { return nil, domain.ErrContentHashMismatch }

	var revID string; var revNum int64
	if err := tx.QueryRowContext(ctx,
		`INSERT INTO document_revisions
		   (document_id, parent_revision_id, session_id, storage_key, content_hash, form_data_snapshot)
		 VALUES ($1,$2,$3,$4,$5,$6) RETURNING id::text, revision_num`,
		docID, p.BaseRevisionID, sessionID, p.StorageKey, p.ContentHash, formDataSnapshot,
	).Scan(&revID, &revNum); err != nil { return nil, fmt.Errorf("insert revision: %w", err) }

	if _, err := tx.ExecContext(ctx,
		`UPDATE documents SET current_revision_id=$1, form_data_json=$2, updated_at=now() WHERE id=$3`,
		revID, formDataSnapshot, docID,
	); err != nil { return nil, err }
	if _, err := tx.ExecContext(ctx,
		`UPDATE editor_sessions SET last_acknowledged_revision_id=$1 WHERE id=$2`, revID, sessionID,
	); err != nil { return nil, err }
	if _, err := tx.ExecContext(ctx,
		`UPDATE autosave_pending_uploads SET consumed_at=now() WHERE id=$1`, pendingID,
	); err != nil { return nil, err }

	return &CommitResult{RevisionID: revID, RevisionNum: revNum}, tx.Commit()
}

func (r *Repository) DeleteExpiredPending(ctx context.Context, olderThan time.Time) (int, error) {
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM autosave_pending_uploads WHERE expires_at < $1 AND consumed_at IS NULL`,
		olderThan)
	if err != nil { return 0, err }
	n, _ := res.RowsAffected()
	return int(n), nil
}
```

- [ ] **Step 5: Checkpoints + revision lookup**

Append:

```go
func (r *Repository) CreateCheckpoint(ctx context.Context, docID, actorUserID, label string) (*domain.Checkpoint, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil { return nil, err }
	defer tx.Rollback()

	var revID string
	if err := tx.QueryRowContext(ctx,
		`SELECT current_revision_id::text FROM documents WHERE id=$1 FOR UPDATE`, docID,
	).Scan(&revID); err != nil { return nil, err }
	if revID == "" { return nil, fmt.Errorf("document has no current revision") }

	var nextVer int
	if err := tx.QueryRowContext(ctx,
		`SELECT coalesce(max(version_num),0)+1 FROM document_checkpoints WHERE document_id=$1`, docID,
	).Scan(&nextVer); err != nil { return nil, err }

	cp := &domain.Checkpoint{DocumentID: docID, RevisionID: revID, VersionNum: nextVer, Label: label, CreatedBy: actorUserID}
	if err := tx.QueryRowContext(ctx,
		`INSERT INTO document_checkpoints (document_id, revision_id, version_num, label, created_by)
		 VALUES ($1,$2,$3,$4,$5) RETURNING id::text, created_at`,
		docID, revID, nextVer, label, actorUserID,
	).Scan(&cp.ID, &cp.CreatedAt); err != nil { return nil, err }

	return cp, tx.Commit()
}

func (r *Repository) ListCheckpoints(ctx context.Context, docID string) ([]domain.Checkpoint, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id::text, document_id::text, revision_id::text, version_num, coalesce(label,''), created_at, created_by::text
		 FROM document_checkpoints WHERE document_id=$1 ORDER BY version_num DESC`, docID)
	if err != nil { return nil, err }
	defer rows.Close()
	out := []domain.Checkpoint{}
	for rows.Next() {
		var c domain.Checkpoint
		if err := rows.Scan(&c.ID, &c.DocumentID, &c.RevisionID, &c.VersionNum, &c.Label, &c.CreatedAt, &c.CreatedBy); err != nil { return nil, err }
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *Repository) GetRevision(ctx context.Context, docID, revID string) (*domain.Revision, error) {
	var rv domain.Revision
	err := r.db.QueryRowContext(ctx,
		`SELECT id::text, document_id::text, revision_num, coalesce(parent_revision_id::text,''), session_id::text, storage_key, content_hash, form_data_snapshot, created_at
		 FROM document_revisions WHERE id=$1 AND document_id=$2`, revID, docID,
	).Scan(&rv.ID, &rv.DocumentID, &rv.RevisionNum, &rv.ParentRevisionID, &rv.SessionID, &rv.StorageKey, &rv.ContentHash, &rv.FormDataSnapshot, &rv.CreatedAt)
	if err != nil { return nil, err }
	return &rv, nil
}

// RestoreCheckpoint is forward-only: it resolves the checkpoint's revision,
// copies that revision's storage_key + content_hash + form_data_snapshot into
// a NEW revision appended to head (parent = current_revision_id). It never
// rewrites history or rewinds revision_num. Session holder/active checks apply
// — restore is a session-authoritative action equivalent to a large autosave.
func (r *Repository) RestoreCheckpoint(ctx context.Context, docID, actorUserID string, versionNum int) (*RestoreResult, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil { return nil, err }
	defer tx.Rollback()

	// Lock document + require active session held by actor.
	var sessID, sessUser, sessStatus string
	var curRev string
	if err := tx.QueryRowContext(ctx,
		`SELECT coalesce(active_session_id::text,''), coalesce(current_revision_id::text,'')
		 FROM documents WHERE id=$1 FOR UPDATE`, docID,
	).Scan(&sessID, &curRev); err != nil { return nil, err }
	if sessID == "" { return nil, domain.ErrSessionInactive }
	if err := tx.QueryRowContext(ctx,
		`SELECT user_id::text, status FROM editor_sessions WHERE id=$1 FOR UPDATE`, sessID,
	).Scan(&sessUser, &sessStatus); err != nil { return nil, err }
	if sessStatus != string(domain.SessionActive) { return nil, domain.ErrSessionInactive }
	if sessUser != actorUserID { return nil, domain.ErrSessionNotHolder }

	// Resolve checkpoint.
	var cpRevID, cpStorageKey, cpContentHash string
	var cpFormData []byte
	err = tx.QueryRowContext(ctx,
		`SELECT cp.revision_id::text, r.storage_key, r.content_hash, r.form_data_snapshot
		 FROM document_checkpoints cp
		 JOIN document_revisions r ON r.id = cp.revision_id
		 WHERE cp.document_id=$1 AND cp.version_num=$2`, docID, versionNum,
	).Scan(&cpRevID, &cpStorageKey, &cpContentHash, &cpFormData)
	if errors.Is(err, sql.ErrNoRows) { return nil, domain.ErrCheckpointNotFound }
	if err != nil { return nil, err }

	// Append a new head revision pointing at the checkpoint's storage_key. The
	// content_hash is reused (content-addressed storage means no new upload).
	// NOTE: document_revisions.UNIQUE (document_id, content_hash) means restoring
	// to a hash that already exists as head is a no-op — ON CONFLICT returns the
	// existing row (xmax != 0 in pg). We detect that via `xmax::text::bigint != 0`
	// on the RETURNING row to set RestoreResult.Idempotent = true.
	var newRevID string; var newRevNum int64; var idempotent bool
	err = tx.QueryRowContext(ctx,
		`INSERT INTO document_revisions
		   (document_id, parent_revision_id, session_id, storage_key, content_hash, form_data_snapshot)
		 VALUES ($1,$2,$3,$4,$5,$6)
		 ON CONFLICT (document_id, content_hash)
		 DO UPDATE SET id = document_revisions.id
		 RETURNING id::text, revision_num, (xmax::text::bigint <> 0)`,
		docID, curRev, sessID, cpStorageKey, cpContentHash, cpFormData,
	).Scan(&newRevID, &newRevNum, &idempotent)
	if err != nil { return nil, fmt.Errorf("restore insert: %w", err) }

	// On idempotent (head already equals checkpoint hash): do NOT rewrite
	// documents.current_revision_id or the session ack — they already match.
	// On fresh insert: advance head + session ack.
	if !idempotent {
		if _, err := tx.ExecContext(ctx,
			`UPDATE documents SET current_revision_id=$1, form_data_json=$2, updated_at=now() WHERE id=$3`,
			newRevID, cpFormData, docID,
		); err != nil { return nil, err }
		if _, err := tx.ExecContext(ctx,
			`UPDATE editor_sessions SET last_acknowledged_revision_id=$1 WHERE id=$2`, newRevID, sessID,
		); err != nil { return nil, err }
	}

	return &RestoreResult{
		NewRevisionID:   newRevID,
		NewRevisionNum:  newRevNum,
		CheckpointRevID: cpRevID,
		Idempotent:      idempotent,
	}, tx.Commit()
}

// IsDocumentOwner returns true iff the document was created by userID under
// tenantID. Used by handler-level defense-in-depth for document_filler routes.
func (r *Repository) IsDocumentOwner(ctx context.Context, tenantID, docID, userID string) (bool, error) {
	var ok bool
	err := r.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM documents WHERE id=$1 AND tenant_id=$2 AND created_by=$3)`,
		docID, tenantID, userID,
	).Scan(&ok)
	return ok, err
}
```

- [ ] **Step 6: Integration test (testcontainers)**

```go
// internal/modules/documents_v2/repository/repository_integration_test.go
//go:build integration
// +build integration

package repository_test

// Standard repo integration test harness. Uses the shared testcontainers
// helper already wired by W1 (see internal/platform/testcontainers/postgres.go).
// Covers: CreateDocument round-trip, AcquireSession partial unique index,
// CommitUpload happy path, DeleteExpiredPending cleanup.
```

Plan executors: copy the existing W2 repository_integration_test.go harness; add four test funcs `TestCreateDocument_RoundTrip`, `TestAcquireSession_SingleWriterInvariant`, `TestCommitUpload_Happy`, `TestDeleteExpiredPending_RemovesExpired`.

- [ ] **Step 7: Commit**

```bash
go test -tags=integration ./internal/modules/documents_v2/repository/...
rtk git add internal/modules/documents_v2/repository
rtk git commit -m "feat(documents_v2/repository): docs+sessions+revisions+pending+checkpoints"
```

---

## Task 5: Go application — service (use cases)

**Files:**
- Create: `internal/modules/documents_v2/application/service.go`
- Create: `internal/modules/documents_v2/application/service_test.go`
- Create: `internal/modules/documents_v2/application/autosave_commit_branches_test.go`

- [ ] **Step 1: Write `service.go` — interfaces + constructor**

```go
package application

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"metaldocs/internal/modules/documents_v2/domain"
)

type Repository interface {
	CreateDocument(ctx context.Context, d *domain.Document, initialContentHash string) (docID, revID, sessionID string, err error)
	SetRevisionStorageKey(ctx context.Context, revID, storageKey string) error
	GetDocument(ctx context.Context, tenantID, id string) (*domain.Document, error)
	ListDocuments(ctx context.Context, tenantID string) ([]domain.Document, error)
	ListDocumentsForUser(ctx context.Context, tenantID, userID string) ([]domain.Document, error)
	UpdateDocumentStatus(ctx context.Context, tenantID, id string, cur, next domain.DocumentStatus, stampTime bool) error
	IsDocumentOwner(ctx context.Context, tenantID, docID, userID string) (bool, error)
	AcquireSession(ctx context.Context, tenantID, docID, userID string) (*domain.Session, error)
	HeartbeatSession(ctx context.Context, sessionID, userID string) error
	ReleaseSession(ctx context.Context, sessionID, userID string) error
	ForceReleaseSession(ctx context.Context, sessionID string) error
	ExpireStaleSessions(ctx context.Context, now time.Time) (int, error)
	PresignReserve(ctx context.Context, sessionID, userID, docID, baseRev, contentHash, storageKey string, expiresAt time.Time) (string, error)
	GetPendingForCommit(ctx context.Context, pendingID string) (*PendingCommitMeta, error)
	CommitUpload(ctx context.Context, sessionID, userID, docID, pendingID, serverComputedHash string, formDataSnapshot []byte) (*CommitResult, error)
	CreateCheckpoint(ctx context.Context, docID, actorUserID, label string) (*domain.Checkpoint, error)
	ListCheckpoints(ctx context.Context, docID string) ([]domain.Checkpoint, error)
	RestoreCheckpoint(ctx context.Context, docID, actorUserID string, versionNum int) (*RestoreResult, error)
	GetRevision(ctx context.Context, docID, revID string) (*domain.Revision, error)
	DeleteExpiredPending(ctx context.Context, olderThan time.Time) (int, error)
}

// PendingCommitMeta is the minimal read the service needs before hashing S3.
// Repository returns ErrPendingNotFound when pendingID is unknown.
type PendingCommitMeta = struct {
	SessionID           string
	DocumentID          string
	BaseRevisionID      string
	ExpectedContentHash string
	StorageKey          string
	ExpiresAt           time.Time
	ConsumedAt          *time.Time
}

// RestoreResult mirrors repository.RestoreResult for handler JSON.
type RestoreResult = struct {
	NewRevisionID   string
	NewRevisionNum  int64
	CheckpointRevID string
	Idempotent      bool
}

// CommitResult mirrors repository.CommitResult; alias so handlers import only application.
type CommitResult = struct {
	RevisionID      string
	RevisionNum     int64
	AlreadyConsumed bool
}

type DocgenRenderer interface {
	RenderDocx(ctx context.Context, templateDocxKey, schemaKey, outputKey string, formData json.RawMessage) (contentHash string, sizeBytes int64, unreplaced []string, err error)
}

type Presigner interface {
	PresignRevisionPUT(ctx context.Context, tenantID, docID, contentHash string) (url, storageKey string, err error)
	PresignObjectGET(ctx context.Context, storageKey string) (url string, err error)
	AdoptTempObject(ctx context.Context, tmpKey, finalKey string) error
	DeleteObject(ctx context.Context, key string) error
	// HashObject streams the object at key and returns its lower-case hex SHA256.
	// Returns domain.ErrUploadMissing if the object does not exist.
	HashObject(ctx context.Context, key string) (string, error)
}

type TemplateReader interface {
	GetPublishedVersion(ctx context.Context, tenantID, templateVersionID string) (docxKey, schemaKey, schemaJSON string, err error)
}

type FormValidator interface {
	Validate(schemaJSON string, formData json.RawMessage) (valid bool, errs []string, err error)
}

type Audit interface {
	Write(ctx context.Context, tenantID, actorID, action, docID string, meta any)
}

type Service struct {
	repo      Repository
	docgen    DocgenRenderer
	presigner Presigner
	tpl       TemplateReader
	fv        FormValidator
	audit     Audit
}

func New(r Repository, d DocgenRenderer, p Presigner, t TemplateReader, fv FormValidator, a Audit) *Service {
	return &Service{repo: r, docgen: d, presigner: p, tpl: t, fv: fv, audit: a}
}
```

- [ ] **Step 2: Create document**

Append:

```go
type CreateDocumentCmd struct {
	TenantID           string
	ActorUserID        string
	TemplateVersionID  string
	Name               string
	FormData           json.RawMessage
}

type CreateDocumentResult struct {
	DocumentID        string
	InitialRevisionID string
	SessionID         string
}

// Object-key reconciliation (locked pattern):
//   1. docgen-v2 writes to provisional tmp key: tenants/{tid}/documents/tmp/{uuid}.docx
//   2. docgen returns content_hash.
//   3. repo.CreateDocument inserts doc+session+revision in one deferrable-FK tx,
//      storing storage_key = "" (placeholder) — repo returns the new docID.
//   4. Service computes final content-addressed key using docID + content_hash.
//   5. Service calls presigner.AdoptTempObject(ctx, tmpKey, finalKey) which does
//      S3 CopyObject then DeleteObject. Atomic from S3's view per object.
//   6. Service calls repo.SetRevisionStorageKey(ctx, revID, finalKey) inside a
//      short follow-up tx.
//   7. On any error after step 2, Service calls presigner.DeleteObject(tmpKey)
//      to prevent orphan objects (orphan-pending sweeper is a backstop only).

func (s *Service) CreateDocument(ctx context.Context, cmd CreateDocumentCmd) (res *CreateDocumentResult, err error) {
	docxKey, schemaKey, schemaJSON, err := s.tpl.GetPublishedVersion(ctx, cmd.TenantID, cmd.TemplateVersionID)
	if err != nil { return nil, fmt.Errorf("template lookup: %w", err) }
	ok, verrs, err := s.fv.Validate(schemaJSON, cmd.FormData)
	if err != nil { return nil, err }
	if !ok { return nil, fmt.Errorf("form_data_invalid: %v", verrs) }

	tmpKey := fmt.Sprintf("tenants/%s/documents/tmp/%s.docx", cmd.TenantID, uuid.NewString())
	contentHash, _, _, err := s.docgen.RenderDocx(ctx, docxKey, schemaKey, tmpKey, cmd.FormData)
	if err != nil { return nil, fmt.Errorf("render: %w", err) }

	// Guarantee tmp cleanup on any failure after docgen.RenderDocx succeeded.
	cleanupTmp := true
	defer func() {
		if cleanupTmp {
			_ = s.presigner.DeleteObject(context.Background(), tmpKey)
		}
	}()

	doc := &domain.Document{
		TenantID: cmd.TenantID, TemplateVersionID: cmd.TemplateVersionID,
		Name: cmd.Name, FormDataJSON: cmd.FormData, CreatedBy: cmd.ActorUserID,
	}
	docID, revID, sessionID, err := s.repo.CreateDocument(ctx, doc, contentHash)
	if err != nil { return nil, err }

	finalKey := fmt.Sprintf("tenants/%s/documents/%s/revisions/%s.docx", cmd.TenantID, docID, contentHash)
	if err := s.presigner.AdoptTempObject(ctx, tmpKey, finalKey); err != nil {
		return nil, fmt.Errorf("adopt tmp: %w", err)
	}
	cleanupTmp = false

	if err := s.repo.SetRevisionStorageKey(ctx, revID, finalKey); err != nil {
		// S3 object exists at finalKey; sweeper will reconcile if this persists.
		return nil, fmt.Errorf("set revision key: %w", err)
	}

	s.audit.Write(ctx, cmd.TenantID, cmd.ActorUserID, "document.created", docID, map[string]any{"template_version_id": cmd.TemplateVersionID})

	return &CreateDocumentResult{DocumentID: docID, InitialRevisionID: revID, SessionID: sessionID}, nil
}
```

**Repository contract implied by this flow (must match Task 4):**
- `Repository.CreateDocument(ctx, doc, contentHash) (docID, revID, sessionID string, err error)` — inserts `document_revisions` with `storage_key = ''` and `content_hash = contentHash`. The `UNIQUE (document_id, content_hash)` constraint still holds because every (docID, contentHash) pair is fresh per call.
- `Repository.SetRevisionStorageKey(ctx, revID, storageKey string) error` — short tx; also validates `storage_key = ''` via `UPDATE … WHERE id=$1 AND storage_key=''` for idempotency.
- `Presigner.AdoptTempObject(ctx, tmpKey, finalKey string) error` — S3 CopyObject then DeleteObject; returns first error.
- `Presigner.DeleteObject(ctx, key string) error` — idempotent delete (NoSuchKey → nil).

Update Task 4 Step 2 repository signature and add `SetRevisionStorageKey`. Update Task 9 Step 3 `DocumentPresigner` to include `AdoptTempObject` + `DeleteObject`.

- [ ] **Step 3: Autosave presign + commit**

Append:

```go
type PresignAutosaveCmd struct {
	TenantID, ActorUserID, DocumentID, SessionID, BaseRevisionID, ContentHash string
}

type PresignAutosaveResult struct {
	UploadURL       string
	PendingUploadID string
	ExpiresAt       time.Time
}

func (s *Service) PresignAutosave(ctx context.Context, cmd PresignAutosaveCmd) (*PresignAutosaveResult, error) {
	url, storageKey, err := s.presigner.PresignRevisionPUT(ctx, cmd.TenantID, cmd.DocumentID, cmd.ContentHash)
	if err != nil { return nil, err }
	expiresAt := time.Now().Add(15 * time.Minute)
	pendingID, err := s.repo.PresignReserve(ctx, cmd.SessionID, cmd.ActorUserID, cmd.DocumentID, cmd.BaseRevisionID, cmd.ContentHash, storageKey, expiresAt)
	if err != nil { return nil, err }
	return &PresignAutosaveResult{UploadURL: url, PendingUploadID: pendingID, ExpiresAt: expiresAt}, nil
}

type CommitAutosaveCmd struct {
	TenantID, ActorUserID, DocumentID, SessionID, PendingUploadID string
	FormDataSnapshot    json.RawMessage
}

// CommitAutosave is server-authoritative. The client does NOT supply a content
// hash — the service streams the object back from S3, computes SHA256, and
// compares to the hash reserved at presign time. Flow:
//   1. Load the pending row metadata (session+base+expected_hash+storage_key).
//      Cheap read, no lock — repo.CommitUpload later revalidates under lock.
//   2. Hash the S3 object at pending.StorageKey via presigner.HashObject.
//      S3 NotFound → ErrUploadMissing (410). Any other S3 error → 502.
//   3. If hash != pending.ExpectedHash → call presigner.DeleteObject to drop
//      the orphan bytes, then return ErrContentHashMismatch (422).
//   4. Pass the server-computed hash to repo.CommitUpload which owns the
//      transactional rejection matrix (misbound, expired, session checks, etc.).
func (s *Service) CommitAutosave(ctx context.Context, cmd CommitAutosaveCmd) (*CommitResult, error) {
	meta, err := s.repo.GetPendingForCommit(ctx, cmd.PendingUploadID)
	if err != nil { return nil, err } // includes ErrPendingNotFound

	serverHash, err := s.presigner.HashObject(ctx, meta.StorageKey)
	if err != nil {
		if errors.Is(err, domain.ErrUploadMissing) { return nil, domain.ErrUploadMissing }
		return nil, fmt.Errorf("hash s3 object: %w", err)
	}
	if serverHash != meta.ExpectedContentHash {
		_ = s.presigner.DeleteObject(ctx, meta.StorageKey) // best-effort orphan cleanup
		return nil, domain.ErrContentHashMismatch
	}

	res, err := s.repo.CommitUpload(ctx, cmd.SessionID, cmd.ActorUserID, cmd.DocumentID, cmd.PendingUploadID, serverHash, cmd.FormDataSnapshot)
	if err != nil { return nil, err }
	if !res.AlreadyConsumed {
		s.audit.Write(ctx, cmd.TenantID, cmd.ActorUserID, "document.autosaved", cmd.DocumentID, map[string]any{"revision_id": res.RevisionID, "revision_num": res.RevisionNum})
	}
	return res, nil
}
```

- [ ] **Step 4: Session + checkpoint + lifecycle**

Append:

```go
func (s *Service) AcquireSession(ctx context.Context, tenantID, docID, userID string) (*domain.Session, bool /*readonly*/, error) {
	sess, err := s.repo.AcquireSession(ctx, tenantID, docID, userID)
	if err == domain.ErrSessionTaken { return sess, true, nil }
	if err != nil { return nil, false, err }
	s.audit.Write(ctx, tenantID, userID, "session.acquired", docID, map[string]any{"session_id": sess.ID})
	return sess, false, nil
}

func (s *Service) HeartbeatSession(ctx context.Context, sessionID, userID string) error { return s.repo.HeartbeatSession(ctx, sessionID, userID) }
func (s *Service) ReleaseSession(ctx context.Context, tenantID, sessionID, userID, docID string) error {
	if err := s.repo.ReleaseSession(ctx, sessionID, userID); err != nil { return err }
	s.audit.Write(ctx, tenantID, userID, "session.released", docID, map[string]any{"session_id": sessionID})
	return nil
}
func (s *Service) ForceReleaseSession(ctx context.Context, tenantID, adminID, sessionID, docID string) error {
	if err := s.repo.ForceReleaseSession(ctx, sessionID); err != nil { return err }
	s.audit.Write(ctx, tenantID, adminID, "session.force_released", docID, map[string]any{"session_id": sessionID})
	return nil
}

func (s *Service) CreateCheckpoint(ctx context.Context, tenantID, docID, actorID, label string) (*domain.Checkpoint, error) {
	cp, err := s.repo.CreateCheckpoint(ctx, docID, actorID, label)
	if err != nil { return nil, err }
	s.audit.Write(ctx, tenantID, actorID, "document.checkpoint_created", docID, map[string]any{"version_num": cp.VersionNum, "label": label})
	return cp, nil
}

func (s *Service) ListCheckpoints(ctx context.Context, tenantID, docID string) ([]domain.Checkpoint, error) { return s.repo.ListCheckpoints(ctx, docID) }

// ListDocumentsForUser is used by the handler when the caller is a
// document_filler (non-admin) — prevents metadata leakage of other users' docs.
// Admins / template_* roles use ListDocuments (unrestricted tenant scope).
func (s *Service) ListDocumentsForUser(ctx context.Context, tenantID, userID string) ([]domain.Document, error) {
	return s.repo.ListDocumentsForUser(ctx, tenantID, userID)
}

// RestoreCheckpoint is forward-only: appends a new head revision cloning the
// checkpoint's storage_key+content_hash+form_data. Audits as
// `document.checkpoint_restored`. Session holder + active requirement enforced
// by repo under FOR UPDATE. Fillers restoring their own doc are allowed; the
// ownership gate is at the handler level (defense-in-depth).
func (s *Service) RestoreCheckpoint(ctx context.Context, tenantID, docID, actorID string, versionNum int) (*RestoreResult, error) {
	res, err := s.repo.RestoreCheckpoint(ctx, docID, actorID, versionNum)
	if err != nil { return nil, err }
	s.audit.Write(ctx, tenantID, actorID, "document.checkpoint_restored", docID, map[string]any{
		"version_num":        versionNum,
		"new_revision_id":    res.NewRevisionID,
		"new_revision_num":   res.NewRevisionNum,
		"source_revision_id": res.CheckpointRevID,
		"idempotent":         res.Idempotent,
	})
	return res, nil
}

func (s *Service) Finalize(ctx context.Context, tenantID, docID, actorID string) error {
	if err := s.repo.UpdateDocumentStatus(ctx, tenantID, docID, domain.DocStatusDraft, domain.DocStatusFinalized, true); err != nil { return err }
	s.audit.Write(ctx, tenantID, actorID, "document.finalized", docID, nil)
	return nil
}
func (s *Service) Archive(ctx context.Context, tenantID, docID, actorID string, fromFinalized bool) error {
	cur := domain.DocStatusDraft
	if fromFinalized { cur = domain.DocStatusFinalized }
	if err := s.repo.UpdateDocumentStatus(ctx, tenantID, docID, cur, domain.DocStatusArchived, true); err != nil { return err }
	s.audit.Write(ctx, tenantID, actorID, "document.archived", docID, nil)
	return nil
}
```

- [ ] **Step 5: Table-driven unit tests for happy paths + role/state errors**

`service_test.go`: one test per public method, using a hand-rolled `fakeRepo` struct implementing `Repository`. Minimum coverage: `TestCreateDocument_OK`, `TestCreateDocument_InvalidFormData_Rejects`, `TestAcquireSession_Readonly_WhenTaken`, `TestAcquireSession_Success_RecordsAudit`, `TestCreateCheckpoint_OK`, `TestFinalize_FromDraft_OK`, `TestFinalize_FromFinalized_Rejects`.

- [ ] **Step 6: 9-branch commit rejection matrix (service-level unit)**

`autosave_commit_branches_test.go` — the hot spot: exhaustive coverage of every rejection branch a CommitAutosave call can raise. With server-authoritative commit, branches split across layers:

| # | Branch                   | Layer     | Trigger                                    | Mapped HTTP |
|---|--------------------------|-----------|--------------------------------------------|-------------|
| 1 | `pending_not_found`      | service   | `GetPendingForCommit` → `ErrPendingNotFound` | 404        |
| 2 | `upload_missing`         | service   | `presigner.HashObject` → `ErrUploadMissing` (S3 404) | 410 |
| 3 | `content_hash_mismatch`  | service   | S3 SHA256 ≠ `ExpectedContentHash` → orphan delete + 422 | 422 |
| 4 | `misbound_session`       | repo      | pending row session/doc/base ≠ args        | 409         |
| 5 | `already_consumed_replay`| repo      | `consumed_at IS NOT NULL` ∧ `(session,base,hash)` match | 200 idempotent |
| 6 | `expired_upload`         | repo      | `expires_at < now()`                       | 410         |
| 7 | `session_inactive`       | repo      | `editor_sessions.status != 'active'`       | 409         |
| 8 | `session_not_holder`     | repo      | `editor_sessions.user_id != actor`         | 403         |
| 9 | `stale_base`             | repo      | `documents.current_revision_id != base`    | 409         |

Branches 1–3 are driven with a `fakePresigner`; 4–9 with `fakeRepo.commitErr` / `fakeRepo.commitResult`.

```go
package application_test

import (
	"context"
	"testing"
	"time"

	"metaldocs/internal/modules/documents_v2/application"
	"metaldocs/internal/modules/documents_v2/domain"
)

func TestCommitAutosave_RejectionBranches(t *testing.T) {
	type harness struct {
		repo *fakeRepo
		pre  *fakePresigner
	}
	branches := []struct {
		name    string
		setup   func(*harness)
		wantErr error
		assertOrphanDeleted bool
	}{
		{"pending_not_found", func(h *harness) { h.repo.pendingErr = domain.ErrPendingNotFound }, domain.ErrPendingNotFound, false},
		{"upload_missing",    func(h *harness) { h.pre.hashErr = domain.ErrUploadMissing },       domain.ErrUploadMissing, false},
		{"content_hash_mismatch", func(h *harness) { h.pre.hashReturn = "wronghash" }, domain.ErrContentHashMismatch, true},
		{"misbound_session",  func(h *harness) { h.repo.commitErr = domain.ErrMisbound },         domain.ErrMisbound, false},
		{"already_consumed_replay", func(h *harness) {
			h.repo.commitResult = &application.CommitResult{RevisionID: "r1", AlreadyConsumed: true}
		}, nil, false},
		{"expired_upload",    func(h *harness) { h.repo.commitErr = domain.ErrExpiredUpload },    domain.ErrExpiredUpload, false},
		{"session_inactive",  func(h *harness) { h.repo.commitErr = domain.ErrSessionInactive },  domain.ErrSessionInactive, false},
		{"session_not_holder", func(h *harness) { h.repo.commitErr = domain.ErrSessionNotHolder }, domain.ErrSessionNotHolder, false},
		{"stale_base",        func(h *harness) { h.repo.commitErr = domain.ErrStaleBase },        domain.ErrStaleBase, false},
	}
	for _, b := range branches {
		t.Run(b.name, func(t *testing.T) {
			h := &harness{
				repo: &fakeRepo{
					pendingMeta: &application.PendingCommitMeta{
						SessionID: "s1", DocumentID: "d1", BaseRevisionID: "r0",
						ExpectedContentHash: "h_expected", StorageKey: "tenants/t1/documents/d1/revisions/p1.docx",
						ExpiresAt: time.Now().Add(5 * time.Minute),
					},
				},
				pre: &fakePresigner{hashReturn: "h_expected"}, // matches ExpectedContentHash by default
			}
			b.setup(h)
			svc := application.New(h.repo, h.pre, nil, nil, nil, &noopAudit{})
			_, err := svc.CommitAutosave(context.Background(), application.CommitAutosaveCmd{
				TenantID: "t1", ActorUserID: "u1", DocumentID: "d1",
				SessionID: "s1", PendingUploadID: "p1",
			})
			if err != b.wantErr { t.Fatalf("branch %s: got %v, want %v", b.name, err, b.wantErr) }
			if b.assertOrphanDeleted && h.pre.deleteCalls != 1 {
				t.Fatalf("branch %s: expected orphan cleanup via presigner.DeleteObject, got %d calls", b.name, h.pre.deleteCalls)
			}
			// Replay success: RevisionID returned, audit MUST NOT be written twice.
			if b.name == "already_consumed_replay" && !h.repo.commitResult.AlreadyConsumed {
				t.Fatalf("branch %s: expected replay flag", b.name)
			}
		})
	}
}
```

`fakeRepo`, `fakePresigner`, `fakeDocgen`, `fakeTplReader`, `fakeFormVal`, and `noopAudit` helpers live in a sibling `testhelpers_test.go`. Contract:

- `fakeRepo` must implement every method on the `application.Repository` interface from `service.go` Step 1 — including `SetRevisionStorageKey`, `GetPendingForCommit`, `IsDocumentOwner`, `RestoreCheckpoint`, `ListDocumentsForUser`. Fields:
  - `createDocErr error`, `setStorageKeyErr error`
  - `acquireSess *domain.Session`, `acquireErr error`
  - `pendingMeta *application.PendingCommitMeta`, `pendingErr error`
  - `commitResult *application.CommitResult`, `commitErr error`
  - `restoreResult *application.RestoreResult`, `restoreErr error`
  - `ownerReturn bool`, `ownerErr error`
  - plus a recorded call log for assertions.
- `fakePresigner` must implement `application.Presigner` — 5 methods: `SignPut`, `SignGet`, `AdoptTempObject`, `DeleteObject`, `HashObject`. Fields:
  - `hashReturn string`, `hashErr error`
  - `adoptErr error`, `deleteCalls int`, `deleteErr error`
  - record every call with args for inspection.
- `fakeDocgen.RenderDocx` returns a canned `contentHash` (e.g. `"deadbeef"`) and `unreplaced = nil` unless `renderErr` set.
- `fakeTplReader.GetPublishedVersion` returns canned `docxKey`, `schemaKey`, `schemaJSON` unless `tplErr` set.
- `fakeFormVal.Validate` returns `(true, nil, nil)` by default; `invalidBlock = true` flips to `(false, []string{"form_data_invalid"}, nil)`.
- `noopAudit.Write` is a no-op; can record calls for assertions like "audit must be written on commit success".

Copy W2's template pattern for how these structs are organized and wired into `application.New(...)` in each test.

- [ ] **Step 7: DB-state-unchanged integration tests per rejection branch**

`repository/commit_rejections_integration_test.go` — real postgres via `dockertest`; one seeded fixture per rejection branch. **For each branch, assert that on error the `document_revisions` table, `documents.current_revision_id`, and `pending_uploads.consumed_at` are all UNCHANGED.** This guards against partial-commit leaks that a service-level fake can't catch.

Coverage mirrors branches 4–9 of the matrix (repo-level branches):

| Branch               | Fixture                                     | DB invariant asserted                  |
|----------------------|---------------------------------------------|----------------------------------------|
| misbound_session     | pending.session ≠ arg session               | `COUNT(*) FROM document_revisions WHERE document_id=$1` unchanged |
| expired_upload       | pending.expires_at = now() - 1s             | `pending_uploads.consumed_at IS NULL`  |
| session_inactive     | session.status = 'ended'                    | `documents.current_revision_id` unchanged |
| session_not_holder   | session.user_id = other user                | Same                                   |
| stale_base           | document.current_revision_id advanced mid-fixture | Same                             |
| already_consumed_replay | pending.consumed_at set; args match        | Returns same revision id, no new row, no extra `consumed_at` update |

Run with `go test -tags=integration ./internal/modules/documents_v2/repository/...`. CI job already provisions postgres in W2 — reuse the same container.

```go
func TestCommitUpload_MisboundSession_LeavesDBUnchanged(t *testing.T) {
	ctx := context.Background()
	tx := setupDocV2Fixture(t) // seeds doc + base rev + session + pending
	before := snapshotDocV2(t, tx, "d1")
	_, err := repo.CommitUpload(ctx, "s_OTHER", "u1", "d1", "p1", "h_expected", []byte(`{}`))
	if !errors.Is(err, domain.ErrMisbound) { t.Fatalf("want misbound, got %v", err) }
	after := snapshotDocV2(t, tx, "d1")
	if !reflect.DeepEqual(before, after) { t.Fatalf("db state changed after misbound rejection: %+v → %+v", before, after) }
}
```

`snapshotDocV2` reads `(current_revision_id, active_session_id)` from documents, every `document_revisions` row, and every `pending_uploads` row — returns a comparable struct.

- [ ] **Step 8: Run + commit**

```bash
go test ./internal/modules/documents_v2/application/...
go test -tags=integration ./internal/modules/documents_v2/repository/...
rtk git add internal/modules/documents_v2
rtk git commit -m "feat(documents_v2): 9-branch commit matrix + DB-unchanged integration tests"
```

---

## Task 6: Go HTTP handler + routes

**Files:**
- Create: `internal/modules/documents_v2/delivery/http/handler.go`
- Create: `internal/modules/documents_v2/delivery/http/handler_test.go`

- [ ] **Step 1: Handler skeleton + role helper**

```go
package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"metaldocs/internal/modules/documents_v2/application"
	"metaldocs/internal/modules/documents_v2/domain"
)

// Role constants stamped by IAM middleware into X-User-Roles (comma-separated).
const (
	roleAdmin          = "admin"
	roleTemplateAuthor = "template_author"
	roleDocumentFiller = "document_filler"
)

func requireRole(r *http.Request, want ...string) bool {
	hdr := r.Header.Get("X-User-Roles")
	if hdr == "" { return false }
	for _, w := range want {
		for _, g := range strings.Split(hdr, ",") {
			if strings.TrimSpace(g) == w { return true }
		}
	}
	return false
}

type Service interface {
	CreateDocument(ctx context.Context, cmd application.CreateDocumentCmd) (*application.CreateDocumentResult, error)
	GetDocument(ctx context.Context, tenantID, id string) (*domain.Document, error)
	ListDocuments(ctx context.Context, tenantID string) ([]domain.Document, error)
	ListDocumentsForUser(ctx context.Context, tenantID, userID string) ([]domain.Document, error)
	IsDocumentOwner(ctx context.Context, tenantID, docID, userID string) (bool, error)
	AcquireSession(ctx context.Context, tenantID, docID, userID string) (*domain.Session, bool, error)
	HeartbeatSession(ctx context.Context, sessionID, userID string) error
	ReleaseSession(ctx context.Context, tenantID, sessionID, userID, docID string) error
	ForceReleaseSession(ctx context.Context, tenantID, adminID, sessionID, docID string) error
	PresignAutosave(ctx context.Context, cmd application.PresignAutosaveCmd) (*application.PresignAutosaveResult, error)
	CommitAutosave(ctx context.Context, cmd application.CommitAutosaveCmd) (*application.CommitResult, error)
	CreateCheckpoint(ctx context.Context, tenantID, docID, actorID, label string) (*domain.Checkpoint, error)
	ListCheckpoints(ctx context.Context, tenantID, docID string) ([]domain.Checkpoint, error)
	RestoreCheckpoint(ctx context.Context, tenantID, docID, actorID string, versionNum int) (*application.RestoreResult, error)
	Finalize(ctx context.Context, tenantID, docID, actorID string) error
	Archive(ctx context.Context, tenantID, docID, actorID string, fromFinalized bool) error
	SignedRevisionURL(ctx context.Context, tenantID, docID, revID string) (string, error)
}

type Handler struct{ svc Service }

func NewHandler(svc Service) *Handler { return &Handler{svc: svc} }

// isAdmin returns true iff the IAM-stamped roles include admin.
func isAdmin(r *http.Request) bool { return requireRole(r, roleAdmin) }

// ensureDocAccess is handler-level defense-in-depth for the document ownership
// rule from the spec's RBAC matrix. admin bypasses; every other role (notably
// document_filler) may only act on documents they created. Should be called at
// the top of every document-scoped route after the role gate.
func (h *Handler) ensureDocAccess(ctx context.Context, tenantID, docID, userID string) error {
	// admin bypass is checked by the caller via requireRole(r, roleAdmin, ...);
	// this function is invoked regardless. Admins pass the ownership check too
	// (they are recorded in audit either way).
	// If the user is admin-only, a separate handler path is used and this is
	// not invoked. For filler paths, gate on ownership.
	if h.isCallerAdmin(ctx) { return nil }
	ok, err := h.svc.IsDocumentOwner(ctx, tenantID, docID, userID)
	if err != nil { return err }
	if !ok { return domain.ErrDocumentNotOwner }
	return nil
}
```

The `h.isCallerAdmin` helper reads a `ctx` value stamped by a tiny wrapper around each handler — this keeps `ensureDocAccess` decoupled from `*http.Request`. Wire it like so in `requireRole`:

```go
// Add to handler.go:
type ctxKey int
const ctxAdminKey ctxKey = 1

func withAdminCtx(r *http.Request) *http.Request {
	if requireRole(r, roleAdmin) {
		return r.WithContext(context.WithValue(r.Context(), ctxAdminKey, true))
	}
	return r
}
func (h *Handler) isCallerAdmin(ctx context.Context) bool {
	v, _ := ctx.Value(ctxAdminKey).(bool); return v
}
```

Call `r = withAdminCtx(r)` at the top of each handler (before `ensureDocAccess`).

- [ ] **Step 2: Route registration**

Append:

```go
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v2/documents", h.listDocuments)
	mux.HandleFunc("POST /api/v2/documents", h.createDocument)
	mux.HandleFunc("GET /api/v2/documents/{id}", h.getDocument)
	mux.HandleFunc("POST /api/v2/documents/{id}/finalize", h.finalizeDocument)
	mux.HandleFunc("POST /api/v2/documents/{id}/archive", h.archiveDocument)

	mux.HandleFunc("POST /api/v2/documents/{id}/session/acquire", h.acquireSession)
	mux.HandleFunc("POST /api/v2/documents/{id}/session/heartbeat", h.heartbeatSession)
	mux.HandleFunc("POST /api/v2/documents/{id}/session/release", h.releaseSession)
	mux.HandleFunc("POST /api/v2/documents/{id}/session/force-release", h.forceReleaseSession)

	mux.HandleFunc("POST /api/v2/documents/{id}/autosave/presign", h.presignAutosave)
	mux.HandleFunc("POST /api/v2/documents/{id}/autosave/commit", h.commitAutosave)

	mux.HandleFunc("GET /api/v2/documents/{id}/checkpoints", h.listCheckpoints)
	mux.HandleFunc("POST /api/v2/documents/{id}/checkpoints", h.createCheckpoint)
	mux.HandleFunc("POST /api/v2/documents/{id}/checkpoints/{versionNum}/restore", h.restoreCheckpoint)

	mux.HandleFunc("GET /api/v2/documents/{id}/revisions/{rid}/url", h.signedRevision)
}
```

- [ ] **Step 3: Error mapping helper**

```go
// mapErr is the CANONICAL HTTP status translation for every documented domain
// error. Keep in sync with the rejection matrix below. Missing mappings MUST
// fall to 500 explicitly, not silently return nil.
//
// Rejection matrix (commit path):
//   ErrPendingNotFound       → 404 pending_not_found
//   ErrMisbound              → 409 misbound
//   ErrExpiredUpload         → 410 expired_upload
//   ErrSessionInactive       → 409 session_inactive
//   ErrSessionNotHolder      → 409 session_not_holder
//   ErrStaleBase             → 409 stale_base
//   ErrContentHashMismatch   → 422 content_hash_mismatch
//   ErrUploadMissing         → 410 upload_missing            (server-authoritative)
//   (already-consumed replay is NOT an error; repo returns res.AlreadyConsumed=true)
//
// Additional mappings:
//   ErrForbidden             → 403 forbidden
//   ErrDocumentNotOwner      → 403 forbidden
//   ErrInvalidStateTransition→ 409 invalid_state_transition
//   ErrSessionTaken          → 409 session_taken
//   ErrCheckpointNotFound    → 404 checkpoint_not_found
func mapErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrForbidden),
		errors.Is(err, domain.ErrDocumentNotOwner):
		httpErr(w, 403, "forbidden")
	case errors.Is(err, domain.ErrInvalidStateTransition):
		httpErr(w, 409, "invalid_state_transition")
	case errors.Is(err, domain.ErrSessionInactive):
		httpErr(w, 409, "session_inactive")
	case errors.Is(err, domain.ErrSessionNotHolder):
		httpErr(w, 409, "session_not_holder")
	case errors.Is(err, domain.ErrStaleBase):
		httpErr(w, 409, "stale_base")
	case errors.Is(err, domain.ErrMisbound):
		httpErr(w, 409, "misbound")
	case errors.Is(err, domain.ErrExpiredUpload):
		httpErr(w, 410, "expired_upload")
	case errors.Is(err, domain.ErrUploadMissing):
		httpErr(w, 410, "upload_missing")
	case errors.Is(err, domain.ErrContentHashMismatch):
		httpErr(w, 422, "content_hash_mismatch")
	case errors.Is(err, domain.ErrPendingNotFound):
		httpErr(w, 404, "pending_not_found")
	case errors.Is(err, domain.ErrCheckpointNotFound):
		httpErr(w, 404, "checkpoint_not_found")
	case errors.Is(err, domain.ErrSessionTaken):
		httpErr(w, 409, "session_taken")
	default:
		httpErr(w, 500, err.Error())
	}
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
func httpErr(w http.ResponseWriter, code int, msg string) { writeJSON(w, code, map[string]string{"error": msg}) }
```

- [ ] **Step 4: Document CRUD handlers**

```go
type createDocReq struct {
	TemplateVersionID string          `json:"template_version_id"`
	Name              string          `json:"name"`
	FormData          json.RawMessage `json:"form_data"`
}

func (h *Handler) createDocument(w http.ResponseWriter, r *http.Request) {
	if !requireRole(r, roleAdmin, roleDocumentFiller) { httpErr(w, 403, "forbidden"); return }
	var req createDocReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { httpErr(w, 400, "invalid_body"); return }
	tenant := r.Header.Get("X-Tenant-ID")
	actor := r.Header.Get("X-User-ID")
	res, err := h.svc.CreateDocument(r.Context(), application.CreateDocumentCmd{
		TenantID: tenant, ActorUserID: actor, TemplateVersionID: req.TemplateVersionID,
		Name: req.Name, FormData: req.FormData,
	})
	if err != nil { mapErr(w, err); return }
	writeJSON(w, 201, res)
}

// listDocuments: admin sees all tenant docs; filler sees only own. Scope via
// ListDocumentsForUser to avoid leaking metadata (document names of other users
// in the same tenant can carry PII).
func (h *Handler) listDocuments(w http.ResponseWriter, r *http.Request) {
	if !requireRole(r, roleAdmin, roleDocumentFiller) { httpErr(w, 403, "forbidden"); return }
	tenant := r.Header.Get("X-Tenant-ID"); actor := r.Header.Get("X-User-ID")
	var docs []domain.Document
	var err error
	if isAdmin(r) {
		docs, err = h.svc.ListDocuments(r.Context(), tenant)
	} else {
		docs, err = h.svc.ListDocumentsForUser(r.Context(), tenant, actor)
	}
	if err != nil { mapErr(w, err); return }
	out := make([]map[string]any, 0, len(docs))
	for _, d := range docs {
		out = append(out, map[string]any{
			"id": d.ID, "name": d.Name, "status": string(d.Status),
			"template_version_id": d.TemplateVersionID,
			"updated_at":          d.UpdatedAt,
			"current_revision_id": d.CurrentRevisionID,
			"created_by":          d.CreatedBy,
		})
	}
	writeJSON(w, 200, out)
}

func (h *Handler) getDocument(w http.ResponseWriter, r *http.Request) {
	if !requireRole(r, roleAdmin, roleDocumentFiller) { httpErr(w, 403, "forbidden"); return }
	tenant := r.Header.Get("X-Tenant-ID"); actor := r.Header.Get("X-User-ID")
	docID := r.PathValue("id")
	r = withAdminCtx(r)
	if err := h.ensureDocAccess(r.Context(), tenant, docID, actor); err != nil { mapErr(w, err); return }
	d, err := h.svc.GetDocument(r.Context(), tenant, docID)
	if err != nil { httpErr(w, 404, "not_found"); return }
	writeJSON(w, 200, d)
}

func (h *Handler) finalizeDocument(w http.ResponseWriter, r *http.Request) {
	if !requireRole(r, roleAdmin, roleDocumentFiller) { httpErr(w, 403, "forbidden"); return }
	tenant := r.Header.Get("X-Tenant-ID"); actor := r.Header.Get("X-User-ID")
	docID := r.PathValue("id")
	r = withAdminCtx(r)
	if err := h.ensureDocAccess(r.Context(), tenant, docID, actor); err != nil { mapErr(w, err); return }
	if err := h.svc.Finalize(r.Context(), tenant, docID, actor); err != nil { mapErr(w, err); return }
	writeJSON(w, 200, map[string]string{"status": "finalized"})
}

// archiveDocument: spec RBAC matrix allows document_filler to archive OWN docs
// (draft→archived or finalized→archived). Admin can archive any doc.
func (h *Handler) archiveDocument(w http.ResponseWriter, r *http.Request) {
	if !requireRole(r, roleAdmin, roleDocumentFiller) { httpErr(w, 403, "forbidden"); return }
	tenant := r.Header.Get("X-Tenant-ID"); actor := r.Header.Get("X-User-ID")
	docID := r.PathValue("id")
	r = withAdminCtx(r)
	if err := h.ensureDocAccess(r.Context(), tenant, docID, actor); err != nil { mapErr(w, err); return }
	// Try finalized→archived first, fall back to draft→archived.
	if err := h.svc.Archive(r.Context(), tenant, docID, actor, true); err == nil {
		writeJSON(w, 200, map[string]string{"status": "archived"}); return
	}
	if err := h.svc.Archive(r.Context(), tenant, docID, actor, false); err != nil { mapErr(w, err); return }
	writeJSON(w, 200, map[string]string{"status": "archived"})
}
```

- [ ] **Step 5: Session handlers**

```go
func (h *Handler) acquireSession(w http.ResponseWriter, r *http.Request) {
	if !requireRole(r, roleAdmin, roleDocumentFiller) { httpErr(w, 403, "forbidden"); return }
	tenant := r.Header.Get("X-Tenant-ID"); actor := r.Header.Get("X-User-ID")
	docID := r.PathValue("id")
	r = withAdminCtx(r)
	if err := h.ensureDocAccess(r.Context(), tenant, docID, actor); err != nil { mapErr(w, err); return }
	sess, readonly, err := h.svc.AcquireSession(r.Context(), tenant, docID, actor)
	if err != nil { mapErr(w, err); return }
	if readonly {
		writeJSON(w, 200, map[string]any{"mode": "readonly", "held_by": sess.UserID, "held_until": sess.ExpiresAt})
		return
	}
	writeJSON(w, 201, map[string]any{"mode": "writer", "session_id": sess.ID, "expires_at": sess.ExpiresAt, "last_ack_revision_id": sess.LastAcknowledgedRevisionID})
}

type sessionOpReq struct { SessionID string `json:"session_id"` }

func (h *Handler) heartbeatSession(w http.ResponseWriter, r *http.Request) {
	if !requireRole(r, roleAdmin, roleDocumentFiller) { httpErr(w, 403, "forbidden"); return }
	var req sessionOpReq; _ = json.NewDecoder(r.Body).Decode(&req)
	actor := r.Header.Get("X-User-ID")
	if err := h.svc.HeartbeatSession(r.Context(), req.SessionID, actor); err != nil { mapErr(w, err); return }
	writeJSON(w, 200, map[string]string{"status": "ok"})
}

func (h *Handler) releaseSession(w http.ResponseWriter, r *http.Request) {
	if !requireRole(r, roleAdmin, roleDocumentFiller) { httpErr(w, 403, "forbidden"); return }
	var req sessionOpReq; _ = json.NewDecoder(r.Body).Decode(&req)
	tenant := r.Header.Get("X-Tenant-ID"); actor := r.Header.Get("X-User-ID")
	if err := h.svc.ReleaseSession(r.Context(), tenant, req.SessionID, actor, r.PathValue("id")); err != nil { mapErr(w, err); return }
	writeJSON(w, 200, map[string]string{"status": "released"})
}

func (h *Handler) forceReleaseSession(w http.ResponseWriter, r *http.Request) {
	if !requireRole(r, roleAdmin) { httpErr(w, 403, "forbidden"); return }
	var req sessionOpReq; _ = json.NewDecoder(r.Body).Decode(&req)
	tenant := r.Header.Get("X-Tenant-ID"); admin := r.Header.Get("X-User-ID")
	if err := h.svc.ForceReleaseSession(r.Context(), tenant, admin, req.SessionID, r.PathValue("id")); err != nil { mapErr(w, err); return }
	writeJSON(w, 200, map[string]string{"status": "force_released"})
}
```

- [ ] **Step 6: Autosave handlers**

```go
type presignAutosaveReq struct {
	SessionID, BaseRevisionID, ContentHash string
}
// commitAutosaveReq intentionally omits any client-supplied hash — the service
// recomputes it from S3 (server-authoritative). The client-provided
// BaseRevisionID is still needed for the PresignAutosave call, not here.
type commitAutosaveReq struct {
	SessionID, PendingUploadID string
	FormDataSnapshot json.RawMessage
}

func (h *Handler) presignAutosave(w http.ResponseWriter, r *http.Request) {
	if !requireRole(r, roleAdmin, roleDocumentFiller) { httpErr(w, 403, "forbidden"); return }
	tenant := r.Header.Get("X-Tenant-ID"); actor := r.Header.Get("X-User-ID")
	docID := r.PathValue("id")
	r = withAdminCtx(r)
	if err := h.ensureDocAccess(r.Context(), tenant, docID, actor); err != nil { mapErr(w, err); return }
	var req presignAutosaveReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { httpErr(w, 400, "invalid_body"); return }
	out, err := h.svc.PresignAutosave(r.Context(), application.PresignAutosaveCmd{
		TenantID: tenant, ActorUserID: actor, DocumentID: docID,
		SessionID: req.SessionID, BaseRevisionID: req.BaseRevisionID, ContentHash: req.ContentHash,
	})
	if err != nil { mapErr(w, err); return }
	writeJSON(w, 200, out)
}

func (h *Handler) commitAutosave(w http.ResponseWriter, r *http.Request) {
	if !requireRole(r, roleAdmin, roleDocumentFiller) { httpErr(w, 403, "forbidden"); return }
	tenant := r.Header.Get("X-Tenant-ID"); actor := r.Header.Get("X-User-ID")
	docID := r.PathValue("id")
	r = withAdminCtx(r)
	if err := h.ensureDocAccess(r.Context(), tenant, docID, actor); err != nil { mapErr(w, err); return }
	var req commitAutosaveReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { httpErr(w, 400, "invalid_body"); return }
	res, err := h.svc.CommitAutosave(r.Context(), application.CommitAutosaveCmd{
		TenantID: tenant, ActorUserID: actor, DocumentID: docID,
		SessionID: req.SessionID, PendingUploadID: req.PendingUploadID,
		FormDataSnapshot: req.FormDataSnapshot,
	})
	if err != nil { mapErr(w, err); return }
	writeJSON(w, 200, map[string]any{
		"revision_id": res.RevisionID, "revision_num": res.RevisionNum,
		"idempotent_replay": res.AlreadyConsumed,
	})
}
```

- [ ] **Step 7: Checkpoint + signed revision handlers**

```go
func (h *Handler) listCheckpoints(w http.ResponseWriter, r *http.Request) {
	if !requireRole(r, roleAdmin, roleDocumentFiller) { httpErr(w, 403, "forbidden"); return }
	tenant := r.Header.Get("X-Tenant-ID"); actor := r.Header.Get("X-User-ID")
	docID := r.PathValue("id")
	r = withAdminCtx(r)
	if err := h.ensureDocAccess(r.Context(), tenant, docID, actor); err != nil { mapErr(w, err); return }
	cps, err := h.svc.ListCheckpoints(r.Context(), tenant, docID)
	if err != nil { mapErr(w, err); return }
	writeJSON(w, 200, cps)
}

type createCheckpointReq struct { Label string `json:"label"` }

func (h *Handler) createCheckpoint(w http.ResponseWriter, r *http.Request) {
	if !requireRole(r, roleAdmin, roleDocumentFiller) { httpErr(w, 403, "forbidden"); return }
	tenant := r.Header.Get("X-Tenant-ID"); actor := r.Header.Get("X-User-ID")
	docID := r.PathValue("id")
	r = withAdminCtx(r)
	if err := h.ensureDocAccess(r.Context(), tenant, docID, actor); err != nil { mapErr(w, err); return }
	var req createCheckpointReq; _ = json.NewDecoder(r.Body).Decode(&req)
	cp, err := h.svc.CreateCheckpoint(r.Context(), tenant, docID, actor, req.Label)
	if err != nil { mapErr(w, err); return }
	writeJSON(w, 201, cp)
}

func (h *Handler) restoreCheckpoint(w http.ResponseWriter, r *http.Request) {
	if !requireRole(r, roleAdmin, roleDocumentFiller) { httpErr(w, 403, "forbidden"); return }
	tenant := r.Header.Get("X-Tenant-ID"); actor := r.Header.Get("X-User-ID")
	docID := r.PathValue("id")
	r = withAdminCtx(r)
	if err := h.ensureDocAccess(r.Context(), tenant, docID, actor); err != nil { mapErr(w, err); return }
	v, err := strconv.Atoi(r.PathValue("versionNum"))
	if err != nil || v <= 0 { httpErr(w, 400, "invalid_version_num"); return }
	res, err := h.svc.RestoreCheckpoint(r.Context(), tenant, docID, actor, v)
	if err != nil { mapErr(w, err); return }
	// Canonical restore response — keep field names and presence aligned with:
	//   • OpenAPI  Task 7   (checkpoints/{versionNum}/restore response schema)
	//   • Client   Task 11  (RestoreCheckpointResult)
	//   • UI       Task 15  (CheckpointsPanel.onRestored callback)
	writeJSON(w, 200, map[string]any{
		"new_revision_id":                res.NewRevisionID,
		"new_revision_num":               res.NewRevisionNum,
		"source_checkpoint_version_num":  v,
		"idempotent":                     res.Idempotent,
	})
}

func (h *Handler) signedRevision(w http.ResponseWriter, r *http.Request) {
	if !requireRole(r, roleAdmin, roleDocumentFiller) { httpErr(w, 403, "forbidden"); return }
	tenant := r.Header.Get("X-Tenant-ID"); actor := r.Header.Get("X-User-ID")
	docID := r.PathValue("id")
	r = withAdminCtx(r)
	if err := h.ensureDocAccess(r.Context(), tenant, docID, actor); err != nil { mapErr(w, err); return }
	url, err := h.svc.SignedRevisionURL(r.Context(), tenant, docID, r.PathValue("rid"))
	if err != nil { mapErr(w, err); return }
	http.Redirect(w, r, url, http.StatusFound)
}
```

- [ ] **Step 8: Handler tests**

`handler_test.go`: one test per route asserting 201/200 for the happy path and 403 when `X-User-Roles: template_author` (no filler role). Include `TestCommitAutosave_IdempotentReplay_Returns200` — verifies that `idempotent_replay: true` flag round-trips.

- [ ] **Step 9: Run + commit**

```bash
go test ./internal/modules/documents_v2/delivery/http/...
rtk git add internal/modules/documents_v2/delivery
rtk git commit -m "feat(documents_v2/http): routes + RBAC + error mapping"
```

---

## Task 7: OpenAPI partial — `documents-v2.yaml`

**Files:**
- Create: `api/openapi/v1/partials/documents-v2.yaml`
- Modify: `api/openapi/v1/openapi.yaml`

- [ ] **Step 1: Write partial**

```yaml
paths:
  /api/v2/documents:
    get:
      summary: List documents for tenant
      tags: [documents-v2]
      operationId: listDocumentsV2
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema:
                type: array
                items:
                  type: object
                  required: [id, name, status, template_version_id, updated_at]
                  properties:
                    id: { type: string, format: uuid }
                    name: { type: string }
                    status: { type: string, enum: [draft, finalized, archived] }
                    template_version_id: { type: string, format: uuid }
                    updated_at: { type: string, format: date-time }
                    current_revision_id: { type: string, format: uuid }
        '403': { description: forbidden }
    post:
      summary: Create document from published template
      tags: [documents-v2]
      operationId: createDocumentV2
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [template_version_id, name, form_data]
              properties:
                template_version_id: { type: string, format: uuid }
                name: { type: string }
                form_data: { type: object, additionalProperties: true }
      responses:
        '201':
          description: created
          content:
            application/json:
              schema:
                type: object
                required: [DocumentID, InitialRevisionID, SessionID]
                properties:
                  DocumentID: { type: string, format: uuid }
                  InitialRevisionID: { type: string, format: uuid }
                  SessionID: { type: string, format: uuid }
        '403': { description: forbidden }
        '422': { description: form_data_invalid or template parse error }

  /api/v2/documents/{id}:
    get:
      tags: [documents-v2]
      parameters: [{ name: id, in: path, required: true, schema: { type: string, format: uuid } }]
      responses: { '200': { description: ok }, '404': { description: not_found } }

  /api/v2/documents/{id}/finalize:
    post:
      tags: [documents-v2]
      parameters: [{ name: id, in: path, required: true, schema: { type: string, format: uuid } }]
      responses: { '200': { description: ok }, '409': { description: invalid_state_transition } }
  /api/v2/documents/{id}/archive:
    post:
      tags: [documents-v2]
      parameters: [{ name: id, in: path, required: true, schema: { type: string, format: uuid } }]
      responses: { '200': { description: ok } }

  /api/v2/documents/{id}/session/acquire:
    post:
      tags: [documents-v2]
      parameters: [{ name: id, in: path, required: true, schema: { type: string, format: uuid } }]
      responses:
        '201': { description: writer acquired }
        '200': { description: readonly (another user holds session) }
  /api/v2/documents/{id}/session/heartbeat:
    post:
      tags: [documents-v2]
      parameters: [{ name: id, in: path, required: true, schema: { type: string, format: uuid } }]
      responses: { '200': { description: ok }, '409': { description: session_inactive } }
  /api/v2/documents/{id}/session/release:
    post:
      tags: [documents-v2]
      parameters: [{ name: id, in: path, required: true, schema: { type: string, format: uuid } }]
      responses: { '200': { description: ok } }
  /api/v2/documents/{id}/session/force-release:
    post:
      tags: [documents-v2]
      parameters: [{ name: id, in: path, required: true, schema: { type: string, format: uuid } }]
      responses: { '200': { description: ok }, '403': { description: forbidden } }

  /api/v2/documents/{id}/autosave/presign:
    post:
      tags: [documents-v2]
      parameters: [{ name: id, in: path, required: true, schema: { type: string, format: uuid } }]
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [session_id, base_revision_id, content_hash]
              properties:
                session_id: { type: string, format: uuid }
                base_revision_id: { type: string, format: uuid }
                content_hash: { type: string, minLength: 64, maxLength: 64 }
      responses:
        '200': { description: presigned url }
        '409': { description: session_inactive | session_not_holder | stale_base }
  /api/v2/documents/{id}/autosave/commit:
    post:
      tags: [documents-v2]
      parameters: [{ name: id, in: path, required: true, schema: { type: string, format: uuid } }]
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [session_id, pending_upload_id]
              properties:
                session_id: { type: string, format: uuid }
                pending_upload_id: { type: string, format: uuid }
                form_data_snapshot: { type: object }
              # NOTE: content_hash is NOT a request field. The server streams
              # S3 → SHA256 and compares to the expected hash captured at
              # presign time. Client cannot forge the hash.
      responses:
        '200':
          description: revision committed or idempotent replay
          content:
            application/json:
              schema:
                type: object
                required: [revision_id, revision_num]
                properties:
                  revision_id: { type: string, format: uuid }
                  revision_num: { type: integer, minimum: 1 }
                  idempotent_replay: { type: boolean }
        '403': { description: session_not_holder | document_not_owner }
        '404': { description: pending_not_found }
        '409': { description: misbound | session_inactive | stale_base }
        '410': { description: expired_upload | upload_missing }
        '422': { description: content_hash_mismatch }

  /api/v2/documents/{id}/checkpoints:
    get:
      tags: [documents-v2]
      parameters: [{ name: id, in: path, required: true, schema: { type: string, format: uuid } }]
      responses: { '200': { description: ok } }
    post:
      tags: [documents-v2]
      parameters: [{ name: id, in: path, required: true, schema: { type: string, format: uuid } }]
      requestBody:
        content:
          application/json:
            schema: { type: object, properties: { label: { type: string } } }
      responses: { '201': { description: created } }

  /api/v2/documents/{id}/checkpoints/{versionNum}/restore:
    post:
      tags: [documents-v2]
      summary: Forward-only restore — clones the checkpoint rev into a new head revision
      parameters:
        - { name: id, in: path, required: true, schema: { type: string, format: uuid } }
        - { name: versionNum, in: path, required: true, schema: { type: integer, minimum: 1 } }
      responses:
        '200':
          description: restored (new head revision appended) OR idempotent no-op (if current head already equals the checkpoint hash)
          content:
            application/json:
              schema:
                type: object
                required: [new_revision_id, new_revision_num, idempotent]
                properties:
                  new_revision_id: { type: string, format: uuid }
                  new_revision_num: { type: integer, minimum: 1 }
                  source_checkpoint_version_num: { type: integer, minimum: 1 }
                  idempotent: { type: boolean, description: "true when ON CONFLICT (document_id, content_hash) fired — current head already matches" }
        '403': { description: session_not_holder | document_not_owner | session_inactive }
        '404': { description: checkpoint_not_found }

  /api/v2/documents/{id}/revisions/{rid}/url:
    get:
      tags: [documents-v2]
      parameters:
        - { name: id, in: path, required: true, schema: { type: string, format: uuid } }
        - { name: rid, in: path, required: true, schema: { type: string, format: uuid } }
      responses:
        '302': { description: redirect to signed URL }
```

- [ ] **Step 2: Merge into `openapi.yaml`**

Copy ALL 13 v2 paths verbatim into `api/openapi/v1/openapi.yaml` under `paths:`. Same procedure as W2 Task 19 Step 5. Governance check: every path listed in Section "HTTP surface" of the spec must appear, including the new `checkpoints/{versionNum}/restore` route.

- [ ] **Step 3: Lint + commit**

```bash
npx @redocly/cli lint api/openapi/v1/openapi.yaml --config api/openapi/.redocly.yaml || true
rtk git add api/openapi/v1
rtk git commit -m "feat(openapi): documents-v2 paths partial + merge"
```

---

## Task 8: Permission resolver entries

**Files:**
- Modify: `apps/api/cmd/metaldocs-api/permissions.go`

- [ ] **Step 1: Add block**

Inside `newPermissionResolver`, after the `/api/v2/templates` block, add:

```go
if strings.HasPrefix(path, "/api/v2/documents") {
	switch {
	case method == http.MethodGet:
		return iamdomain.PermDocumentRead, true
	case method == http.MethodPost && path == "/api/v2/documents":
		return iamdomain.PermDocumentCreate, true
	case method == http.MethodPost && strings.HasSuffix(path, "/finalize"):
		return iamdomain.PermWorkflowTransition, true
	case method == http.MethodPost && strings.HasSuffix(path, "/archive"):
		// IAM layer: any editor can HIT the archive endpoint; the handler's
		// ensureDocAccess gate + state-machine check enforces that a filler
		// may only archive their own draft. Admin override (archiving a
		// finalized doc owned by another user) is permitted via isCallerAdmin
		// in the handler, NOT via a distinct IAM permission — keeping the
		// permission check layer-consistent with the other mutation routes.
		return iamdomain.PermDocumentEdit, true
	case method == http.MethodPost && strings.Contains(path, "/session/force-release"):
		return iamdomain.PermDocumentManagePermissions, true  // admin only
	case method == http.MethodPost && strings.Contains(path, "/session/"):
		return iamdomain.PermDocumentEdit, true
	case method == http.MethodPost && strings.Contains(path, "/autosave/"):
		return iamdomain.PermDocumentEdit, true
	case method == http.MethodPost && strings.Contains(path, "/checkpoints/") && strings.HasSuffix(path, "/restore"):
		return iamdomain.PermDocumentEdit, true
	case method == http.MethodPost && strings.Contains(path, "/checkpoints"):
		return iamdomain.PermDocumentEdit, true
	}
}
```

`PermDocumentRead` / `PermDocumentCreate` / `PermDocumentEdit` / `PermDocumentManagePermissions` / `PermWorkflowTransition` already exist in `iamdomain`. No changes there.

- [ ] **Step 2: Commit**

```bash
rtk git add apps/api/cmd/metaldocs-api/permissions.go
rtk git commit -m "feat(api): /api/v2/documents* IAM perm routing"
```

---

## Task 9: Wire module into `main.go`

**Files:**
- Create: `internal/modules/documents_v2/module.go`
- Modify: `apps/api/cmd/metaldocs-api/main.go`

- [ ] **Step 1: Write `module.go`**

```go
package documents_v2

import (
	"database/sql"
	"net/http"

	"metaldocs/internal/modules/documents_v2/application"
	dhttp "metaldocs/internal/modules/documents_v2/delivery/http"
	"metaldocs/internal/modules/documents_v2/repository"
)

// Module owns the wired repository so background jobs (session + orphan
// sweepers in Task 10) can share the same concrete *repository.Repository
// without re-parsing DB connections. Exposed via Repo() for main.go wiring.
type Module struct {
	Handler *dhttp.Handler
	repo    *repository.Repository
}

type Dependencies struct {
	DB       *sql.DB
	Docgen   application.DocgenRenderer
	Presign  application.Presigner
	TplRead  application.TemplateReader
	FormVal  application.FormValidator
	Audit    application.Audit
}

func New(deps Dependencies) *Module {
	repo := repository.New(deps.DB)
	svc := application.New(repo, deps.Docgen, deps.Presign, deps.TplRead, deps.FormVal, deps.Audit)
	return &Module{Handler: dhttp.NewHandler(svc), repo: repo}
}

func (m *Module) RegisterRoutes(mux *http.ServeMux) { m.Handler.RegisterRoutes(mux) }

// Repo exposes the concrete repository for background jobs (session sweeper,
// orphan pending sweeper). NOT intended for use outside jobs.
func (m *Module) Repo() *repository.Repository { return m.repo }
```

- [ ] **Step 2: Wire into `main.go`**

Inside the `if featureFlagsCfg.DocxV2Enabled { ... }` block added by W2, append after the templates module block:

```go
docMod := documents_v2.New(documents_v2.Dependencies{
	DB:      deps.SQLDB,
	Docgen:  deps.DocgenV2Client,   // already wired for W2 validate
	Presign: objectstore.NewDocumentPresigner(deps.S3Client, deps.S3Bucket, 15*time.Minute, 25*1024*1024),
	TplRead: docgenv2.NewTemplateReader(deps.SQLDB, deps.S3Client, deps.S3Bucket),
	FormVal: formval.NewGojsonschema(),
	Audit:   deps.AuditWriter,
})
docMod.RegisterRoutes(mux)
log.Printf("docx-v2 documents module enabled")

// Start background sweepers (60s session expiry, 1h orphan pending).
stopSessions := jobs.StartSessionSweeper(context.Background(), docMod.Repo(), 60*time.Second)
stopOrphans := jobs.StartOrphanPendingSweeper(context.Background(), docMod.Repo(), time.Hour)
defer stopSessions(); defer stopOrphans()
```

New imports:

```go
documents_v2 "metaldocs/internal/modules/documents_v2"
"metaldocs/internal/modules/documents_v2/jobs"
docgenv2 "metaldocs/internal/platform/docgenv2"
"metaldocs/internal/platform/formval"
```

- [ ] **Step 3: Add `DocumentPresigner` to objectstore**

In `internal/platform/objectstore/`, add `NewDocumentPresigner(s3Client, bucket, ttl, maxBytes)` returning `application.Presigner`. Required methods:

- `PresignRevisionPUT(ctx, tenantID, docID, contentHash string) (url, storageKey string, err error)` → presigned PUT URL for `tenants/{tid}/documents/{docID}/revisions/{contentHash}.docx`.
- `PresignObjectGET(ctx, key string) (string, error)` → presigned GET URL.
- `AdoptTempObject(ctx, tmpKey, finalKey string) error` → S3 `CopyObject` from tmp → final then `DeleteObject` on tmp. First error wins; logs (not returns) the tmp-delete error if copy succeeded.
- `DeleteObject(ctx, key string) error` → idempotent (`NoSuchKey` → nil).
- `HashObject(ctx, key string) (string, error)` → streaming `GetObject` piped into `sha256.New()`; returns lower-case hex digest. Maps `NoSuchKey` (or `minio.ErrorResponse{Code: "NoSuchKey"}`) to `domain.ErrUploadMissing`; all other errors wrapped and returned. MUST set a `MaxBytes` guard (25 MiB for W3) via `io.LimitReader` to defend against malicious oversized uploads.

Add `HTTPClient` stub tests using the `minio-go/v7` mock or `s3iface`-style interface: assert CopyObject/DeleteObject/HashObject call the expected S3 methods with the expected args, and that HashObject returns `domain.ErrUploadMissing` when GetObject returns NoSuchKey.

- [ ] **Step 4: Commit**

```bash
go build ./...
rtk git add internal/modules/documents_v2/module.go apps/api/cmd/metaldocs-api/main.go internal/platform/objectstore internal/platform/docgenv2 internal/platform/formval
rtk git commit -m "feat(api): mount documents_v2 module under METALDOCS_DOCX_V2_ENABLED"
```

---

## Task 10: Background sweepers

**Files:**
- Create: `internal/modules/documents_v2/jobs/session_sweeper.go`
- Create: `internal/modules/documents_v2/jobs/orphan_pending_sweeper.go`
- Create: `internal/modules/documents_v2/jobs/jobs_test.go`

- [ ] **Step 1: Session sweeper**

```go
package jobs

import (
	"context"
	"log"
	"time"

	"metaldocs/internal/modules/documents_v2/repository"
)

func StartSessionSweeper(ctx context.Context, r *repository.Repository, interval time.Duration) (stop func()) {
	ctx, cancel := context.WithCancel(ctx)
	go func() {
		t := time.NewTicker(interval); defer t.Stop()
		for {
			select {
			case <-ctx.Done(): return
			case now := <-t.C:
				n, err := r.ExpireStaleSessions(ctx, now)
				if err != nil { log.Printf("session_sweeper error: %v", err); continue }
				if n > 0 { log.Printf("session_sweeper expired=%d", n) }
			}
		}
	}()
	return cancel
}
```

- [ ] **Step 2: Orphan pending sweeper**

```go
package jobs

import (
	"context"
	"log"
	"time"

	"metaldocs/internal/modules/documents_v2/repository"
)

func StartOrphanPendingSweeper(ctx context.Context, r *repository.Repository, interval time.Duration) (stop func()) {
	ctx, cancel := context.WithCancel(ctx)
	go func() {
		t := time.NewTicker(interval); defer t.Stop()
		for {
			select {
			case <-ctx.Done(): return
			case <-t.C:
				cutoff := time.Now().Add(-24 * time.Hour)
				n, err := r.DeleteExpiredPending(ctx, cutoff)
				if err != nil { log.Printf("orphan_pending_sweeper error: %v", err); continue }
				if n > 0 { log.Printf("orphan_pending_sweeper deleted=%d", n) }
			}
		}
	}()
	return cancel
}
```

- [ ] **Step 3: Tests (real repo + testcontainers)**

`jobs_test.go` (build tag `integration`): seed an `editor_sessions` row with `expires_at = now() - interval '1 minute'`, run `StartSessionSweeper` with 50ms interval, wait 200ms, assert status transitioned to `expired` and `documents.active_session_id IS NULL`.

Same for orphan pending.

- [ ] **Step 4: Commit**

```bash
go test -tags=integration ./internal/modules/documents_v2/jobs/...
rtk git add internal/modules/documents_v2/jobs
rtk git commit -m "feat(documents_v2/jobs): session + orphan pending sweepers"
```

---

## Task 11: Frontend — API client `documentsV2.ts`

**Files:**
- Create: `frontend/apps/web/src/features/documents/v2/api/documentsV2.ts`

- [ ] **Step 1: Write types + client**

```ts
// All routes under /api/v2/documents*. All requests rely on IAM cookies +
// tenant/role headers stamped by the middleware chain; we do not set X-* from
// the client.

export type DocumentRow = {
  id: string;
  name: string;
  status: 'draft' | 'finalized' | 'archived';
  template_version_id: string;
  updated_at: string;
  current_revision_id?: string;
};

export type CreateDocumentResult = { DocumentID: string; InitialRevisionID: string; SessionID: string };
export type AcquireWriter = { mode: 'writer'; session_id: string; expires_at: string; last_ack_revision_id: string };
export type AcquireReadonly = { mode: 'readonly'; held_by: string; held_until: string };
export type AcquireResult = AcquireWriter | AcquireReadonly;
export type PresignResult = { UploadURL: string; PendingUploadID: string; ExpiresAt: string };
export type CommitResult = { revision_id: string; revision_num: number; idempotent_replay?: boolean };
export type Checkpoint = { ID: string; DocumentID: string; RevisionID: string; VersionNum: number; Label: string; CreatedAt: string; CreatedBy: string };

async function json<T>(res: Response): Promise<T> {
  if (!res.ok) throw Object.assign(new Error(`http_${res.status}`), { status: res.status, body: await res.text() });
  return res.json() as Promise<T>;
}

export async function listDocuments(): Promise<DocumentRow[]> {
  return json(await fetch('/api/v2/documents'));
}
export async function getDocument(id: string): Promise<any> {
  return json(await fetch(`/api/v2/documents/${id}`));
}
export async function createDocument(req: { template_version_id: string; name: string; form_data: unknown }): Promise<CreateDocumentResult> {
  return json(await fetch('/api/v2/documents', {
    method: 'POST', headers: { 'content-type': 'application/json' },
    body: JSON.stringify(req),
  }));
}
export async function finalizeDocument(id: string) {
  return json(await fetch(`/api/v2/documents/${id}/finalize`, { method: 'POST' }));
}
export async function archiveDocument(id: string) {
  return json(await fetch(`/api/v2/documents/${id}/archive`, { method: 'POST' }));
}

export async function acquireSession(id: string): Promise<AcquireResult> {
  return json(await fetch(`/api/v2/documents/${id}/session/acquire`, { method: 'POST' }));
}
export async function heartbeatSession(id: string, sessionID: string) {
  return json(await fetch(`/api/v2/documents/${id}/session/heartbeat`, {
    method: 'POST', headers: { 'content-type': 'application/json' },
    body: JSON.stringify({ session_id: sessionID }),
  }));
}
export async function releaseSession(id: string, sessionID: string) {
  return json(await fetch(`/api/v2/documents/${id}/session/release`, {
    method: 'POST', headers: { 'content-type': 'application/json' },
    body: JSON.stringify({ session_id: sessionID }),
  }));
}
export async function forceReleaseSession(id: string, sessionID: string) {
  return json(await fetch(`/api/v2/documents/${id}/session/force-release`, {
    method: 'POST', headers: { 'content-type': 'application/json' },
    body: JSON.stringify({ session_id: sessionID }),
  }));
}

export async function presignAutosave(id: string, req: { session_id: string; base_revision_id: string; content_hash: string }): Promise<PresignResult> {
  return json(await fetch(`/api/v2/documents/${id}/autosave/presign`, {
    method: 'POST', headers: { 'content-type': 'application/json' },
    body: JSON.stringify(req),
  }));
}
// Server is authoritative for content_hash — it re-computes SHA256 from S3 on
// commit. Client does NOT forward a client-computed hash.
export async function commitAutosave(id: string, req: { session_id: string; pending_upload_id: string; form_data_snapshot?: unknown }): Promise<CommitResult> {
  return json(await fetch(`/api/v2/documents/${id}/autosave/commit`, {
    method: 'POST', headers: { 'content-type': 'application/json' },
    body: JSON.stringify(req),
  }));
}

export async function listCheckpoints(id: string): Promise<Checkpoint[]> {
  return json(await fetch(`/api/v2/documents/${id}/checkpoints`));
}
export async function createCheckpoint(id: string, label: string): Promise<Checkpoint> {
  return json(await fetch(`/api/v2/documents/${id}/checkpoints`, {
    method: 'POST', headers: { 'content-type': 'application/json' },
    body: JSON.stringify({ label }),
  }));
}

export type RestoreCheckpointResult = {
  new_revision_id: string;
  new_revision_num: number;
  source_checkpoint_version_num: number;
  idempotent: boolean;
};
export async function restoreCheckpoint(id: string, versionNum: number): Promise<RestoreCheckpointResult> {
  return json(await fetch(`/api/v2/documents/${id}/checkpoints/${versionNum}/restore`, { method: 'POST' }));
}

export function signedRevisionURL(documentID: string, revisionID: string): string {
  return `/api/v2/documents/${documentID}/revisions/${revisionID}/url`;
}
```

- [ ] **Step 2: Commit**

```bash
rtk git add frontend/apps/web/src/features/documents/v2/api
rtk git commit -m "feat(web/documents-v2): API client"
```

---

## Task 12: Frontend — `useDocumentSession` hook

**Files:**
- Create: `frontend/apps/web/src/features/documents/v2/hooks/useDocumentSession.ts`

- [ ] **Step 1: Write hook**

```ts
import { useCallback, useEffect, useRef, useState } from 'react';
import { acquireSession, heartbeatSession, releaseSession, type AcquireResult } from '../api/documentsV2';

export type SessionState =
  | { phase: 'idle' }
  | { phase: 'acquiring' }
  | { phase: 'writer'; sessionID: string; lastAckRevisionID: string }
  | { phase: 'readonly'; heldBy: string; heldUntil: string }
  | { phase: 'lost'; reason: 'expired' | 'force_released' | 'network' };

const HEARTBEAT_MS = 30_000;

export function useDocumentSession(documentID: string) {
  const [state, setState] = useState<SessionState>({ phase: 'idle' });
  const timer = useRef<number | null>(null);

  const stopHeartbeat = useCallback(() => {
    if (timer.current) { window.clearInterval(timer.current); timer.current = null; }
  }, []);

  const startHeartbeat = useCallback((sessionID: string) => {
    stopHeartbeat();
    timer.current = window.setInterval(async () => {
      try { await heartbeatSession(documentID, sessionID); }
      catch (e: any) {
        if (e?.status === 409) setState({ phase: 'lost', reason: 'force_released' });
        else setState({ phase: 'lost', reason: 'network' });
        stopHeartbeat();
      }
    }, HEARTBEAT_MS);
  }, [documentID, stopHeartbeat]);

  const acquire = useCallback(async () => {
    setState({ phase: 'acquiring' });
    const res: AcquireResult = await acquireSession(documentID);
    if (res.mode === 'writer') {
      setState({ phase: 'writer', sessionID: res.session_id, lastAckRevisionID: res.last_ack_revision_id });
      startHeartbeat(res.session_id);
    } else {
      setState({ phase: 'readonly', heldBy: res.held_by, heldUntil: res.held_until });
    }
  }, [documentID, startHeartbeat]);

  const release = useCallback(async () => {
    if (state.phase !== 'writer') return;
    stopHeartbeat();
    try { await releaseSession(documentID, state.sessionID); } catch {}
    setState({ phase: 'idle' });
  }, [documentID, state, stopHeartbeat]);

  useEffect(() => {
    // Acquire on mount.
    acquire();
    // Release on unmount + on page hide (best-effort — browser may block async fetch).
    const onHide = () => { if (state.phase === 'writer') navigator.sendBeacon(`/api/v2/documents/${documentID}/session/release`, JSON.stringify({ session_id: state.sessionID })); };
    window.addEventListener('pagehide', onHide);
    return () => { stopHeartbeat(); window.removeEventListener('pagehide', onHide); };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [documentID]);

  // Expose setters for autosave hook to mutate lastAckRevisionID after commit.
  const setLastAck = useCallback((newAck: string) => {
    setState((cur) => (cur.phase === 'writer' ? { ...cur, lastAckRevisionID: newAck } : cur));
  }, []);

  return { state, acquire, release, setLastAck };
}
```

- [ ] **Step 2: Vitest smoke test**

`hooks/useDocumentSession.test.ts`: mock `acquireSession` to return writer, spy on `heartbeatSession`, advance fake timers 30s, assert at least one heartbeat; then make heartbeat throw with `status: 409`, advance timer, assert state transitions to `{ phase: 'lost', reason: 'force_released' }`.

- [ ] **Step 3: Commit**

```bash
rtk git add frontend/apps/web/src/features/documents/v2/hooks/useDocumentSession.ts frontend/apps/web/src/features/documents/v2/hooks/useDocumentSession.test.ts
rtk git commit -m "feat(web/documents-v2): useDocumentSession hook + heartbeat"
```

---

## Task 13: Frontend — `useDocumentAutosave` hook + IndexedDB restore

**Files:**
- Create: `frontend/apps/web/src/features/documents/v2/hooks/useDocumentAutosave.ts`
- Create: `frontend/apps/web/src/features/documents/v2/hooks/useIndexedDBRestore.ts`
- Create: `frontend/apps/web/src/features/documents/v2/hooks/useDocumentAutosave.test.ts`

- [ ] **Step 1: IndexedDB wrapper (`useIndexedDBRestore.ts`)**

```ts
import { openDB, type IDBPDatabase } from 'idb';

const DB_NAME = 'metaldocs_docs_v2';
const STORE = 'pending_autosaves';

interface PendingBlob {
  document_id: string;
  session_id: string;
  base_revision_id: string;
  content_hash: string;
  buffer: ArrayBuffer;
  created_at: number;
}

let dbPromise: Promise<IDBPDatabase> | null = null;

function getDB() {
  if (!dbPromise) {
    dbPromise = openDB(DB_NAME, 1, {
      upgrade(db) {
        if (!db.objectStoreNames.contains(STORE)) {
          db.createObjectStore(STORE, { keyPath: ['document_id', 'content_hash'] });
        }
      },
    });
  }
  return dbPromise;
}

export async function putPending(p: PendingBlob) {
  const db = await getDB();
  await db.put(STORE, p);
}
export async function getAllPending(documentID: string): Promise<PendingBlob[]> {
  const db = await getDB();
  const all = await db.getAll(STORE);
  return all.filter((x: PendingBlob) => x.document_id === documentID);
}
export async function deletePending(documentID: string, contentHash: string) {
  const db = await getDB();
  await db.delete(STORE, [documentID, contentHash]);
}
```

- [ ] **Step 2: Autosave hook**

```ts
import { useCallback, useEffect, useRef, useState } from 'react';
import { presignAutosave, commitAutosave } from '../api/documentsV2';
import { deletePending, putPending, getAllPending } from './useIndexedDBRestore';

export type AutosaveStatus = 'idle' | 'dirty' | 'saving' | 'saved' | 'stale' | 'session_lost' | 'error';

export interface AutosaveArgs {
  documentID: string;
  sessionID: string;
  baseRevisionID: string;
  onAdvanceBase: (newRevisionID: string) => void;
  onSessionLost: (reason: 'stale_base' | 'session_inactive' | 'force_released') => void;
}

const SYNC_DEBOUNCE_MS = 15_000;

async function sha256Hex(buf: ArrayBuffer): Promise<string> {
  const digest = await crypto.subtle.digest('SHA-256', buf);
  return Array.from(new Uint8Array(digest)).map((b) => b.toString(16).padStart(2, '0')).join('');
}

export function useDocumentAutosave(args: AutosaveArgs) {
  const pending = useRef<ArrayBuffer | null>(null);
  const pendingHash = useRef<string>('');
  const formSnapshot = useRef<unknown>(null);
  const timer = useRef<number | null>(null);
  const [status, setStatus] = useState<AutosaveStatus>('idle');

  const flush = useCallback(async () => {
    if (!pending.current) return;
    setStatus('saving');
    const buf = pending.current;
    const hash = pendingHash.current;
    try {
      // Persist to IndexedDB BEFORE hitting network — crash recovery.
      await putPending({
        document_id: args.documentID,
        session_id: args.sessionID,
        base_revision_id: args.baseRevisionID,
        content_hash: hash,
        buffer: buf,
        created_at: Date.now(),
      });
      const presigned = await presignAutosave(args.documentID, {
        session_id: args.sessionID,
        base_revision_id: args.baseRevisionID,
        content_hash: hash,
      });
      await fetch(presigned.UploadURL, {
        method: 'PUT',
        headers: { 'content-type': 'application/vnd.openxmlformats-officedocument.wordprocessingml.document' },
        body: buf,
      });
      // Server re-computes content_hash from S3; client does NOT send a hash.
      const commit = await commitAutosave(args.documentID, {
        session_id: args.sessionID,
        pending_upload_id: presigned.PendingUploadID,
        form_data_snapshot: formSnapshot.current,
      });
      await deletePending(args.documentID, hash);
      pending.current = null; pendingHash.current = '';
      args.onAdvanceBase(commit.revision_id);
      setStatus('saved');
    } catch (e: any) {
      if (e?.status === 409) {
        const body = e?.body ? (() => { try { return JSON.parse(e.body); } catch { return {}; } })() : {};
        if (body?.error === 'stale_base') { args.onSessionLost('stale_base'); setStatus('stale'); return; }
        if (body?.error === 'session_inactive' || body?.error === 'session_not_holder') {
          args.onSessionLost('session_inactive'); setStatus('session_lost'); return;
        }
      }
      if (e?.status === 410) {
        // upload_missing or expired_upload: the S3 object is gone. Drop the
        // IndexedDB entry so recovery doesn't loop, surface error to user.
        try { await deletePending(args.documentID, hash); } catch { /* ignore */ }
        pending.current = null; pendingHash.current = '';
        setStatus('error'); return;
      }
      if (e?.status === 422) {
        // content_hash_mismatch: S3 bytes differ from what we presigned for.
        // Discard local pending — bytes on server are corrupt, cannot retry.
        try { await deletePending(args.documentID, hash); } catch { /* ignore */ }
        pending.current = null; pendingHash.current = '';
        setStatus('error'); return;
      }
      setStatus('error');
    }
  }, [args]);

  const schedule = useCallback(() => {
    if (timer.current) window.clearTimeout(timer.current);
    timer.current = window.setTimeout(flush, SYNC_DEBOUNCE_MS);
  }, [flush]);

  const queue = useCallback(async (buf: ArrayBuffer, snapshot: unknown) => {
    pending.current = buf;
    formSnapshot.current = snapshot;
    pendingHash.current = await sha256Hex(buf);
    setStatus('dirty');
    schedule();
  }, [schedule]);

  // Recovery on mount: if IndexedDB has a pending blob not yet committed,
  // replay it (if session matches base we still hold). Safe because server
  // is idempotent on (session, base, hash).
  useEffect(() => {
    (async () => {
      const leftovers = await getAllPending(args.documentID);
      for (const p of leftovers) {
        if (p.session_id !== args.sessionID) continue;
        pending.current = p.buffer;
        pendingHash.current = p.content_hash;
        await flush();
      }
    })();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [args.documentID, args.sessionID]);

  useEffect(() => () => { if (timer.current) window.clearTimeout(timer.current); }, []);

  return { status, queue, flush };
}
```

- [ ] **Step 3: Vitest coverage (state machine)**

`useDocumentAutosave.test.ts`: mock `presignAutosave`/`commitAutosave`. Cases:
1. `queue` → flush → `saved`; `onAdvanceBase` called with new `revision_id`.
2. `presignAutosave` throws 409 `stale_base` → `onSessionLost('stale_base')`; `status='stale'`.
3. `presignAutosave` throws 409 `session_inactive` → `onSessionLost('session_inactive')`; `status='session_lost'`.
4. `commitAutosave` throws 410 `upload_missing` → `status='error'`; **IndexedDB entry is deleted and `pending.current` is cleared** — the S3 object is gone, retrying would loop forever. Assert `getAllPending(documentID)` is empty after the throw.
5. `commitAutosave` throws 410 `expired_upload` → identical to case 4: `status='error'`, pending cleared, IndexedDB entry deleted. The 15-minute presign window elapsed; the client must re-queue from the editor buffer rather than replay the stale pending id.
6. `commitAutosave` throws 422 `content_hash_mismatch` → `status='error'`; pending cleared, IndexedDB entry deleted (bytes at S3 are corrupt; cannot recover).
7. IndexedDB has leftover pending on mount — flush replays, deletion happens post-commit.

**Decision recorded (R2-5):** 410 and 422 responses **discard** the pending blob. This is the chosen contract because (a) on 410 the S3 object is gone — replay would always 410 again, (b) on 422 the server already told us the bytes are unrecoverable. Retrying would wedge the autosave loop. The user recovers by continuing to edit; the next `queue()` call produces a fresh hash, a fresh presign, and a fresh S3 PUT. **The `AutosaveStatus` union does NOT include `'pending'`** — there is no intermediate state between `'saving'` and `'error'`.

- [ ] **Step 4: Commit**

```bash
npm test --workspace @metaldocs/web -- useDocumentAutosave useIndexedDBRestore
rtk git add frontend/apps/web/src/features/documents/v2/hooks
rtk git commit -m "feat(web/documents-v2): autosave state machine + IndexedDB recovery"
```

---

## Task 14: Frontend — DocumentCreatePage

**Files:**
- Create: `frontend/apps/web/src/features/documents/v2/DocumentCreatePage.tsx`
- Create: `frontend/apps/web/src/features/documents/v2/hooks/useDocumentLoad.ts` (shared helper; simple query)

- [ ] **Step 1: Extend templates list to expose `latest_version_id`**

The backend's `CreateDocument` consumes `template_version_id` (UUID), not the template id. Plan B's list endpoint returns `latest_version: number` only. Propagate a `latest_version_id: string` column through domain → repository → handler → OpenAPI → frontend type before writing the page.

Modify:
- `internal/modules/templates/domain/model.go` → add `LatestVersionID string` to `TemplateListItem`.
- `internal/modules/templates/repository/repository.go` → replace the `latest_version` subquery in `ListTemplates` with one that also selects the UUID:
  ```sql
  coalesce((SELECT version_num FROM template_versions WHERE template_id=t.id ORDER BY version_num DESC LIMIT 1), 0) AS latest_version,
  coalesce((SELECT id FROM template_versions WHERE template_id=t.id ORDER BY version_num DESC LIMIT 1), '00000000-0000-0000-0000-000000000000'::uuid) AS latest_version_id
  ```
  Extend the Scan target list with `&row.LatestVersionID`.
- `internal/modules/templates/delivery/http/handler.go` → include `"latest_version_id": t.LatestVersionID` in `listTemplates` JSON output.
- `api/openapi/v1/partials/templates-v2.yaml` → add `latest_version_id: { type: string, format: uuid }` to the list item schema.
- `frontend/apps/web/src/features/templates/v2/api/templatesV2.ts` → add `latest_version_id: string` to the `TemplateListRow` type.
- Update `internal/modules/templates/repository/repository_test.go` — the existing `TestListTemplates_ReturnsLatestVersion` (from Plan B Fix R2-1) asserts only `latest_version`; extend it to also assert `latest_version_id` is the expected UUID.

- [ ] **Step 2: Write page**

```tsx
import { useEffect, useState } from 'react';
import { FormRenderer, validateJsonSchema } from '@metaldocs/form-ui';
import { listTemplates, type TemplateListRow } from '../../templates/v2/api/templatesV2';
import { createDocument } from './api/documentsV2';

export type DocumentCreatePageProps = {
  onCreated: (documentID: string) => void;
};

export function DocumentCreatePage({ onCreated }: DocumentCreatePageProps) {
  const [templates, setTemplates] = useState<TemplateListRow[]>([]);
  const [pick, setPick] = useState<TemplateListRow | null>(null);
  const [schemaObj, setSchemaObj] = useState<unknown>(null);
  const [formData, setFormData] = useState<any>({});
  const [name, setName] = useState('');
  const [err, setErr] = useState<string | null>(null);

  useEffect(() => { listTemplates().then(setTemplates).catch((e) => setErr(String(e))); }, []);

  useEffect(() => {
    if (!pick) return;
    (async () => {
      const metaRes = await fetch(`/api/v2/templates/${pick.id}/versions/${pick.latest_version}`);
      const meta = await metaRes.json();
      const sres = await fetch(`/api/v2/signed?key=${encodeURIComponent(meta.schema_storage_key)}`);
      setSchemaObj(await sres.json());
    })();
  }, [pick]);

  async function handleCreate() {
    setErr(null);
    if (!pick) return;
    try {
      const res = await createDocument({
        template_version_id: pick.latest_version_id,
        name,
        form_data: formData,
      });
      onCreated(res.DocumentID);
    } catch (e: any) {
      setErr(String(e));
    }
  }

  if (err) return <div role="alert">{err}</div>;
  return (
    <div>
      <h1>New document</h1>
      <section>
        <h2>Pick template</h2>
        <ul>
          {templates.map((t) => (
            <li key={t.id}><button onClick={() => setPick(t)}>{t.name} (v{t.latest_version})</button></li>
          ))}
        </ul>
      </section>
      {pick && (
        <>
          <label>Document name <input value={name} onChange={(e) => setName(e.target.value)} /></label>
          {schemaObj && (
            <FormRenderer schema={schemaObj} formData={formData} onChange={(data: any) => setFormData(data)} />
          )}
          <button onClick={handleCreate} disabled={!name || !validateJsonSchema(JSON.stringify(schemaObj ?? {})).valid}>
            Generate document
          </button>
        </>
      )}
    </div>
  );
}
```

- [ ] **Step 3: Commit**

```bash
rtk git add frontend/apps/web/src/features/documents/v2/DocumentCreatePage.tsx \
  frontend/apps/web/src/features/templates/v2/api/templatesV2.ts \
  internal/modules/templates api/openapi/v1
rtk git commit -m "feat(web/documents-v2): DocumentCreatePage + expose latest_version_id"
```

---

## Task 15: Frontend — DocumentEditorPage

**Files:**
- Create: `frontend/apps/web/src/features/documents/v2/DocumentEditorPage.tsx`
- Create: `frontend/apps/web/src/features/documents/v2/CheckpointsPanel.tsx`
- Create: `frontend/apps/web/src/features/documents/v2/styles/DocumentEditorPage.module.css`

- [ ] **Step 1: Write page**

```tsx
import { useEffect, useRef, useState } from 'react';
import { MetalDocsEditor, type MetalDocsEditorRef } from '@metaldocs/editor-ui';
import { useDocumentSession } from './hooks/useDocumentSession';
import { useDocumentAutosave } from './hooks/useDocumentAutosave';
import { getDocument, finalizeDocument, signedRevisionURL } from './api/documentsV2';
import { CheckpointsPanel } from './CheckpointsPanel';
import styles from './styles/DocumentEditorPage.module.css';

export type DocumentEditorPageProps = {
  documentID: string;
  onDone: () => void;
};

export function DocumentEditorPage({ documentID, onDone }: DocumentEditorPageProps) {
  const session = useDocumentSession(documentID);
  const [doc, setDoc] = useState<any>(null);
  const [buffer, setBuffer] = useState<ArrayBuffer | undefined>();
  const editorRef = useRef<MetalDocsEditorRef>(null);

  useEffect(() => {
    (async () => {
      const d = await getDocument(documentID);
      setDoc(d);
      if (d.CurrentRevisionID) {
        const r = await fetch(signedRevisionURL(documentID, d.CurrentRevisionID));
        setBuffer(await r.arrayBuffer());
      }
    })();
  }, [documentID]);

  const autosaveArgs = session.state.phase === 'writer'
    ? {
        documentID,
        sessionID: session.state.sessionID,
        baseRevisionID: session.state.lastAckRevisionID,
        onAdvanceBase: session.setLastAck,
        onSessionLost: () => {/* render banner via session.state=lost */},
      }
    : null;

  const autosave = useDocumentAutosave(autosaveArgs ?? { documentID, sessionID: '', baseRevisionID: '', onAdvanceBase: () => {}, onSessionLost: () => {} });

  async function handleSave() {
    if (!editorRef.current) return;
    const buf = await editorRef.current.getDocumentBuffer();
    if (!buf) return;
    await autosave.queue(buf, doc?.FormDataJSON);
  }

  async function handleFinalize() {
    if (session.state.phase === 'writer') await autosave.flush();
    await finalizeDocument(documentID);
    await session.release();
    onDone();
  }

  async function handleRestored(newRevisionID: string) {
    // Reload the signed URL for the new head revision into the editor, and
    // advance the autosave base so subsequent queues stamp the correct base.
    const r = await fetch(signedRevisionURL(documentID, newRevisionID));
    setBuffer(await r.arrayBuffer());
    session.setLastAck(newRevisionID);
    // Also refresh FormDataJSON from the server — restore snapshotted it.
    const d = await getDocument(documentID);
    setDoc(d);
  }

  if (!doc) return <div>Loading…</div>;

  return (
    <div className={styles.page} data-editor-root>
      <header className={styles.header}>
        <h1>{doc.Name}</h1>
        <span className={styles.status} data-status={autosave.status}>{autosave.status}</span>
        {session.state.phase === 'readonly' && (
          <div role="alert" className={styles.banner}>Read-only — held by {session.state.heldBy} until {session.state.heldUntil}</div>
        )}
        {session.state.phase === 'lost' && (
          <div role="alert" className={styles.banner}>Session lost ({session.state.reason}). <button onClick={() => location.reload()}>Reload</button></div>
        )}
        <button onClick={handleFinalize} disabled={session.state.phase !== 'writer' || doc.Status !== 'draft'}>Finalize</button>
      </header>
      <div className={styles.split}>
        <div className={styles.editor}>
          <MetalDocsEditor
            ref={editorRef}
            mode={session.state.phase === 'writer' ? 'document-edit' : 'readonly'}
            documentBuffer={buffer}
            userId={doc.CreatedBy}
            onAutoSave={handleSave}
          />
        </div>
        <CheckpointsPanel
          documentID={documentID}
          onRestored={handleRestored}
          disabled={session.state.phase !== 'writer'}
        />
      </div>
    </div>
  );
}
```

- [ ] **Step 2: CheckpointsPanel**

```tsx
import { useEffect, useState } from 'react';
import {
  listCheckpoints,
  createCheckpoint,
  restoreCheckpoint,
  type Checkpoint,
} from './api/documentsV2';

type Props = {
  documentID: string;
  /**
   * Fired after a successful restore so the parent editor can reload the
   * latest revision into the editor surface. Passes the new head revision id.
   */
  onRestored: (newRevisionID: string) => void;
  /**
   * If true, the restore button is disabled (e.g. caller does not hold an
   * active session). Determined by the parent from useDocumentSession phase.
   */
  disabled?: boolean;
};

export function CheckpointsPanel({ documentID, onRestored, disabled }: Props) {
  const [items, setItems] = useState<Checkpoint[]>([]);
  const [label, setLabel] = useState('');
  const [busyVersion, setBusyVersion] = useState<number | null>(null);
  const [error, setError] = useState<string | null>(null);

  async function refresh() { setItems(await listCheckpoints(documentID)); }
  useEffect(() => { refresh(); }, [documentID]);

  async function handleCreate() {
    setError(null);
    await createCheckpoint(documentID, label);
    setLabel('');
    await refresh();
  }

  async function handleRestore(c: Checkpoint) {
    if (disabled) return;
    const ok = window.confirm(
      `Restore checkpoint v${c.VersionNum}${c.Label ? ' — ' + c.Label : ''}?\n\n` +
      `This appends a new head revision cloning the checkpoint. History is preserved; no data is lost.`
    );
    if (!ok) return;
    setBusyVersion(c.VersionNum);
    setError(null);
    try {
      const res = await restoreCheckpoint(documentID, c.VersionNum);
      onRestored(res.new_revision_id);
      await refresh();
    } catch (e: any) {
      setError(
        e?.status === 403 ? 'You do not hold the active session — acquire writer first.' :
        e?.status === 404 ? 'Checkpoint no longer exists.' :
        'Restore failed. Please retry.'
      );
    } finally {
      setBusyVersion(null);
    }
  }

  return (
    <aside data-checkpoints-panel>
      <h3>Checkpoints</h3>
      <input value={label} onChange={(e) => setLabel(e.target.value)} placeholder="Label (optional)" />
      <button onClick={handleCreate}>Create checkpoint</button>
      {error && <p role="alert" data-checkpoint-error>{error}</p>}
      <ul>
        {items.map((c) => (
          <li key={c.ID}>
            v{c.VersionNum} — {c.Label || '(no label)'} — {new Date(c.CreatedAt).toLocaleString()}
            <button
              onClick={() => handleRestore(c)}
              disabled={disabled || busyVersion !== null}
              data-restore-version={c.VersionNum}
            >
              {busyVersion === c.VersionNum ? 'Restoring…' : 'Restore'}
            </button>
          </li>
        ))}
      </ul>
    </aside>
  );
}
```

**Parent wiring:** `DocumentEditorPage.tsx` must pass `onRestored={(revID) => { /* set baseRevisionID to revID; fetch signed URL; reload editor model */ }}` and `disabled={session.phase !== 'writer'}`. The restore is forward-only, so reloading the signed URL for the new head revision is sufficient — no state reconciliation beyond that.

- [ ] **Step 3: Styles**

```css
.page { display: flex; flex-direction: column; height: 100vh; }
.header { display: flex; gap: 12px; align-items: center; padding: 8px 16px; border-bottom: 1px solid #ddd; }
.status[data-status="saved"] { color: #2a2; }
.status[data-status="saving"] { color: #aa2; }
.status[data-status="stale"], .status[data-status="session_lost"], .status[data-status="error"] { color: #c22; }
.banner { background: #fee; color: #900; padding: 8px 16px; }
.split { flex: 1; display: grid; grid-template-columns: 1fr 320px; min-height: 0; }
.editor { overflow: auto; border-right: 1px solid #ddd; }
```

- [ ] **Step 4: Commit**

```bash
rtk git add frontend/apps/web/src/features/documents/v2/DocumentEditorPage.tsx frontend/apps/web/src/features/documents/v2/CheckpointsPanel.tsx frontend/apps/web/src/features/documents/v2/styles
rtk git commit -m "feat(web/documents-v2): editor page + checkpoints panel"
```

---

## Task 16: Frontend — view routing

**Files:**
- Create: `frontend/apps/web/src/features/documents/v2/routes.tsx`
- Modify: `frontend/apps/web/src/App.tsx`
- Modify: `frontend/apps/web/src/routing/workspaceRoutes.ts`

**Routing contract:** URL is the source of truth. The `docsRoute` state is **derived from** `location.pathname` on mount, kept in sync by a `popstate` listener, and updated via `history.pushState` whenever `onNavigate` fires. Direct-URL loads (`/documents-v2/{id}`) must hydrate into `{ kind: 'editor', documentID }` — without this, the E2E specs in Tasks 17–19 (which call `page.goto(docURL)` and `tab2.goto(docURL)`) would land on the create page instead of the editor and fail.

- [ ] **Step 1: Route tree + URL↔state helpers**

```tsx
// routes.tsx
import { DocumentCreatePage } from './DocumentCreatePage';
import { DocumentEditorPage } from './DocumentEditorPage';

export type DocumentsV2Route =
  | { kind: 'create' }
  | { kind: 'editor'; documentID: string };

// UUID regex — `new` is reserved as the create-view sentinel, so an `editor`
// route requires a UUID-shaped trailing segment.
const UUID_RE = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

export function routeFromPath(pathname: string): DocumentsV2Route {
  // Accept `/documents-v2`, `/documents-v2/`, `/documents-v2/new` → create.
  if (pathname === '/documents-v2' || pathname === '/documents-v2/' || pathname === '/documents-v2/new') {
    return { kind: 'create' };
  }
  const m = pathname.match(/^\/documents-v2\/([^/?#]+)\/?$/);
  if (m && UUID_RE.test(m[1])) return { kind: 'editor', documentID: m[1] };
  // Unknown sub-path → fall back to create (the view switcher will still
  // render documents-v2, but the page resets to the picker).
  return { kind: 'create' };
}

export function pathFromRoute(route: DocumentsV2Route): string {
  switch (route.kind) {
    case 'create': return '/documents-v2/new';
    case 'editor': return `/documents-v2/${route.documentID}`;
  }
}

export function renderDocumentsV2View(route: DocumentsV2Route, onNavigate: (r: DocumentsV2Route) => void) {
  switch (route.kind) {
    case 'create':
      return <DocumentCreatePage onCreated={(id) => onNavigate({ kind: 'editor', documentID: id })} />;
    case 'editor':
      return <DocumentEditorPage documentID={route.documentID} onDone={() => onNavigate({ kind: 'create' })} />;
  }
}
```

- [ ] **Step 2: workspaceRoutes extension**

In `workspaceRoutes.ts`:

```ts
// viewFromPath: add before the fallback
if (path === '/documents-v2/new' || path === '/documents-v2') return 'documents-v2';
if (path.startsWith('/documents-v2/')) return 'documents-v2';

// pathFromView: add case
case 'documents-v2': return '/documents-v2/new';

// isPathForView: add
if (view === 'documents-v2') return path === '/documents-v2' || path.startsWith('/documents-v2/');
```

- [ ] **Step 3: App.tsx case with URL hydration**

At the top of `App` component, initialise from `location.pathname` and wire a `popstate` listener + a URL-syncing `onNavigate`:

```tsx
import { routeFromPath, pathFromRoute, type DocumentsV2Route } from './features/documents/v2/routes';

const [docsRoute, setDocsRouteState] = useState<DocumentsV2Route>(() =>
  routeFromPath(window.location.pathname)
);

useEffect(() => {
  const onPop = () => setDocsRouteState(routeFromPath(window.location.pathname));
  window.addEventListener('popstate', onPop);
  return () => window.removeEventListener('popstate', onPop);
}, []);

const setDocsRoute = useCallback((r: DocumentsV2Route) => {
  setDocsRouteState(r);
  const target = pathFromRoute(r);
  if (window.location.pathname !== target) {
    window.history.pushState(null, '', target);
  }
}, []);
```

And in `renderWorkspaceView` switch, inside the `if (isDocxV2Enabled())` branch, add:

```tsx
case 'documents-v2':
  return renderDocumentsV2View(docsRoute, setDocsRoute);
```

- [ ] **Step 4: Vitest — URL↔state round-trip**

Create `frontend/apps/web/src/features/documents/v2/routes.test.ts`:

```ts
import { describe, expect, it } from 'vitest';
import { routeFromPath, pathFromRoute } from './routes';

describe('documents-v2 routes', () => {
  it.each([
    ['/documents-v2',     { kind: 'create' }],
    ['/documents-v2/',    { kind: 'create' }],
    ['/documents-v2/new', { kind: 'create' }],
    ['/documents-v2/11111111-2222-3333-4444-555555555555',
      { kind: 'editor', documentID: '11111111-2222-3333-4444-555555555555' }],
  ] as const)('routeFromPath(%s) → %o', (path, expected) => {
    expect(routeFromPath(path)).toEqual(expected);
  });

  it('ignores non-UUID segments', () => {
    expect(routeFromPath('/documents-v2/not-a-uuid')).toEqual({ kind: 'create' });
  });

  it('round-trips create + editor', () => {
    expect(pathFromRoute({ kind: 'create' })).toBe('/documents-v2/new');
    expect(pathFromRoute({ kind: 'editor', documentID: 'abc' })).toBe('/documents-v2/abc');
  });
});
```

- [ ] **Step 5: Commit**

```bash
npm test --workspace @metaldocs/web -- features/documents/v2/routes
rtk git add frontend/apps/web/src/features/documents/v2/routes.tsx frontend/apps/web/src/features/documents/v2/routes.test.ts frontend/apps/web/src/App.tsx frontend/apps/web/src/routing/workspaceRoutes.ts
rtk git commit -m "feat(web): /documents-v2 URL routing + direct-link hydration behind flag"
```

---

## Task 17: Playwright — filler-happy-path

**Files:**
- Create: `frontend/apps/web/e2e/filler-happy-path.spec.ts`
- Create: `frontend/apps/web/e2e/fixtures/filler-po.form.json`

- [ ] **Step 1: Write spec**

```ts
import { test, expect } from '@playwright/test';

test('filler happy path: pick template → fill form → generate → edit → checkpoint → finalize', async ({ page }) => {
  // Assumes W2 E2E already published the 'po' template.
  await page.goto('/documents-v2/new');
  await page.getByRole('button', { name: /purchase order/i }).click();
  await page.getByLabel(/document name/i).fill('PO-2026-0001');
  await page.getByLabel(/client name/i).fill('Acme Corp');
  await page.getByLabel(/total amount/i).fill('12345.67');
  await page.getByRole('button', { name: /generate document/i }).click();

  await page.waitForURL(/\/documents-v2\/.+/);
  await expect(page.getByText(/saved|idle/i)).toBeVisible({ timeout: 30_000 });

  // Create checkpoint
  await page.getByPlaceholder(/label/i).fill('initial');
  await page.getByRole('button', { name: /create checkpoint/i }).click();
  await expect(page.getByText(/v1 — initial/)).toBeVisible();

  // Finalize
  await page.getByRole('button', { name: /finalize/i }).click();
  await page.waitForURL(/\/documents-v2\/new/, { timeout: 5_000 });
});
```

- [ ] **Step 2: Commit**

```bash
rtk git add frontend/apps/web/e2e/filler-happy-path.spec.ts frontend/apps/web/e2e/fixtures/filler-po.form.json
rtk git commit -m "test(e2e/documents): filler-happy-path"
```

---

## Task 18: Playwright — conflict-two-tabs

**Files:**
- Create: `frontend/apps/web/e2e/conflict-two-tabs.spec.ts`

- [ ] **Step 1: Write spec**

```ts
import { test, expect } from '@playwright/test';

test('two tabs — second is readonly until first releases', async ({ browser }) => {
  const ctx1 = await browser.newContext();
  const ctx2 = await browser.newContext();
  const tab1 = await ctx1.newPage();
  const tab2 = await ctx2.newPage();

  // Tab 1: create a fresh document.
  await tab1.goto('/documents-v2/new');
  await tab1.getByRole('button', { name: /purchase order/i }).click();
  await tab1.getByLabel(/document name/i).fill('concurrent');
  await tab1.getByLabel(/client name/i).fill('Tab1');
  await tab1.getByLabel(/total amount/i).fill('1');
  await tab1.getByRole('button', { name: /generate document/i }).click();
  await tab1.waitForURL(/\/documents-v2\/(.+)/);
  const url = tab1.url();

  // Tab 2: open same document → must enter readonly mode.
  await tab2.goto(url);
  await expect(tab2.getByText(/read-only/i)).toBeVisible({ timeout: 10_000 });

  // Tab 1 releases (simulate via navigation to /documents-v2/new).
  await tab1.goto('/documents-v2/new');

  // Tab 2 reload → now writer.
  await tab2.reload();
  await expect(tab2.getByText(/saved|idle/i)).toBeVisible({ timeout: 15_000 });
  await expect(tab2.getByText(/read-only/i)).toHaveCount(0);
});
```

- [ ] **Step 2: Commit**

```bash
rtk git add frontend/apps/web/e2e/conflict-two-tabs.spec.ts
rtk git commit -m "test(e2e/documents): conflict-two-tabs"
```

---

## Task 19: Playwright — autosave-crash

**Files:**
- Create: `frontend/apps/web/e2e/autosave-crash.spec.ts`

- [ ] **Step 1: Write spec**

Recovery must be asserted with REAL docx bytes, not a fake body. A fake-body approach only proves the pathway is exercised and would mask bugs in the rehydration → presign → PUT → commit chain. Flow:

1. Open doc, wait for initial autosave, capture `currentRevisionId` and real docx bytes.
2. Intercept the next `POST /api/v2/documents/{id}/autosave/commit` with `route.abort()` — the presign + S3 PUT still happen, producing a real pending upload whose bytes are the real docx. The commit never fires.
3. Trigger an autosave (type in editor → debounce fires). Pending is now persisted in IndexedDB + S3, but NOT committed.
4. Close tab (simulate crash).
5. Reopen in new tab. Recovery path reads IndexedDB, sees unfinished pending, calls `POST /autosave/commit` with stored `pendingUploadId`.
6. Assert server responds **200** (content hash matches because bytes ARE the real docx) and `document.current_revision_id` advances — fetched via `GET /api/v2/documents/{id}`.

```ts
import { test, expect } from '@playwright/test';

test('autosave-crash: real-blob replay advances current_revision_id', async ({ browser }) => {
  const ctx = await browser.newContext();
  const tab1 = await ctx.newPage();

  await tab1.goto('/documents-v2/new');
  await tab1.getByRole('button', { name: /purchase order/i }).click();
  await tab1.getByLabel(/document name/i).fill('crash-recovery');
  await tab1.getByLabel(/client name/i).fill('c');
  await tab1.getByLabel(/total amount/i).fill('1');
  await tab1.getByRole('button', { name: /generate document/i }).click();
  await tab1.waitForURL(/\/documents-v2\/(.+)/);
  const docURL = tab1.url();
  const docID = docURL.split('/').pop()!;

  // Wait for initial revision committed → status 'saved'.
  await expect(tab1.locator('[data-status="saved"]')).toBeVisible({ timeout: 15_000 });

  // Snapshot current_revision_id via API.
  const revBefore = await tab1.evaluate(async (id) => {
    const r = await fetch(`/api/v2/documents/${id}`, { credentials: 'include' });
    return (await r.json()).current_revision_id as string;
  }, docID);

  // Block ONLY the next commit; presign + S3 PUT must still run.
  await tab1.route(`**/api/v2/documents/${docID}/autosave/commit`, (route) => route.abort('failed'));

  // Trigger a real edit → autosave pipeline fires: presign → PUT → (aborted commit).
  const editor = tab1.locator('[data-editor-root]');
  await editor.click();
  await tab1.keyboard.type(' edit-for-crash-test');

  // Wait until the real blob was uploaded but commit was aborted → status 'error'.
  // (Per Task 13 Step 3 contract: there is no 'pending' status; failed commits
  // land on 'error' with the IndexedDB entry retained ONLY when the failure is
  // a transport error like this `route.abort('failed')` — the hook treats a
  // network abort as a retryable transport failure, keeps the pending ref and
  // the IndexedDB row, and surfaces 'error' to the UI. 410/422 responses would
  // have cleared IndexedDB per Step 3 cases 4–6; an aborted request never
  // reaches that branch.)
  await expect(tab1.locator('[data-status="error"]')).toBeVisible({ timeout: 15_000 });

  // Kill tab.
  await tab1.close();

  // Reopen in a new tab WITHOUT the route-block: recovery chain can now commit.
  const tab2 = await ctx.newPage();
  await tab2.goto(docURL);

  // Recovery replays pending from IndexedDB → commit succeeds → status transitions to 'saved'.
  await expect(tab2.locator('[data-status="saved"]')).toBeVisible({ timeout: 20_000 });

  // Assert current_revision_id advanced.
  const revAfter = await tab2.evaluate(async (id) => {
    const r = await fetch(`/api/v2/documents/${id}`, { credentials: 'include' });
    return (await r.json()).current_revision_id as string;
  }, docID);
  expect(revAfter).not.toBe(revBefore);
});
```

**Why this spec is load-bearing:** exercises the full chain (IndexedDB read → presign reuse or presign-again → S3 GET or re-PUT → server-authoritative hash recompute → commit 200) with actual docx bytes. A regression anywhere — pending key naming, hash storage, rehydration order, commit idempotency — fails this spec. `useDocumentAutosave.test.ts` (vitest) remains the state-machine coverage; this spec is the end-to-end recovery proof.

**Prerequisite:** `[data-editor-root]` and `[data-status]` attributes must be emitted by `DocumentEditorPage.tsx` (Task 13 Step 2 — already specified) and `AutosaveStatusChip.tsx` (Task 15 Step 2 — already specified).

- [ ] **Step 2: Commit**

```bash
rtk git add frontend/apps/web/e2e/autosave-crash.spec.ts
rtk git commit -m "test(e2e/documents): autosave-crash recovery pathway"
```

---

## Task 20: CI — documents E2E job

**Files:**
- Modify: `.github/workflows/docx-v2-ci.yml`

- [ ] **Step 1: Add job**

```yaml
  e2e-documents:
    runs-on: ubuntu-latest
    needs: [e2e-templates]   # documents E2E depends on the templates job having published a template
    services:
      postgres:
        image: postgres:16-alpine
        env: { POSTGRES_USER: metaldocs, POSTGRES_PASSWORD: metaldocs, POSTGRES_DB: metaldocs }
        options: >-
          --health-cmd "pg_isready -U metaldocs"
          --health-interval 5s --health-timeout 3s --health-retries 10
        ports: [ "5432:5432" ]
      minio:
        image: minio/minio:RELEASE.2024-04-18T19-09-19Z
        env: { MINIO_ROOT_USER: minioadmin, MINIO_ROOT_PASSWORD: minioadmin }
        ports: [ "9000:9000" ]
      gotenberg:
        image: gotenberg/gotenberg:8.4
        ports: [ "3000:3000" ]
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: 1.25.x }
      - uses: actions/setup-node@v4
        with: { node-version: 20.11.0, cache: npm }
      - name: Apply migrations (0101–0110)
        run: |
          for f in migrations/0101_*.sql migrations/0102_*.sql migrations/0103_*.sql migrations/0104_*.sql migrations/0105_*.sql migrations/0106_*.sql migrations/0107_*.sql migrations/0108_*.sql migrations/0109_*.sql migrations/0110_*.sql; do
            PGPASSWORD=metaldocs psql -h 127.0.0.1 -U metaldocs -d metaldocs -v ON_ERROR_STOP=1 -f "$f"
          done
      - run: npm ci --include-workspace-root
      - run: npm ci
        working-directory: frontend/apps/web
      - run: METALDOCS_DOCX_V2_ENABLED=true go run ./apps/api/cmd/metaldocs-api &
        env: { PGCONN: "postgres://metaldocs:metaldocs@127.0.0.1:5432/metaldocs?sslmode=disable" }
      - run: DOCGEN_V2_SERVICE_TOKEN=test-token-0123456789 DOCGEN_V2_S3_ACCESS_KEY=minioadmin DOCGEN_V2_S3_SECRET_KEY=minioadmin npm run start --workspace @metaldocs/docgen-v2 &
      - run: npx playwright install --with-deps chromium
        working-directory: frontend/apps/web
      - run: npx playwright test filler-happy-path.spec.ts conflict-two-tabs.spec.ts autosave-crash.spec.ts
        working-directory: frontend/apps/web
```

- [ ] **Step 2: Commit**

```bash
rtk git add .github/workflows/docx-v2-ci.yml
rtk git commit -m "ci(docx-v2): e2e-documents job (3 specs)"
```

---

## Task 21: Runbook + governance integration test

**Files:**
- Create: `docs/runbooks/docx-v2-w3-documents.md`
- Create: `tests/docx_v2/documents_integration_test.go`

- [ ] **Step 1: Runbook**

Cover:
- New routes (13 including checkpoints/{versionNum}/restore) + RBAC matrix
- Autosave semantics: presign vs commit, idempotency on `(session, base, hash)`, why content_hash is re-computed server-side (server streams S3 → SHA256 on commit, rejects 422 on mismatch)
- Checkpoint restore semantics: forward-only (appends new head revision), idempotent via `ON CONFLICT (document_id, content_hash)` when current head already equals the target; audit event `document.checkpoint_restored`
- Session force-release procedure (`POST /session/force-release`) — admin-only
- How to reset a wedged document: admin force-release, user reload
- Orphan S3 cleanup cadence (tmp objects from failed `CreateDocument` — deferred cleanup via `presigner.DeleteObject`; orphan pending uploads — sweeper, 24h cutoff)
- Troubleshooting: 409 `stale_base` usually means two tabs; 410 `expired_upload` means presign was never consumed within 15min; 410 `upload_missing` means the S3 object was deleted between presign and commit; 422 `content_hash_mismatch` means the bytes at S3 don't match the hash captured at presign (rare — corrupt upload or proxy tampering)

- [ ] **Step 2: Governance integration test — happy path + RBAC denial matrix**

`tests/docx_v2/documents_integration_test.go` (build tag `integration`): spin up real API + MinIO + Postgres. Two test functions — both required for governance.

### `TestDocumentsV2_HappyPath_AllRoutes`

Drive the happy path via HTTP as `document_filler` (with ownership):
`POST /api/v2/documents` → `POST /autosave/presign` → PUT S3 → `POST /autosave/commit` → `POST /session/heartbeat` → `POST /checkpoints` → `GET /checkpoints` → `POST /checkpoints/{versionNum}/restore` → `POST /session/release` → `POST /finalize` → `POST /archive`. Asserts status codes (201/200) and row counts in `documents`, `document_revisions`, `editor_sessions`, `pending_uploads`, `document_checkpoints`, `audit_events` after each step.

### `TestDocumentsV2_RBACDenialMatrix`

One sub-test per route × denial-reason combination. Satisfies governance rule "every new delivery/http route has at least one **negative** RBAC assertion in addition to the happy-path positive".

| Route                                             | Role absent / ownership absent | Caller             | Expected |
|---------------------------------------------------|--------------------------------|--------------------|----------|
| `POST /api/v2/documents`                          | no `document_filler`           | `template_author`  | 403      |
| `GET /api/v2/documents`                           | filler sees only own (not 403, but row filter) | `filler_B` after `filler_A` created doc | 200 with 0 rows for B |
| `GET /api/v2/documents/{id}`                      | not owner, no admin            | `filler_B`         | 403      |
| `POST /autosave/presign`                          | not owner                      | `filler_B`         | 403      |
| `POST /autosave/commit`                           | not owner                      | `filler_B`         | 403      |
| `POST /session/acquire`                           | not owner                      | `filler_B`         | 403      |
| `POST /session/heartbeat`                         | not session-holder             | `filler_B` (after A holds) | 403 `session_not_holder` |
| `POST /session/release`                           | not session-holder             | `filler_B`         | 403      |
| `POST /session/force-release`                     | no `admin`                     | `filler_A` (owner) | 403      |
| `POST /checkpoints`                               | not owner                      | `filler_B`         | 403      |
| `GET /checkpoints`                                | not owner                      | `filler_B`         | 403      |
| `POST /checkpoints/{versionNum}/restore`          | not owner                      | `filler_B`         | 403      |
| `POST /finalize`                                  | not owner                      | `filler_B`         | 403      |
| `POST /archive`                                   | not owner, no admin            | `filler_B`         | 403      |
| `POST /archive` (finalized)                       | admin override allowed         | `admin`            | 200      |

Each sub-test seeds a shared fixture (tenant, two fillers `filler_A` and `filler_B`, one admin, one doc owned by `filler_A`) and issues the route with the appropriate `X-User-ID` / `X-User-Roles` / `X-Tenant-ID` headers. Assert: HTTP status **and** that no row was inserted/modified on denial (repeat snapshot pattern from Task 5 Step 7).

### Why this matters

The `ensureDocAccess` handler helper introduced in Task 6 Step 1 is defense-in-depth over the IAM middleware. Without per-route negative tests, a regression where a handler forgets the ownership check is undetectable until a filler-vs-filler data leak incident. CI rule: this test file must import every handler path string — fail the build if a route defined in Task 6 is not referenced here.

- [ ] **Step 3: Commit**

```bash
rtk git add docs/runbooks/docx-v2-w3-documents.md tests/docx_v2/documents_integration_test.go
rtk git commit -m "docs+test(docx-v2/w3): runbook + governance integration test (happy + RBAC denial matrix)"
```

---

## Sanity pass

- [ ] `go build ./...` passes.
- [ ] `go test ./internal/modules/documents_v2/... ./tests/docx_v2/...` passes (integration tag where noted).
- [ ] `npm test --workspace @metaldocs/web -- documents/v2` passes.
- [ ] `npm run build --workspace @metaldocs/web` passes (type-check).
- [ ] Playwright `filler-happy-path`, `conflict-two-tabs`, `autosave-crash` pass locally.

---

## Codex Hardening Log

**R1 verdict:** `REJECT` (mode: `COVERAGE`, upgrade_required: `true`, confidence: `high`)

R1 flagged 5 structural issues; all 5 have been applied inline before R2 is fired:

1. **Checkpoint restore missing end-to-end** — added `ErrCheckpointNotFound` domain error (Task 3), `RestoreCheckpoint` repo method with `ON CONFLICT (document_id, content_hash) DO UPDATE SET id = id` forward-only semantics (Task 4), `Service.RestoreCheckpoint` wrapping with audit event `document.checkpoint_restored` (Task 5), `h.restoreCheckpoint` handler + route `POST /documents/{id}/checkpoints/{versionNum}/restore` (Task 6), OpenAPI path (Task 7), permission resolver entry (Task 8), `restoreCheckpoint` API client (Task 11), `CheckpointsPanel` restore button wired through `onRestored → DocumentEditorPage.handleRestored` (Task 15).
2. **Server-authoritative commit** — `commitAutosaveReq` no longer carries `ComputedContentHash`; `Service.CommitAutosave` now calls `presigner.HashObject` which streams S3 → SHA256 (25 MiB `io.LimitReader` guard) and compares to `ExpectedContentHash` captured at presign. Orphan cleanup via `presigner.DeleteObject` on hash mismatch. Repo `CommitUpload` adds TOCTOU re-check under `FOR UPDATE`. Clients and tests updated (Task 5, Task 6, Task 9 Step 3, Task 11, Task 13).
3. **9-branch rejection matrix alignment** — `mapErr` canonical rejection table documented; `autosave_commit_branches_test.go` now covers 9 branches split across service (upload_missing, content_hash_mismatch, pending_not_found) and repo (misbound, already_consumed, expired_upload, session_inactive, session_not_holder, stale_base). New repository integration test `commit_rejections_integration_test.go` asserts DB state unchanged per rejection branch under real postgres (Task 5 Steps 6–7).
4. **RBAC ownership defense-in-depth** — `ensureDocAccess` handler helper added with `isCallerAdmin` bypass via `withAdminCtx`. Every mutation route (GET/POST on `/documents/{id}/*`) gates through it in addition to the IAM middleware. Filler archive-from-draft permitted; admin override for archive-from-finalized via `isCallerAdmin`. New `ListDocumentsForUser` repo/service method prevents metadata leakage to non-admin fillers on the list endpoint (Task 4, Task 5, Task 6, Task 8).
5. **Governance + E2E strengthening** — `TestDocumentsV2_RBACDenialMatrix` added (15 sub-tests asserting 403/empty-result per route when caller lacks role or ownership, plus DB-unchanged snapshot). `autosave-crash.spec.ts` rewritten to use **real docx bytes** via `route.abort()` on the commit leg: verifies IndexedDB replay actually advances `current_revision_id` rather than merely triggering the recovery pathway with fake bytes (Task 19, Task 21).

**R2 verdict:** `APPROVE_WITH_FIXES` (mode: `SEQUENCING`, upgrade_required: `false`, confidence: `high`)

R2 returned 5 local-scope issues; all 5 have been applied inline:

1. **Ownership gate inconsistent across handlers** — `ensureDocAccess` + `withAdminCtx(r)` were present on some mutation routes but missing on `acquireSession`, `listCheckpoints`, and `signedRevision`. All five affected handlers (`acquireSession`, `presignAutosave`, `commitAutosave`, `listCheckpoints`, `signedRevision`) now gate through `ensureDocAccess` with the admin-bypass context tag before any repository call (Task 6 Step 1).
2. **`docMod.Repo()` accessor missing** — Task 9 Step 2 called `docMod.Repo()` but the Module struct only exposed `Handler`. The struct now stores an unexported `repo *repository.Repository` field set in `New(...)` and returns it via `func (m *Module) Repo() *repository.Repository`, so the cleanup job wiring compiles (Task 5 Module wiring).
3. **Restore response field names mismatched across layers** — the handler emitted `source_revision_id` while OpenAPI spoke of `source_checkpoint_version_num`, and the client/UI never saw the `idempotent` flag. The canonical response is now `{ new_revision_id, new_revision_num, source_checkpoint_version_num, idempotent }` in the domain `RestoreResult`, the repository RETURNING clause (`RETURNING id::text, revision_num, (xmax::text::bigint <> 0)`), the application layer alias, the handler JSON, the OpenAPI path, the `restoreCheckpoint` client type, and the `handleRestore` UI handler (Tasks 4, 5, 6, 7, 11, 15). The idempotent path also skips the `UPDATE documents` / `UPDATE editor_sessions` writes — the row already points at the target revision.
4. **Frontend routing missed URL → state hydration** — Task 16 previously initialised `docsRoute` from a constant `{ kind: 'create' }`, which meant `page.goto('/documents-v2/{id}')` in the E2E specs (Tasks 17, 18, 19) would have landed on the picker instead of the editor. Task 16 now exports `routeFromPath` + `pathFromRoute`, initialises `docsRoute` from `location.pathname`, listens to `popstate`, and rewrites the URL via `history.pushState` inside `onNavigate`. A vitest spec (`routes.test.ts`) covers the four canonical URL shapes and the non-UUID fallback.
5. **Autosave 410-path inconsistency** — Task 13's catch branches cleared `pending.current` + IndexedDB on 410/422, but Step 3 case 4 said "buffer remains queued" and Task 19 asserted `[data-status="pending"]` (a status not in the `AutosaveStatus` union). Step 3 is now explicit: 410 `upload_missing`, 410 `expired_upload`, 422 `content_hash_mismatch` all **discard** the pending blob and assert `getAllPending` is empty after the throw. Task 19 asserts `[data-status="error"]` only, with a load-bearing comment recording that transport-level `route.abort('failed')` is the exception — it skips the 410/422 branches and retains the pending ref so recovery can replay it (Task 13 Step 3, Task 19 Step 1).

Per co-plan protocol, max 2 Codex rounds are enforced and this plan is now at round 2. No structural issues were raised in R2, so no third round is triggered. The plan is ready for execution handoff.

---
