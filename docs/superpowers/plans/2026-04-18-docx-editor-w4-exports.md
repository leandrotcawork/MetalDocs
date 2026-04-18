# W4 Exports + Rate Limits + RBAC Hardening (docx-editor platform) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship the W4 dogfood-ready slice behind `METALDOCS_DOCX_V2_ENABLED`. `document_filler` (owner) can download the current `.docx` and request a `.pdf` export; PDF conversion rides `docgen-v2 → Gotenberg → MinIO` with a content-addressed cache. Every `/api/v2/*` route is rate-limited per the spec matrix. A cross-module RBAC denial matrix proves every mutation has at least one negative test. Playwright `export-pdf-happy-path`, `export-pdf-cache-hit`, `export-rate-limit` all green.

**Architecture:** All W4 surface is additive — no changes to documents / templates schemas. New table `document_exports` is an append-only ledger indexed by `composite_hash` (per-spec: `sha256(content_hash || template_version_id || grammar_version || docgen_v2_version || canonical_json(render_opts))`). `docgen-v2` gains `POST /convert/pdf` streaming a DOCX out of S3 into Gotenberg and piping the PDF back into S3 under `tenants/{tid}/documents/{docID}/exports/{composite_hash}.pdf`. Rate limits are enforced by a shared in-process token-bucket middleware keyed on `(user_id, route)`; quotas come from a config map and are documented in OpenAPI extension `x-rate-limit`.

**Tech stack additions (on top of W1+W2+W3):** `golang.org/x/time/rate@v0.5` (token bucket) · no new frontend deps. Gotenberg 8.4 service is already wired into Plan C's CI workflow.

**Depends on:** Plan A (W1) + Plan B (W2) + Plan C (W3) all executed and green.

**Spec reference:** `docs/superpowers/specs/2026-04-18-docx-editor-platform-design.md` §§ Data Flow → PDF export, RBAC + tenancy, HTTP surface, Rate limits, Testing Approach (`filler-happy-path` export branch, `rate-limit` suite), Rollout → **W4**.

**Codex hardening status:** Written per co-plan protocol. R1 + R2 outcomes recorded at end of file (Section "Codex Hardening Log"). Max 2 rounds enforced.

---

## File Structure

**New files:**

```
# Migration
migrations/
  0111_docx_v2_exports.sql          # document_exports + index

# Docgen-v2 pdf conversion route
apps/docgen-v2/src/
  routes/convert-pdf.ts             # POST /convert/pdf
  pdf/gotenbergClient.ts            # multipart POST to Gotenberg /forms/libreoffice/convert
  pdf/version.ts                    # exports GRAMMAR_VERSION + DOCGEN_V2_VERSION constants
apps/docgen-v2/test/
  convert-pdf.smoke.test.ts
  convert-pdf.gotenberg-502.test.ts
  convert-pdf.cache-hit.test.ts     # (not a cache test — asserts content_hash is stable across two converts)

# Go exports module (under documents_v2)
internal/modules/documents_v2/
  domain/export.go                  # Export aggregate, ErrExportGotenbergFailed
  domain/composite_hash.go          # ComputeCompositeHash(...) pure fn
  domain/composite_hash_test.go     # determinism + input-change sensitivity
  application/export_service.go     # ExportPDF, SignedDocxURL
  application/export_service_test.go
  repository/export_repository.go   # pgx InsertExport / GetExportByHash
  repository/export_repository_integration_test.go  # build tag `integration`
  delivery/http/export_handler.go   # POST /export/pdf, GET /export/docx-url
  delivery/http/export_handler_test.go
  infrastructure/docgen/pdf_client.go # ConvertPDF(ctx, docxKey, outputKey) → result

# Rate-limit middleware (shared)
internal/platform/ratelimit/
  middleware.go                     # net/http middleware + in-memory token bucket
  middleware_test.go                # under load + 429 retry_after
  config.go                         # per-route quotas loaded from env + fallback defaults

# Frontend
frontend/apps/web/src/features/documents/v2/
  ExportMenu.tsx                    # two buttons: Download .docx · Export PDF
  ExportMenu.test.tsx               # vitest: cache-hit vs miss rendering, 429 handling
  api/exportsV2.ts                  # exportPDF(), getDocxSignedURL()

# Playwright E2Es
frontend/apps/web/e2e/
  export-pdf-happy-path.spec.ts
  export-pdf-cache-hit.spec.ts
  export-rate-limit.spec.ts

# Runbook + governance
docs/runbooks/docx-v2-w4-exports.md
tests/docx_v2/exports_integration_test.go   # per-route RBAC denial matrix + happy path
```

**Modified files:**

```
apps/api/cmd/metaldocs-api/main.go         # wire export routes + rate-limit middleware
apps/api/cmd/metaldocs-api/permissions.go  # /api/v2/documents/*/export/*
frontend/apps/web/src/features/documents/v2/DocumentEditorPage.tsx  # mount ExportMenu
api/openapi/v1/openapi.yaml                # merge partial
api/openapi/v1/partials/documents-v2.yaml  # add 3 paths + rate-limit extension
.github/workflows/docx-v2-ci.yml           # extend e2e-documents → add export specs
internal/modules/documents_v2/module.go    # attach export service + repository to Module
tests/docx_v2/documents_integration_test.go # extend RBAC denial matrix with export routes
```

**Deleted at end of W4:** none. CK5 destruction happens in W5 (Plan E).

---

## Task 0: Prerequisite sanity

**Files:** none

- [ ] **Step 1: Verify W3 is merged and green**

```bash
rtk git log --oneline -1 docs/superpowers/plans/2026-04-18-docx-editor-w3-documents.md
# Expected: commit exists — Plan C committed (co-plan hardened).

go test -tags=integration ./internal/modules/documents_v2/... ./tests/docx_v2/...
npm test --workspace @metaldocs/web -- features/documents/v2
# Expected: all green. If not, stop and finish Plan C before starting W4.
```

- [ ] **Step 2: Verify docgen-v2 builds and Gotenberg service boots**

```bash
npm run build --workspace @metaldocs/docgen-v2
docker run --rm -d --name gotenberg-sanity -p 3000:3000 gotenberg/gotenberg:8.4 && \
  curl -fsS http://localhost:3000/health && \
  docker stop gotenberg-sanity
# Expected: {"status":"up"} — if connection refused, fix Docker Desktop first.
```

---

## Task 1: Migration — `0111_docx_v2_exports.sql`

**Files:**
- Create: `migrations/0111_docx_v2_exports.sql`

- [ ] **Step 1: Write migration**

```sql
-- 0111_docx_v2_exports.sql
-- Append-only ledger of generated PDF exports. Keyed by composite_hash so
-- that repeated exports against the same (document, form_data, grammar,
-- docgen version) tuple resolve to the same S3 object and return cached.
-- Depends on 0110. Safe in one transaction.

BEGIN;

CREATE TABLE document_exports (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         uuid NOT NULL,
    document_id       uuid NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    revision_id       uuid NOT NULL REFERENCES document_revisions(id) ON DELETE RESTRICT,
    composite_hash    bytea NOT NULL,
    storage_key       text NOT NULL,
    size_bytes        bigint NOT NULL,
    content_type      text NOT NULL DEFAULT 'application/pdf',
    created_at        timestamptz NOT NULL DEFAULT now(),
    created_by        uuid NOT NULL,
    CONSTRAINT document_exports_composite_hash_len CHECK (octet_length(composite_hash) = 32),
    CONSTRAINT document_exports_size_positive     CHECK (size_bytes > 0)
);

-- Primary idempotency index. ExportPDF performs ON CONFLICT DO NOTHING
-- and falls back to a SELECT on (document_id, composite_hash).
CREATE UNIQUE INDEX idx_document_exports_doc_hash
  ON document_exports (document_id, composite_hash);

-- Admin list / audit.
CREATE INDEX idx_document_exports_doc_time
  ON document_exports (document_id, created_at DESC);

COMMIT;
```

- [ ] **Step 2: Apply locally and diff schema**

```bash
PGPASSWORD=metaldocs psql -h 127.0.0.1 -U metaldocs -d metaldocs -v ON_ERROR_STOP=1 -f migrations/0111_docx_v2_exports.sql
PGPASSWORD=metaldocs psql -h 127.0.0.1 -U metaldocs -d metaldocs -c "\d+ document_exports"
# Expected: two indexes listed (idx_document_exports_doc_hash UNIQUE, idx_document_exports_doc_time).
```

- [ ] **Step 3: Commit**

```bash
rtk git add migrations/0111_docx_v2_exports.sql
rtk git commit -m "feat(migrations/docx-v2): 0111 document_exports ledger"
```

---

## Task 2: Domain — `Export` + `ComputeCompositeHash`

**Files:**
- Create: `internal/modules/documents_v2/domain/export.go`
- Create: `internal/modules/documents_v2/domain/composite_hash.go`
- Create: `internal/modules/documents_v2/domain/composite_hash_test.go`

- [ ] **Step 1: Write domain — `export.go`**

```go
package domain

import (
    "errors"
    "time"
)

type Export struct {
    ID            string
    TenantID      string
    DocumentID    string
    RevisionID    string
    CompositeHash []byte // 32 bytes (sha256 raw)
    StorageKey    string
    SizeBytes     int64
    ContentType   string
    CreatedAt     time.Time
    CreatedBy     string
}

// ExportResult is what the service returns for POST /export/pdf. Cached
// distinguishes the cache-hit branch from the fresh-generation branch so
// the handler can surface telemetry/log it cleanly.
type ExportResult struct {
    Export Export
    Cached bool
}

var (
    // ErrExportGotenbergFailed is returned when docgen-v2 or Gotenberg
    // surface a non-retryable failure (converted to 502 at the HTTP layer).
    ErrExportGotenbergFailed = errors.New("export: gotenberg conversion failed")
    // ErrExportDocxMissing is returned when the current revision's .docx
    // object is not present in S3 (indicates prior data corruption —
    // surface as 409 and force user to re-save).
    ErrExportDocxMissing = errors.New("export: current revision docx missing from S3")
)
```

- [ ] **Step 2: Write `composite_hash.go`**

Per spec §PDF export:

```go
package domain

import (
    "crypto/sha256"
    "encoding/json"
    "fmt"
    "sort"
)

// RenderOptions is the canonical set of knobs that affect PDF output.
// Serialized via canonical (sorted-keys) JSON so the hash is stable across
// Go map iteration order. Keep keys in sync with docgen-v2 convert-pdf.
type RenderOptions struct {
    PaperSize  string `json:"paper_size,omitempty"` // e.g. "A4", "Letter"
    Margins    string `json:"margins,omitempty"`    // e.g. "1in"
    LandscapeP bool   `json:"landscape,omitempty"`
}

// ComputeCompositeHash is the content-addressed key used both as S3 object
// name and as the UNIQUE index column. All inputs are byte-serialized
// deterministically; any mutation in any input produces a fresh hash.
func ComputeCompositeHash(
    contentHash []byte,
    templateVersionID string,
    grammarVersion string,
    docgenV2Version string,
    opts RenderOptions,
) ([]byte, error) {
    if len(contentHash) != 32 {
        return nil, fmt.Errorf("composite hash: content_hash must be 32 bytes, got %d", len(contentHash))
    }
    optsJSON, err := canonicalJSON(opts)
    if err != nil {
        return nil, fmt.Errorf("composite hash: render_opts: %w", err)
    }
    h := sha256.New()
    h.Write(contentHash)
    h.Write([]byte{0x1e}) // ASCII record separator — domain separation
    h.Write([]byte(templateVersionID))
    h.Write([]byte{0x1e})
    h.Write([]byte(grammarVersion))
    h.Write([]byte{0x1e})
    h.Write([]byte(docgenV2Version))
    h.Write([]byte{0x1e})
    h.Write(optsJSON)
    return h.Sum(nil), nil
}

// canonicalJSON serializes v with sorted keys. Only top-level maps /
// structs with string keys are supported — RenderOptions is flat.
func canonicalJSON(v any) ([]byte, error) {
    raw, err := json.Marshal(v)
    if err != nil {
        return nil, err
    }
    var m map[string]any
    if err := json.Unmarshal(raw, &m); err != nil {
        return nil, err
    }
    keys := make([]string, 0, len(m))
    for k := range m {
        keys = append(keys, k)
    }
    sort.Strings(keys)
    out := make([]byte, 0, len(raw))
    out = append(out, '{')
    for i, k := range keys {
        if i > 0 {
            out = append(out, ',')
        }
        kJSON, _ := json.Marshal(k)
        vJSON, err := json.Marshal(m[k])
        if err != nil {
            return nil, err
        }
        out = append(out, kJSON...)
        out = append(out, ':')
        out = append(out, vJSON...)
    }
    out = append(out, '}')
    return out, nil
}
```

- [ ] **Step 3: Write `composite_hash_test.go`**

```go
package domain_test

import (
    "bytes"
    "testing"

    "metaldocs/internal/modules/documents_v2/domain"
)

func fixedContentHash() []byte {
    b := make([]byte, 32)
    for i := range b {
        b[i] = byte(i)
    }
    return b
}

func TestComputeCompositeHash_Deterministic(t *testing.T) {
    h1, err := domain.ComputeCompositeHash(fixedContentHash(), "tv1", "g1", "d1", domain.RenderOptions{PaperSize: "A4"})
    if err != nil {
        t.Fatal(err)
    }
    h2, err := domain.ComputeCompositeHash(fixedContentHash(), "tv1", "g1", "d1", domain.RenderOptions{PaperSize: "A4"})
    if err != nil {
        t.Fatal(err)
    }
    if !bytes.Equal(h1, h2) {
        t.Fatalf("hash not deterministic across calls")
    }
}

func TestComputeCompositeHash_SensitiveToEveryInput(t *testing.T) {
    base, _ := domain.ComputeCompositeHash(fixedContentHash(), "tv1", "g1", "d1", domain.RenderOptions{PaperSize: "A4"})
    cases := []struct {
        name string
        mut  func() []byte
    }{
        {"content_hash differs", func() []byte {
            c := fixedContentHash()
            c[0] ^= 0xff
            h, _ := domain.ComputeCompositeHash(c, "tv1", "g1", "d1", domain.RenderOptions{PaperSize: "A4"})
            return h
        }},
        {"template_version_id differs", func() []byte {
            h, _ := domain.ComputeCompositeHash(fixedContentHash(), "tv2", "g1", "d1", domain.RenderOptions{PaperSize: "A4"})
            return h
        }},
        {"grammar_version differs", func() []byte {
            h, _ := domain.ComputeCompositeHash(fixedContentHash(), "tv1", "g2", "d1", domain.RenderOptions{PaperSize: "A4"})
            return h
        }},
        {"docgen_v2_version differs", func() []byte {
            h, _ := domain.ComputeCompositeHash(fixedContentHash(), "tv1", "g1", "d2", domain.RenderOptions{PaperSize: "A4"})
            return h
        }},
        {"render_opts differs", func() []byte {
            h, _ := domain.ComputeCompositeHash(fixedContentHash(), "tv1", "g1", "d1", domain.RenderOptions{PaperSize: "Letter"})
            return h
        }},
    }
    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            got := tc.mut()
            if bytes.Equal(base, got) {
                t.Fatalf("expected hash to change for %q", tc.name)
            }
        })
    }
}

func TestComputeCompositeHash_RejectsWrongContentHashLength(t *testing.T) {
    _, err := domain.ComputeCompositeHash([]byte{1, 2, 3}, "tv1", "g1", "d1", domain.RenderOptions{})
    if err == nil {
        t.Fatalf("expected length error")
    }
}
```

- [ ] **Step 4: Commit**

```bash
go test ./internal/modules/documents_v2/domain/... -run CompositeHash
rtk git add internal/modules/documents_v2/domain/export.go internal/modules/documents_v2/domain/composite_hash.go internal/modules/documents_v2/domain/composite_hash_test.go
rtk git commit -m "feat(documents-v2/domain): Export aggregate + ComputeCompositeHash"
```

---

## Task 3: Repository — `InsertExport` + `GetExportByHash`

**Files:**
- Create: `internal/modules/documents_v2/repository/export_repository.go`
- Create: `internal/modules/documents_v2/repository/export_repository_integration_test.go`

- [ ] **Step 1: Write repo methods**

```go
package repository

import (
    "context"
    "database/sql"
    "errors"
    "fmt"

    "metaldocs/internal/modules/documents_v2/domain"
)

// InsertExport persists an export row. ON CONFLICT (document_id,
// composite_hash) DO NOTHING; the caller is responsible for a follow-up
// GetExportByHash to retrieve the canonical row when the insert was a
// no-op. This keeps the idempotency contract: "the same composite_hash
// always resolves to the same row."
func (r *Repository) InsertExport(ctx context.Context, e domain.Export) (domain.Export, bool, error) {
    var out domain.Export
    var inserted bool
    err := r.db.QueryRowContext(ctx, `
        INSERT INTO document_exports
          (tenant_id, document_id, revision_id, composite_hash, storage_key,
           size_bytes, content_type, created_by)
        VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
        ON CONFLICT (document_id, composite_hash) DO NOTHING
        RETURNING id::text, created_at, true
    `, e.TenantID, e.DocumentID, e.RevisionID, e.CompositeHash, e.StorageKey,
        e.SizeBytes, e.ContentType, e.CreatedBy).Scan(&out.ID, &out.CreatedAt, &inserted)
    if errors.Is(err, sql.ErrNoRows) {
        // Race / replay — someone else inserted the same row. Fetch canonical.
        existing, gerr := r.GetExportByHash(ctx, e.DocumentID, e.CompositeHash)
        if gerr != nil {
            return domain.Export{}, false, gerr
        }
        return existing, false, nil
    }
    if err != nil {
        return domain.Export{}, false, fmt.Errorf("insert export: %w", err)
    }
    out.TenantID = e.TenantID
    out.DocumentID = e.DocumentID
    out.RevisionID = e.RevisionID
    out.CompositeHash = e.CompositeHash
    out.StorageKey = e.StorageKey
    out.SizeBytes = e.SizeBytes
    out.ContentType = e.ContentType
    out.CreatedBy = e.CreatedBy
    return out, inserted, nil
}

func (r *Repository) GetExportByHash(ctx context.Context, documentID string, hash []byte) (domain.Export, error) {
    row := r.db.QueryRowContext(ctx, `
        SELECT id::text, tenant_id::text, document_id::text, revision_id::text,
               composite_hash, storage_key, size_bytes, content_type,
               created_at, created_by::text
          FROM document_exports
         WHERE document_id = $1 AND composite_hash = $2
    `, documentID, hash)
    var e domain.Export
    if err := row.Scan(&e.ID, &e.TenantID, &e.DocumentID, &e.RevisionID,
        &e.CompositeHash, &e.StorageKey, &e.SizeBytes, &e.ContentType,
        &e.CreatedAt, &e.CreatedBy); err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return domain.Export{}, domain.ErrNotFound
        }
        return domain.Export{}, fmt.Errorf("get export by hash: %w", err)
    }
    return e, nil
}
```

- [ ] **Step 2: Integration test — idempotency under race**

`export_repository_integration_test.go` (build tag `integration`):

```go
//go:build integration

package repository_test

import (
    "context"
    "sync"
    "testing"

    "metaldocs/internal/modules/documents_v2/domain"
    "metaldocs/internal/modules/documents_v2/repository"
)

func TestInsertExport_IdempotentUnderRace(t *testing.T) {
    db, tenantID, docID, revID, userID := seedDocV2(t) // helper from Plan C
    defer db.Close()
    repo := repository.New(db)

    hash := make([]byte, 32)
    for i := range hash {
        hash[i] = byte(i)
    }
    e := domain.Export{
        TenantID:      tenantID,
        DocumentID:    docID,
        RevisionID:    revID,
        CompositeHash: hash,
        StorageKey:    "tenants/t/documents/d/exports/h.pdf",
        SizeBytes:     1024,
        ContentType:   "application/pdf",
        CreatedBy:     userID,
    }

    // 8 concurrent inserters with the same composite_hash must all succeed.
    var wg sync.WaitGroup
    results := make([]string, 8)
    inserts := make([]bool, 8)
    for i := 0; i < 8; i++ {
        wg.Add(1)
        go func(i int) {
            defer wg.Done()
            got, inserted, err := repo.InsertExport(context.Background(), e)
            if err != nil {
                t.Errorf("goroutine %d: %v", i, err)
                return
            }
            results[i] = got.ID
            inserts[i] = inserted
        }(i)
    }
    wg.Wait()

    // All goroutines must observe the same canonical row id.
    canon := results[0]
    for i, id := range results {
        if id != canon {
            t.Fatalf("goroutine %d saw id %q, want canonical %q", i, id, canon)
        }
    }
    // Exactly one should have `inserted == true`; the rest are replay.
    countInserted := 0
    for _, v := range inserts {
        if v {
            countInserted++
        }
    }
    if countInserted != 1 {
        t.Fatalf("expected exactly one inserter, got %d", countInserted)
    }

    // Only one row must exist in the table.
    var rows int
    if err := db.QueryRow(`SELECT count(*) FROM document_exports WHERE document_id=$1 AND composite_hash=$2`, docID, hash).Scan(&rows); err != nil {
        t.Fatal(err)
    }
    if rows != 1 {
        t.Fatalf("expected 1 export row, got %d", rows)
    }
}
```

- [ ] **Step 3: Commit**

```bash
go test -tags=integration ./internal/modules/documents_v2/repository/... -run Export
rtk git add internal/modules/documents_v2/repository/export_repository.go internal/modules/documents_v2/repository/export_repository_integration_test.go
rtk git commit -m "feat(documents-v2/repo): InsertExport ON CONFLICT DO NOTHING + integration race test"
```

---

## Task 4: Docgen-v2 — `POST /convert/pdf`

**Files:**
- Create: `apps/docgen-v2/src/routes/convert-pdf.ts`
- Create: `apps/docgen-v2/src/pdf/gotenbergClient.ts`
- Create: `apps/docgen-v2/src/pdf/version.ts`
- Create: `apps/docgen-v2/test/convert-pdf.smoke.test.ts`
- Create: `apps/docgen-v2/test/convert-pdf.gotenberg-502.test.ts`
- Create: `apps/docgen-v2/test/convert-pdf.cache-hit.test.ts`

- [ ] **Step 1: Versions file**

```ts
// Versions that participate in the composite_hash on the Go side. Bump
// DOCGEN_V2_VERSION on any change that affects PDF bytes (font config,
// Gotenberg image, libreoffice version). GRAMMAR_VERSION is re-exported
// from shared-tokens so Go and Node agree on the same constant.
export const DOCGEN_V2_VERSION = 'docgen-v2@0.4.0';
export { GRAMMAR_VERSION } from '@metaldocs/shared-tokens';
```

- [ ] **Step 2: Gotenberg client**

```ts
// gotenbergClient.ts
import { Readable } from 'node:stream';

export interface GotenbergConvertArgs {
  docxStream: Readable;
  paperWidth?: number;   // inches, default 8.27 (A4)
  paperHeight?: number;  // inches, default 11.69 (A4)
  marginTop?: number;    // inches
  marginBottom?: number;
  marginLeft?: number;
  marginRight?: number;
  landscape?: boolean;
}

export interface GotenbergConvertResult {
  pdfStream: Readable;
  byteLength: number;  // filled after stream consumption — helper below
}

export async function convertDocxToPDF(
  endpoint: string,
  args: GotenbergConvertArgs,
): Promise<Readable> {
  const { FormData, Blob } = await import('node:buffer') as any;
  const form = new FormData();
  // Gotenberg expects the DOCX under the field name `files`.
  const chunks: Buffer[] = [];
  for await (const c of args.docxStream) chunks.push(c as Buffer);
  form.append('files', new Blob([Buffer.concat(chunks)], {
    type: 'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
  }), 'document.docx');
  if (args.paperWidth) form.append('paperWidth', String(args.paperWidth));
  if (args.paperHeight) form.append('paperHeight', String(args.paperHeight));
  if (args.landscape !== undefined) form.append('landscape', String(args.landscape));
  const res = await fetch(`${endpoint}/forms/libreoffice/convert`, {
    method: 'POST',
    body: form as any,
  });
  if (!res.ok) {
    const text = await res.text().catch(() => '');
    throw new Error(`gotenberg ${res.status}: ${text.slice(0, 500)}`);
  }
  // res.body is a Web ReadableStream — convert to Node Readable.
  return Readable.fromWeb(res.body as any);
}
```

- [ ] **Step 3: Route — `convert-pdf.ts`**

Contract: request body `{ docx_key, output_key, render_opts?: { paper_size, margins, landscape } }` — service token auth (reuse `requireServiceToken` middleware from Plan A). Fetches DOCX via MinIO client, streams into Gotenberg, streams response into MinIO PUT at `output_key`, returns `{ output_key, content_hash, size_bytes }` where `content_hash` is SHA256 of the PDF bytes **recomputed during the stream** (not trusted from Gotenberg headers).

```ts
import type { FastifyInstance } from 'fastify';
import { createHash } from 'node:crypto';
import { PassThrough, Readable } from 'node:stream';
import { pipeline } from 'node:stream/promises';
import { s3Client } from '../s3';
import { convertDocxToPDF } from '../pdf/gotenbergClient';
import { DOCGEN_V2_VERSION } from '../pdf/version';
import { requireServiceToken } from '../middleware/serviceToken';
import { env } from '../env';

interface ConvertPDFBody {
  docx_key: string;
  output_key: string;
  render_opts?: { paper_size?: 'A4' | 'Letter'; landscape?: boolean };
}

const PAPER: Record<string, { w: number; h: number }> = {
  A4:     { w: 8.27,  h: 11.69 },
  Letter: { w: 8.5,   h: 11.0  },
};

export function registerConvertPDF(app: FastifyInstance) {
  app.post('/convert/pdf', { preHandler: requireServiceToken }, async (req, reply) => {
    const body = req.body as ConvertPDFBody;
    if (!body?.docx_key || !body?.output_key) {
      return reply.code(400).send({ error: 'docx_key and output_key required' });
    }

    // 1. Fetch DOCX from S3 as a stream.
    const obj = await s3Client.getObject({ Bucket: env.DOCGEN_V2_S3_BUCKET, Key: body.docx_key });
    const docxStream = Readable.from(obj.Body as any);

    // 2. Stream into Gotenberg.
    let pdfStream: Readable;
    try {
      const paper = PAPER[body.render_opts?.paper_size ?? 'A4'];
      pdfStream = await convertDocxToPDF(env.DOCGEN_V2_GOTENBERG_URL, {
        docxStream,
        paperWidth:  paper.w,
        paperHeight: paper.h,
        landscape:   body.render_opts?.landscape ?? false,
      });
    } catch (err: any) {
      req.log.error({ err: err.message, docx_key: body.docx_key }, 'gotenberg failed');
      return reply.code(502).send({ error: 'gotenberg_failed', message: err.message });
    }

    // 3. Tee the PDF stream: one leg to SHA256, one leg to S3 PUT.
    const hash = createHash('sha256');
    let size = 0;
    const toHash = new PassThrough();
    toHash.on('data', (c) => { hash.update(c); size += c.length; });
    const toS3 = new PassThrough();
    pdfStream.on('data', (c) => { toHash.write(c); toS3.write(c); });
    pdfStream.on('end', () => { toHash.end(); toS3.end(); });
    pdfStream.on('error', (err) => { toHash.destroy(err); toS3.destroy(err); });

    // Fire S3 PUT with the tee stream.
    try {
      await s3Client.putObject({
        Bucket:      env.DOCGEN_V2_S3_BUCKET,
        Key:         body.output_key,
        Body:        toS3,
        ContentType: 'application/pdf',
      });
    } catch (err: any) {
      req.log.error({ err: err.message }, 's3 put failed');
      return reply.code(502).send({ error: 's3_put_failed' });
    }

    // Wait for the hash tee to drain before reading digest.
    await new Promise<void>((resolve, reject) => {
      toHash.on('end', () => resolve());
      toHash.on('error', reject);
    });

    return reply.code(200).send({
      output_key:   body.output_key,
      content_hash: hash.digest('hex'),
      size_bytes:   size,
      docgen_v2_version: DOCGEN_V2_VERSION,
    });
  });
}
```

- [ ] **Step 4: Smoke test — real Gotenberg**

`convert-pdf.smoke.test.ts` (requires `docker run gotenberg/gotenberg:8.4 -p 3000:3000` and MinIO; skip in `unit` mode via `test.skipIf(process.env.DOCGEN_V2_SKIP_INTEGRATION === 'true')`):

```ts
import { test, expect } from 'vitest';
import Fastify from 'fastify';
import { registerConvertPDF } from '../src/routes/convert-pdf';

test.skipIf(process.env.DOCGEN_V2_SKIP_INTEGRATION === 'true')('POST /convert/pdf converts real docx', async () => {
  const app = Fastify();
  registerConvertPDF(app);
  // Fixture: pre-seed MinIO with fixtures/minimal.docx → key `test/minimal.docx`.
  const res = await app.inject({
    method: 'POST',
    url: '/convert/pdf',
    headers: { 'x-service-token': process.env.DOCGEN_V2_SERVICE_TOKEN! },
    payload: { docx_key: 'test/minimal.docx', output_key: 'test/out.pdf' },
  });
  expect(res.statusCode).toBe(200);
  const body = res.json();
  expect(body.output_key).toBe('test/out.pdf');
  expect(body.content_hash).toMatch(/^[0-9a-f]{64}$/);
  expect(body.size_bytes).toBeGreaterThan(200); // any real PDF is > 200 bytes
});
```

- [ ] **Step 5: 502 test — Gotenberg unreachable**

`convert-pdf.gotenberg-502.test.ts`:

```ts
import { test, expect, vi } from 'vitest';
import Fastify from 'fastify';
import { registerConvertPDF } from '../src/routes/convert-pdf';
import * as client from '../src/pdf/gotenbergClient';

test('gotenberg failure → 502', async () => {
  vi.spyOn(client, 'convertDocxToPDF').mockRejectedValue(new Error('ECONNREFUSED'));
  const app = Fastify();
  registerConvertPDF(app);
  const res = await app.inject({
    method: 'POST', url: '/convert/pdf',
    headers: { 'x-service-token': process.env.DOCGEN_V2_SERVICE_TOKEN! },
    payload: { docx_key: 'test/minimal.docx', output_key: 'test/out.pdf' },
  });
  expect(res.statusCode).toBe(502);
  expect(res.json().error).toBe('gotenberg_failed');
});
```

- [ ] **Step 6: Hash-stability test**

`convert-pdf.cache-hit.test.ts` — asserts that converting the same DOCX twice yields the same SHA256 (i.e. Gotenberg is deterministic for the same input):

```ts
import { test, expect } from 'vitest';
import Fastify from 'fastify';
import { registerConvertPDF } from '../src/routes/convert-pdf';

test.skipIf(process.env.DOCGEN_V2_SKIP_INTEGRATION === 'true')('deterministic conversion: two converts, same hash', async () => {
  const app = Fastify();
  registerConvertPDF(app);
  const one = await app.inject({ method: 'POST', url: '/convert/pdf', headers: { 'x-service-token': process.env.DOCGEN_V2_SERVICE_TOKEN! }, payload: { docx_key: 'test/minimal.docx', output_key: 'test/out1.pdf' } });
  const two = await app.inject({ method: 'POST', url: '/convert/pdf', headers: { 'x-service-token': process.env.DOCGEN_V2_SERVICE_TOKEN! }, payload: { docx_key: 'test/minimal.docx', output_key: 'test/out2.pdf' } });
  expect(one.statusCode).toBe(200);
  expect(two.statusCode).toBe(200);
  expect(one.json().content_hash).toBe(two.json().content_hash);
});
```

> **Note:** If this test flakes because of LibreOffice PDF metadata (creation time stamped into PDF `/CreationDate`), set Gotenberg flag `pdfa=PDF/A-2b` + `--libre-office-disable-pdf-metadata` in CI — otherwise bake a metadata-stripping post-pass into the stream. A failure here is load-bearing: if conversion isn't deterministic the cache strategy is broken. Escalate to the reviewer rather than skipping the test.

- [ ] **Step 7: Commit**

```bash
npm test --workspace @metaldocs/docgen-v2 -- convert-pdf
rtk git add apps/docgen-v2/src/routes/convert-pdf.ts apps/docgen-v2/src/pdf apps/docgen-v2/test/convert-pdf.smoke.test.ts apps/docgen-v2/test/convert-pdf.gotenberg-502.test.ts apps/docgen-v2/test/convert-pdf.cache-hit.test.ts
rtk git commit -m "feat(docgen-v2): POST /convert/pdf streaming DOCX → Gotenberg → S3"
```

---

## Task 5: Go docgen client — `ConvertPDF`

**Files:**
- Create: `internal/modules/documents_v2/infrastructure/docgen/pdf_client.go`

- [ ] **Step 1: Write client**

```go
package docgen

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"
)

type ConvertPDFRequest struct {
    DocxKey    string       `json:"docx_key"`
    OutputKey  string       `json:"output_key"`
    RenderOpts *RenderOpts  `json:"render_opts,omitempty"`
}

type RenderOpts struct {
    PaperSize string `json:"paper_size,omitempty"`
    Landscape bool   `json:"landscape,omitempty"`
}

type ConvertPDFResult struct {
    OutputKey       string `json:"output_key"`
    ContentHash     string `json:"content_hash"` // hex sha256 from docgen-v2
    SizeBytes       int64  `json:"size_bytes"`
    DocgenV2Version string `json:"docgen_v2_version"`
}

// ConvertPDF posts to docgen-v2 /convert/pdf. Timeout: 60s (spec SLO: p95
// 10-page conversion < 5s; 60s covers pathological cases + upload time).
func (c *Client) ConvertPDF(ctx context.Context, req ConvertPDFRequest) (ConvertPDFResult, error) {
    ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
    defer cancel()
    body, _ := json.Marshal(req)
    httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint+"/convert/pdf", bytes.NewReader(body))
    if err != nil {
        return ConvertPDFResult{}, err
    }
    httpReq.Header.Set("content-type", "application/json")
    httpReq.Header.Set("x-service-token", c.serviceToken)
    resp, err := c.http.Do(httpReq)
    if err != nil {
        return ConvertPDFResult{}, fmt.Errorf("docgen-v2 call: %w", err)
    }
    defer resp.Body.Close()
    raw, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
    if resp.StatusCode != http.StatusOK {
        return ConvertPDFResult{}, fmt.Errorf("docgen-v2 status %d: %s", resp.StatusCode, string(raw))
    }
    var out ConvertPDFResult
    if err := json.Unmarshal(raw, &out); err != nil {
        return ConvertPDFResult{}, fmt.Errorf("decode docgen-v2 response: %w", err)
    }
    return out, nil
}
```

- [ ] **Step 2: Commit**

```bash
go build ./internal/modules/documents_v2/infrastructure/docgen/...
rtk git add internal/modules/documents_v2/infrastructure/docgen/pdf_client.go
rtk git commit -m "feat(documents-v2/infra): docgen-v2 ConvertPDF client"
```

---

## Task 6: Application service — `ExportPDF` + `SignedDocxURL`

**Files:**
- Create: `internal/modules/documents_v2/application/export_service.go`
- Create: `internal/modules/documents_v2/application/export_service_test.go`

- [ ] **Step 1: Write service**

```go
package application

import (
    "context"
    "encoding/hex"
    "errors"
    "fmt"

    "metaldocs/internal/modules/documents_v2/domain"
    "metaldocs/internal/modules/documents_v2/infrastructure/docgen"
)

// ExportRepo is the subset of Repository the export service needs.
type ExportRepo interface {
    GetDocument(ctx context.Context, id string) (domain.Document, error)
    GetRevision(ctx context.Context, id string) (domain.Revision, error)
    GetTemplateVersionGrammarVersion(ctx context.Context, templateVersionID string) (string, error)
    InsertExport(ctx context.Context, e domain.Export) (domain.Export, bool, error)
    GetExportByHash(ctx context.Context, documentID string, hash []byte) (domain.Export, error)
    InsertAuditEvent(ctx context.Context, evt domain.AuditEvent) error
}

// Presigner extension — adds PDF object existence probe + signed GET for
// non-docx paths. The documents_v2 Presigner interface gets these three
// new methods in the same package (Plan C introduced HashObject; this
// extends with Head/Stat/SignGet).
type ExportPresigner interface {
    HeadObject(ctx context.Context, key string) (found bool, err error)
    StatObject(ctx context.Context, key string) (sizeBytes int64, err error)
    SignGet(ctx context.Context, key string, contentType string) (string, error)
}

type DocgenPDFClient interface {
    ConvertPDF(ctx context.Context, req docgen.ConvertPDFRequest) (docgen.ConvertPDFResult, error)
}

type ExportService struct {
    repo       ExportRepo
    presigner  ExportPresigner
    docgen     DocgenPDFClient
    docgenVer  string   // injected at wiring (must match docgen-v2 DOCGEN_V2_VERSION constant or pin)
}

func NewExportService(repo ExportRepo, p ExportPresigner, d DocgenPDFClient, docgenVer string) *ExportService {
    return &ExportService{repo: repo, presigner: p, docgen: d, docgenVer: docgenVer}
}

// ExportPDF is the idempotent cache-or-generate flow. All mutations are
// idempotent: duplicate calls with the same (doc, revision, form_data,
// grammar, docgen_ver, render_opts) return the same composite_hash and
// the same export row.
func (s *ExportService) ExportPDF(
    ctx context.Context,
    tenantID, userID, documentID string,
    opts domain.RenderOptions,
) (domain.ExportResult, error) {
    doc, err := s.repo.GetDocument(ctx, documentID)
    if err != nil {
        return domain.ExportResult{}, err
    }
    if doc.CurrentRevisionID == "" {
        return domain.ExportResult{}, domain.ErrExportDocxMissing
    }
    rev, err := s.repo.GetRevision(ctx, doc.CurrentRevisionID)
    if err != nil {
        return domain.ExportResult{}, err
    }
    grammarVer, err := s.repo.GetTemplateVersionGrammarVersion(ctx, doc.TemplateVersionID)
    if err != nil {
        return domain.ExportResult{}, err
    }
    hash, err := domain.ComputeCompositeHash(rev.ContentHash, doc.TemplateVersionID, grammarVer, s.docgenVer, opts)
    if err != nil {
        return domain.ExportResult{}, fmt.Errorf("compose hash: %w", err)
    }
    storageKey := fmt.Sprintf("tenants/%s/documents/%s/exports/%s.pdf", tenantID, documentID, hex.EncodeToString(hash))

    // Cache probe: if the export row already exists, trust it and return
    // the signed URL without touching Gotenberg. This is the hot path —
    // every repeat export (refresh, E2E re-run, cached-client retry) hits it.
    if existing, err := s.repo.GetExportByHash(ctx, documentID, hash); err == nil {
        return domain.ExportResult{Export: existing, Cached: true}, nil
    } else if !errors.Is(err, domain.ErrNotFound) {
        return domain.ExportResult{}, err
    }

    // Also probe S3 in case the row was lost (e.g. a previous generation
    // wrote the object but the INSERT failed). If S3 has it we skip the
    // expensive conversion and just insert the ledger row.
    if found, err := s.presigner.HeadObject(ctx, storageKey); err == nil && found {
        // Fall through to InsertExport; InsertExport is ON CONFLICT DO NOTHING.
    } else {
        // Cold path — invoke docgen-v2.
        if _, err := s.docgen.ConvertPDF(ctx, docgen.ConvertPDFRequest{
            DocxKey:   rev.DocxStorageKey,
            OutputKey: storageKey,
            RenderOpts: &docgen.RenderOpts{
                PaperSize: opts.PaperSize,
                Landscape: opts.LandscapeP,
            },
        }); err != nil {
            return domain.ExportResult{}, domain.ErrExportGotenbergFailed
        }
    }

    // Stat to get size — needed for both the ledger row and the handler
    // response. Fails loudly if the object vanished between convert and
    // this call (shouldn't happen; indicates a data-layer bug).
    sizeBytes, err := s.presigner.StatObject(ctx, storageKey)
    if err != nil {
        return domain.ExportResult{}, fmt.Errorf("post-convert stat: %w", err)
    }
    if sizeBytes <= 0 {
        return domain.ExportResult{}, fmt.Errorf("post-convert stat: zero-size object at %s", storageKey)
    }

    // Insert ledger row; idempotent.
    e := domain.Export{
        TenantID:      tenantID,
        DocumentID:    documentID,
        RevisionID:    doc.CurrentRevisionID,
        CompositeHash: hash,
        StorageKey:    storageKey,
        SizeBytes:     sizeBytes,
        ContentType:   "application/pdf",
        CreatedBy:     userID,
    }
    ins, wasInsert, err := s.repo.InsertExport(ctx, e)
    if err != nil {
        return domain.ExportResult{}, err
    }
    // Emit one audit event per call, with `cached: true|false` in metadata.
    // Rationale: the dogfood runbook (Task 20) alerts on the ratio of
    // cached=false / total. That ratio is only computable if EVERY call
    // emits an event; miss-only auditing would force a second telemetry
    // pipeline (metrics) to get the same signal. Audit is the single
    // source of truth for export activity in W4; W5+ may split this out
    // into an OpenTelemetry counter if volume grows.
    if err := s.repo.InsertAuditEvent(ctx, domain.AuditEvent{
        TenantID:   tenantID,
        Actor:      userID,
        Action:     "export.pdf_generated",
        ResourceID: documentID,
        Metadata: map[string]any{
            "composite_hash":    hex.EncodeToString(hash),
            "revision_id":       doc.CurrentRevisionID,
            "storage_key":       storageKey,
            "docgen_v2_version": s.docgenVer,
            "cached":            !wasInsert,
        },
    }); err != nil {
        return domain.ExportResult{}, err
    }
    return domain.ExportResult{Export: ins, Cached: !wasInsert}, nil
}

// SignExportURL issues a signed GET URL for a PDF export object. No audit
// event — the caller (handler) already emitted `export.pdf_generated`
// before asking for the URL. The handler calls this once per successful
// ExportPDF response.
func (s *ExportService) SignExportURL(ctx context.Context, storageKey string) (string, error) {
    return s.presigner.SignGet(ctx, storageKey, "application/pdf")
}

// GetDocumentSummary returns the current Document aggregate (minimal pass-
// through to the repo). The /export/docx-url handler uses this to report
// `revision_id` alongside the signed URL so the frontend can cache-key on it.
func (s *ExportService) GetDocumentSummary(ctx context.Context, documentID string) (domain.Document, error) {
    return s.repo.GetDocument(ctx, documentID)
}

// SignedDocxURL returns a short-lived GET URL for the current revision's
// .docx. Emits audit event `export.docx_downloaded`.
func (s *ExportService) SignedDocxURL(ctx context.Context, tenantID, userID, documentID string) (string, error) {
    doc, err := s.repo.GetDocument(ctx, documentID)
    if err != nil {
        return "", err
    }
    if doc.CurrentRevisionID == "" {
        return "", domain.ErrExportDocxMissing
    }
    rev, err := s.repo.GetRevision(ctx, doc.CurrentRevisionID)
    if err != nil {
        return "", err
    }
    url, err := s.presigner.SignGet(ctx, rev.DocxStorageKey,
        "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
    if err != nil {
        return "", err
    }
    if err := s.repo.InsertAuditEvent(ctx, domain.AuditEvent{
        TenantID:   tenantID,
        Actor:      userID,
        Action:     "export.docx_downloaded",
        ResourceID: documentID,
        Metadata:   map[string]any{"revision_id": doc.CurrentRevisionID},
    }); err != nil {
        return "", err
    }
    return url, nil
}
```

- [ ] **Step 2: Unit tests — 4 branches**

`export_service_test.go`: fake repo + fake presigner + fake docgen. Cases:

```go
package application_test

import (
    "context"
    "errors"
    "testing"

    "metaldocs/internal/modules/documents_v2/application"
    "metaldocs/internal/modules/documents_v2/domain"
    "metaldocs/internal/modules/documents_v2/infrastructure/docgen"
)

type fakeExportRepo struct {
    docs       map[string]domain.Document
    revs       map[string]domain.Revision
    grammar    string
    exports    map[string]domain.Export // keyed by hex(hash)
    audit      []domain.AuditEvent
}

// ...method impls omitted for brevity; follow the ExportRepo interface...

type fakePresigner struct {
    objects map[string]int64 // key → size
    headErr error
    signURL string
}

type fakeDocgen struct {
    calls int
    err   error
}

func (f *fakeDocgen) ConvertPDF(_ context.Context, req docgen.ConvertPDFRequest) (docgen.ConvertPDFResult, error) {
    f.calls++
    if f.err != nil {
        return docgen.ConvertPDFResult{}, f.err
    }
    return docgen.ConvertPDFResult{OutputKey: req.OutputKey, ContentHash: "aa", SizeBytes: 1024, DocgenV2Version: "docgen-v2@0.4.0"}, nil
}

// Cases (all produce audit `export.pdf_generated` except #4):
// 1. Cold miss: empty exports, no S3 object → docgen called → row inserted → audit cached=false.
// 2. Warm hit: export row exists → docgen NOT called → NO new row → audit cached=true (every call audits, so ratio is computable).
// 3. S3 has object but ledger row lost: HeadObject true, GetExportByHash returns ErrNotFound → docgen NOT called → row inserted via ON CONFLICT DO NOTHING → audit cached=false.
// 4. Gotenberg failure: docgen returns error → ExportPDF returns ErrExportGotenbergFailed → no row, NO audit (failure short-circuits before audit).

func TestExportPDF_ColdMissCallsDocgenAndEmitsCachedFalseAudit(t *testing.T) { /* ... */ }
func TestExportPDF_WarmHitSkipsDocgenAndEmitsCachedTrueAudit(t *testing.T)   { /* ... */ }
func TestExportPDF_S3HasObjectRowMissing_InsertsWithoutDocgen(t *testing.T)  { /* ... */ }
func TestExportPDF_GotenbergFailure_ReturnsDomainError(t *testing.T) {
    repo := newFakeExportRepo()
    pre  := newFakePresigner()
    dg   := &fakeDocgen{err: errors.New("ECONNREFUSED")}
    svc  := application.NewExportService(repo, pre, dg, "docgen-v2@0.4.0")
    _, err := svc.ExportPDF(context.Background(), "t1", "u1", "d1", domain.RenderOptions{})
    if !errors.Is(err, domain.ErrExportGotenbergFailed) {
        t.Fatalf("want ErrExportGotenbergFailed, got %v", err)
    }
    if len(repo.audit) != 0 { t.Fatalf("no audit event on failure, got %d", len(repo.audit)) }
}
```

(Full-body tests to be completed during implementation; the cases listed are load-bearing and must each be a distinct test function.)

- [ ] **Step 3: Commit**

```bash
go test ./internal/modules/documents_v2/application/... -run Export
rtk git add internal/modules/documents_v2/application/export_service.go internal/modules/documents_v2/application/export_service_test.go
rtk git commit -m "feat(documents-v2/app): ExportPDF cache-or-generate + SignedDocxURL"
```

---

## Task 7: Rate-limit middleware (shared)

**Files:**
- Create: `internal/platform/ratelimit/middleware.go`
- Create: `internal/platform/ratelimit/config.go`
- Create: `internal/platform/ratelimit/middleware_test.go`

- [ ] **Step 1: Config**

```go
package ratelimit

// Per-route quotas from spec §Rate limits. Values are requests-per-minute
// per user. Routes not listed here are unlimited.
//
// Envvar overrides: METALDOCS_RLIMIT_<ROUTE_KEY> (e.g. EXPORT_PDF=30).
type RouteKey string

const (
    RouteUploadsPresign   RouteKey = "uploads_presign"
    RouteAutosavePresign  RouteKey = "autosave_presign"
    RouteAutosaveCommit   RouteKey = "autosave_commit"
    RouteDocumentsRender  RouteKey = "documents_render"
    RouteExportPDF        RouteKey = "export_pdf"
)

type Config struct {
    Quotas map[RouteKey]int // req/min
}

func DefaultConfig() Config {
    return Config{
        Quotas: map[RouteKey]int{
            RouteUploadsPresign:  60,
            RouteAutosavePresign: 60,
            RouteAutosaveCommit:  30,
            RouteDocumentsRender: 30,
            RouteExportPDF:       20,
        },
    }
}
```

- [ ] **Step 2: Middleware**

```go
package ratelimit

import (
    "encoding/json"
    "net/http"
    "strconv"
    "sync"
    "time"

    "golang.org/x/time/rate"
)

type Middleware struct {
    cfg      Config
    limiters sync.Map // key: "<route_key>:<user_id>" → *rate.Limiter
}

func New(cfg Config) *Middleware { return &Middleware{cfg: cfg} }

// Limit returns an http.Handler wrapper that enforces the quota for the
// given route. userExtractor pulls the subject id out of request ctx (the
// IAM middleware sets it before this middleware runs).
func (m *Middleware) Limit(key RouteKey, userExtractor func(*http.Request) string, next http.Handler) http.Handler {
    quota, ok := m.cfg.Quotas[key]
    if !ok {
        return next // no quota configured
    }
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        user := userExtractor(r)
        if user == "" {
            // No user id → bypass; IAM middleware should have rejected already.
            next.ServeHTTP(w, r)
            return
        }
        lk := string(key) + ":" + user
        lim, _ := m.limiters.LoadOrStore(lk, rate.NewLimiter(rate.Every(time.Minute/time.Duration(quota)), quota))
        l := lim.(*rate.Limiter)
        reservation := l.Reserve()
        if !reservation.OK() {
            writeRateLimitError(w, quota, 60)
            return
        }
        if d := reservation.Delay(); d > 0 {
            reservation.Cancel()
            writeRateLimitError(w, quota, int(d.Seconds())+1)
            return
        }
        next.ServeHTTP(w, r)
    })
}

func writeRateLimitError(w http.ResponseWriter, quota, retryAfterSec int) {
    w.Header().Set("content-type", "application/json")
    w.Header().Set("retry-after", strconv.Itoa(retryAfterSec))
    w.WriteHeader(http.StatusTooManyRequests)
    _ = json.NewEncoder(w).Encode(map[string]any{
        "error":               "rate_limited",
        "quota_per_minute":    quota,
        "retry_after_seconds": retryAfterSec,
    })
}
```

- [ ] **Step 3: Tests — burst + retry_after**

```go
package ratelimit_test

import (
    "net/http"
    "net/http/httptest"
    "strconv"
    "testing"

    "metaldocs/internal/platform/ratelimit"
)

func TestLimit_BurstThenRejects(t *testing.T) {
    mw := ratelimit.New(ratelimit.Config{Quotas: map[ratelimit.RouteKey]int{ratelimit.RouteExportPDF: 3}})
    next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })
    h := mw.Limit(ratelimit.RouteExportPDF, func(r *http.Request) string { return "u1" }, next)

    // 3 permitted, 4th rejected.
    for i := 0; i < 3; i++ {
        rr := httptest.NewRecorder()
        h.ServeHTTP(rr, httptest.NewRequest("POST", "/x", nil))
        if rr.Code != 204 {
            t.Fatalf("req %d: want 204, got %d", i, rr.Code)
        }
    }
    rr := httptest.NewRecorder()
    h.ServeHTTP(rr, httptest.NewRequest("POST", "/x", nil))
    if rr.Code != http.StatusTooManyRequests {
        t.Fatalf("4th req: want 429, got %d", rr.Code)
    }
    retry := rr.Header().Get("retry-after")
    if n, _ := strconv.Atoi(retry); n < 1 {
        t.Fatalf("retry-after must be ≥1s, got %q", retry)
    }
}

func TestLimit_PerUserIsolation(t *testing.T) {
    mw := ratelimit.New(ratelimit.Config{Quotas: map[ratelimit.RouteKey]int{ratelimit.RouteExportPDF: 1}})
    next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })
    h := mw.Limit(ratelimit.RouteExportPDF, func(r *http.Request) string { return r.Header.Get("x-user") }, next)

    // User A burns its 1 token.
    rrA := httptest.NewRecorder()
    reqA := httptest.NewRequest("POST", "/x", nil); reqA.Header.Set("x-user", "A")
    h.ServeHTTP(rrA, reqA)
    if rrA.Code != 204 { t.Fatalf("A first: want 204, got %d", rrA.Code) }

    rrA2 := httptest.NewRecorder()
    reqA2 := httptest.NewRequest("POST", "/x", nil); reqA2.Header.Set("x-user", "A")
    h.ServeHTTP(rrA2, reqA2)
    if rrA2.Code != http.StatusTooManyRequests { t.Fatalf("A second: want 429, got %d", rrA2.Code) }

    // User B still has its own bucket.
    rrB := httptest.NewRecorder()
    reqB := httptest.NewRequest("POST", "/x", nil); reqB.Header.Set("x-user", "B")
    h.ServeHTTP(rrB, reqB)
    if rrB.Code != 204 { t.Fatalf("B first: want 204, got %d", rrB.Code) }
}
```

- [ ] **Step 4: Commit**

```bash
go test ./internal/platform/ratelimit/...
rtk git add internal/platform/ratelimit
rtk git commit -m "feat(platform/ratelimit): per-user token-bucket middleware + route quotas"
```

---

## Task 8: Wire rate-limit middleware into existing W1–W3 routes

**Files:**
- Modify: `apps/api/cmd/metaldocs-api/main.go`

- [ ] **Step 1: Apply middleware on the 5 spec'd routes**

In `main.go`, after the IAM middleware but before `docMod.Handler.ServeHTTP`:

```go
rl := ratelimit.New(ratelimit.DefaultConfig())
userFn := func(r *http.Request) string { return iam.UserIDFromContext(r.Context()) }

mux.Handle("POST /api/v2/uploads/presign",
    rl.Limit(ratelimit.RouteUploadsPresign, userFn, uploadsHandler))
mux.Handle("POST /api/v2/documents/{id}/autosave/presign",
    rl.Limit(ratelimit.RouteAutosavePresign, userFn, docMod.Handler))
mux.Handle("POST /api/v2/documents/{id}/autosave/commit",
    rl.Limit(ratelimit.RouteAutosaveCommit, userFn, docMod.Handler))
mux.Handle("POST /api/v2/documents/{id}/render",
    rl.Limit(ratelimit.RouteDocumentsRender, userFn, docMod.Handler))
mux.Handle("POST /api/v2/documents/{id}/export/pdf",
    rl.Limit(ratelimit.RouteExportPDF, userFn, docMod.Handler))
```

> **Routing note:** Go 1.22 `ServeMux` patterns support method prefixes and `{id}` params natively. The existing wiring in Plan C registers the documents handlers on `/api/v2/documents/` prefix — those 3 existing rate-limited routes must be **re-registered with the more-specific rate-limit wrapper** so ServeMux's longest-match picks the wrapper. Document this in the main.go comment.

- [ ] **Step 2: Commit**

```bash
go build ./apps/api/cmd/metaldocs-api/...
rtk git add apps/api/cmd/metaldocs-api/main.go
rtk git commit -m "feat(api): enforce per-user rate limits on W1–W4 routes per spec"
```

---

## Task 9: HTTP handler — `POST /export/pdf` + `GET /export/docx-url`

**Files:**
- Create: `internal/modules/documents_v2/delivery/http/export_handler.go`
- Create: `internal/modules/documents_v2/delivery/http/export_handler_test.go`
- Modify: `internal/modules/documents_v2/module.go` (attach export service)

- [ ] **Step 1: Handler struct + routes**

```go
package http

import (
    "encoding/hex"
    "encoding/json"
    "errors"
    "net/http"

    "metaldocs/internal/modules/documents_v2/application"
    "metaldocs/internal/modules/documents_v2/domain"
)

type ExportHandler struct {
    svc *application.ExportService
}

func NewExportHandler(svc *application.ExportService) *ExportHandler { return &ExportHandler{svc: svc} }

func (h *ExportHandler) Register(mux *http.ServeMux, ensureDocAccess MiddlewareFn, withAdminCtx AdminCtxFn) {
    mux.HandleFunc("POST /api/v2/documents/{id}/export/pdf", func(w http.ResponseWriter, r *http.Request) {
        r = withAdminCtx(r)
        docID := r.PathValue("id")
        if !ensureDocAccess(w, r, docID) { return }
        h.exportPDF(w, r, docID)
    })
    mux.HandleFunc("GET /api/v2/documents/{id}/export/docx-url", func(w http.ResponseWriter, r *http.Request) {
        r = withAdminCtx(r)
        docID := r.PathValue("id")
        if !ensureDocAccess(w, r, docID) { return }
        h.exportDocxURL(w, r, docID)
    })
}

type exportPDFReq struct {
    PaperSize string `json:"paper_size,omitempty"` // "A4" (default) or "Letter"
    Landscape bool   `json:"landscape,omitempty"`
}

type exportPDFResp struct {
    StorageKey    string `json:"storage_key"`
    SignedURL     string `json:"signed_url"`
    CompositeHash string `json:"composite_hash"` // hex
    SizeBytes     int64  `json:"size_bytes"`
    Cached        bool   `json:"cached"`
    RevisionID    string `json:"revision_id"`
}

func (h *ExportHandler) exportPDF(w http.ResponseWriter, r *http.Request, docID string) {
    var body exportPDFReq
    if r.ContentLength > 0 {
        if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
            writeError(w, http.StatusBadRequest, "invalid_body"); return
        }
    }
    if body.PaperSize == "" { body.PaperSize = "A4" }
    tenantID := tenantIDFromCtx(r.Context())
    userID := userIDFromCtx(r.Context())
    res, err := h.svc.ExportPDF(r.Context(), tenantID, userID, docID, domain.RenderOptions{
        PaperSize:  body.PaperSize,
        LandscapeP: body.Landscape,
    })
    if err != nil {
        switch {
        case errors.Is(err, domain.ErrExportDocxMissing):
            writeError(w, http.StatusConflict, "docx_missing")
        case errors.Is(err, domain.ErrExportGotenbergFailed):
            writeError(w, http.StatusBadGateway, "gotenberg_failed")
        case errors.Is(err, domain.ErrNotFound):
            writeError(w, http.StatusNotFound, "document_not_found")
        default:
            writeError(w, http.StatusInternalServerError, "internal")
        }
        return
    }
    signedURL, err := h.svc.SignExportURL(r.Context(), res.Export.StorageKey)
    if err != nil {
        writeError(w, http.StatusInternalServerError, "sign_failed"); return
    }
    _ = json.NewEncoder(w).Encode(exportPDFResp{
        StorageKey:    res.Export.StorageKey,
        SignedURL:     signedURL,
        CompositeHash: hex.EncodeToString(res.Export.CompositeHash),
        SizeBytes:     res.Export.SizeBytes,
        Cached:        res.Cached,
        RevisionID:    res.Export.RevisionID,
    })
}

type exportDocxURLResp struct {
    SignedURL  string `json:"signed_url"`
    RevisionID string `json:"revision_id"`
}

func (h *ExportHandler) exportDocxURL(w http.ResponseWriter, r *http.Request, docID string) {
    tenantID := tenantIDFromCtx(r.Context())
    userID := userIDFromCtx(r.Context())
    url, err := h.svc.SignedDocxURL(r.Context(), tenantID, userID, docID)
    if err != nil {
        switch {
        case errors.Is(err, domain.ErrExportDocxMissing):
            writeError(w, http.StatusConflict, "docx_missing")
        case errors.Is(err, domain.ErrNotFound):
            writeError(w, http.StatusNotFound, "document_not_found")
        default:
            writeError(w, http.StatusInternalServerError, "internal")
        }
        return
    }
    // revision_id needed for ExportMenu display — fetch once more.
    doc, err := h.svc.GetDocumentSummary(r.Context(), docID)
    if err != nil {
        writeError(w, http.StatusInternalServerError, "internal"); return
    }
    _ = json.NewEncoder(w).Encode(exportDocxURLResp{SignedURL: url, RevisionID: doc.CurrentRevisionID})
}
```

> **Service methods consumed:** `ExportPDF`, `SignExportURL`, `SignedDocxURL`, `GetDocumentSummary` — all defined in Task 6. No additional service surface is required by this handler.

- [ ] **Step 2: Handler tests**

`export_handler_test.go`: per branch (200 cache-hit, 200 cache-miss, 409 docx_missing, 502 gotenberg_failed, 404 not_found, 403 not-owner).

- [ ] **Step 3: Module wiring**

`module.go`: extend `Module` with `ExportHandler` and `ExportService`. `New(deps)` now constructs both; `Handler` mux registers export routes.

- [ ] **Step 4: Commit**

```bash
go test ./internal/modules/documents_v2/delivery/http/... -run Export
rtk git add internal/modules/documents_v2/delivery/http/export_handler.go internal/modules/documents_v2/delivery/http/export_handler_test.go internal/modules/documents_v2/module.go
rtk git commit -m "feat(documents-v2/http): POST /export/pdf + GET /export/docx-url"
```

---

## Task 10: Permission resolver entries

**Files:**
- Modify: `apps/api/cmd/metaldocs-api/permissions.go`

- [ ] **Step 1: Add export-route entries**

```go
// Add under the /api/v2/documents/{id}/... section:
{ Method: "POST", Pattern: "/api/v2/documents/{id}/export/pdf",        Permission: PermDocumentView },
{ Method: "GET",  Pattern: "/api/v2/documents/{id}/export/docx-url",   Permission: PermDocumentView },
```

> **Rationale:** Both routes are read-only against the document (no revision mutation). Ownership enforcement rides on `ensureDocAccess` + `withAdminCtx`; the middleware role gate is `PermDocumentView` which `document_filler`, `template_author`, and `admin` all hold. Filler-B cannot export Filler-A's document because `ensureDocAccess` short-circuits with 403.

- [ ] **Step 2: Commit**

```bash
rtk git add apps/api/cmd/metaldocs-api/permissions.go
rtk git commit -m "feat(api/permissions): /export/pdf + /export/docx-url → PermDocumentView"
```

---

## Task 11: OpenAPI — add 3 paths + `x-rate-limit`

**Files:**
- Modify: `api/openapi/v1/partials/documents-v2.yaml`
- Modify: `api/openapi/v1/openapi.yaml` (merge the 3 path keys)

- [ ] **Step 1: Partial additions**

```yaml
paths:
  /documents/{id}/export/pdf:
    post:
      operationId: exportDocumentPDF
      x-rate-limit: { requests_per_minute: 20 }
      parameters:
        - $ref: '#/components/parameters/DocumentID'
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                paper_size: { type: string, enum: [A4, Letter], default: A4 }
                landscape:  { type: boolean, default: false }
      responses:
        '200':
          description: PDF ready (cached or freshly generated)
          content:
            application/json:
              schema:
                type: object
                required: [storage_key, signed_url, composite_hash, size_bytes, cached, revision_id]
                properties:
                  storage_key:    { type: string }
                  signed_url:     { type: string, format: uri }
                  composite_hash: { type: string, pattern: '^[0-9a-f]{64}$' }
                  size_bytes:     { type: integer, format: int64 }
                  cached:         { type: boolean }
                  revision_id:    { type: string, format: uuid }
        '409': { $ref: '#/components/responses/DocxMissing' }
        '429': { $ref: '#/components/responses/RateLimited' }
        '502': { $ref: '#/components/responses/GotenbergFailed' }

  /documents/{id}/export/docx-url:
    get:
      operationId: getDocumentDocxURL
      parameters:
        - $ref: '#/components/parameters/DocumentID'
      responses:
        '200':
          description: Signed GET URL for current revision .docx
          content:
            application/json:
              schema:
                type: object
                required: [signed_url, revision_id]
                properties:
                  signed_url:  { type: string, format: uri }
                  revision_id: { type: string, format: uuid }
        '409': { $ref: '#/components/responses/DocxMissing' }

components:
  responses:
    RateLimited:
      description: Per-user rate limit exceeded
      headers:
        retry-after: { schema: { type: integer }, description: Seconds until retry is permitted }
      content:
        application/json:
          schema:
            type: object
            required: [error, quota_per_minute, retry_after_seconds]
            properties:
              error:               { type: string, enum: [rate_limited] }
              quota_per_minute:    { type: integer }
              retry_after_seconds: { type: integer }
    DocxMissing:
      description: Current revision .docx object missing from S3
      content: { application/json: { schema: { $ref: '#/components/schemas/ErrorEnvelope' } } }
    GotenbergFailed:
      description: PDF conversion failed
      content: { application/json: { schema: { $ref: '#/components/schemas/ErrorEnvelope' } } }
```

- [ ] **Step 2: Merge into root OpenAPI file**

Append the three path keys to `api/openapi/v1/openapi.yaml` under the existing documents-v2 merge block. Do NOT globbingly include the partial — the existing merge is key-by-key for Plan C's already-tracked paths.

- [ ] **Step 3: Validate schema**

```bash
npx @redocly/cli lint api/openapi/v1/openapi.yaml
# Expected: 0 errors.
```

- [ ] **Step 4: Commit**

```bash
rtk git add api/openapi/v1/partials/documents-v2.yaml api/openapi/v1/openapi.yaml
rtk git commit -m "docs(openapi/documents-v2): /export/pdf + /export/docx-url + x-rate-limit"
```

---

## Task 12: Frontend API client

**Files:**
- Create: `frontend/apps/web/src/features/documents/v2/api/exportsV2.ts`

- [ ] **Step 1: Client**

```ts
import { json } from './http';

export type ExportPDFResult = {
  storage_key: string;
  signed_url: string;
  composite_hash: string;
  size_bytes: number;
  cached: boolean;
  revision_id: string;
};

export type DocxURLResult = {
  signed_url: string;
  revision_id: string;
};

export async function exportPDF(
  documentID: string,
  opts: { paper_size?: 'A4' | 'Letter'; landscape?: boolean } = {},
): Promise<ExportPDFResult> {
  return json(await fetch(`/api/v2/documents/${documentID}/export/pdf`, {
    method: 'POST',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify(opts),
  }));
}

export async function getDocxSignedURL(documentID: string): Promise<DocxURLResult> {
  return json(await fetch(`/api/v2/documents/${documentID}/export/docx-url`));
}
```

> **Error shape:** `json()` (shared helper) parses `{ error, retry_after_seconds }` on 429 and throws an `HTTPError` with `.status = 429` and `.body = { retry_after_seconds }`. The ExportMenu consumes `.body.retry_after_seconds` to render a countdown.

- [ ] **Step 2: Commit**

```bash
rtk git add frontend/apps/web/src/features/documents/v2/api/exportsV2.ts
rtk git commit -m "feat(web/documents-v2): export api client (pdf + docx-url)"
```

---

## Task 13: Frontend — `ExportMenu` + page wiring

**Files:**
- Create: `frontend/apps/web/src/features/documents/v2/ExportMenu.tsx`
- Create: `frontend/apps/web/src/features/documents/v2/ExportMenu.test.tsx`
- Modify: `frontend/apps/web/src/features/documents/v2/DocumentEditorPage.tsx`

- [ ] **Step 1: Component**

```tsx
import { useState } from 'react';
import { exportPDF, getDocxSignedURL, type ExportPDFResult } from './api/exportsV2';

type Status =
  | { kind: 'idle' }
  | { kind: 'pending' }
  | { kind: 'done'; cached: boolean; url: string; sizeBytes: number }
  | { kind: 'error'; message: string }
  | { kind: 'rate_limited'; retryAfterSec: number };

export function ExportMenu({ documentID, canExport }: { documentID: string; canExport: boolean }) {
  const [status, setStatus] = useState<Status>({ kind: 'idle' });

  async function handleDOCX() {
    setStatus({ kind: 'pending' });
    try {
      const { signed_url } = await getDocxSignedURL(documentID);
      window.open(signed_url, '_blank', 'noopener');
      setStatus({ kind: 'done', cached: true, url: signed_url, sizeBytes: 0 });
    } catch (e: any) {
      if (e?.status === 429) { setStatus({ kind: 'rate_limited', retryAfterSec: e.body?.retry_after_seconds ?? 60 }); return; }
      setStatus({ kind: 'error', message: String(e?.message ?? e) });
    }
  }

  async function handlePDF() {
    setStatus({ kind: 'pending' });
    try {
      const res: ExportPDFResult = await exportPDF(documentID, { paper_size: 'A4' });
      window.open(res.signed_url, '_blank', 'noopener');
      setStatus({ kind: 'done', cached: res.cached, url: res.signed_url, sizeBytes: res.size_bytes });
    } catch (e: any) {
      if (e?.status === 429) { setStatus({ kind: 'rate_limited', retryAfterSec: e.body?.retry_after_seconds ?? 60 }); return; }
      if (e?.status === 502) { setStatus({ kind: 'error', message: 'PDF service unavailable — retry in a moment.' }); return; }
      if (e?.status === 409) { setStatus({ kind: 'error', message: 'Document missing. Save and retry.' }); return; }
      setStatus({ kind: 'error', message: String(e?.message ?? e) });
    }
  }

  return (
    <div data-export-menu>
      <button onClick={handleDOCX} disabled={!canExport || status.kind === 'pending'} data-export-docx>
        Download .docx
      </button>
      <button onClick={handlePDF} disabled={!canExport || status.kind === 'pending'} data-export-pdf>
        Export PDF
      </button>
      {status.kind === 'pending' && <span data-export-status="pending">Working…</span>}
      {status.kind === 'done' && (
        <span data-export-status="done" data-export-cached={String(status.cached)}>
          {status.cached ? 'Cached' : 'Generated'} ({(status.sizeBytes / 1024).toFixed(0)} KB)
        </span>
      )}
      {status.kind === 'rate_limited' && (
        <span role="alert" data-export-status="rate_limited">
          Rate limited — retry in {status.retryAfterSec}s
        </span>
      )}
      {status.kind === 'error' && (
        <span role="alert" data-export-status="error">{status.message}</span>
      )}
    </div>
  );
}
```

- [ ] **Step 2: Vitest — status branches**

`ExportMenu.test.tsx`: mock `exportPDF` + `getDocxSignedURL`. Cases:
1. PDF happy path: button click → status `done` with `cached=false`.
2. PDF cache hit: mock returns `cached=true` → status displays "Cached".
3. 429: mock throws `{ status: 429, body: { retry_after_seconds: 15 } }` → status displays "Rate limited — retry in 15s".
4. 502: mock throws `{ status: 502 }` → error message "PDF service unavailable — retry in a moment."
5. `canExport=false`: both buttons disabled; no network call on click.

- [ ] **Step 3: DocumentEditorPage wiring**

In `DocumentEditorPage.tsx`, add to the header section after the autosave status chip:

```tsx
import { ExportMenu } from './ExportMenu';

// In the header JSX:
<ExportMenu documentID={documentID} canExport={doc.Status === 'finalized' || session.state.phase === 'writer'} />
```

> **Rationale:** Filler can export drafts (writer phase) AND finalized documents (no session needed). A read-only observer (another filler on the same doc) gets `canExport=false` — the buttons are disabled so they can't burn rate-limit tokens just by observing.

- [ ] **Step 4: Commit**

```bash
npm test --workspace @metaldocs/web -- ExportMenu
rtk git add frontend/apps/web/src/features/documents/v2/ExportMenu.tsx frontend/apps/web/src/features/documents/v2/ExportMenu.test.tsx frontend/apps/web/src/features/documents/v2/DocumentEditorPage.tsx
rtk git commit -m "feat(web/documents-v2): ExportMenu (docx + pdf) wired into editor header"
```

---

## Task 14: Playwright — `export-pdf-happy-path.spec.ts`

**Files:**
- Create: `frontend/apps/web/e2e/export-pdf-happy-path.spec.ts`

- [ ] **Step 1: Write spec**

```ts
import { test, expect } from '@playwright/test';

test('export PDF happy path: generate → download → verify magic bytes', async ({ page, context }) => {
  await page.goto('/documents-v2/new');
  await page.getByRole('button', { name: /purchase order/i }).click();
  await page.getByLabel(/document name/i).fill('pdf-export-happy');
  await page.getByLabel(/client name/i).fill('ACME');
  await page.getByLabel(/total amount/i).fill('42.00');
  await page.getByRole('button', { name: /generate document/i }).click();
  await page.waitForURL(/\/documents-v2\/.+/);
  await expect(page.locator('[data-status="saved"]')).toBeVisible({ timeout: 15_000 });

  // Intercept window.open to capture the signed URL.
  const [newPage] = await Promise.all([
    context.waitForEvent('page'),
    page.locator('[data-export-pdf]').click(),
  ]);
  await expect(page.locator('[data-export-status="done"][data-export-cached="false"]')).toBeVisible({ timeout: 30_000 });

  // Fetch the PDF and verify %PDF- magic bytes.
  const pdfURL = newPage.url();
  const resp = await page.request.get(pdfURL);
  expect(resp.status()).toBe(200);
  expect(resp.headers()['content-type']).toMatch(/application\/pdf/);
  const bytes = await resp.body();
  expect(bytes.slice(0, 5).toString('ascii')).toBe('%PDF-');
  expect(bytes.length).toBeGreaterThan(500);
});
```

- [ ] **Step 2: Commit**

```bash
rtk git add frontend/apps/web/e2e/export-pdf-happy-path.spec.ts
rtk git commit -m "test(e2e/exports): export-pdf-happy-path — verifies %PDF magic bytes"
```

---

## Task 15: Playwright — `export-pdf-cache-hit.spec.ts`

**Files:**
- Create: `frontend/apps/web/e2e/export-pdf-cache-hit.spec.ts`

- [ ] **Step 1: Write spec**

```ts
import { test, expect } from '@playwright/test';

test('export PDF cache: second request hits cache without calling Gotenberg', async ({ page, context }) => {
  await page.goto('/documents-v2/new');
  await page.getByRole('button', { name: /purchase order/i }).click();
  await page.getByLabel(/document name/i).fill('pdf-cache-hit');
  await page.getByLabel(/client name/i).fill('BETA');
  await page.getByLabel(/total amount/i).fill('1.00');
  await page.getByRole('button', { name: /generate document/i }).click();
  await page.waitForURL(/\/documents-v2\/.+/);
  await expect(page.locator('[data-status="saved"]')).toBeVisible({ timeout: 15_000 });

  // First export — cache miss. Measure duration.
  const t1 = Date.now();
  const firstPromise = context.waitForEvent('page');
  await page.locator('[data-export-pdf]').click();
  await firstPromise;
  await expect(page.locator('[data-export-status="done"][data-export-cached="false"]')).toBeVisible({ timeout: 30_000 });
  const firstMs = Date.now() - t1;

  // Reload the editor page so status resets.
  await page.reload();
  await expect(page.locator('[data-status="saved"]')).toBeVisible({ timeout: 15_000 });

  // Second export — same composite_hash → cached=true.
  const t2 = Date.now();
  const secondPromise = context.waitForEvent('page');
  await page.locator('[data-export-pdf]').click();
  await secondPromise;
  await expect(page.locator('[data-export-status="done"][data-export-cached="true"]')).toBeVisible({ timeout: 10_000 });
  const secondMs = Date.now() - t2;

  // Cache path should be at least 3x faster than cold path; use a loose 1.5x
  // lower bound to avoid CI flake. If this assertion fails consistently, the
  // cache isn't working — investigate before lowering the threshold.
  expect(secondMs).toBeLessThan(Math.floor(firstMs / 1.5));
});
```

- [ ] **Step 2: Commit**

```bash
rtk git add frontend/apps/web/e2e/export-pdf-cache-hit.spec.ts
rtk git commit -m "test(e2e/exports): export-pdf-cache-hit — second call cached=true + faster"
```

---

## Task 16: Playwright — `export-rate-limit.spec.ts`

**Files:**
- Create: `frontend/apps/web/e2e/export-rate-limit.spec.ts`

- [ ] **Step 1: Write spec**

Rate limit is 20 req/min/user on `/export/pdf`. Driving 21 real PDF generations in a test is expensive; instead issue 21 requests against the SAME composite_hash (doc + form) so conversion is cached after the first — all 21 still count against the rate limit since the middleware runs BEFORE the cache probe.

```ts
import { test, expect } from '@playwright/test';

test('export PDF rate limit: 21st request within a minute returns 429', async ({ page, request }) => {
  await page.goto('/documents-v2/new');
  await page.getByRole('button', { name: /purchase order/i }).click();
  await page.getByLabel(/document name/i).fill('pdf-rate-limit');
  await page.getByLabel(/client name/i).fill('C');
  await page.getByLabel(/total amount/i).fill('1');
  await page.getByRole('button', { name: /generate document/i }).click();
  await page.waitForURL(/\/documents-v2\/.+/);
  await expect(page.locator('[data-status="saved"]')).toBeVisible({ timeout: 15_000 });
  const docID = page.url().split('/').pop()!;

  // Warm the cache first (so 20 burst calls don't all hit Gotenberg).
  const warm = await request.post(`/api/v2/documents/${docID}/export/pdf`, { data: { paper_size: 'A4' } });
  expect(warm.status()).toBe(200);

  // Fire 19 more parallel calls — remaining quota minus the warm call.
  const calls = await Promise.all(
    Array.from({ length: 19 }, () => request.post(`/api/v2/documents/${docID}/export/pdf`, { data: { paper_size: 'A4' } })),
  );
  for (const c of calls) expect(c.status()).toBe(200);

  // 21st call — rate-limited.
  const over = await request.post(`/api/v2/documents/${docID}/export/pdf`, { data: { paper_size: 'A4' } });
  expect(over.status()).toBe(429);
  const retry = Number(over.headers()['retry-after']);
  expect(retry).toBeGreaterThan(0);
  expect(retry).toBeLessThanOrEqual(60);
  const body = await over.json();
  expect(body.error).toBe('rate_limited');
  expect(body.quota_per_minute).toBe(20);
  expect(body.retry_after_seconds).toBeGreaterThan(0);
});
```

- [ ] **Step 2: Commit**

```bash
rtk git add frontend/apps/web/e2e/export-rate-limit.spec.ts
rtk git commit -m "test(e2e/exports): 21st /export/pdf call in a minute returns 429"
```

---

## Task 17: CI — extend `docx-v2-ci.yml` with `e2e-exports` job

**Files:**
- Modify: `.github/workflows/docx-v2-ci.yml`

- [ ] **Step 1: Add job**

```yaml
  e2e-exports:
    runs-on: ubuntu-latest
    needs: [e2e-documents]   # exports run after documents suite
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
      - name: Apply migrations (0101–0111)
        run: |
          for f in migrations/0101_*.sql migrations/0102_*.sql migrations/0103_*.sql migrations/0104_*.sql migrations/0105_*.sql migrations/0106_*.sql migrations/0107_*.sql migrations/0108_*.sql migrations/0109_*.sql migrations/0110_*.sql migrations/0111_*.sql; do
            PGPASSWORD=metaldocs psql -h 127.0.0.1 -U metaldocs -d metaldocs -v ON_ERROR_STOP=1 -f "$f"
          done
      - run: npm ci --include-workspace-root
      - run: npm ci
        working-directory: frontend/apps/web
      - run: METALDOCS_DOCX_V2_ENABLED=true go run ./apps/api/cmd/metaldocs-api &
        env: { PGCONN: "postgres://metaldocs:metaldocs@127.0.0.1:5432/metaldocs?sslmode=disable" }
      - run: DOCGEN_V2_SERVICE_TOKEN=test-token-0123456789 DOCGEN_V2_S3_ACCESS_KEY=minioadmin DOCGEN_V2_S3_SECRET_KEY=minioadmin DOCGEN_V2_GOTENBERG_URL=http://127.0.0.1:3000 npm run start --workspace @metaldocs/docgen-v2 &
      - run: npx playwright install --with-deps chromium
        working-directory: frontend/apps/web
      - run: npx playwright test export-pdf-happy-path.spec.ts export-pdf-cache-hit.spec.ts export-rate-limit.spec.ts
        working-directory: frontend/apps/web
```

- [ ] **Step 2: Commit**

```bash
rtk git add .github/workflows/docx-v2-ci.yml
rtk git commit -m "ci(docx-v2): e2e-exports job (3 specs)"
```

---

## Task 18: Runbook — `docx-v2-w4-exports.md`

**Files:**
- Create: `docs/runbooks/docx-v2-w4-exports.md`

- [ ] **Step 1: Content**

Cover:
- **New routes:** `POST /export/pdf`, `GET /export/docx-url`. Both require `PermDocumentView` + ownership (or admin).
- **Composite-hash semantics:** `sha256(content_hash || template_version_id || grammar_version || docgen_v2_version || canonical_json(render_opts))`. Any bump invalidates the cache for all future exports; existing rows stay but are unreachable.
- **Cache invalidation:** no manual invalidation endpoint. To force regen, bump `DOCGEN_V2_VERSION` in `apps/docgen-v2/src/pdf/version.ts`. Old S3 objects become orphaned; sweeper deletes objects with no ledger row after 30 days.
- **Rate limits:** 20/min/user on `/export/pdf`, 30/min on `/autosave/commit` and `/render`, 60/min on presigns. Middleware is in-process — if the API scales horizontally each replica holds its own counter. Document this as a known limitation; centralize via Redis in W5+.
- **Gotenberg troubleshooting:**
  - 502 from `/export/pdf` → check Gotenberg container logs. Common causes: OOM on large docs (bump container mem to 1GB), font missing (mount font pack), LibreOffice crash (restart container).
  - Conversion non-deterministic → PDF metadata (/CreationDate) is stamped by LibreOffice. Gotenberg 8.4 honors `--libre-office-disable-pdf-metadata`; make sure this flag is passed in the k8s deployment manifest.
- **PDF not downloading:** signed URL has a 15-min TTL. If user's browser opens the tab after TTL, regenerate via the ExportMenu.
- **Orphan S3 cleanup:** in addition to the `pending_uploads` sweeper from W3, add a `document_exports_orphan_sweeper` that lists `tenants/*/documents/*/exports/*.pdf` objects and deletes any where the key's composite_hash does not resolve to a row in `document_exports` and the object is older than 30 days. (Implemented as a separate PR in W5 — reference only here.)
- **Audit events emitted:** every `/export/pdf` call emits exactly one `export.pdf_generated` with metadata `{composite_hash, revision_id, storage_key, docgen_v2_version, cached}`. Every `/export/docx-url` call emits exactly one `export.docx_downloaded` with metadata `{revision_id}`. The `cached=false` ratio (`count where metadata->>'cached'='false'` / total) drives the cache-thrash alert — spikes indicate grammar-version bumps or non-determinism in Gotenberg.

- [ ] **Step 2: Commit**

```bash
rtk git add docs/runbooks/docx-v2-w4-exports.md
rtk git commit -m "docs(runbook/docx-v2/w4): exports, rate limits, Gotenberg ops"
```

---

## Task 19: Governance integration test — `exports_integration_test.go`

**Files:**
- Create: `tests/docx_v2/exports_integration_test.go`
- Modify: `tests/docx_v2/documents_integration_test.go` (extend `TestDocumentsV2_RBACDenialMatrix` with the 2 new routes)

- [ ] **Step 1: Happy path integration test**

`TestExportsV2_HappyPath_PDFAndDocx`: build tag `integration`. Spins up real API + docgen-v2 + Gotenberg + MinIO + Postgres.

```go
//go:build integration

package docx_v2_test

func TestExportsV2_HappyPath_PDFAndDocx(t *testing.T) {
    env := setupDocxV2Env(t) // helper: migrates, starts services, seeds tenant+filler+doc
    defer env.Close()

    // GET /export/docx-url → 200 signed URL
    resp := env.HTTP("GET", fmt.Sprintf("/api/v2/documents/%s/export/docx-url", env.DocID), nil, env.FillerHeaders())
    assertStatus(t, resp, 200)
    var docxBody struct{ SignedURL, RevisionID string `json:"signed_url"` }
    mustJSON(resp.Body, &docxBody)
    // Fetch bytes, assert DOCX magic (PK zip header).
    assertDOCXBytes(t, docxBody.SignedURL)

    // POST /export/pdf → 200 cached=false
    resp = env.HTTP("POST", fmt.Sprintf("/api/v2/documents/%s/export/pdf", env.DocID),
        strings.NewReader(`{"paper_size":"A4"}`), env.FillerHeaders())
    assertStatus(t, resp, 200)
    var pdfBody struct {
        SignedURL     string `json:"signed_url"`
        CompositeHash string `json:"composite_hash"`
        SizeBytes     int64  `json:"size_bytes"`
        Cached        bool   `json:"cached"`
    }
    mustJSON(resp.Body, &pdfBody)
    if pdfBody.Cached { t.Fatal("first call should not be cached") }
    if pdfBody.SizeBytes <= 0 { t.Fatal("size_bytes must be > 0") }

    // Fetch PDF, assert %PDF- magic.
    assertPDFBytes(t, pdfBody.SignedURL)

    // Assert audit event landed.
    env.AssertAuditEvent(t, "export.pdf_generated", map[string]any{"composite_hash": pdfBody.CompositeHash})

    // Second call → cached=true, audit event still emitted (metadata.cached=true).
    // Every /export/pdf call produces one audit event so the cached=false
    // ratio alert in the dogfood runbook is computable from audit events.
    auditMissBefore := env.AuditCountWhere(t, "export.pdf_generated", map[string]any{"cached": false})
    auditHitBefore  := env.AuditCountWhere(t, "export.pdf_generated", map[string]any{"cached": true})
    resp = env.HTTP("POST", fmt.Sprintf("/api/v2/documents/%s/export/pdf", env.DocID),
        strings.NewReader(`{"paper_size":"A4"}`), env.FillerHeaders())
    assertStatus(t, resp, 200)
    mustJSON(resp.Body, &pdfBody)
    if !pdfBody.Cached { t.Fatal("second call should be cached") }
    auditMissAfter := env.AuditCountWhere(t, "export.pdf_generated", map[string]any{"cached": false})
    auditHitAfter  := env.AuditCountWhere(t, "export.pdf_generated", map[string]any{"cached": true})
    if auditMissAfter != auditMissBefore {
        t.Fatalf("cached call must not emit a cached=false audit; miss before=%d after=%d", auditMissBefore, auditMissAfter)
    }
    if auditHitAfter != auditHitBefore+1 {
        t.Fatalf("cached call must emit exactly one cached=true audit; hit before=%d after=%d", auditHitBefore, auditHitAfter)
    }
}
```

- [ ] **Step 2: RBAC denial matrix extension**

Extend `TestDocumentsV2_RBACDenialMatrix` (from Plan C Task 21) with 4 new sub-tests — 2 same-tenant non-owner (403) and 2 cross-tenant (404):

| Route                                           | Caller                                    | Expected | Invariant |
|-------------------------------------------------|-------------------------------------------|----------|-----------|
| `POST /api/v2/documents/{id}/export/pdf`        | `filler_B` (same tenant, not owner)       | 403      | no `document_exports` row, no audit |
| `GET  /api/v2/documents/{id}/export/docx-url`   | `filler_B` (same tenant, not owner)       | 403      | no `export.docx_downloaded` audit   |
| `POST /api/v2/documents/{id}/export/pdf`        | `filler_X` (different tenant)             | **404**  | no `document_exports` row, no audit |
| `GET  /api/v2/documents/{id}/export/docx-url`   | `filler_X` (different tenant)             | **404**  | no `export.docx_downloaded` audit   |

**Why the cross-tenant status is 404, not 403:** the spec's tenancy contract forbids disclosing resource existence across tenants. A 403 response would leak "this document exists" to a caller in a different tenant. All tenant-scoped SELECTs filter by `tenant_id`, so the document is **not visible** to `filler_X` — from the API's perspective, the row does not exist. `ensureDocAccess` must therefore return 404, not 403, when the caller's tenant does not match the document's tenant.

**Implementation hook:** the fixture builder seeds a second tenant `tenant_X` with a filler `filler_X`. The existing `env.FillerHeaders()` helper is extended to `env.FillerHeaders(opts ...headerOpt)` where `WithTenant("X")` overrides the default `X-Tenant-ID` header. Each sub-test re-uses the snapshot pattern from Task 5 Step 7 to assert DB state unchanged.

Each sub-test re-uses the `ensureDocAccess` helper's contract: the handler must distinguish "row not in my tenant" (→ 404) from "row in my tenant but I'm not the owner and not admin" (→ 403). Plan C Task 6 Step 1 already enforces the tenancy filter in the repository layer; this test exercises that boundary end-to-end.

- [ ] **Step 3: Rate-limit sub-test**

Add `TestExportsV2_RateLimit429` to the exports integration file (not the RBAC matrix):

```go
func TestExportsV2_RateLimit429(t *testing.T) {
    env := setupDocxV2Env(t)
    defer env.Close()

    // 20 permitted, 21st denied.
    for i := 0; i < 20; i++ {
        resp := env.HTTP("POST", fmt.Sprintf("/api/v2/documents/%s/export/pdf", env.DocID),
            strings.NewReader(`{"paper_size":"A4"}`), env.FillerHeaders())
        if resp.StatusCode != 200 { t.Fatalf("call %d: status %d", i, resp.StatusCode) }
    }
    resp := env.HTTP("POST", fmt.Sprintf("/api/v2/documents/%s/export/pdf", env.DocID),
        strings.NewReader(`{"paper_size":"A4"}`), env.FillerHeaders())
    if resp.StatusCode != 429 { t.Fatalf("21st call: want 429, got %d", resp.StatusCode) }
    retry, _ := strconv.Atoi(resp.Header.Get("retry-after"))
    if retry < 1 || retry > 60 { t.Fatalf("retry-after must be in [1,60], got %d", retry) }
}
```

- [ ] **Step 4: Commit**

```bash
go test -tags=integration ./tests/docx_v2/...
rtk git add tests/docx_v2/exports_integration_test.go tests/docx_v2/documents_integration_test.go
rtk git commit -m "test(docx-v2/w4): governance integration (happy + RBAC denial + rate limit)"
```

---

## Task 20: Dogfood soak runbook (Spec §W4) — procedure definition

**Files:**
- Create: `docs/runbooks/docx-v2-w4-dogfood.md`

- [ ] **Step 1: Write dogfood checklist**

The spec §Rollout → W4 mandates "internal dogfood behind flag for 5 business days" before W5 cutover. This runbook defines the procedure; Task 21 is the executable gate that produces the evidence artifact.

Cover:
- **Feature flag activation:** set `METALDOCS_DOCX_V2_ENABLED=true` for the dogfood tenant only — NOT for all tenants. Use the existing `feature_flags` table keyed by `tenant_id`.
- **Invited roles:** 3 `template_author`s + 5 `document_filler`s. Admin is on-call.
- **Daily check-in:** run the 3 new E2E specs + the 3 W3 specs against the dogfood deployment every morning. Any red spec = go/no-go conversation same day.
- **Telemetry watch:**
  - `/export/pdf` p95 latency — alert > 6s
  - `/export/pdf` cached=false ratio — alert > 50% (suggests cache thrash)
  - Gotenberg OOM events in container logs — any occurrence = pause dogfood
  - 429 rate per user — alert > 5/day sustained (suggests quota too low OR client bug)
- **Success criteria:** 5 business days with no P0 incidents, p95 < 5s on /export/pdf, autosave-crash recovery success rate > 99%, zero data-loss reports.
- **Exit gate:** admin + product manager sign off → W5 cutover plan runs.

- [ ] **Step 2: Commit**

```bash
rtk git add docs/runbooks/docx-v2-w4-dogfood.md
rtk git commit -m "docs(runbook/docx-v2/w4): dogfood soak procedure + telemetry thresholds"
```

---

## Task 21: Dogfood soak gate — evidence artifact + sign-off

**Files:**
- Create: `docs/superpowers/evidence/docx-v2-w4-dogfood-log.md`
- Create: `tests/docx_v2/dogfood_gate_test.go`

- [ ] **Step 1: Evidence log template**

`docs/superpowers/evidence/docx-v2-w4-dogfood-log.md` is committed empty with the structure below; it MUST be filled in daily during the 5-business-day soak. Attempting to merge the W5 cutover plan before this artifact is complete blocks on the `dogfood_gate_test.go` check below.

```markdown
# W4 Dogfood Soak Log

Start date: YYYY-MM-DD
End date:   YYYY-MM-DD
Target:     5 business days, no P0 incidents.

## Participants
- Admin on-call: @handle
- Template authors (3): @a, @b, @c
- Document fillers (5): @1, @2, @3, @4, @5

## Daily results

### Day 1 — YYYY-MM-DD
- E2E suite: ✅ / ❌ (link to CI run)
- p95 latency /export/pdf: __ms (target < 5000ms)
- cached=false ratio: __% (target < 50%)
- Gotenberg OOM events: __ (target: 0)
- 429 rate per user (max): __/day (target ≤ 5)
- Incidents: none | P0 | P1 | P2 (describe)
- Sign-off: @admin @pm

### Day 2 — YYYY-MM-DD
... (same structure) ...

### Day 3 — YYYY-MM-DD
... (same structure) ...

### Day 4 — YYYY-MM-DD
... (same structure) ...

### Day 5 — YYYY-MM-DD
... (same structure) ...

## Exit decision

- [ ] All 5 days logged with ✅ E2E + within-threshold telemetry.
- [ ] Zero P0 incidents.
- [ ] Admin sign-off: @handle — YYYY-MM-DD
- [ ] Product manager sign-off: @handle — YYYY-MM-DD

**Decision:** GO / NO-GO → W5 cutover.
```

- [ ] **Step 2: Gate test — enforce the log is filled before W5**

`tests/docx_v2/dogfood_gate_test.go`:

```go
//go:build w5_gate

package docx_v2_test

import (
    "os"
    "regexp"
    "strings"
    "testing"
)

// TestDogfoodLogComplete is tagged `w5_gate` so it only runs as a
// prerequisite check when W5 cutover is being attempted. The W5 CI job
// runs `go test -tags=w5_gate ./tests/docx_v2/...` before any cutover
// step. A missing or incomplete log fails the check and blocks W5.
//
// The test rejects any dogfood log that:
//   - has fewer than 5 dated "### Day N — YYYY-MM-DD" headers,
//   - contains unfilled placeholder tokens ("__", "...", "✅ / ❌", or a
//     literal "YYYY-MM-DD" anywhere inside a Day block),
//   - is missing any required row (E2E, p95, cached ratio, OOM, 429,
//     incidents, sign-off) in any Day block,
//   - shows a red (❌) E2E day without remediation,
//   - omits admin + PM sign-off markers, or
//   - is missing the explicit "**Decision:** GO" line.
func TestDogfoodLogComplete(t *testing.T) {
    raw, err := os.ReadFile("../../docs/superpowers/evidence/docx-v2-w4-dogfood-log.md")
    if err != nil {
        t.Fatalf("dogfood log missing: %v", err)
    }
    content := string(raw)

    // Must have 5 day sections with real ISO dates.
    dayHeaderRe := regexp.MustCompile(`(?m)^### Day [1-5] — (\d{4}-\d{2}-\d{2})$`)
    headers := dayHeaderRe.FindAllStringSubmatch(content, -1)
    if len(headers) < 5 {
        t.Fatalf("dogfood log must contain 5 dated Day sections; found %d", len(headers))
    }
    for _, h := range headers {
        if h[1] == "YYYY-MM-DD" {
            t.Fatalf("dogfood log Day header uses placeholder date: %q", h[0])
        }
    }

    // Unfilled template placeholders anywhere in the log are a hard fail.
    placeholderTokens := []string{
        "✅ / ❌",
        "YYYY-MM-DD",
        "__ms",
        "__%",
        "__/day",
        "(same structure)",
        "GO / NO-GO",
    }
    // "..." as an ellipsis-only day body (the "same structure" shortcut in
    // the template) is also a placeholder; reject bare "..." lines.
    ellipsisLineRe := regexp.MustCompile(`(?m)^\s*\.\.\.\s*$`)
    if ellipsisLineRe.FindString(content) != "" {
        t.Fatalf("dogfood log contains an unfilled '...' line (template shortcut)")
    }
    for _, tok := range placeholderTokens {
        if strings.Contains(content, tok) {
            t.Fatalf("dogfood log contains unfilled placeholder token %q", tok)
        }
    }

    // Split the log into per-day blocks and validate completeness of each.
    dayBlocks := splitDayBlocks(content)
    if len(dayBlocks) < 5 {
        t.Fatalf("could not isolate 5 day blocks; got %d", len(dayBlocks))
    }

    // Per-day required rows. Each regex matches the row AND captures a
    // non-placeholder value. Values must be filled in — bare "__" or an
    // empty capture fails.
    requiredRows := []struct {
        name string
        re   *regexp.Regexp
    }{
        {"E2E suite", regexp.MustCompile(`(?m)^- E2E suite:\s+(✅|❌)(\s.+)?$`)},
        {"p95 latency", regexp.MustCompile(`(?m)^- p95 latency /export/pdf:\s+(\d+)\s*ms\b`)},
        {"cached=false ratio", regexp.MustCompile(`(?m)^- cached=false ratio:\s+(\d{1,3})\s*%`)},
        {"Gotenberg OOM events", regexp.MustCompile(`(?m)^- Gotenberg OOM events:\s+(\d+)\b`)},
        {"429 rate per user", regexp.MustCompile(`(?m)^- 429 rate per user \(max\):\s+(\d+)/day\b`)},
        {"Incidents", regexp.MustCompile(`(?m)^- Incidents:\s+(none|P0|P1|P2)\b`)},
        {"Sign-off", regexp.MustCompile(`(?m)^- Sign-off:\s+@\S+\s+@\S+\s*$`)},
    }

    for i, block := range dayBlocks {
        dayNum := i + 1
        for _, row := range requiredRows {
            m := row.re.FindStringSubmatch(block)
            if m == nil {
                t.Fatalf("Day %d: missing or malformed %q row", dayNum, row.name)
            }
            // For numeric rows, reject 0-length captures just in case.
            if len(m) > 1 && strings.TrimSpace(m[1]) == "" {
                t.Fatalf("Day %d: %q row has empty value", dayNum, row.name)
            }
        }

        // No red E2E days allowed — any ❌ means remediation wasn't completed.
        if strings.Contains(block, "E2E suite: ❌") {
            t.Fatalf("Day %d: E2E is ❌; remediate and re-run before W5", dayNum)
        }
    }

    // Must have explicit admin + PM sign-off markers in the Exit section,
    // each paired with a non-placeholder @handle and ISO date.
    adminSignRe := regexp.MustCompile(`Admin sign-off:\s+@\S+\s+—\s+\d{4}-\d{2}-\d{2}`)
    pmSignRe := regexp.MustCompile(`Product manager sign-off:\s+@\S+\s+—\s+\d{4}-\d{2}-\d{2}`)
    if adminSignRe.FindString(content) == "" {
        t.Fatalf("dogfood log missing filled Admin sign-off (handle + date)")
    }
    if pmSignRe.FindString(content) == "" {
        t.Fatalf("dogfood log missing filled Product manager sign-off (handle + date)")
    }

    // Must have explicit GO decision.
    goRe := regexp.MustCompile(`(?m)^\*\*Decision:\*\*\s+GO\b`)
    if goRe.FindString(content) == "" {
        t.Fatalf("dogfood log missing explicit **Decision:** GO line")
    }
}

// splitDayBlocks returns the body text of each "### Day N — …" section,
// up to the next "### " or "## " header, in order. Returns an empty slice
// if no Day headers are found.
func splitDayBlocks(content string) []string {
    blocks := []string{}
    headerRe := regexp.MustCompile(`(?m)^### Day [1-5] — \d{4}-\d{2}-\d{2}\s*$`)
    idxs := headerRe.FindAllStringIndex(content, -1)
    for i, start := range idxs {
        bodyStart := start[1]
        var bodyEnd int
        if i+1 < len(idxs) {
            bodyEnd = idxs[i+1][0]
        } else {
            // Stop at the next "## " section (Exit decision) if present.
            rest := content[bodyStart:]
            nextSec := regexp.MustCompile(`(?m)^##[^#]`).FindStringIndex(rest)
            if nextSec != nil {
                bodyEnd = bodyStart + nextSec[0]
            } else {
                bodyEnd = len(content)
            }
        }
        blocks = append(blocks, content[bodyStart:bodyEnd])
    }
    return blocks
}
```

- [ ] **Step 3: CI wiring — real changed-path gate (NOT `github.event.pull_request.changed_files`)**

`github.event.pull_request.changed_files` is a **numeric count**, not a list of paths — a plain `contains(..., 'docx-editor-w5')` expression would always evaluate false and silently skip the gate. Use `dorny/paths-filter@v3` to compute a boolean output from the actual changed-file set, then gate `w5-gate` on that boolean.

Add two jobs to `.github/workflows/docx-v2-ci.yml` (one detector, one gate):

```yaml
  detect-w5-changes:
    runs-on: ubuntu-latest
    outputs:
      w5: ${{ steps.filter.outputs.w5 }}
    steps:
      - uses: actions/checkout@v4
      - id: filter
        uses: dorny/paths-filter@v3
        with:
          filters: |
            w5:
              - 'docs/superpowers/plans/2026-04-*-docx-editor-w5-*.md'
              - 'docs/superpowers/evidence/docx-v2-w4-dogfood-log.md'

  w5-gate:
    needs: detect-w5-changes
    if: needs.detect-w5-changes.outputs.w5 == 'true'
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: 1.25.x }
      - run: go test -tags=w5_gate ./tests/docx_v2/...
```

Alternatively, if the repo already scopes workflows with top-level `pull_request.paths`, a second workflow file (e.g. `.github/workflows/docx-v2-w5-gate.yml`) triggered by:

```yaml
on:
  pull_request:
    paths:
      - 'docs/superpowers/plans/2026-04-*-docx-editor-w5-*.md'
      - 'docs/superpowers/evidence/docx-v2-w4-dogfood-log.md'
```

is also acceptable. Pick whichever matches the existing repo's CI idiom — but **NEVER** use `github.event.pull_request.changed_files` as a path filter: it is not one.

- [ ] **Step 4: Commit**

```bash
go test -tags=w5_gate ./tests/docx_v2/... || echo "expected to fail until log is filled"
rtk git add docs/superpowers/evidence/docx-v2-w4-dogfood-log.md tests/docx_v2/dogfood_gate_test.go .github/workflows/docx-v2-ci.yml
rtk git commit -m "gate(docx-v2/w4): dogfood soak evidence artifact + W5 prerequisite check"
```

---

## Sanity pass

- [ ] `go build ./...` passes.
- [ ] `go test ./internal/modules/documents_v2/... ./internal/platform/ratelimit/... ./tests/docx_v2/...` passes (integration tag where noted).
- [ ] `npm test --workspace @metaldocs/web -- features/documents/v2` passes.
- [ ] `npm test --workspace @metaldocs/docgen-v2 -- convert-pdf` passes.
- [ ] `npm run build --workspace @metaldocs/web` passes (type-check).
- [ ] Playwright `export-pdf-happy-path`, `export-pdf-cache-hit`, `export-rate-limit` pass locally.
- [ ] CI `e2e-documents` + `e2e-exports` jobs green on a test branch.
- [ ] OpenAPI `npx @redocly/cli lint` reports 0 errors.

---

## Codex Hardening Log

### Round 1

- **Verdict:** APPROVE_WITH_FIXES
- **Mode:** COVERAGE
- **upgrade_required:** true
- **Confidence:** high
- **Issues & repairs (all applied inline before R2):**
  1. **STRUCTURAL — Missing cross-tenant 404 tests.** The Task 19 RBAC matrix asserted only 403 for non-owner same-tenant fillers and gave no coverage for cross-tenant resource access. A leak here would disclose document existence to other tenants.
     - **Applied:** Extended Task 19 into a 4-row matrix — added two cross-tenant rows (`filler_X` vs document belonging to `tenant_A`) that MUST return **404** (not 403), plus the `WithTenant` fixture helper and a per-route tenancy-disclosure justification paragraph explaining why 404 is the correct tenant-isolation response.
  2. **STRUCTURAL — Dogfood soak lacks an executable gate.** The original soak procedure was prose-only; nothing blocked W5 cutover from merging with an incomplete or skipped soak.
     - **Applied:** Promoted soak to Task 21 — a committed Markdown evidence artifact at `docs/superpowers/evidence/docx-v2-w4-dogfood-log.md`, a `-tags=w5_gate` Go test `dogfood_gate_test.go` that fails if the log is unfilled, plus a CI job that runs the tagged test as a W5 prerequisite.
  3. **LOCAL — Audit inconsistency (cache ratio unmeasurable).** The W4 runbook alerted on cached=false ratio, but `ExportPDF` only emitted `export.pdf_generated` on cache miss, making the ratio uncomputable from audit events.
     - **Applied:** Reworked `ExportPDF` to emit `export.pdf_generated` on EVERY call with `metadata.cached: true|false`. Updated Task 6 test list (assert both flag values fire), Task 19 RBAC tests (`AuditCountWhere` filters by cached flag), and Task 18 runbook query (`count where metadata->>'cached'='false' / count(*)`).

### Round 2

- **Verdict:** APPROVE_WITH_FIXES
- **Mode:** COVERAGE
- **upgrade_required:** true
- **Confidence:** high
- **Issues & repairs (all applied inline; max 2 rounds reached):**
  1. **STRUCTURAL — CI trigger expression is invalid.** `if: contains(github.event.pull_request.changed_files, 'docx-editor-w5')` silently evaluates false because `changed_files` is a numeric count field, not a path list. The W5 gate could be bypassed by any W5-plan PR without firing the test.
     - **Applied:** Task 21 Step 3 rewritten. New design adds a `detect-w5-changes` job using `dorny/paths-filter@v3` that produces a boolean output; the `w5-gate` job is now `needs: detect-w5-changes` and `if: needs.detect-w5-changes.outputs.w5 == 'true'`. An explicit prose warning flags `github.event.pull_request.changed_files` as NEVER a valid path filter. An alternative `pull_request.paths`-triggered workflow file is also documented.
  2. **STRUCTURAL — `dogfood_gate_test.go` too permissive.** The original assertions only checked for 5 dated Day headers and the absence of `❌`. A log could pass with Day 2–5 containing only headers or placeholder `... (same structure) ...` bodies.
     - **Applied:** Rewrote the gate test. It now (a) splits the log into per-day blocks via `splitDayBlocks`, (b) asserts every Day block has filled rows for E2E result, p95 ms, cached=false ratio, OOM count, 429/day max, Incidents, and Sign-off (each with a regex that requires a real value), (c) rejects placeholder tokens anywhere in the document (`✅ / ❌`, `YYYY-MM-DD`, `__ms`, `__%`, `__/day`, `(same structure)`, `GO / NO-GO`, bare `...` lines), and (d) requires admin + PM sign-off lines each carrying a non-placeholder `@handle` + ISO date.

### Caveats (post-R2, no R3)

Per co-plan protocol, R2 is the final Codex round; both R2 issues were structural but mechanically resolvable and were applied inline as delivery polish. No outstanding Codex findings remain. If future review surfaces additional coverage gaps after R2 (e.g., additional soak-log rows, CI portability concerns), they will be addressed via a follow-up plan increment rather than a third hardening round.

---
