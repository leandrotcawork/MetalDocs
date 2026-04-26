# Plan — Document Submit Pipeline Unblock
**Date:** 2026-04-26
**Status:** In progress
**Goal:** Fix every gap blocking document submit → approval → fanout. Every document submission currently fails due to `enforce_snapshot_on_submit_trg` trigger requiring 6 snapshot columns that are never populated.

---

## Root Cause Chain

1. `repository.CreateDocument` INSERT omits `profile_code_snapshot` + `process_area_code_snapshot` (migration 0129 marks NOT NULL)
2. `SnapshotTemplateReader` interface has no production implementation — only test fakes
3. `module.go` uses `NewService` not `NewServiceWithSnapshot`; `Dependencies` has no snapshot fields
4. `main.go` never builds `SnapshotService`; `docDeps` has no snapshot fields
5. `main.go:243-247` — `DecisionService` silently installs nil `freezeInvoker` when `METALDOCS_FANOUT_URL` unset → crashes on approval

**Architecture decision:** Snapshot at document creation time (not submit). Design already exists (`SnapshotService`, `NewServiceWithSnapshot`, `snapshot_wire_test.go`) — this is a wiring problem.

**`composition_config` for catalog-only templates:** `{}` — freeze service handles this at `freeze_service.go:202-205`.

---

## Wave A — Parallel (no dependencies between tasks)

### Task 1 — Fix CreateDocument INSERT
**File:** `internal/modules/documents_v2/repository/repository.go`
**Change:** Add `profile_code_snapshot, process_area_code_snapshot` to INSERT (lines 43-49). Domain struct already has these fields (`d.ProfileCodeSnapshot`, `d.ProcessAreaCodeSnapshot`).
```sql
INSERT INTO documents (tenant_id, template_version_id, name, status, form_data_json,
                       created_by, controlled_document_id,
                       profile_code_snapshot, process_area_code_snapshot)
VALUES ($1, $2, $3, 'draft', $4, $5, $6, $7, $8) RETURNING id
```
**Test:** Add `TestCreateDocument_PersistsBridgeSnapshotColumns` to `repository/repository_integration_test.go` (build tag `integration`).
**Verify:** `go test ./internal/modules/documents_v2/repository/... -tags=integration -count=1`

---

### Task 2 — Implement production SnapshotTemplateReader
**New file:** `internal/platform/docgenv2/templates_v2_snapshot_reader.go`
**Package:** `docgenv2`
**Type:** `TemplatesV2SnapshotReader struct { db *sql.DB }`
**Constructor:** `func NewTemplatesV2SnapshotReader(db *sql.DB) *TemplatesV2SnapshotReader`
**Method:** `LoadForSnapshot(ctx, tenantID, templateVersionID string) (domain.TemplateSnapshot, error)`

SQL:
```sql
SELECT coalesce(tv.placeholder_schema::text, '{}'),
       coalesce(tv.docx_storage_key, '')
  FROM templates_v2_template_version tv
  JOIN templates_v2_template tpl ON tpl.id = tv.template_id
 WHERE tv.id = $1::uuid
   AND tpl.tenant_id = $2::uuid
   AND tv.status = 'published'
```

Returns: `domain.TemplateSnapshot{ PlaceholderSchemaJSON: []byte(phJSON), CompositionJSON: []byte("{}"), BodyDocxBytes: nil, BodyDocxS3Key: docxKey }`
On `sql.ErrNoRows`: return `domain.ErrSnapshotTemplateNotFound`

**Also add to** `internal/modules/documents_v2/domain/snapshot.go`:
```go
var ErrSnapshotTemplateNotFound = errors.New("snapshot_template_not_found")
```

**Test:** `internal/platform/docgenv2/templates_v2_snapshot_reader_test.go` (build tag `integration`). Seed minimal template rows, assert returned snapshot fields.
**Verify:** `go build ./... && go test ./internal/platform/docgenv2/... -tags=integration -count=1`

---

### Task 3 — Wire SnapshotService into module.go + Dependencies
**File:** `internal/modules/documents_v2/module.go`

Add to `Dependencies`:
```go
SnapshotReader application.SnapshotTemplateReader
SnapshotWriter application.SnapshotWriter
```

In `New(deps)`, replace service construction with:
```go
var svc *application.Service
if deps.SnapshotReader != nil && deps.SnapshotWriter != nil {
    snapSvc := application.NewSnapshotService(deps.SnapshotReader, deps.SnapshotWriter)
    svc = application.NewServiceWithSnapshot(repo, deps.Docgen, deps.Presign, deps.TplRead, deps.FormVal, deps.Audit, deps.RegistryReader, deps.AuthzChecker, deps.ProfileDefaults, snapSvc)
} else {
    svc = application.NewService(repo, deps.Docgen, deps.Presign, deps.TplRead, deps.FormVal, deps.Audit, deps.RegistryReader, deps.AuthzChecker, deps.ProfileDefaults)
}
```

**Note:** Verify exact parameter names/order from `NewService`/`NewServiceWithSnapshot` signatures before writing.

**Test:** `internal/modules/documents_v2/module_test.go` — `TestNew_WithSnapshotReader_NoPanic`
**Verify:** `go build ./...`

---

### Task 5 — Fail fast on missing freeze config
**File:** `apps/api/cmd/metaldocs-api/main.go` (lines ~243-247)

When `fanoutURL == ""`:
- If `os.Getenv("METALDOCS_REQUIRE_FANOUT") == "true"` → `log.Fatalf("METALDOCS_FANOUT_URL is required")`
- Otherwise → `slog.Warn("METALDOCS_FANOUT_URL unset; approval signoff will fail at freeze step")`

Keep existing conditional `if freezeSvc != nil { approvalServices.Decision = ... }` block.

**Verify:** `METALDOCS_REQUIRE_FANOUT=true METALDOCS_FANOUT_URL= go build ./apps/api/cmd/metaldocs-api && go run ./apps/api/cmd/metaldocs-api` exits with fatal message.

---

### Task 7 — Wiki update
**Files to update (stamp `Last verified: 2026-04-26`):**
- `wiki/README.md` — add note: snapshot columns populated at create-time by `SnapshotService` via `Dependencies.SnapshotReader/SnapshotWriter`
- `wiki/architecture/system-overview.md` — correct any wording suggesting snapshot at submit time
- Check `wiki/` for any stale snapshot/submit references

**Verify:** `grep -R "Last verified: 2026-04-26" wiki/` returns updated files

---

## Wave B — Sequential (after Wave A complete)

### Task 4 — Wire SnapshotService from main.go
**File:** `apps/api/cmd/metaldocs-api/main.go`
**After Task 2 + Task 3 complete.**

Before `docDeps := documents_v2.Dependencies{...}`:
```go
snapshotReader := docgenv2.NewTemplatesV2SnapshotReader(deps.SQLDB)
snapshotWriter := docrepo.NewSnapshotRepository(deps.SQLDB)
```

Add to `docDeps` literal:
```go
SnapshotReader: snapshotReader,
SnapshotWriter: snapshotWriter,
```

**Verify:** `go build ./apps/api/cmd/metaldocs-api`

---

### Task 6 — Integration test: create → all 8 columns non-NULL
**File:** `internal/modules/documents_v2/application/create_document_snapshot_integration_test.go`
**After Task 1 + Task 2 + Task 3 complete.**

Test: `TestCreateDocument_PopulatesAllSnapshotColumns` (build tag `integration`)
- Seed: `controlled_documents`, `templates_v2_template`, `templates_v2_template_version` (status=`published`)
- Build real `SnapshotTemplateReader`, `SnapshotRepository`, `NewServiceWithSnapshot`
- Call `svc.CreateDocument(...)`
- Assert 8 columns non-NULL: `placeholder_schema_snapshot`, `placeholder_schema_hash`, `composition_config_snapshot`, `composition_config_hash`, `body_docx_snapshot_s3_key`, `body_docx_hash`, `profile_code_snapshot`, `process_area_code_snapshot`

**Verify:** `go test ./internal/modules/documents_v2/application/... -run TestCreateDocument_PopulatesAllSnapshotColumns -tags=integration -count=1`

---

## Execution summary
```
Wave A (parallel): Task 1 | Task 2 | Task 3 | Task 5 | Task 7
Wave B (sequential after A): Task 4 | Task 6
Final: go build ./... + go test ./...
```
