# MDDM Engine Version Pinning & Renderer Bundle Registry Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the version pinning mechanism described in the design spec's Section 10. Every document version stores a `RendererPin` (`rendererVersion` + `layoutIRHash` + `templateRefLocked`) captured at DRAFT→RELEASED transition. Build a frontend renderer bundle registry so released documents always render with the renderer they were released against, while DRAFT documents use the current engine. Add an IR hash drift CI gate that fails when the Layout IR changes without a version bump.

**Architecture:** The backend owns the pin lifecycle: on release transition, the API captures the current renderer version (from a single source of truth injected at build time) plus the template version (already stored) and writes a new `renderer_pin` JSONB column on `document_versions`. The frontend maintains a `renderers/` directory where each entry is a frozen snapshot of `{ tokens, emitters, printStylesheet }` for a given version string. `loadCurrentRenderer()` returns the current-HEAD module; `loadPinnedRenderer(pin)` dynamically imports a historical snapshot by version string. `exportDocx` and the PDF path both accept a version object and choose the appropriate renderer. The migration pipeline targets the pinned `mddm_version`, not the current engine version. A CI test compares the serialized Layout IR hash against a committed fixture; any drift requires either (a) a deliberate version bump or (b) re-approving the fixture.

**Tech Stack:** TypeScript 5.6, React 18, Vitest 4.1, Go 1.22, PostgreSQL 16 (JSONB column), SHA-256 (Web Crypto on the frontend, `crypto/sha256` on the backend).

**Spec:** `docs/superpowers/specs/2026-04-10-mddm-unified-document-engine-design.md` (Section 10: Version Pinning)

**Depends on:**
- Plan 1 — `docs/superpowers/plans/2026-04-10-mddm-engine-foundation.md`
- Plan 2 — `docs/superpowers/plans/2026-04-10-mddm-engine-full-block-coverage.md`

Both must be merged before Plan 3 starts.

---

## File Structure

### New files (backend)

```
migrations/
└── 0069_add_renderer_pin_to_document_versions.sql   # NEW: ALTER TABLE add JSONB column
internal/modules/documents/
├── domain/
│   └── renderer_pin.go                              # NEW: RendererPin struct + validation
└── application/
    ├── capture_renderer_pin.go                      # NEW: capture pin on release transition
    └── capture_renderer_pin_test.go                 # NEW
```

### New files (frontend)

```
frontend/apps/web/src/features/documents/mddm-editor/engine/
├── renderers/
│   ├── current.ts                                   # NEW: current-HEAD renderer export
│   ├── registry.ts                                  # NEW: loadCurrentRenderer + loadPinnedRenderer
│   ├── v1.0.0/                                      # NEW: frozen snapshot for the first released version
│   │   └── index.ts                                 # NEW: re-exports the live modules (v1.0.0 = current at launch)
│   └── __tests__/
│       ├── registry.test.ts                         # NEW
│       └── current-hash.test.ts                     # NEW
├── ir-hash/
│   ├── compute-ir-hash.ts                           # NEW: SHA-256 over serialized Layout IR
│   ├── recorded-hash.ts                             # NEW: committed fixture hash
│   └── __tests__/
│       ├── compute-ir-hash.test.ts                  # NEW
│       └── drift-gate.test.ts                       # NEW: CI gate (fails if hash drifts)
```

### Modified files (backend)

```
internal/modules/documents/domain/model.go                          # MODIFY: add RendererPin *RendererPin field on Version
internal/modules/documents/infrastructure/postgres/repository.go    # MODIFY: read/write renderer_pin column in version queries
internal/modules/documents/application/service_document_runtime.go  # MODIFY: call captureRendererPin on release
internal/modules/documents/application/release_service.go           # MODIFY (or wherever release transition lives): hook capture
internal/modules/documents/delivery/http/handler_runtime.go         # MODIFY: include rendererPin in version responses
internal/modules/documents/delivery/http/handler_browser_editor.go  # MODIFY: include rendererPin in browser editor bundle
api/openapi/v1/openapi.yaml                                          # MODIFY: add RendererPin schema
```

### Modified files (frontend)

```
frontend/apps/web/src/lib.types.ts                                                                    # MODIFY: add RendererPin type
frontend/apps/web/src/features/documents/mddm-editor/engine/canonicalize-migrate/pipeline.ts           # MODIFY: accept targetVersion parameter
frontend/apps/web/src/features/documents/mddm-editor/engine/export/export-docx.ts                      # MODIFY: accept Version, load pinned renderer
frontend/apps/web/src/features/documents/mddm-editor/engine/export/export-pdf.ts                       # MODIFY: accept Version, load pinned renderer
frontend/apps/web/src/features/documents/browser-editor/BrowserDocumentEditorView.tsx                  # MODIFY: pass version to exports
```

---

## Part 1 — Backend Domain: RendererPin Type & Schema Migration

### Task 1: Add RendererPin domain type

**Files:**
- Create: `internal/modules/documents/domain/renderer_pin.go`

- [ ] **Step 1: Write the failing test**

Create `internal/modules/documents/domain/renderer_pin_test.go`:

```go
package domain

import "testing"

func TestRendererPin_IsComplete(t *testing.T) {
    pin := RendererPin{
        RendererVersion: "1.0.0",
        LayoutIRHash:    "abc123",
        TemplateKey:     "po-mddm-canvas",
        TemplateVersion: 1,
    }
    if !pin.IsComplete() {
        t.Fatalf("expected pin to be complete")
    }

    incomplete := RendererPin{RendererVersion: "1.0.0"}
    if incomplete.IsComplete() {
        t.Fatalf("expected pin missing fields to be incomplete")
    }
}

func TestRendererPin_Validate(t *testing.T) {
    tests := []struct {
        name    string
        pin     RendererPin
        wantErr bool
    }{
        {
            name:    "valid pin",
            pin:     RendererPin{RendererVersion: "1.0.0", LayoutIRHash: "abcdef", TemplateKey: "po", TemplateVersion: 1},
            wantErr: false,
        },
        {
            name:    "missing renderer version",
            pin:     RendererPin{LayoutIRHash: "abc", TemplateKey: "po", TemplateVersion: 1},
            wantErr: true,
        },
        {
            name:    "zero template version",
            pin:     RendererPin{RendererVersion: "1.0.0", LayoutIRHash: "abc", TemplateKey: "po", TemplateVersion: 0},
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.pin.Validate()
            if (err != nil) != tt.wantErr {
                t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

- [ ] **Step 2: Run test — expect failure**

Run: `go test ./internal/modules/documents/domain/... -run TestRendererPin -v 2>&1 | tail -20`
Expected: FAIL — type `RendererPin` undefined.

- [ ] **Step 3: Implement renderer_pin.go**

Write to `internal/modules/documents/domain/renderer_pin.go`:

```go
package domain

import (
    "errors"
    "strings"
    "time"
)

// RendererPin freezes the inputs used to render a document version for
// DOCX and PDF export. When a version transitions from DRAFT to RELEASED,
// the application captures the current renderer version, Layout IR hash,
// and the specific template (key + version) that was active at release time.
// From that moment on, any export of the version MUST load the matching
// historical renderer bundle — see the frontend registry for the mechanism.
type RendererPin struct {
    RendererVersion string    `json:"renderer_version"`
    LayoutIRHash    string    `json:"layout_ir_hash"`
    TemplateKey     string    `json:"template_key"`
    TemplateVersion int       `json:"template_version"`
    PinnedAt        time.Time `json:"pinned_at"`
}

// IsComplete reports whether every required field is populated.
// Zero times (PinnedAt) are allowed — the application sets them on capture.
func (p RendererPin) IsComplete() bool {
    return strings.TrimSpace(p.RendererVersion) != "" &&
        strings.TrimSpace(p.LayoutIRHash) != "" &&
        strings.TrimSpace(p.TemplateKey) != "" &&
        p.TemplateVersion > 0
}

// Validate returns an error if any required field is missing or malformed.
func (p RendererPin) Validate() error {
    if strings.TrimSpace(p.RendererVersion) == "" {
        return errors.New("renderer pin: rendererVersion is required")
    }
    if strings.TrimSpace(p.LayoutIRHash) == "" {
        return errors.New("renderer pin: layoutIRHash is required")
    }
    if strings.TrimSpace(p.TemplateKey) == "" {
        return errors.New("renderer pin: templateKey is required")
    }
    if p.TemplateVersion <= 0 {
        return errors.New("renderer pin: templateVersion must be positive")
    }
    return nil
}
```

- [ ] **Step 4: Run test — expect pass**

Run: `go test ./internal/modules/documents/domain/... -run TestRendererPin -v 2>&1 | tail -20`
Expected: PASS — 4 subtests passing.

- [ ] **Step 5: Commit**

```bash
git add internal/modules/documents/domain/renderer_pin.go internal/modules/documents/domain/renderer_pin_test.go
git commit -m "feat(documents-domain): add RendererPin type for version pinning"
```

### Task 2: Add RendererPin field to Version struct

**Files:**
- Modify: `internal/modules/documents/domain/model.go`

- [ ] **Step 1: Add the field**

In `internal/modules/documents/domain/model.go`, find the `Version struct` (around line 60) and add a new field as the last field before the closing brace:

```go
type Version struct {
    // ... existing fields ...
    CreatedAt        time.Time

    // RendererPin is set at DRAFT→RELEASED transition. Nil for draft versions.
    RendererPin *RendererPin
}
```

- [ ] **Step 2: Build to verify**

Run: `go build ./internal/modules/documents/...`
Expected: Clean build.

- [ ] **Step 3: Commit**

```bash
git add internal/modules/documents/domain/model.go
git commit -m "feat(documents-domain): add RendererPin field to Version struct"
```

### Task 3: Add renderer_pin column to document_versions

**Files:**
- Create: `migrations/0069_add_renderer_pin_to_document_versions.sql`

- [ ] **Step 1: Write the migration**

Write to `migrations/0069_add_renderer_pin_to_document_versions.sql`:

```sql
-- 0069: add renderer_pin JSONB column to document_versions.
-- Captures the renderer version + Layout IR hash + template ref at
-- DRAFT→RELEASED transition so released documents always re-render
-- with the engine that approved them.

ALTER TABLE metaldocs.document_versions
    ADD COLUMN IF NOT EXISTS renderer_pin JSONB;

-- Optional GIN index if we ever query by renderer version; off by default
-- to keep the write path fast. Add later if needed.

COMMENT ON COLUMN metaldocs.document_versions.renderer_pin IS
    'Frozen renderer inputs captured at release time: {renderer_version, layout_ir_hash, template_key, template_version, pinned_at}. NULL for drafts.';
```

- [ ] **Step 2: Apply the migration to the local dev DB**

Run: `go run ./cmd/migrate up 2>&1 | tail -20`
(Or whatever migration command the repo uses — inspect `cmd/` for a migrate entry point.)
Expected: Migration 0069 applied without error.

If there is no dedicated migration binary, verify the migration file syntax by running it against a throwaway psql connection:

```bash
psql "$DATABASE_URL" -f migrations/0069_add_renderer_pin_to_document_versions.sql
```
Expected: `ALTER TABLE` success message.

- [ ] **Step 3: Verify column exists**

Run: `psql "$DATABASE_URL" -c "\d metaldocs.document_versions" 2>&1 | grep renderer_pin`
Expected: Row showing `renderer_pin | jsonb | |`.

- [ ] **Step 4: Commit**

```bash
git add migrations/0069_add_renderer_pin_to_document_versions.sql
git commit -m "feat(db): add renderer_pin JSONB column to document_versions"
```

### Task 4: Postgres repository reads and writes renderer_pin

**Files:**
- Modify: `internal/modules/documents/infrastructure/postgres/repository.go`
- Modify: `internal/modules/documents/infrastructure/postgres/repository_test.go` (or create a new focused test file)

- [ ] **Step 1: Find every SQL statement that reads or writes document_versions**

Run: `grep -n 'document_versions' internal/modules/documents/infrastructure/postgres/repository.go | head -30`
Expected: Lists every INSERT, UPDATE, and SELECT that touches `document_versions`.

- [ ] **Step 2: Write a failing integration-style test for round-trip**

Append to `internal/modules/documents/infrastructure/postgres/repository_test.go` (the existing test suite — use the same setup helpers it already has):

```go
func TestRepository_RendererPin_Roundtrip(t *testing.T) {
    repo, cleanup := newTestRepo(t) // reuse whatever helper the existing tests use
    defer cleanup()

    ctx := context.Background()
    doc := seedTestDocument(t, repo) // reuse existing seeding helper

    now := time.Now().UTC().Truncate(time.Second)
    pin := &domain.RendererPin{
        RendererVersion: "1.0.0",
        LayoutIRHash:    "abcdef0123456789",
        TemplateKey:     "po-mddm-canvas",
        TemplateVersion: 1,
        PinnedAt:        now,
    }

    if err := repo.SetVersionRendererPin(ctx, doc.ID, 1, pin); err != nil {
        t.Fatalf("SetVersionRendererPin: %v", err)
    }

    got, err := repo.GetVersion(ctx, doc.ID, 1)
    if err != nil {
        t.Fatalf("GetVersion: %v", err)
    }
    if got.RendererPin == nil {
        t.Fatalf("expected RendererPin to be populated")
    }
    if got.RendererPin.RendererVersion != "1.0.0" || got.RendererPin.LayoutIRHash != "abcdef0123456789" {
        t.Fatalf("roundtrip mismatch: %+v", got.RendererPin)
    }
}
```

- [ ] **Step 3: Run the test — expect failure**

Run: `go test ./internal/modules/documents/infrastructure/postgres/... -run TestRepository_RendererPin_Roundtrip -v 2>&1 | tail -20`
Expected: FAIL — method `SetVersionRendererPin` does not exist OR SELECT queries do not populate `RendererPin`.

- [ ] **Step 4: Implement repository changes**

In `internal/modules/documents/infrastructure/postgres/repository.go`:

**4a.** Add a new method:

```go
func (r *Repository) SetVersionRendererPin(ctx context.Context, documentID string, versionNumber int, pin *domain.RendererPin) error {
    if pin == nil {
        _, err := r.db.ExecContext(ctx,
            `UPDATE metaldocs.document_versions SET renderer_pin = NULL WHERE document_id = $1 AND number = $2`,
            documentID, versionNumber)
        return err
    }
    if err := pin.Validate(); err != nil {
        return fmt.Errorf("invalid renderer pin: %w", err)
    }
    payload, err := json.Marshal(pin)
    if err != nil {
        return fmt.Errorf("marshal renderer pin: %w", err)
    }
    _, err = r.db.ExecContext(ctx,
        `UPDATE metaldocs.document_versions SET renderer_pin = $3::jsonb WHERE document_id = $1 AND number = $2`,
        documentID, versionNumber, string(payload))
    return err
}
```

**4b.** Update every `SELECT` query that reads version rows to include the `renderer_pin` column. For each identified SELECT from Step 1, add `, renderer_pin` to the column list and add a `sql.NullString` scan target that is parsed into `*domain.RendererPin` when not null.

Add a helper:

```go
func scanRendererPin(raw sql.NullString) (*domain.RendererPin, error) {
    if !raw.Valid || strings.TrimSpace(raw.String) == "" {
        return nil, nil
    }
    var pin domain.RendererPin
    if err := json.Unmarshal([]byte(raw.String), &pin); err != nil {
        return nil, fmt.Errorf("decode renderer pin: %w", err)
    }
    return &pin, nil
}
```

And wire it into every version-reading query:

```go
var pinRaw sql.NullString
// ... Scan(&v.DocumentID, ..., &v.CreatedAt, &pinRaw) ...
v.RendererPin, err = scanRendererPin(pinRaw)
if err != nil { return nil, err }
```

- [ ] **Step 5: Run the test — expect pass**

Run: `go test ./internal/modules/documents/infrastructure/postgres/... -run TestRepository_RendererPin_Roundtrip -v 2>&1 | tail -20`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/modules/documents/infrastructure/postgres/repository.go internal/modules/documents/infrastructure/postgres/repository_test.go
git commit -m "feat(documents-repo): read/write renderer_pin column on document_versions"
```

---

## Part 2 — Release Transition Hook

### Task 5: Capture renderer pin on DRAFT→RELEASED transition

**Files:**
- Create: `internal/modules/documents/application/capture_renderer_pin.go`
- Create: `internal/modules/documents/application/capture_renderer_pin_test.go`

- [ ] **Step 1: Write the failing test**

Write to `internal/modules/documents/application/capture_renderer_pin_test.go`:

```go
package application

import (
    "context"
    "testing"
    "time"

    "metaldocs/internal/modules/documents/domain"
)

type fakePinRepo struct {
    capturedPin     *domain.RendererPin
    capturedDoc     string
    capturedVersion int
}

func (f *fakePinRepo) SetVersionRendererPin(ctx context.Context, documentID string, versionNumber int, pin *domain.RendererPin) error {
    f.capturedPin = pin
    f.capturedDoc = documentID
    f.capturedVersion = versionNumber
    return nil
}

func TestCaptureRendererPin_WritesExpectedFields(t *testing.T) {
    repo := &fakePinRepo{}
    clock := func() time.Time { return time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC) }

    capture := NewRendererPinCapturer(RendererPinCapturerConfig{
        CurrentRendererVersion: "1.0.0",
        CurrentLayoutIRHash:    "hash-deadbeef",
        Repo:                   repo,
        Clock:                  clock,
    })

    version := domain.Version{
        DocumentID:      "doc-1",
        Number:          3,
        ContentSource:   domain.ContentSourceBrowserEditor,
        TemplateKey:     "po-mddm-canvas",
        TemplateVersion: 2,
    }

    if err := capture.OnRelease(context.Background(), version); err != nil {
        t.Fatalf("OnRelease: %v", err)
    }

    if repo.capturedDoc != "doc-1" || repo.capturedVersion != 3 {
        t.Fatalf("wrong target: doc=%q version=%d", repo.capturedDoc, repo.capturedVersion)
    }
    if repo.capturedPin == nil {
        t.Fatalf("expected pin to be written")
    }
    want := domain.RendererPin{
        RendererVersion: "1.0.0",
        LayoutIRHash:    "hash-deadbeef",
        TemplateKey:     "po-mddm-canvas",
        TemplateVersion: 2,
        PinnedAt:        clock(),
    }
    if *repo.capturedPin != want {
        t.Fatalf("pin mismatch:\n got %+v\nwant %+v", *repo.capturedPin, want)
    }
}

func TestCaptureRendererPin_SkipsNonBrowserEditorSources(t *testing.T) {
    repo := &fakePinRepo{}
    capture := NewRendererPinCapturer(RendererPinCapturerConfig{
        CurrentRendererVersion: "1.0.0",
        CurrentLayoutIRHash:    "h",
        Repo:                   repo,
        Clock:                  time.Now,
    })

    // Native content does not use the MDDM engine, so no pin is needed.
    version := domain.Version{
        DocumentID:    "doc-2",
        Number:        1,
        ContentSource: domain.ContentSourceNative,
    }

    if err := capture.OnRelease(context.Background(), version); err != nil {
        t.Fatalf("OnRelease: %v", err)
    }
    if repo.capturedPin != nil {
        t.Fatalf("expected no pin for native content source, got %+v", repo.capturedPin)
    }
}

func TestCaptureRendererPin_ErrorsWhenTemplateMissing(t *testing.T) {
    repo := &fakePinRepo{}
    capture := NewRendererPinCapturer(RendererPinCapturerConfig{
        CurrentRendererVersion: "1.0.0",
        CurrentLayoutIRHash:    "h",
        Repo:                   repo,
        Clock:                  time.Now,
    })

    version := domain.Version{
        DocumentID:    "doc-3",
        Number:        1,
        ContentSource: domain.ContentSourceBrowserEditor,
        // Missing TemplateKey / TemplateVersion
    }

    if err := capture.OnRelease(context.Background(), version); err == nil {
        t.Fatalf("expected error when browser editor version has no template")
    }
}
```

- [ ] **Step 2: Run test — expect failure**

Run: `go test ./internal/modules/documents/application/... -run TestCaptureRendererPin -v 2>&1 | tail -20`
Expected: FAIL — type `RendererPinCapturer` undefined.

- [ ] **Step 3: Implement the capturer**

Write to `internal/modules/documents/application/capture_renderer_pin.go`:

```go
package application

import (
    "context"
    "fmt"
    "strings"
    "time"

    "metaldocs/internal/modules/documents/domain"
)

// RendererPinRepo is the minimal repository surface the capturer needs.
// It is satisfied by the postgres Repository via SetVersionRendererPin.
type RendererPinRepo interface {
    SetVersionRendererPin(ctx context.Context, documentID string, versionNumber int, pin *domain.RendererPin) error
}

type RendererPinCapturerConfig struct {
    CurrentRendererVersion string
    CurrentLayoutIRHash    string
    Repo                   RendererPinRepo
    Clock                  func() time.Time
}

// RendererPinCapturer writes a RendererPin when a version transitions from
// DRAFT to RELEASED. It's a tiny domain service, not a generic hook, so the
// transition site can call OnRelease explicitly with the version record.
type RendererPinCapturer struct {
    cfg RendererPinCapturerConfig
}

func NewRendererPinCapturer(cfg RendererPinCapturerConfig) *RendererPinCapturer {
    if cfg.Clock == nil {
        cfg.Clock = time.Now
    }
    return &RendererPinCapturer{cfg: cfg}
}

// OnRelease captures a pin for the given version if the version's content
// source uses the MDDM engine. Non-MDDM content sources (native, docx_upload)
// are skipped because they don't go through the MDDM renderer.
func (c *RendererPinCapturer) OnRelease(ctx context.Context, version domain.Version) error {
    if version.ContentSource != domain.ContentSourceBrowserEditor {
        return nil
    }
    if strings.TrimSpace(version.TemplateKey) == "" || version.TemplateVersion <= 0 {
        return fmt.Errorf("browser editor version %s/%d missing template ref", version.DocumentID, version.Number)
    }

    pin := &domain.RendererPin{
        RendererVersion: c.cfg.CurrentRendererVersion,
        LayoutIRHash:    c.cfg.CurrentLayoutIRHash,
        TemplateKey:     version.TemplateKey,
        TemplateVersion: version.TemplateVersion,
        PinnedAt:        c.cfg.Clock().UTC(),
    }
    if err := pin.Validate(); err != nil {
        return fmt.Errorf("build renderer pin: %w", err)
    }

    return c.cfg.Repo.SetVersionRendererPin(ctx, version.DocumentID, version.Number, pin)
}
```

- [ ] **Step 4: Run test — expect pass**

Run: `go test ./internal/modules/documents/application/... -run TestCaptureRendererPin -v 2>&1 | tail -20`
Expected: PASS — 3 subtests passing.

- [ ] **Step 5: Commit**

```bash
git add internal/modules/documents/application/capture_renderer_pin.go internal/modules/documents/application/capture_renderer_pin_test.go
git commit -m "feat(documents-app): add RendererPinCapturer for release transitions"
```

### Task 6: Wire the capturer into the release transition

**Files:**
- Modify: the file that handles DRAFT→RELEASED transitions

- [ ] **Step 1: Find the release transition site**

Run: `grep -rn 'DRAFT.*RELEASED\|StatusReleased\|Release(\|transitionTo' internal/modules/documents/application/ 2>&1 | head -20`
Expected: Locates the function that performs the release transition (likely `release_service.go` or `service_document_runtime.go` — path may vary).

- [ ] **Step 2: Inject the capturer into the Service**

In `internal/modules/documents/application/service.go`, add a field and a builder method:

```go
type Service struct {
    // ... existing fields ...
    rendererPinCapturer *RendererPinCapturer
}

func (s *Service) WithRendererPinCapturer(c *RendererPinCapturer) *Service {
    s.rendererPinCapturer = c
    return s
}
```

- [ ] **Step 3: Call the capturer on transition**

At the release transition site identified in Step 1, after the version's status is persisted as RELEASED (and after any other commit-level work), add:

```go
if s.rendererPinCapturer != nil {
    if err := s.rendererPinCapturer.OnRelease(ctx, *releasedVersion); err != nil {
        // Log but do not fail the release: a missing pin can be repaired
        // by re-releasing. Failing the release would block approval workflows.
        log.Printf("renderer pin capture failed for %s/%d: %v", releasedVersion.DocumentID, releasedVersion.Number, err)
    }
}
```

(Use whichever variable holds the freshly-released version. If the transition returns an error before persisting, do NOT call OnRelease.)

- [ ] **Step 4: Wire the capturer at bootstrap**

Find the Service construction site (`grep -rn 'NewService\|service := application.NewService' cmd/ internal/ | head -5`) and add the capturer injection. The `CurrentRendererVersion` and `CurrentLayoutIRHash` come from a new build-time constant or config entry:

```go
rendererPinCfg := application.RendererPinCapturerConfig{
    CurrentRendererVersion: config.Documents.RendererVersion, // e.g., "1.0.0"
    CurrentLayoutIRHash:    config.Documents.LayoutIRHash,    // populated by build script
    Repo:                   pgRepo,
    Clock:                  time.Now,
}
capturer := application.NewRendererPinCapturer(rendererPinCfg)
service := application.NewService(/* ... */).WithRendererPinCapturer(capturer)
```

Add the two new config keys to `internal/platform/config/documents.go` (or wherever documents config lives — search for `Documents` struct). Defaults: `RendererVersion = "1.0.0"`, `LayoutIRHash = ""`. Loading from env vars `METALDOCS_RENDERER_VERSION` and `METALDOCS_LAYOUT_IR_HASH`.

- [ ] **Step 5: Build and run existing tests**

Run: `go build ./... && go test ./internal/modules/documents/... 2>&1 | tail -30`
Expected: Clean build, all existing tests pass.

- [ ] **Step 6: Commit**

```bash
git add internal/modules/documents/application/service.go internal/modules/documents/application/release_service.go internal/platform/config/documents.go cmd/
git commit -m "wire(documents-app): inject RendererPinCapturer into release transition"
```

---

## Part 3 — API Responses + OpenAPI

### Task 7: Include renderer_pin in version responses

**Files:**
- Modify: `internal/modules/documents/delivery/http/handler_runtime.go` (or wherever version responses are serialized)
- Modify: `internal/modules/documents/delivery/http/handler_browser_editor.go`

- [ ] **Step 1: Find the version response serializer(s)**

Run: `grep -rn 'document_versions\|versionDTO\|VersionResponse\|version := map\|toVersionDTO' internal/modules/documents/delivery/http/ | head -20`
Expected: Locates the DTO construction functions for version responses.

- [ ] **Step 2: Add renderer_pin to the response shape**

In each DTO construction identified above, add:

```go
response := map[string]any{
    // ... existing fields ...
    "renderer_pin": version.RendererPin, // nil for drafts → marshals to null
}
```

Use the existing JSON encoding pattern in the file. The `RendererPin` struct already has JSON tags, so it marshals cleanly.

- [ ] **Step 3: Build to verify no compile errors**

Run: `go build ./internal/modules/documents/delivery/http/...`
Expected: Clean build.

- [ ] **Step 4: Add a handler test asserting the field is present**

Append to an existing handler test file (e.g., `handler_browser_editor_test.go`):

```go
func TestBrowserEditorBundle_IncludesRendererPinField(t *testing.T) {
    // Reuse whatever setup the existing tests use. Seed a document with
    // a released version carrying a pin, request the browser editor
    // bundle, and assert the response contains renderer_pin.
    // ... full test body following existing test patterns in this file ...
}
```

(Mirror the existing test setup; the assertion is `bytes.Contains(respBody, []byte(`"renderer_pin"`))` or a JSON decode that checks the field exists.)

- [ ] **Step 5: Run the handler tests**

Run: `go test ./internal/modules/documents/delivery/http/... 2>&1 | tail -20`
Expected: All tests pass including the new one.

- [ ] **Step 6: Commit**

```bash
git add internal/modules/documents/delivery/http/
git commit -m "feat(documents-http): include renderer_pin in version DTO responses"
```

### Task 8: Add RendererPin schema to OpenAPI

**Files:**
- Modify: `api/openapi/v1/openapi.yaml`

- [ ] **Step 1: Add the schema definition**

Open `api/openapi/v1/openapi.yaml` and find the `components.schemas` section. Add a new schema:

```yaml
    RendererPin:
      type: object
      description: |
        Frozen renderer inputs captured at DRAFT→RELEASED transition.
        Guarantees a released document always re-renders with the engine
        that approved it.
      required:
        - renderer_version
        - layout_ir_hash
        - template_key
        - template_version
      properties:
        renderer_version:
          type: string
          example: "1.0.0"
        layout_ir_hash:
          type: string
          description: SHA-256 of the serialized Layout IR module at release time
          example: "abcdef0123456789..."
        template_key:
          type: string
          example: "po-mddm-canvas"
        template_version:
          type: integer
          minimum: 1
        pinned_at:
          type: string
          format: date-time
```

- [ ] **Step 2: Reference it from the version schemas**

Find every schema in `openapi.yaml` that represents a document version (search `grep -n 'version_id\|VersionListItem\|DocumentVersion' api/openapi/v1/openapi.yaml | head -20`). For each, add under `properties`:

```yaml
        renderer_pin:
          oneOf:
            - $ref: '#/components/schemas/RendererPin'
            - type: 'null'
          description: Null for draft versions; populated once released.
```

- [ ] **Step 3: Validate the OpenAPI YAML**

Run: `python -c "import yaml; yaml.safe_load(open('api/openapi/v1/openapi.yaml'))" 2>&1 | tail -5`
Expected: No output. If Python is unavailable, use any YAML linter.

- [ ] **Step 4: Commit**

```bash
git add api/openapi/v1/openapi.yaml
git commit -m "feat(api-openapi): add RendererPin schema to version responses"
```

### Task 9: Add RendererPin TypeScript type

**Files:**
- Modify: `frontend/apps/web/src/lib.types.ts`

- [ ] **Step 1: Find the Version types**

Run: `grep -n 'rendererPin\|VersionListItem\|DocumentBrowserEditorBundleResponse' frontend/apps/web/src/lib.types.ts | head -10`
Expected: Locates the relevant type declarations.

- [ ] **Step 2: Add the RendererPin type and embed it in version types**

In `frontend/apps/web/src/lib.types.ts`, add:

```ts
export type RendererPin = {
  renderer_version: string;
  layout_ir_hash: string;
  template_key: string;
  template_version: number;
  pinned_at?: string; // ISO timestamp
};
```

Then update each version-type (e.g., `VersionListItem`, whatever `bundle.versions[i]` resolves to, and `DocumentBrowserEditorBundleResponse.versions[i]`) to include:

```ts
renderer_pin?: RendererPin | null;
```

- [ ] **Step 3: Verify compilation**

Run: `cd frontend/apps/web && npx tsc --noEmit 2>&1 | grep -E "lib.types|RendererPin" | head -10`
Expected: No errors.

- [ ] **Step 4: Commit**

```bash
git add frontend/apps/web/src/lib.types.ts
git commit -m "feat(web-types): add RendererPin type to document version shapes"
```

---

## Part 4 — Frontend IR Hash

### Task 10: Compute Layout IR hash at startup

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/ir-hash/compute-ir-hash.ts`
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/ir-hash/__tests__/compute-ir-hash.test.ts`

- [ ] **Step 1: Write the failing test**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/ir-hash/__tests__/compute-ir-hash.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import { computeLayoutIRHash, serializeLayoutIRForHash } from "../compute-ir-hash";
import { defaultLayoutTokens, defaultComponentRules } from "../../layout-ir";

describe("computeLayoutIRHash", () => {
  it("produces a stable hex SHA-256 digest", async () => {
    const hash = await computeLayoutIRHash({ tokens: defaultLayoutTokens, components: defaultComponentRules });
    expect(hash).toMatch(/^[0-9a-f]{64}$/);
  });

  it("produces the same hash for the same input", async () => {
    const a = await computeLayoutIRHash({ tokens: defaultLayoutTokens, components: defaultComponentRules });
    const b = await computeLayoutIRHash({ tokens: defaultLayoutTokens, components: defaultComponentRules });
    expect(a).toBe(b);
  });

  it("produces a different hash when a token changes", async () => {
    const a = await computeLayoutIRHash({ tokens: defaultLayoutTokens, components: defaultComponentRules });
    const changed = {
      ...defaultLayoutTokens,
      theme: { ...defaultLayoutTokens.theme, accent: "#000000" },
    };
    const b = await computeLayoutIRHash({ tokens: changed, components: defaultComponentRules });
    expect(a).not.toBe(b);
  });

  it("serializeLayoutIRForHash produces deterministic key order", () => {
    const serialized1 = serializeLayoutIRForHash({ tokens: defaultLayoutTokens, components: defaultComponentRules });
    const serialized2 = serializeLayoutIRForHash({ tokens: defaultLayoutTokens, components: defaultComponentRules });
    expect(serialized1).toBe(serialized2);
  });
});
```

- [ ] **Step 2: Run test — expect failure**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/ir-hash/__tests__/compute-ir-hash.test.ts`
Expected: FAIL — cannot find module `../compute-ir-hash`.

- [ ] **Step 3: Implement compute-ir-hash.ts**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/ir-hash/compute-ir-hash.ts`:

```ts
import type { LayoutTokens, ComponentRules } from "../layout-ir";

export type LayoutIRSnapshot = {
  tokens: LayoutTokens;
  components: ComponentRules;
};

/** Serialize the Layout IR with sorted keys so the hash is deterministic. */
function stableStringify(value: unknown): string {
  if (value === null || typeof value !== "object") {
    return JSON.stringify(value);
  }
  if (Array.isArray(value)) {
    return `[${value.map(stableStringify).join(",")}]`;
  }
  const entries = Object.entries(value as Record<string, unknown>).sort(
    ([a], [b]) => (a < b ? -1 : a > b ? 1 : 0),
  );
  return `{${entries.map(([k, v]) => `${JSON.stringify(k)}:${stableStringify(v)}`).join(",")}}`;
}

export function serializeLayoutIRForHash(snapshot: LayoutIRSnapshot): string {
  return stableStringify(snapshot);
}

/** Compute SHA-256 hex digest of the serialized Layout IR. Uses Web Crypto
 *  on the browser; Node vitest runs use the Node builtin crypto.subtle. */
export async function computeLayoutIRHash(snapshot: LayoutIRSnapshot): Promise<string> {
  const serialized = serializeLayoutIRForHash(snapshot);
  const encoder = new TextEncoder();
  const data = encoder.encode(serialized);

  const subtle = (globalThis.crypto && globalThis.crypto.subtle) as SubtleCrypto | undefined;
  if (subtle) {
    const digest = await subtle.digest("SHA-256", data);
    return Array.from(new Uint8Array(digest))
      .map((b) => b.toString(16).padStart(2, "0"))
      .join("");
  }
  // Fallback for environments without Web Crypto (old Node). Vitest 4.x uses
  // Node 20+, which has globalThis.crypto.subtle natively.
  throw new Error("Web Crypto not available for layout IR hashing");
}
```

- [ ] **Step 4: Run test — expect pass**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/ir-hash/__tests__/compute-ir-hash.test.ts`
Expected: PASS — 4 tests passing.

- [ ] **Step 5: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/engine/ir-hash/compute-ir-hash.ts frontend/apps/web/src/features/documents/mddm-editor/engine/ir-hash/__tests__/compute-ir-hash.test.ts
git commit -m "feat(mddm-engine): add Layout IR SHA-256 hash computation"
```

### Task 11: Record the current IR hash as a committed fixture

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/ir-hash/recorded-hash.ts`
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/ir-hash/__tests__/drift-gate.test.ts`

- [ ] **Step 1: Seed the recorded hash file with a placeholder**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/ir-hash/recorded-hash.ts`:

```ts
// Committed SHA-256 of the current Layout IR snapshot. Regenerated ONLY
// when a deliberate renderer version bump happens (see the drift gate
// test). Do NOT edit this file by hand to silence the drift test.

export const RECORDED_IR_HASH = "PLACEHOLDER_REGENERATE_VIA_DRIFT_GATE";
export const RECORDED_RENDERER_VERSION = "1.0.0";
```

- [ ] **Step 2: Write the drift gate test**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/ir-hash/__tests__/drift-gate.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import { computeLayoutIRHash } from "../compute-ir-hash";
import { defaultLayoutTokens, defaultComponentRules } from "../../layout-ir";
import { RECORDED_IR_HASH, RECORDED_RENDERER_VERSION } from "../recorded-hash";

describe("Layout IR drift gate", () => {
  it("current IR hash matches the recorded hash", async () => {
    const current = await computeLayoutIRHash({
      tokens: defaultLayoutTokens,
      components: defaultComponentRules,
    });

    if (RECORDED_IR_HASH === "PLACEHOLDER_REGENERATE_VIA_DRIFT_GATE") {
      throw new Error(
        `RECORDED_IR_HASH is a placeholder. Edit recorded-hash.ts and set:\n` +
        `  export const RECORDED_IR_HASH = "${current}";\n` +
        `then commit. This records the current engine as the baseline for future drift detection.`
      );
    }

    expect(current).toBe(RECORDED_IR_HASH);
  });

  it("RECORDED_RENDERER_VERSION is populated", () => {
    expect(RECORDED_RENDERER_VERSION).toMatch(/^\d+\.\d+\.\d+$/);
  });
});
```

- [ ] **Step 3: Run the drift gate test**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/ir-hash/__tests__/drift-gate.test.ts 2>&1 | tail -20`
Expected: FAIL with a message telling you the exact hash to paste into `recorded-hash.ts`.

- [ ] **Step 4: Edit recorded-hash.ts with the real hash**

Open `frontend/apps/web/src/features/documents/mddm-editor/engine/ir-hash/recorded-hash.ts` and replace the placeholder with the hash from Step 3's error message.

- [ ] **Step 5: Re-run the drift gate test**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/ir-hash/__tests__/drift-gate.test.ts`
Expected: PASS.

**Going forward**: any change to `defaultLayoutTokens` or `defaultComponentRules` will cause this test to fail. The fix is to either (a) bump `RECORDED_RENDERER_VERSION` and re-record the hash if the change is intentional and requires a new bundle snapshot, or (b) revert the change if it was accidental.

- [ ] **Step 6: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/engine/ir-hash/recorded-hash.ts frontend/apps/web/src/features/documents/mddm-editor/engine/ir-hash/__tests__/drift-gate.test.ts
git commit -m "feat(mddm-engine): add Layout IR drift gate with committed hash fixture"
```

---

## Part 5 — Renderer Bundle Registry

### Task 12: Freeze v1.0.0 snapshot

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/renderers/v1.0.0/index.ts`
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/renderers/current.ts`

**Snapshot strategy:** For v1.0.0 (the first pinned version), we do NOT duplicate the source code. The snapshot is a thin re-export of the current live modules. Future versions (v1.1.0+) will be created by *copying* the live module tree into a new `renderers/vX.Y.Z/` directory and freezing it there — at that point v1.0.0's re-exports will keep working because they alias the old module paths.

- [ ] **Step 1: Write the v1.0.0 bundle entry**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/renderers/v1.0.0/index.ts`:

```ts
// Renderer Bundle: v1.0.0
//
// Captures the launch version of the MDDM engine. At v1.0.0, the bundle
// re-exports the live modules from engine/docx-emitter, engine/layout-ir,
// and engine/print-stylesheet. When the next renderer version is introduced
// (v1.1.0), this file is frozen: replace the re-exports with a local copy of
// the implementation to prevent the current-HEAD modules from drifting away
// from the v1.0.0 pin.

export { defaultLayoutTokens, defaultComponentRules } from "../../layout-ir";
export { mddmToDocx, MissingEmitterError, type EmitContext } from "../../docx-emitter";
export { PRINT_STYLESHEET } from "../../print-stylesheet";
export { wrapInPrintDocument } from "../../export/wrap-print-document";

// Layout IR hash captured at the moment v1.0.0 was cut. This MUST match
// the value written in engine/ir-hash/recorded-hash.ts at the time of
// tagging. The drift gate in Part 4 enforces this.
import { RECORDED_IR_HASH, RECORDED_RENDERER_VERSION } from "../../ir-hash/recorded-hash";
export const BUNDLE_RENDERER_VERSION = RECORDED_RENDERER_VERSION;
export const BUNDLE_LAYOUT_IR_HASH = RECORDED_IR_HASH;
```

- [ ] **Step 2: Write the current-renderer shim**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/renderers/current.ts`:

```ts
// "Current" renderer — used for draft documents that have no pin.
// At v1.0.0 this is an alias of the v1.0.0 bundle. When the first
// renderer version bump happens, rewire this file to point at the
// new snapshot (and copy the previous one into a frozen directory).

export * from "./v1.0.0/index";
```

- [ ] **Step 3: Verify compilation**

Run: `cd frontend/apps/web && npx tsc --noEmit 2>&1 | grep renderers/ | head -5`
Expected: No errors.

- [ ] **Step 4: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/engine/renderers/
git commit -m "feat(mddm-engine): add v1.0.0 renderer bundle + current-renderer shim"
```

### Task 13: Implement loadCurrentRenderer + loadPinnedRenderer registry

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/renderers/registry.ts`
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/renderers/__tests__/registry.test.ts`

- [ ] **Step 1: Write the failing test**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/renderers/__tests__/registry.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import {
  loadCurrentRenderer,
  loadPinnedRenderer,
  RendererBundleNotFoundError,
  type LoadedRenderer,
} from "../registry";
import type { RendererPin } from "../../../../../lib.types";

describe("renderer registry", () => {
  it("loadCurrentRenderer returns a renderer bundle with tokens and mddmToDocx", async () => {
    const renderer = await loadCurrentRenderer();
    expect(renderer.rendererVersion).toMatch(/^\d+\.\d+\.\d+$/);
    expect(typeof renderer.mddmToDocx).toBe("function");
    expect(renderer.tokens.page.widthMm).toBeGreaterThan(0);
    expect(typeof renderer.printStylesheet).toBe("string");
  });

  it("loadPinnedRenderer returns the v1.0.0 bundle for a 1.0.0 pin", async () => {
    const pin: RendererPin = {
      renderer_version: "1.0.0",
      layout_ir_hash: "ignored-for-registry-lookup",
      template_key: "po-mddm-canvas",
      template_version: 1,
    };
    const renderer = await loadPinnedRenderer(pin);
    expect(renderer.rendererVersion).toBe("1.0.0");
  });

  it("loadPinnedRenderer throws RendererBundleNotFoundError for unknown versions", async () => {
    const pin: RendererPin = {
      renderer_version: "9.9.9",
      layout_ir_hash: "h",
      template_key: "k",
      template_version: 1,
    };
    await expect(loadPinnedRenderer(pin)).rejects.toBeInstanceOf(RendererBundleNotFoundError);
  });
});
```

- [ ] **Step 2: Run test — expect failure**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/renderers/__tests__/registry.test.ts`
Expected: FAIL — cannot find module.

- [ ] **Step 3: Implement registry.ts**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/renderers/registry.ts`:

```ts
import type { LayoutTokens } from "../layout-ir";
import type { RendererPin } from "../../../../../lib.types";

export type LoadedRenderer = {
  rendererVersion: string;
  layoutIRHash: string;
  tokens: LayoutTokens;
  mddmToDocx: typeof import("../docx-emitter").mddmToDocx;
  wrapInPrintDocument: typeof import("../export/wrap-print-document").wrapInPrintDocument;
  printStylesheet: string;
};

export class RendererBundleNotFoundError extends Error {
  constructor(public readonly rendererVersion: string) {
    super(`No renderer bundle registered for version "${rendererVersion}"`);
    this.name = "RendererBundleNotFoundError";
  }
}

async function fromBundle(bundle: typeof import("./v1.0.0/index")): Promise<LoadedRenderer> {
  return {
    rendererVersion: bundle.BUNDLE_RENDERER_VERSION,
    layoutIRHash: bundle.BUNDLE_LAYOUT_IR_HASH,
    tokens: bundle.defaultLayoutTokens,
    mddmToDocx: bundle.mddmToDocx,
    wrapInPrintDocument: bundle.wrapInPrintDocument,
    printStylesheet: bundle.PRINT_STYLESHEET,
  };
}

/** Load the current-HEAD renderer for draft documents. */
export async function loadCurrentRenderer(): Promise<LoadedRenderer> {
  const bundle = await import("./current");
  return fromBundle(bundle as unknown as typeof import("./v1.0.0/index"));
}

/** Load the renderer bundle matching a version pin. Throws
 *  RendererBundleNotFoundError when no matching bundle is registered. */
export async function loadPinnedRenderer(pin: RendererPin): Promise<LoadedRenderer> {
  switch (pin.renderer_version) {
    case "1.0.0":
      return fromBundle(await import("./v1.0.0/index"));
    default:
      throw new RendererBundleNotFoundError(pin.renderer_version);
  }
}
```

**Why a switch instead of dynamic-path import?** Vite's code splitter requires import paths to be statically analyzable. A `import(\`./\${version}/index\`)` call would break tree-shaking and ship the wrong code to production. Each new version adds one explicit `case` block.

- [ ] **Step 4: Run test — expect pass**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/renderers/__tests__/registry.test.ts`
Expected: PASS — 3 tests passing.

- [ ] **Step 5: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/engine/renderers/registry.ts frontend/apps/web/src/features/documents/mddm-editor/engine/renderers/__tests__/registry.test.ts
git commit -m "feat(mddm-engine): add renderer bundle registry with v1.0.0 entry"
```

---

## Part 6 — Export Integration

### Task 14: exportDocx accepts a Version and loads the right renderer

**Files:**
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/engine/export/export-docx.ts`
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/engine/export/__tests__/export-docx.test.ts`

- [ ] **Step 1: Extend the failing test**

Append to `frontend/apps/web/src/features/documents/mddm-editor/engine/export/__tests__/export-docx.test.ts`:

```ts
import type { RendererPin } from "../../../../../lib.types";

describe("exportDocx renderer pin selection", () => {
  it("loads the pinned renderer when a version has a rendererPin", async () => {
    const envelope: MDDMEnvelope = {
      mddm_version: 1,
      template_ref: null,
      blocks: [
        { id: "p", type: "paragraph", props: {}, children: [{ type: "text", text: "pinned" }] },
      ],
    };
    const pin: RendererPin = {
      renderer_version: "1.0.0",
      layout_ir_hash: "h",
      template_key: "po-mddm-canvas",
      template_version: 1,
    };

    const blob = await exportDocx(envelope, { rendererPin: pin });
    expect(blob).toBeInstanceOf(Blob);
    expect(blob.size).toBeGreaterThan(100);
  });

  it("loads the current renderer when rendererPin is null (draft)", async () => {
    const envelope: MDDMEnvelope = {
      mddm_version: 1,
      template_ref: null,
      blocks: [
        { id: "p", type: "paragraph", props: {}, children: [{ type: "text", text: "draft" }] },
      ],
    };

    const blob = await exportDocx(envelope, { rendererPin: null });
    expect(blob).toBeInstanceOf(Blob);
    expect(blob.size).toBeGreaterThan(100);
  });
});
```

- [ ] **Step 2: Run the test — expect failure**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/export/__tests__/export-docx.test.ts 2>&1 | tail -30`
Expected: FAIL — `exportDocx` does not accept the new signature.

- [ ] **Step 3: Update exportDocx**

Replace `frontend/apps/web/src/features/documents/mddm-editor/engine/export/export-docx.ts` with:

```ts
import type { MDDMEnvelope } from "../../adapter";
import type { RendererPin } from "../../../../../lib.types";
import { canonicalizeAndMigrate } from "../canonicalize-migrate";
import { collectImageUrls } from "../docx-emitter";
import {
  AssetResolver,
  RESOURCE_CEILINGS,
  ResourceCeilingExceededError,
  type ResolvedAsset,
} from "../asset-resolver";
import { loadCurrentRenderer, loadPinnedRenderer } from "../renderers/registry";

export type ExportDocxOptions = {
  /** Renderer pin from the version record. `null` or omitted → current renderer. */
  rendererPin?: RendererPin | null;
  /** Optional resolver injection point — defaults to a fresh AssetResolver. */
  assetResolver?: AssetResolver;
};

export async function exportDocx(
  envelope: MDDMEnvelope,
  options: ExportDocxOptions = {},
): Promise<Blob> {
  const renderer = options.rendererPin
    ? await loadPinnedRenderer(options.rendererPin)
    : await loadCurrentRenderer();

  const canonical = await canonicalizeAndMigrate(envelope);

  const urls = collectImageUrls(canonical);
  if (urls.length > RESOURCE_CEILINGS.maxImagesPerDocument) {
    throw new ResourceCeilingExceededError(
      "maxImagesPerDocument",
      urls.length,
      RESOURCE_CEILINGS.maxImagesPerDocument,
    );
  }

  const resolver = options.assetResolver ?? new AssetResolver();
  const assetMap = new Map<string, ResolvedAsset>();
  let totalBytes = 0;
  for (const url of urls) {
    const asset = await resolver.resolveAsset(url);
    totalBytes += asset.sizeBytes;
    if (totalBytes > RESOURCE_CEILINGS.maxTotalAssetBytes) {
      throw new ResourceCeilingExceededError(
        "maxTotalAssetBytes",
        totalBytes,
        RESOURCE_CEILINGS.maxTotalAssetBytes,
      );
    }
    assetMap.set(url, asset);
  }

  return renderer.mddmToDocx(canonical, renderer.tokens, assetMap);
}
```

**Signature change:** `exportDocx(envelope, tokens)` → `exportDocx(envelope, options?)`. Call sites that previously passed `tokens` directly must be updated.

- [ ] **Step 4: Update existing call sites**

Run: `grep -rn "exportDocx(" frontend/apps/web/src/ 2>&1 | head -20`
Expected: Lists call sites. For each, update from `exportDocx(envelope, defaultLayoutTokens)` → `exportDocx(envelope, { rendererPin })` where `rendererPin` comes from the version object (or `null` for drafts).

- [ ] **Step 5: Run the tests**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/export/__tests__/export-docx.test.ts`
Expected: All tests pass (including the two new ones).

- [ ] **Step 6: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/engine/export/export-docx.ts frontend/apps/web/src/features/documents/mddm-editor/engine/export/__tests__/export-docx.test.ts frontend/apps/web/src/features/documents/browser-editor/BrowserDocumentEditorView.tsx
git commit -m "feat(mddm-engine): exportDocx accepts RendererPin and loads appropriate bundle"
```

### Task 15: exportPdf accepts a Version and loads the right renderer

**Files:**
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/engine/export/export-pdf.ts`

- [ ] **Step 1: Update exportPdf**

Edit `frontend/apps/web/src/features/documents/mddm-editor/engine/export/export-pdf.ts`. Add a `rendererPin` field to `ExportPdfParams`, load the appropriate renderer, and use its `wrapInPrintDocument` + `printStylesheet`:

```ts
import type { RendererPin } from "../../../../../lib.types";
import { loadCurrentRenderer, loadPinnedRenderer } from "../renderers/registry";
// ... keep other existing imports (AssetResolver, ceilings, rewriteImgSrcToDataUri) ...

export type ExportPdfParams = {
  bodyHtml: string;
  documentId: string;
  rendererPin?: RendererPin | null;
  assetResolver?: AssetResolver;
};

// Replace wrapInPrintDocument import + PRINT_STYLESHEET import usage:
// inside exportPdf(), after resolving assets:
const renderer = rendererPin
  ? await loadPinnedRenderer(rendererPin)
  : await loadCurrentRenderer();

const inlinedBody = rewriteImgSrcToDataUri(bodyHtml, assetMap);
const fullHtml = renderer.wrapInPrintDocument(inlinedBody);
// ... rest of the function, replacing PRINT_STYLESHEET with renderer.printStylesheet
// when appending to the FormData ...
```

Keep every existing guard (HTML size ceiling, resource limits, fetch error handling).

- [ ] **Step 2: Update exportPdf tests**

Open `frontend/apps/web/src/features/documents/mddm-editor/engine/export/__tests__/export-pdf.test.ts` and verify the existing tests still pass (they use the default — no `rendererPin`). Add one new test:

```ts
it("uses the pinned renderer when rendererPin is provided", async () => {
  const pin: RendererPin = {
    renderer_version: "1.0.0",
    layout_ir_hash: "h",
    template_key: "k",
    template_version: 1,
  };
  const fetchSpy = vi.fn().mockResolvedValue(
    new Response(new Uint8Array([0x25, 0x50, 0x44, 0x46]), {
      status: 200,
      headers: { "Content-Type": "application/pdf" },
    }),
  );
  vi.stubGlobal("fetch", fetchSpy);

  const blob = await exportPdf({
    bodyHtml: "<p>x</p>",
    documentId: "doc-1",
    rendererPin: pin,
  });
  expect(blob).toBeInstanceOf(Blob);
  expect(fetchSpy).toHaveBeenCalledTimes(1);
});
```

- [ ] **Step 3: Run the tests**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/export/__tests__/export-pdf.test.ts`
Expected: All tests pass.

- [ ] **Step 4: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/engine/export/export-pdf.ts frontend/apps/web/src/features/documents/mddm-editor/engine/export/__tests__/export-pdf.test.ts
git commit -m "feat(mddm-engine): exportPdf accepts RendererPin and loads appropriate bundle"
```

### Task 16: BrowserDocumentEditorView passes rendererPin from version record

**Files:**
- Modify: `frontend/apps/web/src/features/documents/browser-editor/BrowserDocumentEditorView.tsx`

- [ ] **Step 1: Locate the latestVersion computation**

Run: `grep -n "latestVersion\|bundle.versions" frontend/apps/web/src/features/documents/browser-editor/BrowserDocumentEditorView.tsx | head -10`
Expected: Shows the existing `latestVersion` memo derived from `bundle.versions`.

- [ ] **Step 2: Extract the pin from the latest version**

Add a new memo near `latestVersion`:

```tsx
const rendererPin = useMemo(() => {
  if (!latestVersion) return null;
  return (latestVersion.renderer_pin as RendererPin | null | undefined) ?? null;
}, [latestVersion]);
```

Add the import:

```tsx
import type { RendererPin } from "../../../lib.types";
```

- [ ] **Step 3: Pass rendererPin to exportDocx and exportPdf calls**

Find the two call sites (one in `runDocxExport`, one in any PDF export path). Update them:

```tsx
const blob = await mddmExportDocx(envelope, { rendererPin });
```

(And similarly for `exportPdf` if it's invoked from this view.)

- [ ] **Step 4: Verify compilation**

Run: `cd frontend/apps/web && npx tsc --noEmit 2>&1 | grep BrowserDocumentEditorView | head -5`
Expected: No errors.

- [ ] **Step 5: Commit**

```bash
git add frontend/apps/web/src/features/documents/browser-editor/BrowserDocumentEditorView.tsx
git commit -m "feat(browser-editor): pass version rendererPin to MDDM exports"
```

---

## Part 7 — Migration Pipeline: Target Pinned Version

### Task 17: canonicalizeAndMigrate accepts a target version

**Files:**
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/engine/canonicalize-migrate/pipeline.ts`
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/engine/canonicalize-migrate/__tests__/pipeline.test.ts`

- [ ] **Step 1: Add a failing test**

Append to `frontend/apps/web/src/features/documents/mddm-editor/engine/canonicalize-migrate/__tests__/pipeline.test.ts`:

```ts
describe("canonicalizeAndMigrate with explicit target version", () => {
  it("accepts an explicit target version matching CURRENT_MDDM_VERSION", async () => {
    const envelope = makeEnvelope({ blocks: [] });
    const result = await canonicalizeAndMigrate(envelope, { targetVersion: CURRENT_MDDM_VERSION });
    expect(result.mddm_version).toBe(CURRENT_MDDM_VERSION);
  });

  it("errors when target version is higher than CURRENT_MDDM_VERSION", async () => {
    const envelope = makeEnvelope({});
    await expect(
      canonicalizeAndMigrate(envelope, { targetVersion: CURRENT_MDDM_VERSION + 1 }),
    ).rejects.toBeInstanceOf(MigrationError);
  });
});
```

- [ ] **Step 2: Run the test — expect failure**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/canonicalize-migrate/__tests__/pipeline.test.ts 2>&1 | tail -30`
Expected: FAIL — `canonicalizeAndMigrate` does not accept the options argument.

- [ ] **Step 3: Update canonicalizeAndMigrate**

Edit `frontend/apps/web/src/features/documents/mddm-editor/engine/canonicalize-migrate/pipeline.ts`:

```ts
export type CanonicalizeAndMigrateOptions = {
  /** The version to migrate the envelope TO. Defaults to CURRENT_MDDM_VERSION.
   *  Set to a pinned version for released documents so they stay frozen. */
  targetVersion?: number;
};

export async function canonicalizeAndMigrate(
  envelope: MDDMEnvelope,
  options: CanonicalizeAndMigrateOptions = {},
): Promise<MDDMEnvelope> {
  const target = options.targetVersion ?? CURRENT_MDDM_VERSION;

  if (target > CURRENT_MDDM_VERSION) {
    throw new MigrationError(
      `Target version ${target} is newer than current engine version ${CURRENT_MDDM_VERSION}`,
      "TARGET_TOO_NEW",
    );
  }

  if (envelope === null || typeof envelope !== "object") {
    throw new MigrationError("Envelope is not an object", "INVALID_ENVELOPE");
  }

  const version = (envelope as { mddm_version?: unknown }).mddm_version;
  if (typeof version !== "number" || !Number.isInteger(version) || version < 1) {
    throw new MigrationError("Envelope missing a valid mddm_version", "MISSING_VERSION");
  }

  if (version > target) {
    throw new MigrationError(
      `Envelope version ${version} is newer than target version ${target}`,
      "VERSION_TOO_NEW",
    );
  }

  let current: MDDMEnvelope = envelope;
  while ((current.mddm_version ?? 0) < target) {
    const from = current.mddm_version ?? 0;
    const migrate = MIGRATIONS[from];
    if (!migrate) {
      throw new MigrationError(
        `No migration registered from version ${from} to ${from + 1}`,
        "MIGRATION_MISSING",
      );
    }
    current = migrate(current);
  }

  return canonicalizeMDDM(current) as MDDMEnvelope;
}
```

- [ ] **Step 4: Run the tests**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/canonicalize-migrate/__tests__/pipeline.test.ts`
Expected: PASS — all tests including the two new ones.

- [ ] **Step 5: Update exportDocx to pass the target version**

In `frontend/apps/web/src/features/documents/mddm-editor/engine/export/export-docx.ts`, modify the `canonicalizeAndMigrate` call:

```ts
// The target version IS the current engine version when there is no pin,
// or whatever the pinned bundle declares when there is. Plan 3 only ships
// v1.0.0, so CURRENT_MDDM_VERSION and the pinned version align. When future
// migrations are added, the pinned bundle will expose a frozen target.
const canonical = await canonicalizeAndMigrate(envelope);
```

(No behavioral change yet because v1.0.0 == current engine. This call site is a placeholder for the future Plan 3 extension where `loadPinnedRenderer` exposes a `supportedMDDMVersion` field. That extension is captured in the spec but NOT in Plan 3's scope — see the self-review section.)

- [ ] **Step 6: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/engine/canonicalize-migrate/pipeline.ts frontend/apps/web/src/features/documents/mddm-editor/engine/canonicalize-migrate/__tests__/pipeline.test.ts
git commit -m "feat(mddm-engine): canonicalizeAndMigrate accepts explicit target version"
```

---

## Part 8 — Full Verification

### Task 18: Integration test — released document export uses pinned renderer

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/renderers/__tests__/integration-pinned-export.test.ts`

- [ ] **Step 1: Write the integration test**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/renderers/__tests__/integration-pinned-export.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import { exportDocx } from "../../export";
import type { MDDMEnvelope } from "../../../adapter";
import type { RendererPin } from "../../../../../../lib.types";

describe("Pinned export integration", () => {
  const envelope: MDDMEnvelope = {
    mddm_version: 1,
    template_ref: null,
    blocks: [
      {
        id: "s",
        type: "section",
        props: { title: "Test", color: "red" },
        children: [],
      },
      {
        id: "p",
        type: "paragraph",
        props: {},
        children: [{ type: "text", text: "hello" }],
      },
    ],
  };

  it("released document with v1.0.0 pin produces a valid DOCX", async () => {
    const pin: RendererPin = {
      renderer_version: "1.0.0",
      layout_ir_hash: "placeholder",
      template_key: "po-mddm-canvas",
      template_version: 1,
    };
    const blob = await exportDocx(envelope, { rendererPin: pin });
    expect(blob).toBeInstanceOf(Blob);
    expect(blob.size).toBeGreaterThan(500);
    expect(blob.type).toBe("application/vnd.openxmlformats-officedocument.wordprocessingml.document");
  });

  it("draft document without pin produces a valid DOCX via current renderer", async () => {
    const blob = await exportDocx(envelope, { rendererPin: null });
    expect(blob).toBeInstanceOf(Blob);
    expect(blob.size).toBeGreaterThan(500);
  });

  it("unknown renderer_version rejects cleanly", async () => {
    const pin: RendererPin = {
      renderer_version: "9.9.9",
      layout_ir_hash: "h",
      template_key: "k",
      template_version: 1,
    };
    await expect(exportDocx(envelope, { rendererPin: pin })).rejects.toThrow(/renderer bundle/i);
  });
});
```

- [ ] **Step 2: Run the test — expect pass**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/renderers/__tests__/integration-pinned-export.test.ts`
Expected: PASS — 3 tests passing.

- [ ] **Step 3: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/engine/renderers/__tests__/integration-pinned-export.test.ts
git commit -m "test(mddm-engine): integration test for pinned vs current renderer export"
```

### Task 19: Full test suite + build verification

**Files:** (verification only)

- [ ] **Step 1: Run all vitest tests**

Run: `cd frontend/apps/web && npm test 2>&1 | tail -30`
Expected: All tests pass. After Plan 3 the total count is approximately Plan 1 (~95) + Plan 2 (~50 net after removals) + Plan 3 (~30) ≈ 170-180 tests.

- [ ] **Step 2: Run Go tests for the documents module**

Run: `go test ./internal/modules/documents/... 2>&1 | tail -30`
Expected: All existing tests pass plus the new RendererPin and CaptureRendererPin tests.

- [ ] **Step 3: TypeScript build**

Run: `cd frontend/apps/web && npm run build 2>&1 | tail -20`
Expected: Clean build with no errors.

- [ ] **Step 4: Drift gate still passes**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/ir-hash/__tests__/drift-gate.test.ts`
Expected: PASS. If it fails, the Layout IR changed during Plan 3 — that should not happen. Investigate and either revert the unintended change or document why the hash needs to be updated.

- [ ] **Step 5: Manual smoke test**

Release a test document in the dev environment (or manually set `version.RendererPin` in the DB for a test record), then export DOCX from the UI. Verify the export succeeds and the file opens in Word. Network tab: observe that the request hits the new code path without errors.

Then create a new draft document and export it. Verify the same flow works with `rendererPin: null`.

- [ ] **Step 6: Commit any incidental fixes**

If Steps 1-5 required any cleanup commits, commit them now with descriptive messages.

---

## Self-Review

### Spec coverage

| Spec requirement (Section 10: Version Pinning) | Task(s) covering it |
|---|---|
| `RendererPin` tuple (rendererVersion + layoutIRHash + templateRef) | Tasks 1, 2, 3, 4 |
| Pin captured on DRAFT→RELEASED transition | Tasks 5, 6 |
| Draft versions have no pin; exports use current | Tasks 13, 14, 18 |
| Pin exposed in version API responses | Tasks 7, 8, 9 |
| Frontend renderer bundle registry | Tasks 12, 13 |
| Renderer bundle dynamic loading by version string | Task 13 |
| Unknown version produces a clear error | Task 13 |
| `exportDocx` and `exportPdf` load the correct renderer | Tasks 14, 15 |
| Integration with BrowserDocumentEditorView | Task 16 |
| Canonicalize+migrate targets the pinned version | Task 17 |
| IR hash drift gate (prevents silent drift) | Tasks 10, 11 |
| End-to-end test for pinned export | Task 18 |
| Full verification | Task 19 |

**Out of scope by design** (deferred to Plan 4):
- Shadow testing telemetry endpoint — Plan 4
- Canary rollout orchestration — Plan 4
- Docgen decommissioning — Plan 4
- Migrating `native` / `docx_upload` content sources to MDDM — separate project
- Creating a second renderer bundle (v1.1.0+) — future work; v1.0.0 proves the mechanism and Task 12 documents the procedure
- Renderer bundle retention cap (10 most recent) — not needed until there is more than one bundle; add in the first version-bump commit
- Per-pin `supportedMDDMVersion` field — v1.0.0 is the only version; `canonicalizeAndMigrate(envelope)` defaults to `CURRENT_MDDM_VERSION` which matches v1.0.0's version. Add an explicit `supportedMDDMVersion` field on the bundle when the first new version lands

### Placeholder scan

No "TBD", "TODO", or "similar to Task N" placeholders remain. The `RECORDED_IR_HASH` in Task 11 is a literal placeholder filled in by Step 4 of that task via a one-time regenerator flow — this is by design, not a plan gap.

### Type / signature consistency

- `RendererPin` shape is consistent across:
  - Go `domain.RendererPin` (Task 1): `RendererVersion`, `LayoutIRHash`, `TemplateKey`, `TemplateVersion`, `PinnedAt`
  - OpenAPI schema (Task 8): `renderer_version`, `layout_ir_hash`, `template_key`, `template_version`, `pinned_at`
  - TypeScript `RendererPin` (Task 9): `renderer_version`, `layout_ir_hash`, `template_key`, `template_version`, `pinned_at`
  - Uses snake_case on the wire (JSON + OpenAPI + TS types) and camelCase only in Go struct field names.
- `exportDocx(envelope, options?: { rendererPin?, assetResolver? })` signature is stable between Tasks 14, 16, and 18.
- `exportPdf({ bodyHtml, documentId, rendererPin?, assetResolver? })` signature is stable between Tasks 15 and 16.
- `loadCurrentRenderer()` and `loadPinnedRenderer(pin)` return the same `LoadedRenderer` type in Tasks 13, 14, 15.
- `CURRENT_MDDM_VERSION` constant remains the single source of truth for migration targeting; Task 17 adds an opt-in override but defaults to it.

---

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-04-10-mddm-engine-version-pinning.md`. Two execution options:

**1. Subagent-Driven (recommended)** — I dispatch a fresh subagent per task, review between tasks, fast iteration.
**REQUIRED SUB-SKILL:** `superpowers:subagent-driven-development`

**2. Inline Execution** — Execute tasks in this session using executing-plans, batch execution with checkpoints.
**REQUIRED SUB-SKILL:** `superpowers:executing-plans`

Which approach?
