# Approval E2E Coverage Map

Every invariant from Phases 1-10 + every Phase 9 hardening fix maps to ≥1 E2E spec ID.
CI gate: `.github/workflows/e2e-coverage-gate.yml` fails build if any invariant is unmapped.

## Legend

| Column | Meaning |
|--------|---------|
| Invariant ID | Phase + sequential number |
| Source | Plan section that declared the invariant |
| Spec ID | `flows/<file>.spec.ts :: <test title>` |
| Status | ✅ covered / ⚠ partial / ❌ missing |

---

## Phase 1 — Database Schema + RLS Invariants

| Invariant ID | Description | Spec ID | Status |
|---|---|---|---|
| P1-I01 | `documents_v2` never stores doc body — only metadata + state | `happy_path :: publishes document` | ✅ |
| P1-I02 | `approval_instances` always references existing `approval_routes` | `happy_path :: submits with valid route` | ✅ |
| P1-I03 | `signoffs` FK → `approval_instances` + cascade on delete | `reject_flow :: clears signoffs on reject` | ✅ |
| P1-I04 | `governance_events` immutable after insert (RLS no UPDATE) | `happy_path :: governance event chain` | ✅ |
| P1-I05 | Tenant isolation: cross-tenant read returns 0 rows | `sod_violation :: cross-tenant isolation` | ✅ |

## Phase 2 — IAM + Capability Invariants

| Invariant ID | Description | Spec ID | Status |
|---|---|---|---|
| P2-I01 | `authz.Require` blocks unauthorized calls | `sod_violation :: reviewer cannot submit` | ✅ |
| P2-I02 | Capability hash frozen post-Phase 2 | `route_admin :: deactivate requires admin cap` | ✅ |
| P2-I03 | Area RBAC: user without area role cannot see docs in that area | `edit_lock :: second user restricted area` | ⚠ |

## Phase 3 — State Machine Invariants

| Invariant ID | Description | Spec ID | Status |
|---|---|---|---|
| P3-I01 | Illegal state transitions rejected at DB trigger layer | `trigger_bypass :: illegal transition blocked` | ✅ |
| P3-I02 | `draft → under_review` only via submit, not direct UPDATE | `happy_path :: submit triggers state change` | ✅ |
| P3-I03 | `published` is terminal — no further transitions | `happy_path :: badge stays published` | ✅ |
| P3-I04 | `rejected → draft` auto-transition on rejection | `reject_flow :: returns to draft` | ✅ |
| P3-I05 | `scheduled` → `published` via scheduler tick only | `scheduled_publish :: clock-advance publishes` | ✅ |

## Phase 4 — Governance Events + Audit Trail

| Invariant ID | Description | Spec ID | Status |
|---|---|---|---|
| P4-I01 | Every state transition emits matching governance_event | `happy_path :: governance event chain` | ✅ |
| P4-I02 | Causation ID chain: each event references triggering event | `happy_path :: causation chain` | ✅ |
| P4-I03 | Outbox in same tx as mutation — no split-brain | `happy_path :: rollback drops event` | ✅ |
| P4-I04 | `governance_events.created_at` monotonically increasing per instance | `happy_path :: events monotonic order` | ✅ |

## Phase 5 — Approval Service + Signoff Invariants

| Invariant ID | Description | Spec ID | Status |
|---|---|---|---|
| P5-I01 | SoD: submitter cannot sign any stage of own submission | `sod_violation :: author sign blocked` | ✅ |
| P5-I02 | `m_of_n`: stage passes when m approvals reached, not before | `quorum_m_of_n :: requires 2 of 3` | ✅ |
| P5-I03 | `m_of_n`: first rejection fails stage immediately | `quorum_m_of_n :: single reject fails stage` | ✅ |
| P5-I04 | Only assigned stage members can sign | `sod_violation :: non-member sign blocked` | ✅ |
| P5-I05 | Duplicate signoff (same user, same stage) rejected | `happy_path :: double-sign rejected` | ⚠ |

## Phase 6 — Idempotency Invariants

| Invariant ID | Description | Spec ID | Status |
|---|---|---|---|
| P6-I01 | Same key → same result, single governance row | `happy_path :: idempotent replay` | ✅ |
| P6-I02 | Same key + different body → 409 `idempotency.key_conflict` | `happy_path :: key conflict 409` | ✅ |
| P6-I03 | Expired key (TTL=24h) → new operation, not replay | `happy_path :: ttl expiry new op` | ⚠ |
| P6-I04 | Concurrent same-key requests → one wins, other replays | `happy_path :: concurrent idempotency` | ✅ |

## Phase 7 — HTTP API Invariants

| Invariant ID | Description | Spec ID | Status |
|---|---|---|---|
| P7-I01 | OCC: stale `If-Match` → 412 | `happy_path :: stale etag 412` | ✅ |
| P7-I02 | `Idempotency-Key` header required on mutating endpoints | `happy_path :: idempotency key present` | ✅ |
| P7-I03 | `ETag` on GET; updated on mutation | `happy_path :: etag updated after submit` | ✅ |
| P7-I04 | 423 when document locked | `edit_lock :: locked returns 423` | ✅ |
| P7-I05 | 403 SoD violation surfaced as typed error | `sod_violation :: 403 with code` | ✅ |
| P7-I06 | Rate-limit 429 → toast shown, retry after respected | `happy_path :: 429 toast` | ⚠ |

## Phase 8 — Scheduler + Watchdog Invariants

| Invariant ID | Description | Spec ID | Status |
|---|---|---|---|
| P8-I01 | Fencing epoch prevents stale leader from publishing | `scheduled_publish :: old leader blocked` | ✅ |
| P8-I02 | `release_lease` expires in-place, epoch monotonic | `scheduled_publish :: epoch monotonic` | ✅ |
| P8-I03 | Backpressure SkipOnPressure skips non-critical jobs | `scheduled_publish :: backpressure skip` | ⚠ |
| P8-I04 | Stuck-instance watchdog auto-cancels after 7d | `happy_path :: watchdog cancel` | ⚠ |

## Phase 9 — Frontend Hardening Fixes (F1-F11)

| Fix ID | Description | Spec ID | Status |
|---|---|---|---|
| F1 | `mutationClient` auto-injects `Idempotency-Key` (UUIDv7) | `happy_path :: idempotency key uuidv7` | ✅ |
| F2 | `mutationClient` auto-injects `If-Match` from etagCache | `happy_path :: stale etag 412` | ✅ |
| F3 | `etagCache` updated from response `ETag` header | `happy_path :: etag updated after submit` | ✅ |
| F4 | 412 → toast "document changed, refresh" | `happy_path :: 412 toast shown` | ✅ |
| F5 | 401 → redirect to `/login` | `happy_path :: 401 redirect` | ⚠ |
| F6 | 403 typed error → inline dialog message | `sod_violation :: 403 inline dialog` | ✅ |
| F7 | 429 → toast with retry-after seconds | `happy_path :: 429 toast` | ⚠ |
| F8 | `SignoffDialog` 8-state machine (idle→loading→success→error…) | `happy_path :: signoff dialog states` | ✅ |
| F9 | `LockBadge` shows lock holder + relative time | `edit_lock :: lock badge shows holder` | ✅ |
| F10 | `StateBadge` single source for all 9 states | `happy_path :: state badge transitions` | ✅ |
| F11 | `SupersedePublishDialog` schedule datetime ≥ now+5min validation | `scheduled_publish :: past datetime error` | ✅ |

## Phase 10 — Integration Test Invariants

| Invariant ID | Description | Spec ID | Status |
|---|---|---|---|
| P10-I01 | OCC race: concurrent updates → one 409, one succeeds | `happy_path :: concurrent occ` | ✅ |
| P10-I02 | SKIP LOCKED: second worker doesn't process same row | `scheduled_publish :: no double process` | ✅ |
| P10-I03 | Cascade: approve chain publishes all versions | `happy_path :: cascade publish` | ✅ |
| P10-I04 | AST: no BeginTx outside allowed packages | `route_admin :: tx ownership` | ✅ |
| P10-I05 | Outbox in same tx: rollback drops event | `happy_path :: rollback drops event` | ✅ |

---

## ⚠ Partial Coverage — Follow-up Required

Items marked ⚠ have known gaps. Backlog in `docs/superpowers/plans/followups/spec2-gaps.md`:
- P2-I03: area RBAC negative case needs dedicated flow
- P5-I05: duplicate signoff needs explicit UI path  
- P6-I03: TTL expiry tested in integration but not E2E (requires clock advance + 24h)
- P7-I06 / F7: 429 path exercised only if rate-limit config injectable
- P8-I03 / P8-I04: scheduler E2E requires full stack with real scheduler (staging only)
- F5: 401 redirect requires session expiry injection
