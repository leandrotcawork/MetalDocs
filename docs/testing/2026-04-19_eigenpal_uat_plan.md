# Eigenpal migration — UAT test plan

Companion to [run log](./2026-04-19_eigenpal_uat_log.md). Plan = matrix of scenarios + status. Log = chronological evidence trail.

Scope: verify MetalDocs end-to-end after CK5 → `@eigenpal/docx-js-editor` swap (sprints P1–P6).
Tester: Claude via `preview_*`. Account: `leandro_theodoro` / admin. Frontend `:4174` · API `:8081`.

---

## Legend

| Mark | Meaning |
|------|---------|
| ✅ | pass — evidence in run log |
| ⚠️ | defect, non-blocker — walk continues |
| 🛑 | blocker — feature broken end-to-end |
| 🩹 | fixed mid-run |
| ⏳ | under test now |
| ⏸️ | queued, not started |
| ➖ | out of scope this UAT |

---

## Matrix

### A. Bootstrap + shell
| # | Scenario | Status | Notes |
|---|----------|:---:|------|
| A1 | Vite dev server boots, no unresolved imports | 🩹 | eigenpal subpath alias fix. Log §0. |
| A2 | API boots, `health/live` → 200 | ✅ | |
| A3 | Session cookie persists; land on Dashboard | ✅ | Log §1. |
| A4 | Sidebar renders all groups | ✅ | |
| A5 | Pre-existing v1 404s (profiles/areas/depts/subjects/stream) | ⚠️ | Out of scope — document only. |

### B. Templates V2
| # | Scenario | Status | Notes |
|---|----------|:---:|------|
| B1 | List page renders | ✅ | |
| B2 | Open template → editor panel (DOCX URL + schema) | ✅ | |
| B3 | Publish new version | ⏸️ | Not yet exercised. |
| B4 | New template creation flow | ⏸️ | |
| B5 | Template audit log view | ⏸️ | |

### C. Documents V2 — create + open
| # | Scenario | Status | Notes |
|---|----------|:---:|------|
| C1 | `POST /api/v2/documents` → 201 | ✅ | |
| C2 | Initial revision + session returned | ✅ | |
| C3 | Sidebar "Novo documento" routes to v2 | 🛑 | Routes to legacy `/create`. |
| C4 | `DocumentCreatePage` reachable from UI | 🛑 | No entry point. |
| C5 | Editor mounts at `#/documents-v2/{id}` | ✅ | |
| C6 | Merge-field sidebar populated | ✅ | |

### D. Autosave + heartbeat
| # | Scenario | Status | Notes |
|---|----------|:---:|------|
| D1 | Debounced autosave on typing | ✅ | 1500ms. |
| D2 | Presign → MinIO PUT → commit round-trip | ✅ | |
| D3 | Heartbeat 204 every N seconds | ✅ | |
| D4 | Content survives reload | ✅ | Revision DOCX re-downloadable; file format valid. |
| D5 | Concurrent session detection (take-writer) | ⏸️ | Needs second real user; deferred. |
| D6 | Autosave rejected on finalized doc | ⚠️ | Returns 409 `stale_base` instead of semantic `finalized`/`forbidden`. Defect. |
| D7 | `net::ERR_ABORTED` on MinIO PUT despite 200 | ⚠️ | Cosmetic. |

### E. Comments (P5.1)
| # | Scenario | Status | Notes |
|---|----------|:---:|------|
| E1 | `GET /comments` → 200 empty | 🩹 | Migration 0118 applied. |
| E2 | `POST /comments` → 201 | ✅ | API verified 01:02Z. |
| E3 | List after create returns new comment | ✅ | |
| E4 | Reply (parent_library_id) persists | ✅ | |
| E5 | Resolve toggles `resolved_at` + `resolved_by` | 🩹 | UpdateComment SQL had unused `$4` (pgx SQLSTATE 42P18). Renumbered params + removed duplicate userID arg. Now PATCH `{"done":true}` → 200 w/ `resolved_at`. |
| E6 | Unresolve clears both | ✅ | |
| E7 | Delete cascades replies | ✅ | |
| E8 | `useDocumentComments` hook polls + renders | ⏸️ | UI flow deferred. |

### F. Exports (P4)
| # | Scenario | Status | Notes |
|---|----------|:---:|------|
| F1 | `GET /export/docx-url` → 200 w/ signed URL | 🛑 | Code wired; ops env `METALDOCS_DOCGEN_V2_URL` + token pending. |
| F2 | `POST /export/pdf` → 200 w/ signed URL | 🛑 | Same. Needs docgen-v2 reachable + token matching. |
| F3 | `ExportMenu` enables when session writer/readonly | ⏸️ | Gate on `canExport`. |
| F4 | Download DOCX from signed URL | ⏸️ | |
| F5 | Open PDF from signed URL | ⏸️ | |
| F6 | `ExportMenuButton` PT label + theme | ⚠️ | Cosmetic. |

### G. Finalize + checkpoints
| # | Scenario | Status | Notes |
|---|----------|:---:|------|
| G1 | Finalize → locks document, flips session to readonly | ✅ | 204, Status=finalized, FinalizedAt set. |
| G2 | Checkpoint create | ✅ | 201 w/ version_num=1. |
| G3 | Restore from checkpoint | ✅ | 200 w/ new_revision_num=5, idempotent=true. |
| G4 | Signed revision URL (`/revisions/{rid}/url`) | ✅ | 200 w/ MinIO presigned GET. DOCX 1677 bytes, valid. |

### H. Documents list (v1 shells still used)
| # | Scenario | Status | Notes |
|---|----------|:---:|------|
| H1 | `/documents` library | ✅ | 25 docs. |
| H2 | `/documents/mine` | ✅ | |
| H3 | `/documents/recent` | ✅ | |
| H4 | "Tipos de documento" panel | ⚠️ | Empty — v1 profile endpoint 404s. |

### I. Legacy retirement stubs
| # | Scenario | Status | Notes |
|---|----------|:---:|------|
| I1 | `/content-builder` shows retirement stub | ✅ | |
| I2 | Legacy template editor shows retirement stub | ✅ | |

### J. Cross-cutting
| # | Scenario | Status | Notes |
|---|----------|:---:|------|
| J1 | HashRouter footgun documented | ⚠️ | |
| J2 | Non-admin role → 403 on v2 routes | ✅ | `X-User-Roles: viewer` → 403 forbidden. |
| J3 | Multi-tenant isolation on v2 docs | ⚠️ | Cross-tenant GET returns **500** instead of 404. Silent SQL error (no log line). Defect. |
| J4 | Audit events emitted on doc create/rename/finalize | 🩹 | Adapter in `main.go` never set `Event.ID` / `OccurredAt` / `TraceID` → PK collision / silent drop → only 1 row ever landed. Fixed by generating UUID + UTC timestamp in adapter. `document.renamed` now lands (table count +1). |
| J5 | Notifications on comment/finalize | ➖ | Not wired yet. |

---

## Running tallies

- Total scenarios: 52
- ✅ pass: 25
- 🩹 fixed mid-run: 5
- ⚠️ defect: 8 (adds D6 finalized autosave semantic + J3 500 on wrong tenant)
- 🛑 blocker: 3 (C3, C4, F1/F2 pair)
- ⏸️ queued: 10
- ➖ out of scope: 1

### Newly surfaced defects
- **D6 semantic**: autosave on finalized doc returns `409 stale_base`; should return explicit `finalized`/`forbidden`.
- **J3 silent 500**: cross-tenant GET `/documents/{id}` returns 500 with no backend log. Handler likely scans a `sql.ErrNoRows` path without mapping → `mapErr` fallthrough. Should be 404.
- **Rename on finalized doc accepted** (observed during J4): PATCH succeeded against a doc in `finalized` state — should be 409/403. File separately.

---

## Procedure

1. Pick next ⏸️ in rank order (E → F → G → D reload → J auth → B publish).
2. Drive via `preview_*` on `:4174`. Log evidence inline in run log under matching section.
3. Update this plan's `Status` column. Never rewrite history — new attempts append, prior marks stay with timestamp if retested.
4. On 🛑: root-cause, land fix, re-run same scenario, flip to 🩹.
5. End of UAT: tally + ship summary to stakeholder.

## Ranking for next session

1. **E2–E8** — comments CRUD full sweep (backend reachable, migration applied).
2. **F1–F5** — exports, after ops sets docgen-v2 env (out-of-band).
3. **G1–G4** — finalize + checkpoints (needed before shipping any doc).
4. **D4–D6** — autosave edge cases (reload, take-writer race, MinIO 5xx).
5. **J2–J4** — auth/tenant/audit cross-cutting.
6. **B3–B5** — templates V2 remaining flows.
