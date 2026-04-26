# Placeholder Fixed Catalog Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use nexus:subagent-driven-development to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace user-fillable placeholder system with a fixed catalog of 7 metadata-only tokens that auto-resolve at document open and freeze.

**Architecture:** Lock placeholders to a backend-defined enum exposed via `GET /api/v2/templates/v2/placeholder-catalog`. Template author can no longer create or type-configure placeholders — they pick from the catalog. Document editor drops the right-side fill panel; tokens stay literal `{token}` text in writer mode (substituted by backend at finalize/freeze). `applyVariables` is **NOT** applied at open in writer mode because eigenpal's `applyVariables` mutates the underlying DOCX and would destroy original tokens on autosave. Catalog reference panel in template author + document editor shows what each token will resolve to. Existing `Placeholder` domain model stays but `Type` is forced to `computed`, `Computed=true`, and `ResolverKey` set automatically from the token name.

**Tech Stack:** Go 1.23 (backend), React 18 + TypeScript + Vite (frontend), eigenpal `@eigenpal/docx-js-editor` (DOCX rendering), vitest (frontend tests), `go test` (backend).

**Execution model (per user request, trimmed for ship):**
- **Codex (`codex:rescue`)** — implementation of every code task, prompted /caveman-style.
- **Haiku** — trivial test scaffolding and file moves.
- **`/simplify`** — invoked on every diff before commit.
- **Codex plan validation** — already run, blockers resolved.
- **Parallel subagents** — Phase 1 runs 3 tasks in parallel; Phase 4 runs 2 in parallel.
- **Final review only** — single Opus reviewer + Wiki Keeper at branch end (skip per-phase ceremony).

---

## File Structure

**New files:**
- `internal/modules/render/resolvers/doc_title.go` — `DocTitleResolver` impl
- `internal/modules/render/resolvers/doc_title_test.go` — resolver test
- `internal/modules/templates_v2/delivery/http/routes_catalog.go` — `GET /api/v2/templates/v2/placeholder-catalog`
- `internal/modules/templates_v2/delivery/http/routes_catalog_test.go` — endpoint test
- `frontend/apps/web/src/features/documents/v2/hooks/useResolvedValues.ts` — fetch resolver map for applyVariables
- `frontend/apps/web/src/features/templates/v2/PlaceholderCatalogPanel.tsx` — read-only catalog reference panel
- `wiki/decisions/0008-placeholder-fixed-catalog.md` — ADR

**Modified files:**
- `internal/modules/render/resolvers/builtins.go` — register `DocTitleResolver`
- `internal/modules/render/resolvers/approvers.go` — empty list returns `"[aguardando aprovação]"`
- `internal/modules/render/resolvers/approvers_test.go` — assert pre-approval label
- `internal/modules/templates_v2/application/schema.go` — `ValidatePlaceholders` enforces catalog enum
- `internal/modules/templates_v2/application/schema_test.go` — test enum rejection
- `internal/modules/templates_v2/delivery/http/handler.go` — register catalog route
- `internal/modules/documents_v2/http/fillin_handler.go` — auto-populate values from resolvers, drop user PUT support
- `frontend/apps/web/src/features/documents/v2/DocumentEditorPage.tsx` — remove `<aside>` fill panel, wire `applyVariables`
- `frontend/apps/web/src/features/templates/v2/TemplateAuthorPage.tsx` — replace placeholder management with catalog reference + token auto-detect
- `frontend/apps/web/src/features/templates/__tests__/template-author-page-convergence.test.tsx` — rewrite for new behavior
- `wiki/concepts/placeholders.md` — rewrite to new model

**Deleted files:**
- `frontend/apps/web/src/features/documents/placeholder-form.tsx` — fill form, dead
- `frontend/apps/web/src/features/templates/placeholder-inspector.tsx` — type/constraints UI, dead

---

## Phase 0: Plan validation (Codex review)

**Before any code lands, validate the whole plan with Codex.**

- [ ] **Step 0.1: Dispatch `codex:rescue` agent for plan review**

Prompt (caveman):
```
read plan docs/superpowers/plans/2026-04-26-placeholder-fixed-catalog.md.
critique:
- gaps in spec coverage?
- ordering wrong? hidden coupling between phases?
- file paths real? (cross-check)
- TDD steps testable?
- migration risk (existing schemas with text/select placeholders)?
return: punch list, max 30 lines.
```

- [ ] **Step 0.2: Address Codex findings inline before proceeding**

If Codex flags gaps, fix the plan file. Re-run only if structural changes.

---

## Phase 1: Backend resolvers + catalog endpoint (parallel-safe)

**Three independent tasks. Dispatch 3 implementer subagents in parallel.**

### Task 1A: DocTitleResolver

**Files:**
- Create: `internal/modules/render/resolvers/doc_title.go`
- Create: `internal/modules/render/resolvers/doc_title_test.go`
- Modify: `internal/modules/render/resolvers/builtins.go`

- [ ] **Step 1A.1: Write failing test**

`internal/modules/render/resolvers/doc_title_test.go`:
```go
package resolvers

import (
	"context"
	"testing"
)

type fakeDocReader struct{ title string }

func (f fakeDocReader) GetDocumentTitle(_ context.Context, _, _ string) (string, error) {
	return f.title, nil
}

func TestDocTitleResolver_Key(t *testing.T) {
	if got := (DocTitleResolver{}).Key(); got != "doc_title" {
		t.Fatalf("Key = %q, want %q", got, "doc_title")
	}
}

func TestDocTitleResolver_Resolve(t *testing.T) {
	r := DocTitleResolver{}
	in := ResolveInput{
		TenantID:    "t1",
		RevisionID:  "rev1",
		DocumentReader: fakeDocReader{title: "E2E Workflow Test - Rev 1"},
	}
	out, err := r.Resolve(context.Background(), in)
	if err != nil {
		t.Fatalf("Resolve err = %v", err)
	}
	if out.Value != "E2E Workflow Test - Rev 1" {
		t.Fatalf("Value = %q", out.Value)
	}
	if out.ResolverKey != "doc_title" {
		t.Fatalf("ResolverKey = %q", out.ResolverKey)
	}
}
```

- [ ] **Step 1A.2: Run test to verify it fails**

Run: `go test ./internal/modules/render/resolvers/ -run DocTitleResolver -v`
Expected: FAIL — `DocTitleResolver` undefined.

- [ ] **Step 1A.3: Add `DocumentReader` to `ResolveInput`**

Edit `internal/modules/render/resolvers/resolver.go` — add field:
```go
type DocumentReader interface {
	GetDocumentTitle(ctx context.Context, tenantID, revisionID string) (string, error)
}

type ResolveInput struct {
	// ... existing fields
	DocumentReader DocumentReader
}
```

- [ ] **Step 1A.4: Implement `DocTitleResolver`**

`internal/modules/render/resolvers/doc_title.go`:
```go
package resolvers

import (
	"context"
	"time"
)

type DocTitleResolver struct{}

func (DocTitleResolver) Key() string  { return "doc_title" }
func (DocTitleResolver) Version() int { return 1 }

func (DocTitleResolver) Resolve(ctx context.Context, in ResolveInput) (ResolvedValue, error) {
	title, err := in.DocumentReader.GetDocumentTitle(ctx, in.TenantID, in.RevisionID)
	if err != nil {
		return ResolvedValue{}, err
	}
	inputsHash, err := hashInputs(struct {
		TenantID   string `json:"tenant_id"`
		RevisionID string `json:"revision_id"`
	}{in.TenantID, in.RevisionID})
	if err != nil {
		return ResolvedValue{}, err
	}
	return ResolvedValue{
		Value:       title,
		ResolverKey: "doc_title",
		ResolverVer: 1,
		InputsHash:  inputsHash,
		ComputedAt:  time.Now().UTC(),
	}, nil
}
```

- [ ] **Step 1A.5: Register in builtins**

Edit `internal/modules/render/resolvers/builtins.go`:
```go
func RegisterBuiltins(r *Registry) {
	r.Register(DocCodeResolver{})
	r.Register(DocTitleResolver{})
	r.Register(RevisionNumberResolver{})
	r.Register(EffectiveDateResolver{})
	r.Register(ControlledByAreaResolver{})
	r.Register(AuthorResolver{})
	r.Register(ApproversResolver{})
	r.Register(ApprovalDateResolver{})
}
```

- [ ] **Step 1A.6: Implement `RevisionReader.GetDocumentTitle` in postgres reader**

Edit `internal/modules/documents_v2/repository/resolver_readers.go` — add:
```go
func (r *RevisionReader) GetDocumentTitle(ctx context.Context, tenantID, revisionID string) (string, error) {
	var title string
	err := r.db.QueryRowContext(ctx,
		`SELECT name FROM documents WHERE tenant_id=$1::uuid AND id=$2::uuid`,
		tenantID, revisionID,
	).Scan(&title)
	return title, err
}
```

(Note: `revisionID` actually equals document ID in this codebase — verify before commit. If revisions have own titles, query by `revision_id`.)

- [ ] **Step 1A.7: Run tests to verify pass**

Run: `go test ./internal/modules/render/resolvers/ -run DocTitleResolver -v`
Expected: PASS.

Run: `go test ./internal/modules/render/resolvers/ -run TestRegisterBuiltins -v`
Expected: PASS.

- [ ] **Step 1A.8: `/simplify` the diff**

Invoke `/simplify` skill on changed files.

- [ ] **Step 1A.9: Commit**

```bash
git add internal/modules/render/resolvers/doc_title.go \
        internal/modules/render/resolvers/doc_title_test.go \
        internal/modules/render/resolvers/builtins.go \
        internal/modules/render/resolvers/resolver.go \
        internal/modules/documents_v2/repository/resolver_readers.go
git commit -m "feat(resolvers): add doc_title computed placeholder"
```

---

### Task 1B: ApproversResolver pre-approval label

**Files:**
- Modify: `internal/modules/render/resolvers/approvers.go`
- Modify: `internal/modules/render/resolvers/approvers_test.go`

- [ ] **Step 1B.1: Write failing test**

Append to `internal/modules/render/resolvers/approvers_test.go`:
```go
func TestApproversResolver_NoApprovers_ReturnsPortuguesePending(t *testing.T) {
	r := ApproversResolver{}
	in := ResolveInput{
		TenantID:       "t1",
		RevisionID:     "rev1",
		WorkflowReader: fakeWorkflowReader{approvers: nil},
	}
	out, err := r.Resolve(context.Background(), in)
	if err != nil {
		t.Fatalf("Resolve err = %v", err)
	}
	if out.Value != "[aguardando aprovação]" {
		t.Fatalf("Value = %q, want %q", out.Value, "[aguardando aprovação]")
	}
}
```

- [ ] **Step 1B.2: Run test to verify it fails**

Run: `go test ./internal/modules/render/resolvers/ -run TestApproversResolver_NoApprovers -v`
Expected: FAIL.

- [ ] **Step 1B.3: Modify resolver to return formatted string**

Current code returns the raw approvers slice as `Value`. We need a string for token substitution. Edit `internal/modules/render/resolvers/approvers.go`:
```go
package resolvers

import (
	"context"
	"strings"
	"time"
)

type ApproversResolver struct{}

func (ApproversResolver) Key() string  { return "approvers" }
func (ApproversResolver) Version() int { return 1 }

func (ApproversResolver) Resolve(ctx context.Context, in ResolveInput) (ResolvedValue, error) {
	approvers, err := in.WorkflowReader.GetApprovers(ctx, in.TenantID, in.RevisionID)
	if err != nil {
		return ResolvedValue{}, err
	}

	var value string
	if len(approvers) == 0 {
		value = "[aguardando aprovação]"
	} else {
		names := make([]string, 0, len(approvers))
		for _, a := range approvers {
			if a.DisplayName != "" {
				names = append(names, a.DisplayName)
			}
		}
		if len(names) == 0 {
			value = "[aguardando aprovação]"
		} else {
			value = strings.Join(names, ", ")
		}
	}

	inputsHash, err := hashInputs(struct {
		TenantID   string `json:"tenant_id"`
		RevisionID string `json:"revision_id"`
	}{in.TenantID, in.RevisionID})
	if err != nil {
		return ResolvedValue{}, err
	}

	return ResolvedValue{
		Value:       value,
		ResolverKey: "approvers",
		ResolverVer: 1,
		InputsHash:  inputsHash,
		ComputedAt:  time.Now().UTC(),
	}, nil
}
```

(Note: this changes `Value` type from slice to string. Callers that consumed the slice must adapt — Codex review will catch any.)

- [ ] **Step 1B.4: Run test to verify pass**

Run: `go test ./internal/modules/render/resolvers/ -run TestApproversResolver -v`
Expected: PASS (all approver tests).

- [ ] **Step 1B.5: `/simplify` the diff**

- [ ] **Step 1B.6: Commit**

```bash
git add internal/modules/render/resolvers/approvers.go \
        internal/modules/render/resolvers/approvers_test.go
git commit -m "feat(resolvers): empty approvers returns aguardando aprovacao label"
```

---

### Task 1C: Placeholder catalog HTTP endpoint

**Files:**
- Create: `internal/modules/templates_v2/delivery/http/routes_catalog.go`
- Create: `internal/modules/templates_v2/delivery/http/routes_catalog_test.go`
- Modify: `internal/modules/templates_v2/delivery/http/handler.go`

- [ ] **Step 1C.1: Write failing test**

`routes_catalog_test.go` (uses existing `newMux(t, authz, repo)` helper from `routes_create_test.go:177`):
```go
package http_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPlaceholderCatalog_Returns7Entries(t *testing.T) {
	repo := newFakeRepo()
	mux := newMux(t, func(_ *http.Request, _, _, _ string) error { return nil }, repo)
	req := httptest.NewRequest("GET", "/api/v2/templates/v2/placeholder-catalog", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var body struct {
		Items []struct {
			Key         string `json:"key"`
			Label       string `json:"label"`
			Description string `json:"description"`
		} `json:"items"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if len(body.Items) != 7 {
		t.Fatalf("items len = %d, want 7", len(body.Items))
	}
	wantKeys := []string{"doc_code", "doc_title", "revision_number", "author", "effective_date", "approvers", "controlled_by_area"}
	for i, k := range wantKeys {
		if body.Items[i].Key != k {
			t.Errorf("items[%d].Key = %q, want %q", i, body.Items[i].Key, k)
		}
		if body.Items[i].Label == "" {
			t.Errorf("items[%d].Label empty", i)
		}
	}
}
```

- [ ] **Step 1C.2: Run test to verify it fails**

Run: `go test ./internal/modules/templates_v2/delivery/http/ -run PlaceholderCatalog -v`
Expected: FAIL — 404.

- [ ] **Step 1C.3: Implement catalog handler**

`internal/modules/templates_v2/delivery/http/routes_catalog.go`:
```go
package http

import (
	"net/http"

	"metaldocs/internal/platform/httpresponse"
)

type catalogEntry struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Description string `json:"description"`
}

var placeholderCatalog = []catalogEntry{
	{"doc_code", "Código do documento", "Código gerado automaticamente do documento controlado."},
	{"doc_title", "Título do documento", "Nome atual do documento."},
	{"revision_number", "Número da revisão", "Versão atual do documento."},
	{"author", "Autor", "Usuário que criou o documento."},
	{"effective_date", "Data efetiva", "Data efetiva (criação enquanto rascunho, data de aprovação após publicação)."},
	{"approvers", "Aprovadores", "Lista de aprovadores ou '[aguardando aprovação]'."},
	{"controlled_by_area", "Área controladora", "Nome da área de processo responsável."},
}

func (h *Handler) listPlaceholderCatalog(w http.ResponseWriter, r *http.Request) {
	httpresponse.WriteJSON(w, http.StatusOK, map[string]any{"items": placeholderCatalog})
}
```

- [ ] **Step 1C.4: Register route**

Edit `internal/modules/templates_v2/delivery/http/handler.go` — append to `Register` method (the actual method name, line 28):
```go
mux.HandleFunc("GET /api/v2/templates/v2/placeholder-catalog", h.listPlaceholderCatalog)
```

- [ ] **Step 1C.5: Run test to verify pass**

Run: `go test ./internal/modules/templates_v2/delivery/http/ -run PlaceholderCatalog -v`
Expected: PASS.

- [ ] **Step 1C.6: `/simplify` the diff**

- [ ] **Step 1C.7: Commit**

```bash
git add internal/modules/templates_v2/delivery/http/routes_catalog.go \
        internal/modules/templates_v2/delivery/http/routes_catalog_test.go \
        internal/modules/templates_v2/delivery/http/handler.go
git commit -m "feat(templates): expose placeholder catalog endpoint"
```

---

---

## Phase 2: Backend schema enforcement (sequential, depends on Phase 1)

### Task 2: Lock `ValidatePlaceholders` to catalog enum

**Files:**
- Modify: `internal/modules/templates_v2/application/schema.go`
- Modify: `internal/modules/templates_v2/application/schema_test.go`

- [ ] **Step 2.1: Write failing test**

Append to `schema_test.go`:
```go
func TestValidatePlaceholders_RejectsNonCatalogName(t *testing.T) {
	phs := []domain.Placeholder{
		{ID: "p1", Name: "customer_name", Label: "Customer", Type: domain.PHText},
	}
	err := ValidatePlaceholders(phs)
	if !errors.Is(err, domain.ErrPlaceholderNotInCatalog) {
		t.Fatalf("err = %v, want ErrPlaceholderNotInCatalog", err)
	}
}

func TestValidatePlaceholders_AcceptsCatalogName(t *testing.T) {
	phs := []domain.Placeholder{
		{ID: "p1", Name: "doc_code", Label: "Codigo", Type: domain.PHComputed, ResolverKey: ptr("doc_code")},
	}
	if err := ValidatePlaceholders(phs); err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
}
```

- [ ] **Step 2.2: Run test to verify it fails**

Run: `go test ./internal/modules/templates_v2/application/ -run TestValidatePlaceholders_Rejects -v`
Expected: FAIL — `ErrPlaceholderNotInCatalog` undefined.

- [ ] **Step 2.3: Add error sentinel + catalog set + force computed shape**

Edit `internal/modules/templates_v2/domain/errors.go`:
```go
var ErrPlaceholderNotInCatalog = errors.New("placeholder name not in fixed catalog")
var ErrPlaceholderNotComputed = errors.New("catalog placeholders must be computed with resolver_key")
```

Edit `internal/modules/templates_v2/application/schema.go` — add catalog set + check at top of `ValidatePlaceholders` loop (after seenNames check):
```go
var placeholderCatalogSet = map[string]struct{}{
	"doc_code": {}, "doc_title": {}, "revision_number": {},
	"author": {}, "effective_date": {}, "approvers": {}, "controlled_by_area": {},
}

// inside ValidatePlaceholders loop, after seenNames check:
if _, ok := placeholderCatalogSet[p.Name]; !ok {
    return fmt.Errorf("placeholder[%s] name %q: %w", p.ID, p.Name, domain.ErrPlaceholderNotInCatalog)
}
// catalog placeholders MUST be computed with matching ResolverKey
if p.Type != domain.PHComputed || !p.Computed || p.ResolverKey == nil || *p.ResolverKey != p.Name {
    return fmt.Errorf("placeholder[%s] %q: %w", p.ID, p.Name, domain.ErrPlaceholderNotComputed)
}
```

Add a third test:
```go
func TestValidatePlaceholders_RejectsCatalogNameWithWrongShape(t *testing.T) {
	rk := "doc_code"
	phs := []domain.Placeholder{
		{ID: "p1", Name: "doc_code", Label: "X", Type: domain.PHText, ResolverKey: &rk}, // wrong type
	}
	if err := ValidatePlaceholders(phs); !errors.Is(err, domain.ErrPlaceholderNotComputed) {
		t.Fatalf("err = %v", err)
	}
}
```

- [ ] **Step 2.4: Run tests to verify pass**

Run: `go test ./internal/modules/templates_v2/application/ -v`
Expected: All PASS.

- [ ] **Step 2.5: `/simplify`**

- [ ] **Step 2.6: Commit**

```bash
git add internal/modules/templates_v2/domain/errors.go \
        internal/modules/templates_v2/application/schema.go \
        internal/modules/templates_v2/application/schema_test.go
git commit -m "feat(templates): enforce placeholder catalog enum in validation"
```

---

### Task 2.5: Migration — wipe legacy non-catalog placeholder schemas

**Why:** Phase 2 enum check rejects pre-existing template versions on next save. Need to clear those rows so authors can re-edit.

**Files:**
- Create: `internal/modules/templates_v2/repository/postgres_migration_purge_legacy_placeholders.go` (one-shot SQL helper)
- Or simpler: a `make` target that runs raw SQL.

- [ ] **Step 2.5.1: Inspect existing template version placeholder rows**

Run SQL via DB CLI:
```sql
SELECT template_id, version_num, jsonb_array_length(placeholder_schema) AS n_phs
FROM template_versions
WHERE jsonb_array_length(placeholder_schema) > 0;
```

- [ ] **Step 2.5.2: Wipe non-catalog placeholders (one-shot SQL, dev DB only)**

```sql
UPDATE template_versions
SET placeholder_schema = '[]'::jsonb
WHERE jsonb_array_length(placeholder_schema) > 0;
```

(Plan ships dev-DB-only because we have no prod data yet. Add a real migration later if/when prod has rows.)

- [ ] **Step 2.5.3: Verify**

Run: `psql $DB_URL -c "SELECT count(*) FROM template_versions WHERE placeholder_schema != '[]'::jsonb;"`
Expected: 0.

- [ ] **Step 2.5.4: Document in commit**

```bash
git commit --allow-empty -m "chore(templates): wipe legacy placeholder schemas (dev DB)

Phase 2 enum enforcement rejects pre-existing text/select rows. Dev DB has no
real data; clearing placeholder_schema to '[]'. Template authors will rebuild
via catalog auto-detect on next save."
```

---

## Phase 3: Frontend document editor cleanup (single atomic task)

### Task 3: Kill fill panel + dead fill code

**Decision (per Codex EIGENPAL blocker):** Do NOT call `applyVariables` at open in writer mode. Eigenpal mutates the underlying DOCX, and autosave would persist the substituted bytes — destroying the original `{token}` strings forever and breaking finalize/freeze re-substitution. For ship: tokens stay literal in the editor; backend substitutes at finalize/freeze (existing fanout pipeline already does this). The catalog reference panel in template author tells the user what each token will resolve to. A future phase can wire a separate "preview" mode (readonly) using `applyVariables` once we have a clear two-buffer story.

**Files:**
- Modify: `frontend/apps/web/src/features/documents/v2/DocumentEditorPage.tsx`
- Delete: `frontend/apps/web/src/features/documents/placeholder-form.tsx`
- Delete: `frontend/apps/web/src/features/documents/fill-in-loader.ts`
- Delete: `frontend/apps/web/src/features/documents/submit-button.tsx` (if only the fill flow uses it — verify)

- [ ] **Step 3.1: Verify `submit-button.tsx` callers**

Run: Grep `SubmitButton` across `frontend/apps/web/src/`. If only `DocumentEditorPage.tsx` uses it, mark for deletion. If broader use, keep file but remove the placeholder-aware code path.

- [ ] **Step 3.2: Edit `DocumentEditorPage.tsx`**

In `frontend/apps/web/src/features/documents/v2/DocumentEditorPage.tsx`:
- Delete `const [fillIn, setFillIn] = useState<FillInData | null>(null);` (line 31)
- Delete `setFillIn(null);` and `setFillIn(fillInData);` lines (54, 68)
- Delete `loadFillInData` from the `Promise.all` (lines 55-63 → leave only `getDocument(documentID)`)
- Delete `handlePlaceholderSave` function (lines 136-144)
- Delete the entire `<aside>` fill panel block (lines 236-251)
- Remove imports: `loadFillInData`, `FillInData`, `PlaceholderForm`, `SubmitButton` (if confirmed unused), `putPlaceholderValue`, `getPlaceholderValues`

- [ ] **Step 3.3: Delete dead files in same commit**

```bash
git rm frontend/apps/web/src/features/documents/placeholder-form.tsx
git rm frontend/apps/web/src/features/documents/fill-in-loader.ts
# only if step 3.1 confirmed:
# git rm frontend/apps/web/src/features/documents/submit-button.tsx
```

- [ ] **Step 3.4: Remove dead API helpers from `documentsV2.ts`**

In `frontend/apps/web/src/features/documents/v2/api/documentsV2.ts`, delete `putPlaceholderValue` and `getPlaceholderValues` exports (and `PlaceholderValueDTO` if unused after).

- [ ] **Step 3.5: Run frontend type-check**

Run: `pnpm --filter @metaldocs/web tsc --noEmit`
Expected: 0 errors.

- [ ] **Step 3.6: Run document tests**

Run: `pnpm --filter @metaldocs/web test -- documents`
Expected: PASS, or fail only on tests asserting the fill panel — delete those test cases (don't keep tests for dead code).

- [ ] **Step 3.7: `/simplify`**

- [ ] **Step 3.8: Commit (atomic — file deletes + page edits in one commit so build never breaks)**

```bash
git add frontend/apps/web/src/features/documents/v2/DocumentEditorPage.tsx \
        frontend/apps/web/src/features/documents/v2/api/documentsV2.ts
git rm frontend/apps/web/src/features/documents/placeholder-form.tsx \
       frontend/apps/web/src/features/documents/fill-in-loader.ts
git commit -m "refactor(documents): remove placeholder fill panel and dead loaders"
```

---

## Phase 4: Frontend template author lockdown (parallel-safe)

**Two parallel tasks. Both touch `TemplateAuthorPage.tsx` but different sections — coordinate to avoid merge conflict (use one commit at end of phase).**

### Task 4A: Replace placeholder management with catalog reference panel

**Files:**
- Create: `frontend/apps/web/src/features/templates/v2/PlaceholderCatalogPanel.tsx`
- Create: `frontend/apps/web/src/features/templates/v2/api/catalog.ts`
- Modify: `frontend/apps/web/src/features/templates/v2/TemplateAuthorPage.tsx`
- Delete: `frontend/apps/web/src/features/templates/placeholder-inspector.tsx`
- Delete: `frontend/apps/web/src/features/templates/placeholder-chip.tsx` (if not used elsewhere; verify)

- [ ] **Step 4A.1: Write failing test for catalog fetch**

`frontend/apps/web/src/features/templates/v2/api/__tests__/catalog.test.ts`:
```ts
import { describe, expect, it, vi } from 'vitest';
import { fetchPlaceholderCatalog } from '../catalog';

describe('fetchPlaceholderCatalog', () => {
  it('returns 7 catalog entries from the API', async () => {
    vi.stubGlobal('fetch', vi.fn(() => Promise.resolve({
      ok: true,
      json: () => Promise.resolve({ items: [
        { key: 'doc_code', label: 'Código do documento', description: '' },
        { key: 'doc_title', label: 'Título do documento', description: '' },
        { key: 'revision_number', label: 'Número da revisão', description: '' },
        { key: 'author', label: 'Autor', description: '' },
        { key: 'effective_date', label: 'Data efetiva', description: '' },
        { key: 'approvers', label: 'Aprovadores', description: '' },
        { key: 'controlled_by_area', label: 'Área controladora', description: '' },
      ] }),
    })));
    const items = await fetchPlaceholderCatalog();
    expect(items).toHaveLength(7);
    expect(items[0].key).toBe('doc_code');
  });
});
```

- [ ] **Step 4A.2: Run test — expect fail**

Run: `pnpm --filter @metaldocs/web test -- catalog`
Expected: FAIL.

- [ ] **Step 4A.3: Implement `catalog.ts`**

```ts
export interface PlaceholderCatalogEntry {
  key: string;
  label: string;
  description: string;
}

export async function fetchPlaceholderCatalog(): Promise<PlaceholderCatalogEntry[]> {
  const r = await fetch('/api/v2/templates/v2/placeholder-catalog');
  if (!r.ok) throw new Error(`http_${r.status}`);
  const body = await r.json() as { items: PlaceholderCatalogEntry[] };
  return body.items ?? [];
}
```

- [ ] **Step 4A.4: Implement `PlaceholderCatalogPanel.tsx`**

```tsx
import React, { useEffect, useState } from 'react';
import { fetchPlaceholderCatalog, type PlaceholderCatalogEntry } from './api/catalog';

export interface PlaceholderCatalogPanelProps {
  detected: string[]; // tokens currently in document
}

export function PlaceholderCatalogPanel({ detected }: PlaceholderCatalogPanelProps): React.ReactElement {
  const [items, setItems] = useState<PlaceholderCatalogEntry[]>([]);
  useEffect(() => { void fetchPlaceholderCatalog().then(setItems); }, []);

  const detectedSet = new Set(detected);
  return (
    <aside style={{ width: 280, borderLeft: '1px solid #e2e8f0', padding: 12 }}>
      <h3>Placeholders disponíveis</h3>
      <p style={{ fontSize: 12, color: '#64748b' }}>
        Digite o nome entre chaves no documento, ex.: {'{doc_code}'}
      </p>
      <ul style={{ listStyle: 'none', padding: 0 }}>
        {items.map((it) => (
          <li
            key={it.key}
            data-testid={`catalog-${it.key}`}
            data-detected={detectedSet.has(it.key)}
            style={{
              padding: 6, borderRadius: 4, marginBottom: 4,
              background: detectedSet.has(it.key) ? '#dcfce7' : '#f1f5f9',
            }}
          >
            <code>{`{${it.key}}`}</code>
            <div style={{ fontSize: 11, color: '#475569' }}>{it.label}</div>
          </li>
        ))}
      </ul>
    </aside>
  );
}
```

- [ ] **Step 4A.5: Run tests**

Run: `pnpm --filter @metaldocs/web test -- catalog`
Expected: PASS.

- [ ] **Step 4A.6: Replace right-side panel in `TemplateAuthorPage.tsx`**

Remove inspector / chip rendering blocks (lines around `addPlaceholder`, `placeholder-chip`, `placeholder-inspector` usage). Insert:
```tsx
<PlaceholderCatalogPanel detected={detectedVariables} />
```

Where `detectedVariables` is the result of `editorRef.current?.getAgent?.()?.getVariables() ?? []` (already exists via `syncPlaceholdersFromDocument`).

- [ ] **Step 4A.7: Delete inspector + chip files**

```bash
git rm frontend/apps/web/src/features/templates/placeholder-inspector.tsx
# verify placeholder-chip.tsx — keep if referenced elsewhere; otherwise:
git rm frontend/apps/web/src/features/templates/placeholder-chip.tsx
```

- [ ] **Step 4A.8: `/simplify`**

(commit at end of phase, after 4B)

---

### Task 4B: Auto-detect tokens, save only catalog tokens to schema

**Files:**
- Modify: `frontend/apps/web/src/features/templates/v2/TemplateAuthorPage.tsx`
- Modify: `frontend/apps/web/src/features/templates/__tests__/template-author-page-convergence.test.tsx`

- [ ] **Step 4B.1: Rewrite convergence test**

Replace contents of `template-author-page-convergence.test.tsx` to assert:
1. Detected tokens IN catalog → saved to `schemas.placeholders` as `{ id, name: <key>, label: <portuguese label>, type: 'computed', resolverKey: <key> }`.
2. Detected tokens NOT in catalog → silently ignored (no save, no error).
3. No "+ Add manually" button rendered.
4. Catalog panel shows all 7 entries; ones present in document marked `data-detected=true`.

Full test (replaces existing file):
```tsx
import React from 'react';
import { act, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { TemplateAuthorPage } from '../v2/TemplateAuthorPage';
import type { TemplateSchemas } from '../v2/api/templatesV2';

let detectedVariables: string[] = [];
const saveSchemas = vi.fn();
const queueDocx = vi.fn();
const flush = vi.fn();

const baseSchemas: TemplateSchemas = {
  placeholders: [],
  composition: { headerSubBlocks: [], footerSubBlocks: [], subBlockParams: {} },
};

vi.mock('@eigenpal/docx-js-editor/styles.css', () => ({}));
vi.mock('@eigenpal/docx-js-editor/core', () => ({ createEmptyDocument: () => ({ type: 'empty-doc' }) }));
vi.mock('@eigenpal/docx-js-editor/react', () => ({
  DocxEditor: React.forwardRef(({ onChange }: { onChange?: () => void }, ref) => {
    React.useImperativeHandle(ref, () => ({
      save: () => Promise.resolve(new ArrayBuffer(1)),
      getAgent: () => ({ getVariables: () => detectedVariables }),
      getEditorRef: () => null,
    }));
    return <button data-testid="mock-editor-change" onClick={() => onChange?.()}>change</button>;
  }),
}));

vi.mock('../v2/hooks/useTemplateDraft', () => ({
  useTemplateDraft: () => ({ loading: false, error: null,
    template: { template_id: 'tpl-1', name: 'Test Template' },
    version: { template_id: 'tpl-1', version_num: 1, status: 'draft', docx_storage_key: null },
    docxBytes: null }),
}));
vi.mock('../v2/hooks/useTemplateAutosave', () => ({
  useTemplateAutosave: () => ({ queueDocx, flush, status: 'idle', hasPending: () => false }),
}));
vi.mock('../v2/hooks/useTemplateSchemas', () => ({
  useTemplateSchemas: () => ({ schemas: baseSchemas, loading: false, error: null, save: saveSchemas, saving: false }),
}));
vi.mock('../v2/api/catalog', () => ({
  fetchPlaceholderCatalog: () => Promise.resolve([
    { key: 'doc_code', label: 'Código do documento', description: '' },
    { key: 'doc_title', label: 'Título do documento', description: '' },
    { key: 'revision_number', label: 'Número da revisão', description: '' },
    { key: 'author', label: 'Autor', description: '' },
    { key: 'effective_date', label: 'Data efetiva', description: '' },
    { key: 'approvers', label: 'Aprovadores', description: '' },
    { key: 'controlled_by_area', label: 'Área controladora', description: '' },
  ]),
}));
vi.mock('sonner', () => ({ toast: { error: vi.fn() } }));

function renderPage() {
  return render(<TemplateAuthorPage templateId="tpl-1" versionNum={1} />);
}

async function triggerEditorChange() {
  fireEvent.click(screen.getByTestId('mock-editor-change'));
  await act(async () => { await Promise.resolve(); vi.advanceTimersByTime(400); });
}

describe('TemplateAuthorPage placeholder catalog', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    detectedVariables = [];
    saveSchemas.mockReset();
    queueDocx.mockReset();
    flush.mockReset();
    vi.stubGlobal('crypto', { ...crypto, randomUUID: vi.fn(() => 'generated-id') });
  });

  it('renders the 7 catalog entries', async () => {
    renderPage();
    await waitFor(() => expect(screen.getByTestId('catalog-doc_code')).toBeInTheDocument());
    expect(screen.getByTestId('catalog-doc_title')).toBeInTheDocument();
    expect(screen.getByTestId('catalog-revision_number')).toBeInTheDocument();
    expect(screen.getByTestId('catalog-author')).toBeInTheDocument();
    expect(screen.getByTestId('catalog-effective_date')).toBeInTheDocument();
    expect(screen.getByTestId('catalog-approvers')).toBeInTheDocument();
    expect(screen.getByTestId('catalog-controlled_by_area')).toBeInTheDocument();
  });

  it('does not render "+ Add manually" button', () => {
    renderPage();
    expect(screen.queryByRole('button', { name: '+ Add manually' })).toBeNull();
  });

  it('marks detected catalog tokens as detected in the panel', async () => {
    renderPage();
    detectedVariables = ['doc_code', 'author'];
    await triggerEditorChange();
    await waitFor(() => {
      expect(screen.getByTestId('catalog-doc_code').getAttribute('data-detected')).toBe('true');
      expect(screen.getByTestId('catalog-author').getAttribute('data-detected')).toBe('true');
      expect(screen.getByTestId('catalog-doc_title').getAttribute('data-detected')).toBe('false');
    });
  });

  it('saves catalog tokens to schema with computed type and resolverKey', async () => {
    renderPage();
    detectedVariables = ['doc_code', 'author'];
    await triggerEditorChange();
    await act(async () => { vi.advanceTimersByTime(400); await Promise.resolve(); });
    await waitFor(() => {
      expect(saveSchemas).toHaveBeenCalledWith({
        placeholders: [
          { id: 'generated-id', name: 'doc_code', label: 'Código do documento', type: 'computed', resolverKey: 'doc_code' },
          { id: 'generated-id', name: 'author', label: 'Autor', type: 'computed', resolverKey: 'author' },
        ],
        composition: baseSchemas.composition,
      });
    });
  });

  it('ignores non-catalog tokens silently', async () => {
    renderPage();
    detectedVariables = ['customer_name', 'doc_code'];
    await triggerEditorChange();
    await act(async () => { vi.advanceTimersByTime(400); await Promise.resolve(); });
    await waitFor(() => {
      expect(saveSchemas).toHaveBeenCalledWith({
        placeholders: [
          { id: 'generated-id', name: 'doc_code', label: 'Código do documento', type: 'computed', resolverKey: 'doc_code' },
        ],
        composition: baseSchemas.composition,
      });
    });
  });
});
```

- [ ] **Step 4B.2: Run test — expect fail**

Run: `pnpm --filter @metaldocs/web test -- template-author-page-convergence`
Expected: FAIL.

- [ ] **Step 4B.3: Rewrite `syncPlaceholdersFromDocument` in `TemplateAuthorPage.tsx`**

Replace existing convergence logic. Pseudocode:
```ts
const [catalog, setCatalog] = useState<PlaceholderCatalogEntry[]>([]);
useEffect(() => { void fetchPlaceholderCatalog().then(setCatalog); }, []);
const catalogByKey = useMemo(() => new Map(catalog.map((c) => [c.key, c])), [catalog]);

const syncPlaceholdersFromDocument = useCallback(() => {
  if (!isDraft) return;
  const detected = editorRef.current?.getAgent?.()?.getVariables?.() ?? [];
  const valid = detected.filter((name) => catalogByKey.has(name));
  const placeholders = valid.map((name) => {
    const entry = catalogByKey.get(name)!;
    return {
      id: crypto.randomUUID(),
      name,
      label: entry.label,
      type: 'computed' as const,
      resolverKey: name,
    };
  });
  saveSchemas({
    ...localSchemas,
    placeholders,
  });
}, [isDraft, catalogByKey, saveSchemas, localSchemas]);
```

Remove:
- `+ Add manually` button + `addPlaceholder` callback
- Orphan tracking logic (orphans no longer apply — schema is purely derived)
- Inspector dialog state

- [ ] **Step 4B.4: Run tests**

Run: `pnpm --filter @metaldocs/web test -- template-author-page-convergence`
Expected: PASS.

Run: `pnpm --filter @metaldocs/web tsc --noEmit`
Expected: 0 errors.

- [ ] **Step 4B.5: `/simplify`**

- [ ] **Step 4B.6: Commit (combined with 4A)**

```bash
git add frontend/apps/web/src/features/templates/v2/PlaceholderCatalogPanel.tsx \
        frontend/apps/web/src/features/templates/v2/api/catalog.ts \
        frontend/apps/web/src/features/templates/v2/TemplateAuthorPage.tsx \
        frontend/apps/web/src/features/templates/__tests__/template-author-page-convergence.test.tsx
git rm frontend/apps/web/src/features/templates/placeholder-inspector.tsx \
       frontend/apps/web/src/features/templates/placeholder-chip.tsx
git commit -m "refactor(templates): replace placeholder editor with fixed catalog panel"
```

---


---

## Phase 5: E2E rebuild

### Task 5: Rebuild test artefacts and run E2E

**Files:**
- Modify: `docs/superpowers/e2e-workflow-test-plan-2026-04-25.md` — Stages 7-11 reflect catalog flow
- Use: existing `frontend/apps/web/src/features/templates/v2/TemplateAuthorPage.tsx` via UI

- [ ] **Step 5.1: Generate fresh test DOCX**

Generate `C:/tmp/e2e-catalog.docx` with 7 paragraphs, one token each:
```
{doc_code}
{doc_title}
{revision_number}
{author}
{effective_date}
{approvers}
{controlled_by_area}
```

Use existing Python zipfile script (referenced in summary).

- [ ] **Step 5.2: Recreate template via UI**

Drive browser through `mcp__Claude_Preview__preview_*` tools:
1. Login as `e2e.admin`.
2. Templates V2 → New Template.
3. Upload `C:/tmp/e2e-catalog.docx`.
4. Verify catalog panel marks all 7 tokens as detected (`data-detected=true`).
5. Save (autosave + schema converge).
6. Submit for review.
7. Logout, login as `e2e.approver`, publish.

- [ ] **Step 5.3: Bind template to `e2e-test` profile (via UI)**

Tipos Documentais → Editar `e2e-test` → set default template version → save.

- [ ] **Step 5.4: Create CD via UI**

Docs Controlados → +Novo → profile `e2e-test`, area `general`, owner `e2e-admin`.

- [ ] **Step 5.5: Create document instance via UI**

+Novo documento → pick CD → name "E2E Catalog Test" → Generate.

- [ ] **Step 5.6: Verify resolved values render in editor**

Editor should display:
- `{doc_code}` → CD code (e.g. `E2E-TEST-03`)
- `{doc_title}` → "E2E Catalog Test"
- `{revision_number}` → `1`
- `{author}` → `E2E Admin`
- `{effective_date}` → today
- `{approvers}` → `[aguardando aprovação]`
- `{controlled_by_area}` → `General`

NO `<aside>` fill panel rendered.

- [ ] **Step 5.7: Submit for approval, approve as `e2e.approver`**

- [ ] **Step 5.8: Verify frozen artifact has substituted values**

Download the published DOCX. Confirm zero `{token}` strings remain.

- [ ] **Step 5.9: Update E2E plan doc**

Edit `docs/superpowers/e2e-workflow-test-plan-2026-04-25.md` Stages 7-11 to reflect new flow (no fill form, applyVariables at open, catalog panel in template author).

- [ ] **Step 5.10: Commit E2E plan + new test DOCX path note**

```bash
git add docs/superpowers/e2e-workflow-test-plan-2026-04-25.md
git commit -m "docs(e2e): update workflow test plan for placeholder catalog"
```

---

## Phase 6: Final review + Wiki

- [ ] **Dispatch final Opus reviewer** for entire branch (caveman prompt):

```
review whole branch vs spec docs/superpowers/plans/2026-04-26-placeholder-fixed-catalog.md.
checks:
- 7 catalog entries everywhere consistent (doc_code, doc_title, revision_number, author, effective_date, approvers, controlled_by_area)
- no fill UI surface left (placeholder-form.tsx, fill-in-loader.ts deleted)
- no user-typeable placeholder names (validation rejects, frontend has no input)
- approvers pre-approval label = "[aguardando aprovação]"
- backend at finalize substitutes tokens correctly via fanout
- E2E passed end-to-end
report blockers, max 30 lines.
```

- [ ] **Dispatch single Wiki Keeper subagent** at end of branch:

```
wiki keeper. final pass for placeholder catalog rework.

read: wiki/README.md, wiki/concepts/placeholders.md, wiki/modules/editor-ui-eigenpal.md, wiki/decisions/.

update:
- wiki/concepts/placeholders.md — rewrite to catalog model, table of 7 entries with resolver source, drop legacy fill workflow.
- wiki/decisions/0008-placeholder-fixed-catalog.md — new ADR (motivation: ship simple, defer fill flexibility, eigenpal autosave constraint).
- wiki/modules/editor-ui-eigenpal.md — note that applyVariables is NOT used in writer mode; tokens substitute at finalize.
- wiki/README.md — add ADR link, bump Last verified: 2026-04-26 on touched docs.

constraints:
- keep file:line anchors precise
- 1-line changelog at top of each touched doc
- no new docs unless concept genuinely new

return: list of files touched.
```

- [ ] **Use nexus:finishing-a-development-branch** to close out.

---

## Self-Review Checklist (run before Phase 1)

**Spec coverage:**
- 7 catalog tokens — ✅ Phase 1A (DocTitle), 1B (Approvers label), 1C (catalog endpoint).
- Pre-approval portuguese label — ✅ 1B.
- `doc_title` resolver — ✅ 1A.
- Lock template author to enum — ✅ Phase 4.
- Kill chip panel in document editor — ✅ Phase 3.
- Auto-substitute at document open — ❌ DEFERRED (eigenpal autosave conflict). Backend substitutes at finalize/freeze; catalog panel describes tokens in editor. See Phase 3 decision block.
- Backend rejects non-catalog names — ✅ Phase 2.
- Migration of legacy schemas — ✅ Phase 2.5.
- E2E rebuild — ✅ Phase 5.

**Placeholder scan:** No "TBD" / "implement later" / "similar to" markers — verified.

**Type consistency:**
- `Placeholder.type` = `'computed'` only (frontend) — Phase 4B test asserts.
- Resolver keys catalog endpoint = 7 entries (no `approval_date`); `RegisterBuiltins` keeps `approval_date` registered for legacy/future use but it's not exposed in the catalog. The two lists are intentionally different.
- `ResolveInput.DocumentReader` interface added in 1A.3, used in 1A.4.
- `ApproversResolver.Value` changes from slice to string — verified caller adapt in Phase 1B note.

---

## Execution Handoff

Plan saved to `docs/superpowers/plans/2026-04-26-placeholder-fixed-catalog.md`.

**Recommend Subagent-Driven execution**: Phase 1 has 3 parallel-safe tasks, Phase 4 has 2. Codex implements; Opus reviews phase-end; Wiki Keeper updates docs continuously.

After Phase 0 (Codex plan validation) clears, dispatch the Phase 1 trio in parallel.
