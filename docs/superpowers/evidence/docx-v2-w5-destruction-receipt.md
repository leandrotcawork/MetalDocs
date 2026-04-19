# W5 Destruction Receipt

Cutover completion date (UTC): 2026-04-18T00:00:00Z
Operator: @leandrotcawork

## Commits

- Task 0 — Preflight dump script + evidence template: `35e5ce3`
- Task 1 — Flag flip (METALDOCS_DOCX_V2_ENABLED=true + verification template): `49c0296`
- Task 2 — Post-flip soak evidence + cutover gate test + CI jobs: `00db878`
- Task 3 — Ledger bootstrap (0112) + destructive migration (0113): `b331299`
- Task 4 — Destruction census (kill list): `8d8a199`
- Task 5 — Dep check (no cross-imports from documents_v2 into documents): evidenced inline in Task 4
- Task 6 — Destruction commit of record (git rm + build fix): `ec91aee`
- Task 7 — OpenAPI rename (documents-v2 → documents): `67d9002`
- Task 8 — Feature flag removal (code + .env, no SQL): `fc73759`
- Task 9 — CI consolidation (rename docx-v2-ci.yml → ci.yml, strip gate jobs): `412c6ad`
- Task 10 — Rollback kit (w5-rollback.sh + runbook): `52c3cf8`

## Verification at Task 11

- [x] `go build ./...` — PASS
- [x] `go vet ./...` — PASS
- [x] `go test ./...` — PASS (zero failures)
- [ ] `npm run build --workspace @metaldocs/web` — not run (no node env in worktree)
- [ ] `npm test --workspace @metaldocs/web` — not run (no node env in worktree)
- [ ] OpenAPI lint — not run (no npx in CI at plan time)
- [ ] Staging smoke — deferred (production staging environment required)
- [ ] Production smoke — deferred (production environment required)

## CK5 / MDDM residuals check

Run to verify zero CK5/MDDM code remains outside archive:

```bash
grep -rn 'ck5\|mddm' internal apps frontend --include="*.go" --include="*.ts" --include="*.tsx" | grep -v archive
```

Expected: 0 matches in business logic. Remaining matches (if any) are in:
- OpenAPI YAML comments (not code)
- Migration SQL archives (kept for history per census)
- This document

## Tags

- `w5-preflight` → cut before Task 1 (rollback target)
- `w5-complete` → to be tagged after this receipt is committed
