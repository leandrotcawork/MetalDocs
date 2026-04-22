# Phase 12 Opus Review — Whole-Plan Cross-Check

**Reviewer:** Opus cross-check (Task 12.12)  
**Date:** 2026-04-22  
**Scope:** Every invariant in Phases 1-11 → CI gate exists

---

## Verdict: APPROVED WITH CONDITIONS

All 12 phases are implemented and committed. Every invariant has ≥1 CI gate. Two conditions must be met before staging release tag.

---

## Invariant → CI Gate Coverage Table

| Phase | Invariant | CI Gate | Status |
|-------|-----------|---------|--------|
| P1 | Schema structure (approval tables, FKs) | `integration tests` (schema_lockdown_test.go) | ✅ |
| P1 | RLS isolation | `integration tests` (trigger_bypass_test.go) | ✅ |
| P2 | authz.Require on all exported methods | `cilint authzrequire` (invariants.yml) | ✅ |
| P2 | Capability catalog frozen | `capability-catalog-hash` (invariants.yml) | ✅ |
| P3 | Illegal state transitions blocked | `integration tests` (trigger_bypass_test.go) | ✅ |
| P3 | State machine transitions correct | `integration tests` + E2E flows | ✅ |
| P4 | Governance events emitted per transition | `cilint outboxpair` + `outbox_same_tx_test.go` | ✅ |
| P4 | Causation chain integrity | `e2e happy_path governance event chain` | ✅ |
| P5 | SoD: submitter cannot sign | `integration tests` + `e2e sod_violation` | ✅ |
| P5 | m_of_n quorum logic | `integration tests` + `e2e quorum_m_of_n` | ✅ |
| P6 | Idempotency replay | `integration tests` (idempotency_test.go) + `e2e happy_path` | ✅ |
| P6 | Key conflict 409 | `integration tests` + `e2e happy_path` | ✅ |
| P7 | OCC / If-Match 412 | `integration tests` (concurrency_test.go) + `e2e happy_path` | ✅ |
| P7 | ETag lifecycle | `e2e happy_path` | ✅ |
| P8 | Fencing epoch monotonic | `integration tests` (TestProbe11) + `e2e scheduled_publish` | ✅ |
| P8 | Backpressure hysteresis | `unit tests` (TestScheduler_BackpressureSkip) | ✅ |
| P8 | Stuck watchdog auto-cancel | `unit tests` (stuck_instance_watchdog/job.go) | ⚠ E2E gap (noted in COVERAGE.md) |
| P9 | Frontend: Idempotency-Key injected | `e2e happy_path` network intercept | ✅ |
| P9 | Frontend: If-Match injected | `e2e happy_path` network intercept | ✅ |
| P9 | Frontend: 412/403/429 surfaced | `e2e happy_path` + `sod_violation` | ✅ |
| P10 | OCC race N=50 | `integration tests` (concurrency_test.go, INTEGRATION_STRESS_N) | ✅ |
| P10 | AST scan BeginTx | `cilint txownership` (invariants.yml) + `tx_ownership_test.go` | ✅ |
| P10 | Legacy vocab absent | `cilint legacyvocab` (invariants.yml) + `legacy_absent_test.go` | ✅ |
| P11 | E2E happy path | `e2e-coverage-gate.yml` | ✅ |
| P11 | All invariants mapped in COVERAGE.md | `e2e-coverage-gate.yml` (coverage-map-check job) | ✅ |
| P11 | Axe a11y zero critical | `e2e-coverage-gate.yml` (axe-baseline-check job) | ✅ |
| P12 | Migration gapless | `migration-gapless` (invariants.yml) | ✅ |
| P12 | OpenAPI drift | `openapi-drift` (invariants.yml) | ✅ |
| P12 | Perf thresholds | `perf.yml` (REDUCED PR gate + full on main) | ✅ |
| P12 | Prod smoke ≤5min | `smoke.yml` (cron */5) | ✅ |

---

## Conditions Before Staging Release

**CONDITION-1 (Important):** `ops/canary/controller.go` `fetchRecentSamples` is stubbed. Before first canary ramp in production, wire Prometheus query (or Grafana API) to return real 1-min metric buckets. Without this, breach detection is non-functional.

**CONDITION-2 (Important):** `httpGet` in controller.go is a stub. Replace with real `net/http.Get` before production canary controller runs. The function currently always returns 200.

---

## Non-Blocking Follow-ups

These items are tracked in `docs/superpowers/plans/followups/spec2-gaps.md`:

1. E2E gap: P8-I04 stuck-instance watchdog E2E test (requires clock advance > 7d — staging only)
2. E2E gap: P2-I03 area RBAC negative case in E2E (needs dedicated flow)
3. E2E gap: F5 (401 redirect) needs session expiry injection in E2E
4. Canary: metrics integration (Prometheus client) — operator task, blocks production canary
5. cosign: image signing + verification gate in controller — operator task

---

## Summary

Spec 2 — Foundation Doc Approval State Machine — is **complete across all 12 phases**. 147 tasks executed. Every major invariant has ≥1 automated CI gate. Two stub implementations in canary controller must be wired before production use. All E2E gaps are documented in COVERAGE.md and spec2-gaps.md.

**Approved for staging deployment.**
