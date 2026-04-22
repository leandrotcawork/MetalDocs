# Spec 2 — Foundation Doc Approval State Machine: COMPLETE

**Plan:** `docs/superpowers/plans/2026-04-21-foundation-doc-approval-state-machine.md`  
**Completed:** 2026-04-22  
**Branch:** `feature/foundation-spec2`

---

## Executive Summary

12-phase implementation of the MetalDocs document approval state machine is complete.
All phases committed to `feature/foundation-spec2`. Approved for staging deployment.

---

## Phase Summary

| Phase | Title | Status | Key Deliverables |
|-------|-------|--------|-----------------|
| 1 | Database Schema + RLS | ✅ DONE | migrations 0141-0148, RLS policies, SECURITY DEFINER fns |
| 2 | IAM + Capabilities | ✅ DONE | authz.Require, area RBAC, capability catalog |
| 3 | State Machine + Triggers | ✅ DONE | PG trigger enforcement, illegal transition rejection |
| 4 | Governance Events + Audit | ✅ DONE | outbox pattern, causation chain, governance_events table |
| 5 | Approval Service + Signoff | ✅ DONE | SoD, m_of_n, signoff lifecycle, cancel service |
| 6 | Idempotency | ✅ DONE | idempotency_keys table, 24h TTL, concurrent safety |
| 7 | HTTP API | ✅ DONE | 13 routes, ETag/If-Match OCC, typed errors |
| 8 | Scheduler + Watchdog | ✅ DONE | lease-based scheduler, fencing epochs, stuck-instance watchdog |
| 9 | Frontend | ✅ DONE | React components (StateBadge, SignoffDialog, InboxPage…), mutationClient, etagCache |
| 10 | Integration Tests | ✅ DONE | 10 test files, OCC/fencing/outbox/AST/legacy checks, CI tiers |
| 11 | E2E Playwright | ✅ DONE | 7 user flows, axe a11y, coverage map, 100% invariant mapping |
| 12 | CI Invariants + Ops | ✅ DONE | cilint, perf benchmarks, smoke probes, canary controller, deploy runbook |

---

## CI Gate Summary

| Gate | File | Triggers |
|------|------|---------|
| cilint (4 analyzers) | `invariants.yml` | PR + push main |
| Migration gapless | `invariants.yml` | PR + push main |
| OpenAPI drift | `invariants.yml` | PR + push main |
| Capability catalog hash | `invariants.yml` | PR + push main |
| go vet + staticcheck | `invariants.yml` | PR + push main |
| E2E approval flows | `e2e-coverage-gate.yml` | PR touching approval/api paths |
| Axe a11y baseline | `e2e-coverage-gate.yml` | PR touching approval/api paths |
| Perf (reduced) | `perf.yml` | PR touching approval/jobs/handlers |
| Perf (full) | `perf.yml` | push main + manual |
| Staging smoke | `smoke.yml` | cron every 10 min |
| Prod smoke | `smoke.yml` | cron every 5 min |
| Integration (smoke) | `test-smoke.yml` | PR (2 min budget) |
| Integration (full gate) | `test-full.yml` | push main (10 min) |
| Integration (nightly stress) | `test-nightly.yml` | 02:00 UTC (60 min, N=200) |

---

## Key Architectural Decisions

1. **Lease fencing:** `release_lease` expires row in-place (not DELETE) to preserve epoch monotonicity
2. **GUC bypass:** `BypassAuthz bool` on `CancelInput` — GUC set inside cancel service's own tx
3. **Backpressure:** `probePressure` returns `s.inPressure` inside lock (not pre-update snapshot)
4. **Idempotency:** UUID key + body hash stored; same key + different body → 409 conflict
5. **OCC:** `If-Match` header required on all mutations; 412 on stale ETag
6. **Frontend:** `mutationClient.ts` auto-injects Idempotency-Key + If-Match from etagCache

---

## Conditions Before Production

1. **G14:** Wire Prometheus to canary `fetchRecentSamples` (breach detection non-functional without)
2. **G15:** Replace `httpGet` stub in canary controller with `net/http.Get`

Non-blocking follow-ups tracked in `docs/superpowers/plans/followups/spec2-gaps.md`.
