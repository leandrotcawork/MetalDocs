---
name: metaldocs-adr
description: Full ADR lifecycle for MetalDocs. An ADR is DONE only when documented with a runnable acceptance test, implementation verified, and git commit made. Required for: architecture changes, boundary changes, security/auth changes, schema migrations, deploy/rollback strategy changes, new relevant dependencies.
---

# MetalDocs ADR

## When an ADR is required (from AGENTS.md)
- Architecture, boundary, or public contract change
- Security, auth, or authorization change
- Schema change, destructive migration, or retention change
- Deploy, rollback, or observability strategy change
- New relevant dependency

## 4 stages — all required

### Stage 1 — Write
File: `docs/adr/<sequence>-<kebab-title>.md`
Use existing ADRs as format reference (see `docs/adr/0001-modular-monolith.md`)

Required sections:
- **Status:** Proposed
- **Context:** why this decision was needed
- **Decision:** precise — not vague
- **Consequences:** trade-offs and impacts
- **Acceptance test:** concrete and runnable

Good acceptance tests:
- ✅ `go test ./...` passes with no boundary violations
- ✅ `scripts/e2e-smoke.ps1` passes after migration
- ✅ `scripts/check-governance.ps1` passes
- ❌ "implementation reviewed"
- ❌ "looks correct"

### Stage 2 — Update docs
Update `docs/plans/MASTER_IMPLEMENTATION_PLAN.md` if this changes the implementation plan.
Update relevant runbook in `docs/runbooks/` if operational behavior changes.

### Stage 3 — Verify
Run the acceptance test from Stage 1.
If fails → fix → re-run. Do not proceed until green.

### Stage 4 — Commit
```bash
git commit -m "docs(adr): <sequence>-<title> — verified and closed"
```
Update ADR Status: Proposed → Accepted.

## Workflow
1. Read relevant existing ADRs in `docs/adr/`
2. Write ADR with acceptance test (Stage 1)
3. Update docs (Stage 2)
4. Verify (Stage 3)
5. Commit (Stage 4)

## References
- `references/adr-checklist.md`
- Existing ADRs: `docs/adr/`
