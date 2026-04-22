# Spec 2 Plan — COMPLETE

**Plan:** `2026-04-21-foundation-doc-approval-state-machine.md` (~4620 lines)
**Sidecar:** `.tasks.json` (~137 tasks across 12 phases)
**Date finalized:** 2026-04-22
**Status:** ready for execution

## Phase totals

| Phase | Tasks | Codex mode | Opus review | Verdict JSON |
|------|------|------------|-------------|--------------|
| 1 | 13 | OPERATIONS (2 rounds) | — | `reviews/phase-1-round-{1,2}.json` |
| 2 | 9 | COVERAGE | ✅ | `reviews/phase-2-*.json` |
| 3 | 9 | QUALITY | — | `reviews/phase-3-round-1.json` |
| 4 | 12 | ARCHITECTURE | ✅ | `reviews/phase-4-*.json` |
| 5 | 13 | COVERAGE | — | `reviews/phase-5-round-1.json` |
| 6 | 12 | ARCHITECTURE | ✅ | `reviews/phase-6-*.json` |
| 7 | 11 | QUALITY | — | `reviews/phase-7-round-1.json` |
| 8 | 10 | OPERATIONS | ✅ | `reviews/phase-8-*.json` |
| 9 | 11 | COVERAGE | — | `reviews/phase-9-round-1.json` |
| 10 | 13 | QUALITY | ✅ | `reviews/phase-10-*.json` |
| 11 | 11 | COVERAGE | — | `reviews/phase-11-round-1.json` |
| 12 | 12 | OPERATIONS | ✅ | `reviews/phase-12-*.json` |
| — | final | SEQUENCING | — | `reviews/final-sequencing-round-1.json` |

## Core invariants locked

- 8-state lifecycle + `cancelled` instance state.
- OCC via `revision_version` + `ETag "v<n>"` + `If-Match` (428/412).
- Idempotency-Key UUIDv7 + payload-hash 24h TTL + replay contract.
- Capability tripwire `{capability, area_id}` GUC; DB trigger catches missing `authz.Require`.
- Lease fencing epoch `bigint` — no double-publish under GC pause.
- Tx ownership allowlist: `approval/application`, `jobs/*`, `iam/area_membership` — cilint AST-enforced.
- SECURITY DEFINER hardening: owner `metaldocs_admin`, `SET search_path`, `REVOKE FROM PUBLIC`.
- Mutation client interceptor centralizes UUIDv7 + If-Match + 412/401/403/offline.
- Outbox same-tx: state row + governance_event row committed together (contract-tested).
- UTC timestamptz throughout; browser tz for display.
- Legacy vocab (`finalized`/`archived`) stripped; cilint `legacyvocab` catches regressions.

## Phase Contracts (sequencing-locked)

1. **P2a vs P2b:** additive+seed in phase; enforcement migration 0142b gated by P12 canary.
2. **P5 idempotency-agnostic:** services compile/test without 0146; decorator wired in P7.
3. **P5 lease-epoch opaque:** `LeaseAsserter` interface; storage-backed impl in P8 only.

## Execution path

1. `superpowers:subagent-driven-development` (recommended) or `superpowers:executing-plans`.
2. Codex `gpt-5.3-codex` medium/high implements; Haiku/Sonnet for trivial edits.
3. Opus coordinates + reviews phase-end on phases 2, 4, 6, 8, 10, 12.
4. Every phase commit gated by Codex verdict JSON + passing tests.
5. Deploy per `ops/DEPLOY.md` (built in Task 12.6): additive → code → smoke → canary → enforcement → enable jobs → final smoke.

## Backlog

10 non-blocking follow-ups in `followups/spec2-gaps.md` (G1-G10) — post-P1-release hardening.
