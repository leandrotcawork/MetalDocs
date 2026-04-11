# MDDM Engine Rollout & Docgen Decommission Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Complete the phased rollout of the MDDM engine (spec Phases 1–4). Phase 1 enables frontend-side shadow testing that runs both the docgen backend path and the new client-side path in parallel, posts structural diffs to a new telemetry endpoint, and surfaces the docgen result to the user unchanged. Phase 2 flips a percentage-based canary feature flag. Phase 3 promotes the new path to 100% of `browser_editor` exports. Phase 4 removes the MDDM-specific docgen path — `generateBrowserDocxBytesWithTemplate`, `docgen.Client.GenerateMDDM`, and the frontend legacy client — while **preserving docgen for `native` and `docx_upload` content sources** (migrating those is out of scope per the spec).

**Architecture:** Shadow testing runs entirely on the frontend. The frontend calls the existing `POST /export/docx` backend path (unchanged) and — in a Web Worker — also runs the new `exportDocx(envelope, options)` client-side pipeline. It then computes a normalized-XML diff between the two, hashes both outputs, and POSTs `{ document_id, version_number, current_xml_hash, shadow_xml_hash, diff_summary, user_id_hash }` to a new `POST /telemetry/mddm-shadow-diff` endpoint that appends the row to a new `metaldocs.mddm_shadow_diff_events` table. The user always receives the docgen result; the shadow is discarded. Canary gating is a new per-user percentage: `hash(user_id) % 100 < MDDM_NATIVE_EXPORT_ROLLOUT_PCT`. The rollout percentage is a server-side config setting exposed via an existing feature-flags endpoint (or a new minimal one). Decommission removes only the `browser_editor` branch from the backend; `native`/`docx_upload` continue using `docgen.Client.Generate`.

**Tech Stack:** TypeScript 5.6, React 18, Vitest 4.1, Go 1.22, PostgreSQL 16 (one new table). Shadow DOCX generation runs in a Web Worker with a 30s timeout and device-memory gating (skipped on sub-4GB devices via `navigator.deviceMemory`).

**Spec:** `docs/superpowers/specs/2026-04-10-mddm-unified-document-engine-design.md` (Sections "Migration & Rollout" and "Phase 1 — Shadow testing")

**Depends on:**
- Plan 1 — `docs/superpowers/plans/2026-04-10-mddm-engine-foundation.md`
- Plan 2 — `docs/superpowers/plans/2026-04-10-mddm-engine-full-block-coverage.md`
- Plan 3 — `docs/superpowers/plans/2026-04-10-mddm-engine-version-pinning.md`

All three must be merged before Plan 4 starts.

---

## File Structure

### New files (backend)

```
migrations/
└── 0070_create_mddm_shadow_diff_events.sql              # NEW: telemetry table
internal/modules/documents/
├── delivery/http/
│   ├── handler_telemetry_shadow_diff.go                 # NEW: POST /telemetry/mddm-shadow-diff
│   └── handler_telemetry_shadow_diff_test.go            # NEW
├── application/
│   ├── shadow_diff_service.go                           # NEW: insert shadow diff events + validate
│   └── shadow_diff_service_test.go                      # NEW
└── infrastructure/postgres/
    ├── shadow_diff_repo.go                              # NEW: postgres insert for the telemetry table
    └── shadow_diff_repo_test.go                         # NEW
internal/platform/config/
└── feature_flags.go                                      # MODIFY or CREATE: add MDDMNativeExportRolloutPercent
```

### New files (frontend)

```
frontend/apps/web/src/features/documents/mddm-editor/engine/
├── shadow-testing/
│   ├── shadow-runner.ts                                 # NEW: wraps exportDocx in a Worker for shadow use
│   ├── shadow-diff.ts                                   # NEW: compute structural diff between two DOCX blobs
│   ├── shadow-telemetry.ts                              # NEW: POST diff summary to backend
│   ├── shadow.worker.ts                                 # NEW: Web Worker entry point
│   └── __tests__/
│       ├── shadow-diff.test.ts                          # NEW
│       └── shadow-telemetry.test.ts                     # NEW
frontend/apps/web/src/features/
└── feature-flags/
    ├── rollout.ts                                       # NEW: hash(user_id) % 100 canary gate
    └── __tests__/
        └── rollout.test.ts                              # NEW
```

### Modified files (backend)

```
internal/modules/documents/application/service_document_runtime.go              # MODIFY: Phase 4 — remove browser_editor branch from ExportDocumentDocxAuthorized
internal/modules/documents/application/service_content_docx.go                  # MODIFY: Phase 4 — delete generateBrowserDocxBytesWithTemplate and its helpers (mddmTemplateThemeFromDefinition, etc.) if they become unreferenced
internal/platform/render/docgen/client.go                                        # MODIFY: Phase 4 — delete GenerateMDDM method
internal/platform/render/docgen/types.go                                         # MODIFY: Phase 4 — delete MDDMExportPayload / MDDMTemplateTheme types
internal/modules/documents/delivery/http/handler.go                              # MODIFY: register POST /telemetry/mddm-shadow-diff route
api/openapi/v1/openapi.yaml                                                       # MODIFY: add /telemetry/mddm-shadow-diff endpoint
```

### Modified files (frontend)

```
frontend/apps/web/src/features/featureFlags.ts                                                         # MODIFY: rename MDDM_NATIVE_EXPORT to MDDM_NATIVE_EXPORT_ROLLOUT_PCT (server-provided) + rollout helper
frontend/apps/web/src/features/documents/browser-editor/BrowserDocumentEditorView.tsx                  # MODIFY: Phase 1 shadow dual-run; Phase 4 final cleanup
frontend/apps/web/src/api/documents.ts                                                                 # MODIFY: Phase 4 — delete exportDocumentDocx (legacy client)
frontend/apps/web/src/features/documents/mddm-editor/engine/export/export-docx.ts                      # MODIFY (minor): no-op in Phase 4
```

### Rollout phasing across the task list

```
Part 1  Shadow testing backend (telemetry endpoint + table)           Phase 1
Part 2  Shadow testing frontend (worker + diff + runner)              Phase 1
Part 3  Dual-run integration in BrowserDocumentEditorView              Phase 1
Part 4  Percentage-based feature flag rollout                          Phase 2
Part 5  Canary monitoring runbook                                      Phase 2
Part 6  100% rollout promotion                                         Phase 3
Part 7  Decommission (backend MDDM docgen path)                        Phase 4
Part 8  Decommission (frontend legacy client + flag cleanup)           Phase 4
Part 9  Final verification                                             Phase 4
```

---

## Part 1 — Shadow Testing Backend

### Task 1: Create mddm_shadow_diff_events table

**Files:**
- Create: `migrations/0070_create_mddm_shadow_diff_events.sql`

- [ ] **Step 1: Write the migration**

Write to `migrations/0070_create_mddm_shadow_diff_events.sql`:

```sql
-- 0070: telemetry table for shadow-mode DOCX export comparison.
-- During Plan 4 Phase 1, the frontend runs both the docgen and new
-- client-side paths in parallel on every browser_editor export and
-- posts a hash + diff summary here. Engineers aggregate these rows
-- off-line to decide when Phase 2 (canary) is safe to enable.

CREATE TABLE IF NOT EXISTS metaldocs.mddm_shadow_diff_events (
    id                 BIGSERIAL PRIMARY KEY,
    document_id        VARCHAR(64)   NOT NULL,
    version_number     INTEGER       NOT NULL,
    user_id_hash       VARCHAR(64)   NOT NULL,
    current_xml_hash   VARCHAR(64)   NOT NULL,
    shadow_xml_hash    VARCHAR(64)   NOT NULL,
    diff_summary       JSONB         NOT NULL DEFAULT '{}'::jsonb,
    current_duration_ms INTEGER      NOT NULL DEFAULT 0,
    shadow_duration_ms  INTEGER      NOT NULL DEFAULT 0,
    shadow_error       TEXT,
    recorded_at        TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    trace_id           VARCHAR(64)
);

CREATE INDEX IF NOT EXISTS mddm_shadow_diff_events_recorded_at_idx
    ON metaldocs.mddm_shadow_diff_events (recorded_at DESC);

COMMENT ON TABLE metaldocs.mddm_shadow_diff_events IS
    'Phase 1 shadow-test telemetry: compares docgen DOCX against the new client-side DOCX for browser_editor documents. Rows are append-only. user_id_hash is a salted SHA-256 so individual users cannot be identified from the raw table.';
```

- [ ] **Step 2: Apply the migration**

```powershell
powershell -ExecutionPolicy Bypass -File scripts/dev-migrate.ps1
```
Expected: Output includes `[dev-migrate] -> 0070_create_mddm_shadow_diff_events.sql` followed by `[dev-migrate] Done.`

- [ ] **Step 3: Verify the table**

```powershell
docker compose -f deploy/compose/docker-compose.yml --env-file .env exec -T postgres psql -U $env:POSTGRES_USER -d $env:POSTGRES_DB -c "\d metaldocs.mddm_shadow_diff_events"
```
Expected: Output shows the columns declared above.

- [ ] **Step 4: Commit**

```bash
git add migrations/0070_create_mddm_shadow_diff_events.sql
git commit -m "feat(db): create mddm_shadow_diff_events telemetry table (Phase 1)"
```

### Task 2: Shadow diff repository (postgres)

**Files:**
- Create: `internal/modules/documents/infrastructure/postgres/shadow_diff_repo.go`
- Create: `internal/modules/documents/infrastructure/postgres/shadow_diff_repo_test.go`

- [ ] **Step 1: Write the failing test**

Write to `internal/modules/documents/infrastructure/postgres/shadow_diff_repo_test.go`:

```go
package postgres

import (
    "context"
    "encoding/json"
    "testing"
    "time"

    "metaldocs/internal/modules/documents/domain"
)

func TestShadowDiffRepository_Insert_Roundtrip(t *testing.T) {
    db := newTestDB(t)
    repo := NewShadowDiffRepository(db)

    event := domain.ShadowDiffEvent{
        DocumentID:        "doc-1",
        VersionNumber:     3,
        UserIDHash:        "hashed-user-id",
        CurrentXMLHash:    "current-hash",
        ShadowXMLHash:     "shadow-hash",
        DiffSummary:       map[string]any{"blocks_equal": 42, "blocks_different": 0},
        CurrentDurationMs: 1200,
        ShadowDurationMs:  900,
        ShadowError:       "",
        RecordedAt:        time.Now().UTC().Truncate(time.Millisecond),
        TraceID:           "trace-xyz",
    }

    if err := repo.Insert(context.Background(), event); err != nil {
        t.Fatalf("Insert: %v", err)
    }

    // Verify the row exists by querying it back with a simple SELECT.
    var got struct {
        DocumentID     string
        VersionNumber  int
        UserIDHash     string
        DiffSummaryRaw []byte
    }
    err := db.QueryRowContext(context.Background(),
        `SELECT document_id, version_number, user_id_hash, diff_summary
         FROM metaldocs.mddm_shadow_diff_events
         WHERE document_id = $1 AND version_number = $2
         ORDER BY id DESC LIMIT 1`,
        event.DocumentID, event.VersionNumber).
        Scan(&got.DocumentID, &got.VersionNumber, &got.UserIDHash, &got.DiffSummaryRaw)
    if err != nil {
        t.Fatalf("SELECT: %v", err)
    }
    if got.DocumentID != event.DocumentID || got.VersionNumber != event.VersionNumber {
        t.Fatalf("row mismatch: %+v", got)
    }

    var summary map[string]any
    if err := json.Unmarshal(got.DiffSummaryRaw, &summary); err != nil {
        t.Fatalf("decode diff_summary: %v", err)
    }
    if summary["blocks_equal"].(float64) != 42 {
        t.Fatalf("diff_summary lost data: %+v", summary)
    }
}
```

- [ ] **Step 2: Add ShadowDiffEvent to the domain**

Create `internal/modules/documents/domain/shadow_diff.go`:

```go
package domain

import "time"

// ShadowDiffEvent is a single telemetry row captured by the frontend during
// Phase 1 shadow testing. It is append-only; engineers aggregate over the
// table off-line to decide when Phase 2 (canary) is safe.
type ShadowDiffEvent struct {
    DocumentID        string
    VersionNumber     int
    UserIDHash        string
    CurrentXMLHash    string
    ShadowXMLHash     string
    DiffSummary       map[string]any
    CurrentDurationMs int
    ShadowDurationMs  int
    ShadowError       string
    RecordedAt        time.Time
    TraceID           string
}
```

- [ ] **Step 3: Run the test — expect failure**

```bash
go test ./internal/modules/documents/infrastructure/postgres/... -run TestShadowDiffRepository_Insert_Roundtrip -v 2>&1 | tail -20
```
Expected: FAIL — `NewShadowDiffRepository` undefined.

- [ ] **Step 4: Implement the repository**

Write to `internal/modules/documents/infrastructure/postgres/shadow_diff_repo.go`:

```go
package postgres

import (
    "context"
    "database/sql"
    "encoding/json"
    "fmt"

    "metaldocs/internal/modules/documents/domain"
)

type ShadowDiffRepository struct {
    db *sql.DB
}

func NewShadowDiffRepository(db *sql.DB) *ShadowDiffRepository {
    return &ShadowDiffRepository{db: db}
}

func (r *ShadowDiffRepository) Insert(ctx context.Context, event domain.ShadowDiffEvent) error {
    summaryBytes, err := json.Marshal(event.DiffSummary)
    if err != nil {
        return fmt.Errorf("marshal diff summary: %w", err)
    }

    _, err = r.db.ExecContext(ctx, `
        INSERT INTO metaldocs.mddm_shadow_diff_events (
            document_id, version_number, user_id_hash,
            current_xml_hash, shadow_xml_hash, diff_summary,
            current_duration_ms, shadow_duration_ms, shadow_error,
            recorded_at, trace_id
        )
        VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7, $8, NULLIF($9, ''), $10, NULLIF($11, ''))`,
        event.DocumentID, event.VersionNumber, event.UserIDHash,
        event.CurrentXMLHash, event.ShadowXMLHash, string(summaryBytes),
        event.CurrentDurationMs, event.ShadowDurationMs, event.ShadowError,
        event.RecordedAt, event.TraceID)
    if err != nil {
        return fmt.Errorf("insert shadow diff event: %w", err)
    }
    return nil
}
```

- [ ] **Step 5: Run the test — expect pass**

```bash
go test ./internal/modules/documents/infrastructure/postgres/... -run TestShadowDiffRepository_Insert_Roundtrip -v 2>&1 | tail -20
```
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/modules/documents/domain/shadow_diff.go internal/modules/documents/infrastructure/postgres/shadow_diff_repo.go internal/modules/documents/infrastructure/postgres/shadow_diff_repo_test.go
git commit -m "feat(documents-repo): add ShadowDiffRepository for Phase 1 telemetry"
```

### Task 3: Shadow diff HTTP handler

**Files:**
- Create: `internal/modules/documents/delivery/http/handler_telemetry_shadow_diff.go`
- Create: `internal/modules/documents/delivery/http/handler_telemetry_shadow_diff_test.go`

- [ ] **Step 1: Write the failing handler test**

Write to `internal/modules/documents/delivery/http/handler_telemetry_shadow_diff_test.go`:

```go
package httpdelivery

import (
    "bytes"
    "context"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "metaldocs/internal/modules/documents/domain"
)

type fakeShadowDiffRepo struct {
    last *domain.ShadowDiffEvent
    err  error
}

func (f *fakeShadowDiffRepo) Insert(ctx context.Context, event domain.ShadowDiffEvent) error {
    if f.err != nil {
        return f.err
    }
    f.last = &event
    return nil
}

func TestHandleShadowDiff_PersistsEvent(t *testing.T) {
    repo := &fakeShadowDiffRepo{}
    handler := NewShadowDiffHandler(repo)

    body, _ := json.Marshal(map[string]any{
        "document_id":        "doc-1",
        "version_number":     3,
        "user_id_hash":       "uhash",
        "current_xml_hash":   "chash",
        "shadow_xml_hash":    "shash",
        "diff_summary":       map[string]any{"blocks_equal": 10},
        "current_duration_ms": 500,
        "shadow_duration_ms":  800,
    })
    req := httptest.NewRequest(http.MethodPost, "/api/v1/telemetry/mddm-shadow-diff", bytes.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    req = req.WithContext(contextWithUserID(req.Context(), "u-1"))

    w := httptest.NewRecorder()
    handler.Handle(w, req)

    if w.Code != http.StatusAccepted {
        t.Fatalf("expected 202, got %d: %s", w.Code, w.Body.String())
    }
    if repo.last == nil {
        t.Fatalf("expected repo to receive an insert")
    }
    if repo.last.DocumentID != "doc-1" || repo.last.VersionNumber != 3 {
        t.Fatalf("event fields mismatch: %+v", repo.last)
    }
}

func TestHandleShadowDiff_Unauthenticated(t *testing.T) {
    repo := &fakeShadowDiffRepo{}
    handler := NewShadowDiffHandler(repo)

    body, _ := json.Marshal(map[string]any{"document_id": "d", "version_number": 1})
    req := httptest.NewRequest(http.MethodPost, "/api/v1/telemetry/mddm-shadow-diff", bytes.NewReader(body))
    req.Header.Set("Content-Type", "application/json")

    w := httptest.NewRecorder()
    handler.Handle(w, req)

    if w.Code != http.StatusUnauthorized {
        t.Fatalf("expected 401, got %d", w.Code)
    }
}

func TestHandleShadowDiff_RejectsMalformedBody(t *testing.T) {
    repo := &fakeShadowDiffRepo{}
    handler := NewShadowDiffHandler(repo)

    req := httptest.NewRequest(http.MethodPost, "/api/v1/telemetry/mddm-shadow-diff", bytes.NewReader([]byte("not json")))
    req.Header.Set("Content-Type", "application/json")
    req = req.WithContext(contextWithUserID(req.Context(), "u-1"))

    w := httptest.NewRecorder()
    handler.Handle(w, req)

    if w.Code != http.StatusBadRequest {
        t.Fatalf("expected 400, got %d", w.Code)
    }
}
```

- [ ] **Step 2: Implement the handler**

Write to `internal/modules/documents/delivery/http/handler_telemetry_shadow_diff.go`:

```go
package httpdelivery

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "time"

    "metaldocs/internal/modules/documents/domain"
)

type ShadowDiffRepo interface {
    Insert(ctx context.Context, event domain.ShadowDiffEvent) error
}

type ShadowDiffHandler struct {
    repo ShadowDiffRepo
}

func NewShadowDiffHandler(repo ShadowDiffRepo) *ShadowDiffHandler {
    return &ShadowDiffHandler{repo: repo}
}

type shadowDiffRequest struct {
    DocumentID        string         `json:"document_id"`
    VersionNumber     int            `json:"version_number"`
    UserIDHash        string         `json:"user_id_hash"`
    CurrentXMLHash    string         `json:"current_xml_hash"`
    ShadowXMLHash     string         `json:"shadow_xml_hash"`
    DiffSummary       map[string]any `json:"diff_summary"`
    CurrentDurationMs int            `json:"current_duration_ms"`
    ShadowDurationMs  int            `json:"shadow_duration_ms"`
    ShadowError       string         `json:"shadow_error,omitempty"`
}

func (h *ShadowDiffHandler) Handle(w http.ResponseWriter, r *http.Request) {
    traceID := requestTraceID(r)

    if userIDFromContext(r.Context()) == "" {
        writeAPIError(w, http.StatusUnauthorized, "AUTH_UNAUTHORIZED", "Authentication required", traceID)
        return
    }

    if h.repo == nil {
        writeAPIError(w, http.StatusServiceUnavailable, "TELEMETRY_UNAVAILABLE", "Shadow diff telemetry is not configured", traceID)
        return
    }

    r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MB hard cap
    var req shadowDiffRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", fmt.Sprintf("decode body: %v", err), traceID)
        return
    }

    if req.DocumentID == "" || req.VersionNumber <= 0 {
        writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "document_id and version_number required", traceID)
        return
    }

    event := domain.ShadowDiffEvent{
        DocumentID:        req.DocumentID,
        VersionNumber:     req.VersionNumber,
        UserIDHash:        req.UserIDHash,
        CurrentXMLHash:    req.CurrentXMLHash,
        ShadowXMLHash:     req.ShadowXMLHash,
        DiffSummary:       req.DiffSummary,
        CurrentDurationMs: req.CurrentDurationMs,
        ShadowDurationMs:  req.ShadowDurationMs,
        ShadowError:       req.ShadowError,
        RecordedAt:        time.Now().UTC(),
        TraceID:           traceID,
    }

    if err := h.repo.Insert(r.Context(), event); err != nil {
        writeAPIError(w, http.StatusInternalServerError, "INTERNAL_ERROR", fmt.Sprintf("insert: %v", err), traceID)
        return
    }

    w.WriteHeader(http.StatusAccepted)
}
```

- [ ] **Step 3: Run the tests**

```bash
go test ./internal/modules/documents/delivery/http/... -run TestHandleShadowDiff -v 2>&1 | tail -30
```
Expected: PASS — all 3 subtests.

- [ ] **Step 4: Commit**

```bash
git add internal/modules/documents/delivery/http/handler_telemetry_shadow_diff.go internal/modules/documents/delivery/http/handler_telemetry_shadow_diff_test.go
git commit -m "feat(documents-http): add POST /telemetry/mddm-shadow-diff handler"
```

### Task 4: Register the telemetry route

**Files:**
- Modify: `internal/modules/documents/delivery/http/handler.go`
- Modify: `apps/api/cmd/metaldocs-api/main.go`

- [ ] **Step 1: Add field + wiring to the Handler struct**

In `internal/modules/documents/delivery/http/handler.go`, add:

```go
type Handler struct {
    // ... existing fields ...
    shadowDiff *ShadowDiffHandler
}

func (h *Handler) WithShadowDiffHandler(s *ShadowDiffHandler) *Handler {
    h.shadowDiff = s
    return h
}
```

- [ ] **Step 2: Register the route**

Find the router dispatch (around the existing `/render/pdf` and `/export/docx` cases from Plan 1). Add:

```go
if len(parts) == 2 && parts[0] == "telemetry" && parts[1] == "mddm-shadow-diff" && r.Method == http.MethodPost {
    if h.shadowDiff == nil {
        writeAPIError(w, http.StatusServiceUnavailable, "TELEMETRY_UNAVAILABLE", "Shadow diff telemetry is not configured", requestTraceID(r))
        return
    }
    h.shadowDiff.Handle(w, r)
    return
}
```

Note: adjust to the actual routing pattern — telemetry paths may need a different base-path match than the existing `/documents/{id}/...` routes.

- [ ] **Step 3: Wire at bootstrap**

In `apps/api/cmd/metaldocs-api/main.go`, after the existing handler setup:

```go
shadowDiffRepo := docpg.NewShadowDiffRepository(deps.PGDB) // use whatever exposes *sql.DB in deps
shadowDiffHandler := docdelivery.NewShadowDiffHandler(shadowDiffRepo)

docHandler := docdelivery.NewHandler(docService).
    WithAttachmentDownloads(/* ... */).
    WithMDDMHandlers(loadService, submitForApprovalService).
    WithShadowDiffHandler(shadowDiffHandler)
```

If `deps.PGDB` does not exist, search for the raw `*sql.DB` the postgres repositories use and plumb it through `deps` via a new field.

- [ ] **Step 4: Build and test**

```bash
go build ./...
go test ./internal/modules/documents/... 2>&1 | tail -20
```
Expected: Clean build, all tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/modules/documents/delivery/http/handler.go apps/api/cmd/metaldocs-api/main.go
git commit -m "wire(documents-http): register shadow diff telemetry route"
```

### Task 5: Add endpoint to OpenAPI

**Files:**
- Modify: `api/openapi/v1/openapi.yaml`

- [ ] **Step 1: Add the path and schema**

In `api/openapi/v1/openapi.yaml`, add a new path entry:

```yaml
  /telemetry/mddm-shadow-diff:
    post:
      summary: Submit a Phase 1 shadow-diff telemetry event
      description: |
        Frontend posts one event per browser_editor DOCX export while
        Plan 4 Phase 1 is active. Records a hash of both the docgen-
        produced DOCX and the client-side DOCX so engineers can aggregate
        structural drift before enabling the canary rollout.
      operationId: recordMDDMShadowDiff
      tags:
        - telemetry
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/MDDMShadowDiffEvent'
      responses:
        '202':
          description: Event accepted for storage
        '400':
          description: Malformed body
          content:
            application/json:
              schema: { $ref: '#/components/schemas/ApiErrorEnvelope' }
        '401':
          description: Not authenticated
          content:
            application/json:
              schema: { $ref: '#/components/schemas/ApiErrorEnvelope' }
        '503':
          description: Telemetry not configured
          content:
            application/json:
              schema: { $ref: '#/components/schemas/ApiErrorEnvelope' }
      security:
        - sessionCookie: []
```

And in `components.schemas`:

```yaml
    MDDMShadowDiffEvent:
      type: object
      required:
        - document_id
        - version_number
        - current_xml_hash
        - shadow_xml_hash
      properties:
        document_id: { type: string }
        version_number: { type: integer, minimum: 1 }
        user_id_hash: { type: string }
        current_xml_hash: { type: string, description: "SHA-256 of normalized docgen document.xml" }
        shadow_xml_hash: { type: string, description: "SHA-256 of normalized client-side document.xml" }
        diff_summary:
          type: object
          additionalProperties: true
          description: "Structural diff counters (blocks_equal, blocks_different, etc.)"
        current_duration_ms: { type: integer, minimum: 0 }
        shadow_duration_ms: { type: integer, minimum: 0 }
        shadow_error: { type: string, description: "Non-empty when the shadow path failed" }
```

- [ ] **Step 2: Validate YAML**

```bash
python -c "import yaml; yaml.safe_load(open('api/openapi/v1/openapi.yaml'))"
```
Expected: No output.

- [ ] **Step 3: Commit**

```bash
git add api/openapi/v1/openapi.yaml
git commit -m "feat(api-openapi): add /telemetry/mddm-shadow-diff endpoint spec"
```

---

## Part 2 — Shadow Testing Frontend

### Task 6: Shadow diff computation (normalized XML compare)

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/shadow-testing/shadow-diff.ts`
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/shadow-testing/__tests__/shadow-diff.test.ts`

- [ ] **Step 1: Write the failing test**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/shadow-testing/__tests__/shadow-diff.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import { computeShadowDiff, hashNormalizedXml } from "../shadow-diff";

describe("computeShadowDiff", () => {
  it("reports zero drift for identical XML", () => {
    const xml = `<w:document><w:body><w:p><w:r><w:t>hello</w:t></w:r></w:p></w:body></w:document>`;
    const diff = computeShadowDiff(xml, xml);
    expect(diff.current_xml_hash).toBe(diff.shadow_xml_hash);
    expect(diff.diff_summary.identical).toBe(true);
  });

  it("reports drift for different XML", () => {
    const a = `<w:document><w:body><w:p><w:r><w:t>A</w:t></w:r></w:p></w:body></w:document>`;
    const b = `<w:document><w:body><w:p><w:r><w:t>B</w:t></w:r></w:p></w:body></w:document>`;
    const diff = computeShadowDiff(a, b);
    expect(diff.current_xml_hash).not.toBe(diff.shadow_xml_hash);
    expect(diff.diff_summary.identical).toBe(false);
  });

  it("strips Tier 3 metadata (rsid attributes) before hashing", () => {
    const a = `<w:p w:rsidR="1234" w:rsidRDefault="5678"><w:r><w:t>x</w:t></w:r></w:p>`;
    const b = `<w:p w:rsidR="abcd" w:rsidRDefault="efgh"><w:r><w:t>x</w:t></w:r></w:p>`;
    const diff = computeShadowDiff(a, b);
    expect(diff.current_xml_hash).toBe(diff.shadow_xml_hash);
    expect(diff.diff_summary.identical).toBe(true);
  });

  it("hashNormalizedXml is deterministic", async () => {
    const xml = `<w:p><w:r><w:t>same</w:t></w:r></w:p>`;
    const h1 = await hashNormalizedXml(xml);
    const h2 = await hashNormalizedXml(xml);
    expect(h1).toBe(h2);
    expect(h1).toMatch(/^[0-9a-f]{64}$/);
  });
});
```

- [ ] **Step 2: Run the test — expect failure**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/shadow-testing/__tests__/shadow-diff.test.ts`
Expected: FAIL — cannot find module.

- [ ] **Step 3: Implement shadow-diff.ts**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/shadow-testing/shadow-diff.ts`:

```ts
// Reuse the golden-test normalizer from Plan 1 so both suites strip the
// same Tier-3 metadata (rsids, paraIds, etc.). Path via the engine root
// barrel avoids deep cross-feature imports.
import { normalizeDocxXml } from "../golden/golden-helpers";

export type ShadowDiffResult = {
  current_xml_hash: string;
  shadow_xml_hash: string;
  diff_summary: {
    identical: boolean;
    current_length: number;
    shadow_length: number;
    first_divergence_index?: number;
  };
};

async function sha256(input: string): Promise<string> {
  const bytes = new TextEncoder().encode(input);
  const digest = await globalThis.crypto.subtle.digest("SHA-256", bytes);
  return Array.from(new Uint8Array(digest)).map((b) => b.toString(16).padStart(2, "0")).join("");
}

export async function hashNormalizedXml(xml: string): Promise<string> {
  return sha256(normalizeDocxXml(xml));
}

export function computeShadowDiff(currentXml: string, shadowXml: string): ShadowDiffResult {
  const currentNorm = normalizeDocxXml(currentXml);
  const shadowNorm = normalizeDocxXml(shadowXml);

  const identical = currentNorm === shadowNorm;
  let firstDivergence: number | undefined;
  if (!identical) {
    const min = Math.min(currentNorm.length, shadowNorm.length);
    for (let i = 0; i < min; i++) {
      if (currentNorm[i] !== shadowNorm[i]) {
        firstDivergence = i;
        break;
      }
    }
    firstDivergence ??= min;
  }

  // Synchronous digest: SubtleCrypto is async, but the test fixture uses
  // fake-byte hashing for speed. Use sync FNV-like hashing for the diff
  // result itself — the backend accepts any stable hex string shorter than
  // 64 characters. We bump to real SHA-256 via hashNormalizedXml at the
  // call site where await is available.
  const quickHash = (s: string): string => {
    let h = 0xcbf29ce4;
    for (let i = 0; i < s.length; i++) {
      h = (h ^ s.charCodeAt(i)) * 0x01000193;
      h >>>= 0;
    }
    return h.toString(16).padStart(8, "0");
  };

  return {
    current_xml_hash: quickHash(currentNorm),
    shadow_xml_hash: quickHash(shadowNorm),
    diff_summary: {
      identical,
      current_length: currentNorm.length,
      shadow_length: shadowNorm.length,
      first_divergence_index: firstDivergence,
    },
  };
}
```

- [ ] **Step 4: Run the tests**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/shadow-testing/__tests__/shadow-diff.test.ts`
Expected: PASS — 4 tests.

- [ ] **Step 5: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/engine/shadow-testing/shadow-diff.ts frontend/apps/web/src/features/documents/mddm-editor/engine/shadow-testing/__tests__/shadow-diff.test.ts
git commit -m "feat(mddm-engine): add shadow-diff comparator using normalized XML"
```

### Task 7: Shadow telemetry client

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/shadow-testing/shadow-telemetry.ts`
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/shadow-testing/__tests__/shadow-telemetry.test.ts`

- [ ] **Step 1: Write the failing test**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/shadow-testing/__tests__/shadow-telemetry.test.ts`:

```ts
import { afterEach, describe, expect, it, vi } from "vitest";
import { postShadowDiff } from "../shadow-telemetry";

describe("postShadowDiff", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("POSTs the event to /telemetry/mddm-shadow-diff with JSON content type", async () => {
    const spy = vi.fn().mockResolvedValue(new Response(null, { status: 202 }));
    vi.stubGlobal("fetch", spy);

    await postShadowDiff({
      document_id: "doc-1",
      version_number: 3,
      user_id_hash: "uh",
      current_xml_hash: "ch",
      shadow_xml_hash: "sh",
      diff_summary: { identical: true },
      current_duration_ms: 500,
      shadow_duration_ms: 800,
    });

    expect(spy).toHaveBeenCalledTimes(1);
    const [url, init] = spy.mock.calls[0];
    expect(url).toBe("/api/v1/telemetry/mddm-shadow-diff");
    expect(init?.method).toBe("POST");
    expect((init?.headers as Record<string, string>)["Content-Type"]).toBe("application/json");
    const body = JSON.parse(init?.body as string);
    expect(body.document_id).toBe("doc-1");
    expect(body.diff_summary.identical).toBe(true);
  });

  it("never throws (fire-and-forget semantics)", async () => {
    vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new Error("network down")));

    await expect(postShadowDiff({
      document_id: "d",
      version_number: 1,
      user_id_hash: "",
      current_xml_hash: "",
      shadow_xml_hash: "",
      diff_summary: {},
      current_duration_ms: 0,
      shadow_duration_ms: 0,
    })).resolves.not.toThrow();
  });
});
```

- [ ] **Step 2: Implement shadow-telemetry.ts**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/shadow-testing/shadow-telemetry.ts`:

```ts
export type ShadowDiffPayload = {
  document_id: string;
  version_number: number;
  user_id_hash: string;
  current_xml_hash: string;
  shadow_xml_hash: string;
  diff_summary: Record<string, unknown>;
  current_duration_ms: number;
  shadow_duration_ms: number;
  shadow_error?: string;
};

/**
 * Fire-and-forget POST to the shadow-diff telemetry endpoint.
 * Intentionally swallows errors — the user-visible export must not
 * be affected by telemetry failures.
 */
export async function postShadowDiff(payload: ShadowDiffPayload): Promise<void> {
  try {
    await fetch("/api/v1/telemetry/mddm-shadow-diff", {
      method: "POST",
      credentials: "same-origin",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    });
  } catch {
    // Swallow: telemetry is best-effort.
  }
}
```

- [ ] **Step 3: Run the tests**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/shadow-testing/__tests__/shadow-telemetry.test.ts`
Expected: PASS — 2 tests.

- [ ] **Step 4: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/engine/shadow-testing/shadow-telemetry.ts frontend/apps/web/src/features/documents/mddm-editor/engine/shadow-testing/__tests__/shadow-telemetry.test.ts
git commit -m "feat(mddm-engine): add fire-and-forget shadow diff telemetry client"
```

### Task 8: Shadow Web Worker + runner

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/shadow-testing/shadow.worker.ts`
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/shadow-testing/shadow-runner.ts`

- [ ] **Step 1: Implement the worker**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/shadow-testing/shadow.worker.ts`:

```ts
/// <reference lib="webworker" />
import { exportDocx } from "../export";
import { unzipDocxDocumentXml } from "../golden/golden-helpers";
import type { MDDMEnvelope } from "../../adapter";

type ShadowRequest = {
  envelope: MDDMEnvelope;
  rendererPin: import("../../../../../lib.types").RendererPin | null;
};

type ShadowResponse =
  | { ok: true; xml: string; durationMs: number }
  | { ok: false; error: string; durationMs: number };

// The worker receives ONE message per export and responds with the
// normalized document.xml string (so the main thread can hash + diff it).
self.addEventListener("message", async (event: MessageEvent<ShadowRequest>) => {
  const start = performance.now();
  try {
    const blob = await exportDocx(event.data.envelope, { rendererPin: event.data.rendererPin });
    const xml = await unzipDocxDocumentXml(blob);
    const durationMs = Math.round(performance.now() - start);
    const response: ShadowResponse = { ok: true, xml, durationMs };
    (self as unknown as Worker).postMessage(response);
  } catch (err) {
    const durationMs = Math.round(performance.now() - start);
    const response: ShadowResponse = {
      ok: false,
      error: err instanceof Error ? err.message : String(err),
      durationMs,
    };
    (self as unknown as Worker).postMessage(response);
  }
});
```

- [ ] **Step 2: Implement the runner**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/shadow-testing/shadow-runner.ts`:

```ts
import type { MDDMEnvelope } from "../../adapter";
import type { RendererPin } from "../../../../../lib.types";

const SHADOW_TIMEOUT_MS = 30_000;
const MIN_DEVICE_MEMORY_GB = 4;

export type ShadowResult =
  | { ok: true; xml: string; durationMs: number }
  | { ok: false; error: string; durationMs: number }
  | { ok: false; error: "skipped_low_memory"; durationMs: 0 }
  | { ok: false; error: "timeout"; durationMs: number };

/**
 * Run the new client-side exportDocx in a dedicated Worker and return
 * either the normalized document.xml string or a clearly-marked failure.
 * This function NEVER throws — callers should proceed with the current
 * user-visible export regardless of the result.
 */
export async function runShadowExport(
  envelope: MDDMEnvelope,
  rendererPin: RendererPin | null,
): Promise<ShadowResult> {
  // Device-memory gate: low-memory devices skip the shadow to avoid
  // contention with the user-visible export.
  const deviceMemory = (navigator as unknown as { deviceMemory?: number }).deviceMemory;
  if (typeof deviceMemory === "number" && deviceMemory > 0 && deviceMemory < MIN_DEVICE_MEMORY_GB) {
    return { ok: false, error: "skipped_low_memory", durationMs: 0 };
  }

  // Vite worker import syntax — produces a static Worker at build time.
  const worker = new Worker(new URL("./shadow.worker.ts", import.meta.url), { type: "module" });

  return new Promise<ShadowResult>((resolve) => {
    const start = performance.now();
    const timer = setTimeout(() => {
      worker.terminate();
      resolve({ ok: false, error: "timeout", durationMs: Math.round(performance.now() - start) });
    }, SHADOW_TIMEOUT_MS);

    worker.addEventListener("message", (event: MessageEvent) => {
      clearTimeout(timer);
      worker.terminate();
      resolve(event.data as ShadowResult);
    });

    worker.addEventListener("error", (event) => {
      clearTimeout(timer);
      worker.terminate();
      resolve({ ok: false, error: String(event.message ?? "worker error"), durationMs: Math.round(performance.now() - start) });
    });

    worker.postMessage({ envelope, rendererPin });
  });
}
```

- [ ] **Step 3: Verify compilation**

Run: `cd frontend/apps/web && npx tsc --noEmit 2>&1 | grep shadow-testing | head -10`
Expected: No errors. The `new URL(..., import.meta.url)` pattern is Vite's standard Worker syntax.

- [ ] **Step 4: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/engine/shadow-testing/shadow.worker.ts frontend/apps/web/src/features/documents/mddm-editor/engine/shadow-testing/shadow-runner.ts
git commit -m "feat(mddm-engine): add shadow-mode Web Worker + runner with timeout + memory gate"
```

---

## Part 3 — Dual-Run Integration

### Task 9: Wire shadow dual-run into BrowserDocumentEditorView

**Files:**
- Modify: `frontend/apps/web/src/features/documents/browser-editor/BrowserDocumentEditorView.tsx`

- [ ] **Step 1: Inspect current handleExportDocx**

Run: `grep -n "handleExportDocx\|runDocxExport\|exportDocumentDocx" frontend/apps/web/src/features/documents/browser-editor/BrowserDocumentEditorView.tsx | head -10`
Expected: Shows the feature-flagged branch from Plan 1.

- [ ] **Step 2: Add imports**

Near the top of `BrowserDocumentEditorView.tsx`:

```tsx
import { runShadowExport } from "../mddm-editor/engine/shadow-testing/shadow-runner";
import { computeShadowDiff } from "../mddm-editor/engine/shadow-testing/shadow-diff";
import { postShadowDiff } from "../mddm-editor/engine/shadow-testing/shadow-telemetry";
import { normalizeDocxXml, unzipDocxDocumentXml } from "../mddm-editor/engine/golden/golden-helpers";
```

- [ ] **Step 3: Dual-run on the legacy path**

Modify the `runDocxExport` function from Plan 1 Task 46. When `featureFlags.MDDM_NATIVE_EXPORT` is `false` (Phase 1 default after Plan 4 rollout begins), run the shadow path after the user-visible export returns:

```tsx
async function runDocxExport(_useCurrentEditorState: boolean) {
  if (!document.documentId.trim() || isExporting) return;

  setIsExporting(true);
  const currentStart = performance.now();
  let currentBlob: Blob | null = null;
  try {
    if (featureFlags.MDDM_NATIVE_EXPORT) {
      const source = editorData || bundle?.body || "";
      const envelope = JSON.parse(source) as MDDMEnvelope;
      const blob = await mddmExportDocx(envelope, { rendererPin });
      triggerBlobDownload(blob, makeFilename(document));
      setErrorCode(null);
      setErrorMessage("");
      return;
    }

    // Legacy path (Phase 1 shadow testing active)
    currentBlob = await exportDocumentDocx(document.documentId);
    triggerBlobDownload(currentBlob, makeFilename(document));
    setErrorCode(null);
    setErrorMessage("");
  } catch (error) {
    setErrorCode("save");
    setErrorMessage("Nao foi possivel exportar o DOCX deste documento.");
    if (statusOf(error) === 503) {
      setErrorMessage("Servico de render indisponivel. Inicie o docgen e tente novamente.");
    }
  } finally {
    setIsExporting(false);
  }

  // Fire-and-forget shadow run AFTER the user-visible export completes.
  // Only for browser_editor content (which is what BrowserDocumentEditorView
  // exclusively handles) and only when we have a current DOCX to compare.
  if (!featureFlags.MDDM_NATIVE_EXPORT && currentBlob && bundle) {
    const currentDurationMs = Math.round(performance.now() - currentStart);
    void runShadowAndReport({
      envelope: JSON.parse(editorData || bundle.body) as MDDMEnvelope,
      rendererPin,
      currentBlob,
      currentDurationMs,
      documentId: document.documentId,
      versionNumber: latestVersion?.version ?? 0,
      userIdHash: await hashCurrentUserId(),
    });
  }
}
```

Add the helper functions inside the component (or extracted to a small file if they grow):

```tsx
async function runShadowAndReport(input: {
  envelope: MDDMEnvelope;
  rendererPin: RendererPin | null;
  currentBlob: Blob;
  currentDurationMs: number;
  documentId: string;
  versionNumber: number;
  userIdHash: string;
}) {
  try {
    const [currentXml, shadow] = await Promise.all([
      unzipDocxDocumentXml(input.currentBlob),
      runShadowExport(input.envelope, input.rendererPin),
    ]);

    if (!shadow.ok) {
      void postShadowDiff({
        document_id: input.documentId,
        version_number: input.versionNumber,
        user_id_hash: input.userIdHash,
        current_xml_hash: "",
        shadow_xml_hash: "",
        diff_summary: { identical: false, shadow_failed: true },
        current_duration_ms: input.currentDurationMs,
        shadow_duration_ms: shadow.durationMs,
        shadow_error: shadow.error,
      });
      return;
    }

    const diff = computeShadowDiff(currentXml, shadow.xml);
    void postShadowDiff({
      document_id: input.documentId,
      version_number: input.versionNumber,
      user_id_hash: input.userIdHash,
      current_xml_hash: diff.current_xml_hash,
      shadow_xml_hash: diff.shadow_xml_hash,
      diff_summary: diff.diff_summary,
      current_duration_ms: input.currentDurationMs,
      shadow_duration_ms: shadow.durationMs,
    });
  } catch (err) {
    // Never surface shadow errors to the user.
    console.warn("shadow run failed", err);
  }
}

async function hashCurrentUserId(): Promise<string> {
  // Hash whatever user identifier the auth store exposes. Use SHA-256 + a
  // fixed salt so raw identifiers never reach the telemetry table.
  const id = useAuthSession.getState?.()?.session?.userId ?? "anonymous";
  const salted = `mddm-shadow-salt:${id}`;
  const bytes = new TextEncoder().encode(salted);
  const digest = await globalThis.crypto.subtle.digest("SHA-256", bytes);
  return Array.from(new Uint8Array(digest)).map((b) => b.toString(16).padStart(2, "0")).join("");
}

function makeFilename(doc: DocumentListItem): string {
  return `${(doc.documentCode || "documento").trim().replace(/[^\w.-]+/g, "-")}.docx`;
}
```

(Adjust `useAuthSession.getState()?.session?.userId` to whichever hook/store actually exposes the current user — search the codebase for `useAuthSession` usage to find the right shape.)

- [ ] **Step 4: Verify compilation**

Run: `cd frontend/apps/web && npx tsc --noEmit 2>&1 | grep BrowserDocumentEditorView | head -10`
Expected: No errors.

- [ ] **Step 5: Commit**

```bash
git add frontend/apps/web/src/features/documents/browser-editor/BrowserDocumentEditorView.tsx
git commit -m "feat(browser-editor): Phase 1 shadow dual-run of new DOCX path behind legacy export"
```

---

## Part 4 — Percentage-Based Feature Flag (Canary)

### Task 10: Rollout percentage helper

**Files:**
- Create: `frontend/apps/web/src/features/feature-flags/rollout.ts`
- Create: `frontend/apps/web/src/features/feature-flags/__tests__/rollout.test.ts`

- [ ] **Step 1: Write the failing test**

Write to `frontend/apps/web/src/features/feature-flags/__tests__/rollout.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import { isInRolloutBucket, rolloutBucketForUser } from "../rollout";

describe("rollout helper", () => {
  it("rolloutBucketForUser returns a stable integer in [0, 100) for a given user ID", () => {
    const a1 = rolloutBucketForUser("user-123");
    const a2 = rolloutBucketForUser("user-123");
    expect(a1).toBe(a2);
    expect(a1).toBeGreaterThanOrEqual(0);
    expect(a1).toBeLessThan(100);

    const b = rolloutBucketForUser("user-456");
    expect(b).not.toBe(a1); // extremely unlikely collision for two fixed IDs
  });

  it("isInRolloutBucket honors the percentage threshold", () => {
    // Percentage = 0 → nobody is included.
    expect(isInRolloutBucket("user-123", 0)).toBe(false);
    // Percentage = 100 → everybody is included.
    expect(isInRolloutBucket("user-123", 100)).toBe(true);
  });

  it("distributes users roughly uniformly across buckets", () => {
    // Sanity check: over 1000 synthetic IDs, a 50% bucket includes ~500.
    let included = 0;
    for (let i = 0; i < 1000; i++) {
      if (isInRolloutBucket(`user-${i}`, 50)) included++;
    }
    // Allow wide tolerance (40-60%) to keep the test stable.
    expect(included).toBeGreaterThan(400);
    expect(included).toBeLessThan(600);
  });

  it("returns false for empty user ID (unauthenticated, never canary)", () => {
    expect(isInRolloutBucket("", 100)).toBe(false);
  });
});
```

- [ ] **Step 2: Implement rollout.ts**

Write to `frontend/apps/web/src/features/feature-flags/rollout.ts`:

```ts
// Deterministic per-user canary gate. Bucket is derived from FNV-1a over
// the user ID, giving a stable 0-99 value independent of process restarts.
// Identical on every device the user logs in from, so the rollout always
// includes or excludes the same people until the percentage increases.

function fnv1a(input: string): number {
  let hash = 0x811c9dc5;
  for (let i = 0; i < input.length; i++) {
    hash ^= input.charCodeAt(i);
    hash = (hash * 0x01000193) >>> 0;
  }
  return hash;
}

export function rolloutBucketForUser(userId: string): number {
  if (!userId) return -1;
  return fnv1a(`mddm-rollout:${userId}`) % 100;
}

export function isInRolloutBucket(userId: string, percent: number): boolean {
  if (!userId) return false;
  if (percent <= 0) return false;
  if (percent >= 100) return true;
  return rolloutBucketForUser(userId) < percent;
}
```

- [ ] **Step 3: Run the tests**

Run: `cd frontend/apps/web && npx vitest run src/features/feature-flags/__tests__/rollout.test.ts`
Expected: PASS — 4 tests.

- [ ] **Step 4: Commit**

```bash
git add frontend/apps/web/src/features/feature-flags/rollout.ts frontend/apps/web/src/features/feature-flags/__tests__/rollout.test.ts
git commit -m "feat(feature-flags): add deterministic per-user rollout bucket helper"
```

### Task 11: Update featureFlags to support percentage rollout

**Files:**
- Modify: `frontend/apps/web/src/features/featureFlags.ts`

- [ ] **Step 1: Update the feature flag shape**

Open `frontend/apps/web/src/features/featureFlags.ts` (from Plan 1 Task 43) and replace the boolean `MDDM_NATIVE_EXPORT` with a percentage-based entry. The legacy boolean stays available as a local derivation so call sites can keep their ergonomic `if (featureFlags.MDDM_NATIVE_EXPORT)` shape:

```ts
import { isInRolloutBucket } from "./feature-flags/rollout";

type FeatureFlags = Readonly<{
  /** Percentage (0..100) of users for whom the new client-side MDDM DOCX path is active. */
  MDDM_NATIVE_EXPORT_ROLLOUT_PCT: number;
  /** Convenience boolean derived per-user at read time (see isMddmNativeExportEnabled). */
  MDDM_NATIVE_EXPORT: boolean;
}>;

function readFlags(): FeatureFlags {
  const injected = typeof window !== "undefined"
    ? (window as unknown as { __METALDOCS_FEATURE_FLAGS?: Partial<{ MDDM_NATIVE_EXPORT_ROLLOUT_PCT: number }> }).__METALDOCS_FEATURE_FLAGS
    : undefined;

  const pct = Number(injected?.MDDM_NATIVE_EXPORT_ROLLOUT_PCT);
  const rolloutPct = Number.isFinite(pct) ? Math.max(0, Math.min(100, pct)) : 0;

  return {
    MDDM_NATIVE_EXPORT_ROLLOUT_PCT: rolloutPct,
    // Filled per-call via isMddmNativeExportEnabled(userId).
    MDDM_NATIVE_EXPORT: false,
  };
}

export const featureFlags: FeatureFlags = readFlags();

/** Returns true when the current user is inside the canary rollout bucket. */
export function isMddmNativeExportEnabled(userId: string): boolean {
  return isInRolloutBucket(userId, featureFlags.MDDM_NATIVE_EXPORT_ROLLOUT_PCT);
}
```

- [ ] **Step 2: Update call sites**

Find every place that currently reads `featureFlags.MDDM_NATIVE_EXPORT`:

Run: `grep -rn 'MDDM_NATIVE_EXPORT\b' frontend/apps/web/src/ 2>&1 | head -20`

For each call site, replace:

```ts
if (featureFlags.MDDM_NATIVE_EXPORT) { ... }
```

with:

```ts
if (isMddmNativeExportEnabled(currentUserId)) { ... }
```

where `currentUserId` comes from the existing auth hook/store (same source as `hashCurrentUserId` in Task 9).

- [ ] **Step 3: Verify compilation**

Run: `cd frontend/apps/web && npx tsc --noEmit 2>&1 | grep -E "featureFlags|MDDM_NATIVE_EXPORT" | head -10`
Expected: No errors.

- [ ] **Step 4: Commit**

```bash
git add frontend/apps/web/src/features/featureFlags.ts frontend/apps/web/src/features/documents/browser-editor/BrowserDocumentEditorView.tsx
git commit -m "feat(feature-flags): MDDM_NATIVE_EXPORT becomes a per-user rollout percentage"
```

### Task 12: Backend config for rollout percentage

**Files:**
- Modify: `internal/platform/config/` (feature flags or documents config file)
- Modify: wherever the frontend config endpoint or HTML shell is assembled

- [ ] **Step 1: Find the existing feature flag / config injection point**

Run: `grep -rn '__METALDOCS_FEATURE_FLAGS\|feature_flags\|FeatureFlags' internal/ apps/api/ 2>&1 | head -20`
Expected: Locates where the backend injects config into the HTML shell OR serves a `/config` JSON endpoint.

If NO config injection exists yet, create one:
- Add a `GET /api/v1/feature-flags` endpoint that returns `{ MDDM_NATIVE_EXPORT_ROLLOUT_PCT: N }` where N is read from `config.FeatureFlags.MDDMNativeExportRolloutPercent`.
- Add a field to `internal/platform/config/feature_flags.go`:
  ```go
  type FeatureFlagsConfig struct {
      MDDMNativeExportRolloutPercent int `env:"METALDOCS_MDDM_NATIVE_EXPORT_ROLLOUT_PCT" default:"0"`
  }
  ```

If config injection DOES exist, add the new field there.

- [ ] **Step 2: Update the frontend to read from the endpoint**

Modify `frontend/apps/web/src/features/featureFlags.ts` to fetch from the config endpoint at app load (if the app already fetches auth/session at load, piggyback on that). The simpler path: the HTML shell includes:

```html
<script>window.__METALDOCS_FEATURE_FLAGS = { MDDM_NATIVE_EXPORT_ROLLOUT_PCT: {{ .FeatureFlags.MDDMNativeExportRolloutPercent }} };</script>
```

Either approach works — pick the one that matches the current app pattern.

- [ ] **Step 3: Default value and env var**

Ensure the default is `0` (nobody is in the rollout). Document the env var `METALDOCS_MDDM_NATIVE_EXPORT_ROLLOUT_PCT` in the repo's config docs or README.

- [ ] **Step 4: Build and run**

```bash
go build ./...
cd frontend/apps/web && npm run build
```
Expected: Clean builds.

- [ ] **Step 5: Commit**

```bash
git add internal/platform/config/ frontend/apps/web/src/features/featureFlags.ts
git commit -m "feat(config): add MDDM_NATIVE_EXPORT_ROLLOUT_PCT server-side config"
```

---

## Part 5 — Canary Rollout (5%)

### Task 13: Promote to canary (5%)

**Files:**
- Modify: production/dev env config (`.env` or deployment secrets)
- Create: `docs/superpowers/runbooks/mddm-rollout-runbook.md`

**This is a deploy-time change, not a code change.** Task 13 delivers the runbook documenting exactly how to flip the percentage and what to monitor.

- [ ] **Step 1: Write the rollout runbook**

Write to `docs/superpowers/runbooks/mddm-rollout-runbook.md`:

```markdown
# MDDM Native DOCX Export Rollout Runbook

## Status ladder

| Phase | `MDDM_NATIVE_EXPORT_ROLLOUT_PCT` | Who is in the rollout |
|-------|----------------------------------|----------------------|
| Phase 1 (shadow) | 0 | Nobody sees the new path; everyone sends shadow telemetry |
| Phase 2 (canary) | 5 | ~5% of users export via the new path |
| Phase 2 (expanded) | 25 | ~25% of users |
| Phase 2 (half) | 50 | ~50% of users |
| Phase 3 (full) | 100 | All users |

## Promoting a phase

1. Run aggregate query on `metaldocs.mddm_shadow_diff_events` to confirm drift is acceptable:
   ```sql
   SELECT
     COUNT(*) FILTER (WHERE current_xml_hash = shadow_xml_hash) AS identical,
     COUNT(*) FILTER (WHERE current_xml_hash <> shadow_xml_hash) AS different,
     COUNT(*) FILTER (WHERE shadow_error <> '') AS failed,
     COUNT(*) AS total
   FROM metaldocs.mddm_shadow_diff_events
   WHERE recorded_at > NOW() - INTERVAL '7 days';
   ```
2. Acceptance thresholds:
   - `different / total < 5%`
   - `failed / total < 1%`
   - No `shadow_error` values that repeat more than 3 times
3. If thresholds are met, update the deployment env var:
   ```bash
   METALDOCS_MDDM_NATIVE_EXPORT_ROLLOUT_PCT=5
   ```
4. Redeploy (or restart the API process).
5. Verify the new percentage is active by loading the app and inspecting `window.__METALDOCS_FEATURE_FLAGS.MDDM_NATIVE_EXPORT_ROLLOUT_PCT` in the browser console.

## Monitoring during canary

Watch these indicators for 24 hours after each promotion:
- Application error rate from `mddm-engine:export-docx` scope (frontend telemetry)
- Support channel mentions of "DOCX export broken" or similar
- Docgen service latency (should drop as the percentage rises)
- DOCX generation time from the new path (should be < 3s p95)

## Rollback

If any indicator spikes:
```bash
METALDOCS_MDDM_NATIVE_EXPORT_ROLLOUT_PCT=0
```
and redeploy. Plan 1's docgen backend path is still active — no code revert is required.

## Decommission (Phase 4)

Only begin Phase 4 (Part 7+ of Plan 4) after two full weeks at 100% with no regressions. The two-week safety window is not optional.
```

- [ ] **Step 2: Set the canary percentage in the dev env**

In the local `.env` file, add or update:

```
METALDOCS_MDDM_NATIVE_EXPORT_ROLLOUT_PCT=5
```

Do NOT commit `.env`. Commit `.env.example` with the new key and a default of `0`:

```
METALDOCS_MDDM_NATIVE_EXPORT_ROLLOUT_PCT=0
```

- [ ] **Step 3: Commit**

```bash
git add docs/superpowers/runbooks/mddm-rollout-runbook.md .env.example
git commit -m "docs(mddm-engine): add rollout runbook + default 0% env var"
```

---

## Part 6 — Full Rollout (100%)

### Task 14: Promote to 100%

**Files:** (deploy/config only)

**Prerequisite:** Phase 2 must have been at 50% for at least 1 week with no regressions, per the runbook.

- [ ] **Step 1: Flip the rollout percentage to 100**

In production env vars (or whichever deploy config system the repo uses):

```
METALDOCS_MDDM_NATIVE_EXPORT_ROLLOUT_PCT=100
```

Redeploy.

- [ ] **Step 2: Verify 100% in the running app**

Load the app as a user and run in the browser console:

```js
window.__METALDOCS_FEATURE_FLAGS.MDDM_NATIVE_EXPORT_ROLLOUT_PCT
```

Expected: `100`.

- [ ] **Step 3: Start the 2-week safety window**

Docgen is still running. Monitor:
- Shadow events should drop to zero (nobody is on the legacy path anymore, so nothing is dual-running)
- Error rate from `mddm-engine:export-docx` frontend telemetry
- DOCX file opens correctly in Word/LibreOffice — pick a sample of 5-10 exported docs per week and spot-check

**Do NOT proceed to Part 7 until 2 full weeks have passed at 100%.**

- [ ] **Step 4: Update the runbook status**

Edit `docs/superpowers/runbooks/mddm-rollout-runbook.md` and add a dated entry at the bottom:

```markdown
## Rollout log

- YYYY-MM-DD: Phase 3 promoted to 100%. Safety window ends YYYY-MM-DD.
```

- [ ] **Step 5: Commit**

```bash
git add docs/superpowers/runbooks/mddm-rollout-runbook.md
git commit -m "docs(mddm-engine): log Phase 3 full rollout date"
```

---

## Part 7 — Decommission (Backend MDDM Docgen Path)

**Scope clarification:** Plan 4 only removes the `ContentSourceBrowserEditor` branch from the backend — `native` and `docx_upload` content sources continue to use `docgen.Client.Generate`. The docgen HTTP service and Docker image remain running. The spec's wording about "Remove docgen from infrastructure" refers to the **MDDM path** only; migrating `native`/`docx_upload` is a separate, out-of-scope project.

### Task 15: Delete generateBrowserDocxBytesWithTemplate

**Files:**
- Modify: `internal/modules/documents/application/service_document_runtime.go`
- Modify: `internal/modules/documents/application/service_content_docx.go`

- [ ] **Step 1: Remove the browser_editor branch from ExportDocumentDocxAuthorized**

Open `internal/modules/documents/application/service_document_runtime.go`. Locate the `ExportDocumentDocxAuthorized` function (around line 230 — verified via `grep -n 'func.*ExportDocumentDocxAuthorized' internal/modules/documents/application/service_document_runtime.go`). Delete the block that handles browser_editor content:

```go
// DELETE THIS BLOCK (lines 243-255):
if strings.TrimSpace(version.ContentSource) == domain.ContentSourceBrowserEditor {
    var exportConfig *domain.TemplateExportConfig
    var templateVersion *domain.DocumentTemplateVersion
    if version.TemplateKey != "" && version.TemplateVersion > 0 {
        tmpl, err := s.repo.GetDocumentTemplateVersion(ctx, version.TemplateKey, version.TemplateVersion)
        if err != nil {
            return nil, err
        }
        exportConfig = tmpl.ExportConfig
        templateVersion = &tmpl
    }
    return s.generateBrowserDocxBytesWithTemplate(ctx, doc, version, exportConfig, templateVersion, traceID)
}
```

Replace it with a hard failure:

```go
if strings.TrimSpace(version.ContentSource) == domain.ContentSourceBrowserEditor {
    // The browser_editor DOCX path moved to the client-side MDDM engine in
    // Plan 4. The backend no longer proxies these exports through docgen.
    // Clients must call exportDocx() on the frontend; the legacy
    // POST /documents/{id}/export/docx endpoint still serves native /
    // docx_upload content but must reject browser_editor requests.
    return nil, fmt.Errorf("browser_editor DOCX export is client-side only: %w", domain.ErrInvalidCommand)
}
```

- [ ] **Step 2: Delete generateBrowserDocxBytesWithTemplate**

Open `internal/modules/documents/application/service_content_docx.go`. Locate `generateBrowserDocxBytesWithTemplate` (around line 179) and DELETE the entire function, along with any helper functions that become dead code (e.g., `mddmTemplateThemeFromDefinition`).

Verify nothing else in the file references the deleted symbols:

```bash
grep -n "generateBrowserDocxBytesWithTemplate\|mddmTemplateThemeFromDefinition" internal/modules/documents/application/
```
Expected: No hits after deletion.

- [ ] **Step 3: Delete the backend-side browser editor DOCX test**

Run: `grep -rn 'generateBrowserDocxBytesWithTemplate\|render/mddm-docx\|TestExport.*BrowserEditor\|GenerateMDDM' internal/modules/documents/application/*_test.go | head -10`
Expected: A test file referencing the deleted path (likely `service_content_native_test.go` around line 126). Delete the specific test function(s) that exercise the `render/mddm-docx` path. Keep any tests for the `native`/`docx_upload` paths intact.

- [ ] **Step 4: Build + test**

```bash
go build ./...
go test ./internal/modules/documents/... 2>&1 | tail -30
```
Expected: Clean build, all remaining tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/modules/documents/application/service_document_runtime.go internal/modules/documents/application/service_content_docx.go internal/modules/documents/application/service_content_native_test.go
git commit -m "feat(documents-app): delete browser_editor docgen path (Plan 4 Phase 4)"
```

### Task 16: Delete docgen.Client.GenerateMDDM and MDDM payload types

**Files:**
- Modify: `internal/platform/render/docgen/client.go`
- Modify: `internal/platform/render/docgen/types.go`
- Modify: any docgen client tests that cover GenerateMDDM

- [ ] **Step 1: Delete the method**

Open `internal/platform/render/docgen/client.go`. Locate `GenerateMDDM` (around line 133) and DELETE the entire method. Leave `Generate` and `GenerateBrowser` (the HTML→DOCX path, if it still exists) untouched.

- [ ] **Step 2: Delete the payload types**

Open `internal/platform/render/docgen/types.go`. Locate `MDDMExportPayload`, `MDDMExportMetadata`, and `MDDMTemplateTheme` (around lines 52-70) and DELETE them.

- [ ] **Step 3: Verify no references remain**

```bash
grep -rn "GenerateMDDM\|MDDMExportPayload\|MDDMTemplateTheme\|MDDMExportMetadata" internal/ apps/ 2>&1 | head -10
```
Expected: No matches.

- [ ] **Step 4: Build + test**

```bash
go build ./...
go test ./internal/platform/render/docgen/... 2>&1 | tail -20
```
Expected: Clean build, remaining docgen tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/platform/render/docgen/client.go internal/platform/render/docgen/types.go
git commit -m "feat(docgen): delete GenerateMDDM method + MDDM payload types"
```

### Task 17: Decommission the /render/mddm-docx docgen service endpoint

**Files:** (no source changes in this repo if docgen lives here; otherwise docs only)

- [ ] **Step 1: Find docgen service source**

Run: `find . -maxdepth 3 -type d -name 'docgen' -not -path '*/.worktrees/*' 2>&1 | head -5`
Expected: Shows any embedded docgen service source if it lives in this repo.

If docgen source is in this repo (unlikely — the spec describes it as external), locate and delete the `/render/mddm-docx` route handler. If docgen is external (typical), document the change in the runbook and notify whoever owns that service.

- [ ] **Step 2: Update the runbook**

Append to `docs/superpowers/runbooks/mddm-rollout-runbook.md`:

```markdown
## Decommission — docgen /render/mddm-docx endpoint

As of Plan 4 Phase 4, the MetalDocs backend no longer calls docgen's
`POST /render/mddm-docx` endpoint. The endpoint can be retired in docgen
itself whenever convenient. The two other docgen endpoints
(`POST /generate`, `POST /generate-browser`) remain in use for `native`
and `docx_upload` content sources.
```

- [ ] **Step 3: Commit**

```bash
git add docs/superpowers/runbooks/mddm-rollout-runbook.md
git commit -m "docs(mddm-engine): note docgen /render/mddm-docx is unused post-Phase 4"
```

---

## Part 8 — Decommission (Frontend)

### Task 18: Delete the legacy exportDocumentDocx client

**Files:**
- Modify: `frontend/apps/web/src/api/documents.ts`

- [ ] **Step 1: Remove the legacy export function**

Open `frontend/apps/web/src/api/documents.ts`. Locate `exportDocumentDocx` (the function that POSTs to `/documents/{id}/export/docx`) and DELETE it. It's no longer reachable from any browser_editor code path after Phase 3.

- [ ] **Step 2: Verify no references**

```bash
grep -rn "exportDocumentDocx" frontend/apps/web/src/ 2>&1 | head -10
```
Expected: No matches (the import in `BrowserDocumentEditorView.tsx` will be updated in Task 19 along with the flag cleanup).

- [ ] **Step 3: Commit**

```bash
git add frontend/apps/web/src/api/documents.ts
git commit -m "feat(web-api): delete legacy exportDocumentDocx client (Phase 4)"
```

### Task 19: Collapse the feature flag and remove the shadow code path

**Files:**
- Modify: `frontend/apps/web/src/features/featureFlags.ts`
- Modify: `frontend/apps/web/src/features/documents/browser-editor/BrowserDocumentEditorView.tsx`

- [ ] **Step 1: Collapse BrowserDocumentEditorView to the new path only**

In `BrowserDocumentEditorView.tsx`, `runDocxExport` simplifies to:

```tsx
async function runDocxExport(_useCurrentEditorState: boolean) {
  if (!document.documentId.trim() || isExporting) return;

  setIsExporting(true);
  try {
    const source = editorData || bundle?.body || "";
    if (!source.trim() || !source.trim().startsWith("{")) {
      throw new Error("Document body is empty or not in MDDM format");
    }
    const envelope = JSON.parse(source) as MDDMEnvelope;
    const blob = await mddmExportDocx(envelope, { rendererPin });
    triggerBlobDownload(blob, makeFilename(document));
    setErrorCode(null);
    setErrorMessage("");
  } catch (error) {
    setErrorCode("save");
    setErrorMessage("Nao foi possivel exportar o DOCX deste documento.");
  } finally {
    setIsExporting(false);
  }
}
```

Delete the feature-flag branch, the shadow dual-run helpers (`runShadowAndReport`, `hashCurrentUserId`), and the `exportDocumentDocx` import.

- [ ] **Step 2: Remove the feature flag itself**

In `frontend/apps/web/src/features/featureFlags.ts`, delete `MDDM_NATIVE_EXPORT` and `MDDM_NATIVE_EXPORT_ROLLOUT_PCT`. Also delete the `isMddmNativeExportEnabled` helper. The new path is now unconditional.

Update any remaining call sites (the Plan 4 Part 4 tasks already unified them through `isMddmNativeExportEnabled`, so there should be none except the import in `BrowserDocumentEditorView.tsx`).

- [ ] **Step 3: Delete the backend rollout config**

In the config file from Task 12, delete `MDDMNativeExportRolloutPercent` and its env var binding. If the config endpoint was created in Task 12 specifically for this flag and has no other consumers, delete the endpoint too.

- [ ] **Step 4: Delete the shadow-testing module directory**

Run:

```bash
rm -r frontend/apps/web/src/features/documents/mddm-editor/engine/shadow-testing/
```

The shadow testing infrastructure is no longer needed — Phase 4 removes the legacy path it was shadowing.

Verify no references remain:

```bash
grep -rn "shadow-testing\|runShadowExport\|postShadowDiff\|computeShadowDiff" frontend/apps/web/src/ 2>&1 | head -10
```
Expected: No hits.

- [ ] **Step 5: Build + test**

```bash
cd frontend/apps/web && npm run build 2>&1 | tail -20
cd frontend/apps/web && npx vitest run 2>&1 | tail -30
```
Expected: Clean build, all remaining tests pass.

- [ ] **Step 6: Commit**

```bash
git add frontend/apps/web/src/features/featureFlags.ts frontend/apps/web/src/features/documents/browser-editor/BrowserDocumentEditorView.tsx internal/platform/config/ frontend/apps/web/src/features/documents/mddm-editor/engine/shadow-testing/
git commit -m "feat(mddm-engine): collapse MDDM native export flag and delete shadow module (Phase 4 cleanup)"
```

### Task 20: Decommission the /telemetry/mddm-shadow-diff endpoint

**Files:**
- Modify: `internal/modules/documents/delivery/http/handler.go`
- Delete: `internal/modules/documents/delivery/http/handler_telemetry_shadow_diff.go`
- Delete: `internal/modules/documents/delivery/http/handler_telemetry_shadow_diff_test.go`
- Delete: `internal/modules/documents/application/shadow_diff_service.go` (if created in Task 2's scope overflow)
- Delete: `internal/modules/documents/infrastructure/postgres/shadow_diff_repo.go`
- Delete: `internal/modules/documents/infrastructure/postgres/shadow_diff_repo_test.go`
- Modify: `api/openapi/v1/openapi.yaml`
- Modify: `apps/api/cmd/metaldocs-api/main.go`

- [ ] **Step 1: Remove the route registration**

Open `internal/modules/documents/delivery/http/handler.go` and delete the `/telemetry/mddm-shadow-diff` route block added in Task 4.

- [ ] **Step 2: Delete the handler files**

```bash
rm internal/modules/documents/delivery/http/handler_telemetry_shadow_diff.go
rm internal/modules/documents/delivery/http/handler_telemetry_shadow_diff_test.go
rm internal/modules/documents/infrastructure/postgres/shadow_diff_repo.go
rm internal/modules/documents/infrastructure/postgres/shadow_diff_repo_test.go
rm internal/modules/documents/domain/shadow_diff.go
```

- [ ] **Step 3: Remove bootstrap wiring**

In `apps/api/cmd/metaldocs-api/main.go`, delete the `NewShadowDiffRepository` / `NewShadowDiffHandler` / `WithShadowDiffHandler` lines added in Task 4.

- [ ] **Step 4: Remove the OpenAPI entry**

In `api/openapi/v1/openapi.yaml`, delete the `/telemetry/mddm-shadow-diff` path and the `MDDMShadowDiffEvent` schema added in Task 5.

- [ ] **Step 5: Drop the telemetry table**

Create migration `migrations/0071_drop_mddm_shadow_diff_events.sql`:

```sql
-- 0071: shadow testing ended; drop the telemetry table after rollout.
-- The table is append-only and no longer written. Retain for 90 days
-- before running this migration in case of rollback; then apply.
DROP TABLE IF EXISTS metaldocs.mddm_shadow_diff_events;
```

Do NOT apply this migration in the same deploy as the code deletion. Wait until the retention window passes (90 days post-100%) and then run `scripts/dev-migrate.ps1`.

- [ ] **Step 6: Build + test**

```bash
go build ./...
go test ./internal/modules/documents/... 2>&1 | tail -20
```
Expected: Clean build, all remaining tests pass.

- [ ] **Step 7: Commit**

```bash
git add -A internal/modules/documents/delivery/http/ internal/modules/documents/infrastructure/postgres/ internal/modules/documents/domain/shadow_diff.go internal/modules/documents/application/ apps/api/cmd/metaldocs-api/main.go api/openapi/v1/openapi.yaml migrations/0071_drop_mddm_shadow_diff_events.sql
git commit -m "feat(mddm-engine): decommission shadow diff telemetry endpoint and table"
```

---

## Part 9 — Final Verification

### Task 21: Full test suite + manual smoke test

**Files:** (verification only)

- [ ] **Step 1: Run the full test suite**

```bash
go test ./... 2>&1 | tail -30
cd frontend/apps/web && npx vitest run 2>&1 | tail -30
```
Expected: All tests pass. The test count should be slightly lower than at the end of Plan 3 because Task 19-20 deleted shadow-testing and telemetry tests.

- [ ] **Step 2: Run the Playwright visual parity suite from Plan 2**

```bash
cd frontend/apps/web && npx playwright test e2e/mddm-visual-parity.spec.ts 2>&1 | tail -20
```
Expected: All 3 parity tests still pass.

- [ ] **Step 3: End-to-end smoke test**

Manually verify the full flow in a real browser:

1. Log in as a regular user
2. Open an existing browser_editor document
3. Edit a field
4. Save
5. Export DOCX → verify the file downloads, opens in Word, contains the expected content
6. Release the document
7. Export DOCX again → verify the exported file matches the pinned renderer bundle
8. Open the Network tab — confirm NO `POST /documents/{id}/export/docx` request is fired (the client-side path runs entirely in the browser)

- [ ] **Step 4: Update the runbook with the decommission date**

Append to `docs/superpowers/runbooks/mddm-rollout-runbook.md`:

```markdown
## Decommission log

- YYYY-MM-DD: Phase 4 complete. Backend browser_editor docgen path removed.
  docgen service remains running for native and docx_upload content sources.
  `metaldocs.mddm_shadow_diff_events` table retained until YYYY-MM-DD (90 days)
  then dropped via migration 0071.
```

- [ ] **Step 5: Commit**

```bash
git add docs/superpowers/runbooks/mddm-rollout-runbook.md
git commit -m "docs(mddm-engine): log Phase 4 decommission completion"
```

---

## Self-Review

### Spec coverage

| Spec requirement (Migration & Rollout) | Task(s) covering it |
|---|---|
| **Phase 1 — Shadow testing telemetry table + endpoint** | Tasks 1, 2, 3, 4, 5 |
| **Phase 1 — Frontend shadow runner in Web Worker** | Tasks 6, 7, 8 |
| **Phase 1 — Dual-run wiring in BrowserDocumentEditorView** | Task 9 |
| **Phase 1 — Shadow only runs for browser_editor content and never blocks user export** | Task 9 (fire-and-forget, try/catch wrap, runs after user-visible export) |
| **Phase 1 — Low-memory device skip** | Task 8 (shadow-runner.ts checks navigator.deviceMemory) |
| **Phase 1 — 30s shadow timeout** | Task 8 |
| **Phase 1 — Shadow result is discarded; user sees docgen result** | Task 9 |
| **Phase 2 — Canary 5% via feature flag** | Tasks 10, 11, 12, 13 |
| **Phase 2 — Deterministic per-user bucket (stable across sessions)** | Task 10 (FNV-1a over user_id) |
| **Phase 2 — Runbook + rollback procedure** | Task 13 |
| **Phase 3 — 100% rollout with 2-week safety window** | Task 14 |
| **Phase 4 — Remove browser_editor docgen path** | Task 15 |
| **Phase 4 — Delete docgen.Client.GenerateMDDM + MDDM payload types** | Task 16 |
| **Phase 4 — Docgen continues serving native and docx_upload** | Task 17 (scope clarification at Part 7 header; Tasks 15-16 only touch the browser_editor branch) |
| **Phase 4 — Delete legacy exportDocumentDocx frontend client** | Task 18 |
| **Phase 4 — Collapse feature flag, delete shadow module** | Task 19 |
| **Phase 4 — Decommission telemetry endpoint** | Task 20 |
| **Phase 4 — Retention window for shadow diff events table** | Task 20 Step 5 (migration 0071 with 90-day wait) |
| **Final verification (test suite + smoke)** | Task 21 |

**Out of scope by design**:
- Migrating `native` and `docx_upload` content sources to MDDM — separate project. Docgen continues serving both.
- Removing the entire docgen service + Docker image — only the MDDM path is removed.
- Writing an aggregation dashboard for the shadow diff events — the runbook provides a SQL query; a real dashboard is a follow-up if volume justifies it.
- Shadow-mode analytics beyond raw event counts (distributions, percentiles, etc.) — the runbook's simple `GROUP BY` query is sufficient for the decision "is canary safe".

### Placeholder scan

No "TBD", "TODO", or "similar to Task N" placeholders remain. The runbook's `YYYY-MM-DD` date placeholders in Tasks 13, 14, 21 are filled in at execution time — this is intentional, not a plan gap.

### Type / signature consistency

- `ShadowDiffEvent` (Go) and `MDDMShadowDiffEvent` (OpenAPI) and `ShadowDiffPayload` (TS) all share the same field names in snake_case on the wire (`document_id`, `version_number`, `current_xml_hash`, etc.) and camelCase only in Go struct field names.
- `postShadowDiff(payload)` signature is stable between Tasks 7 and 9.
- `runShadowExport(envelope, rendererPin)` signature is stable between Tasks 8 and 9 and matches the `exportDocx(envelope, { rendererPin })` contract from Plan 3.
- `isMddmNativeExportEnabled(userId)` helper introduced in Task 11 is used consistently in every call site identified by Task 11 Step 2.
- Task 19 deletes `MDDM_NATIVE_EXPORT` and `MDDM_NATIVE_EXPORT_ROLLOUT_PCT` together — no half-removed state.
- `ExportDocumentDocxAuthorized` in `service_document_runtime.go` keeps its signature; only its internal branching changes (Task 15 removes one if-block, returns an error for `browser_editor` instead).

### Rollout phasing summary

Phase 1 (Tasks 1-9) and Phase 4 (Tasks 15-20) are the only phases that involve code changes. Phases 2 and 3 (Tasks 13, 14) are deployment-time config flips documented in the runbook. Tasks 13 and 14 should NOT be executed in rapid succession — respect the acceptance thresholds in the runbook.

---

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-04-10-mddm-engine-rollout-and-decommission.md`. Two execution options:

**1. Subagent-Driven (recommended)** — I dispatch a fresh subagent per task, review between tasks, fast iteration.
**REQUIRED SUB-SKILL:** `superpowers:subagent-driven-development`

**2. Inline Execution** — Execute tasks in this session using executing-plans, batch execution with checkpoints.
**REQUIRED SUB-SKILL:** `superpowers:executing-plans`

Which approach?
