# Phase 10 Review — Opus

**Verdict**: PASS_WITH_NOTES

---

## Invariant Coverage Matrix

| # | Invariant | Test file | Status |
|---|-----------|-----------|--------|
| 1 | Trigger bypass blocked | `scenarios/trigger_bypass_test.go` — `TestTriggerBypassBlocked`, `TestIllegalTransitionBlocked` | PASS |
| 2 | Membership fn gate | `scenarios/membership_fn_test.go` — `TestDirectInsertUserProcessAreasBlocked`, `TestGrantAreaMembershipFn`, `TestGrantAreaMembershipIdempotent` | PASS |
| 3 | Schema lockdown | `scenarios/schema_lockdown_test.go` — `TestWriterCannotDropTable`, `TestWriterCannotAlterTable`, `TestWriterCannotCreateTable`, `TestWriterCanReadApprovalTables` | PASS |
| 4 | Concurrency races | `scenarios/concurrency_test.go` — `TestConcurrencyScenarios` (OCC stale revision, SKIP LOCKED, lease fencing, signoff unique, N=50 stress) | PASS WITH NOTES |
| 5 | Obsolete cascade | `scenarios/obsolete_cascade_test.go` — `TestObsoleteCascade_ParentAndChildren`, `TestObsoleteCascade_NoStaleOCC`, `TestLegalTransition_ObsoleteFromPublished` | PASS |
| 6 | Tx ownership | `scenarios/tx_ownership_test.go` — `TestReflect_RepositoryNoBeginTx`, `TestHTTPHandlers_NoBeginTx` | PASS |
| 7 | Idempotency | `scenarios/idempotency_test.go` — `TestIdempotency_SameKeyReplay`, `TestIdempotency_SameKeyDifferentPayload`, `TestIdempotency_Expired_NewEntry`, `TestIdempotency_Concurrent_OnlyOneWins` | PASS WITH NOTES |
| 8 | Legacy vocabulary absent | `scenarios/legacy_absent_test.go` — `TestNoLegacyStatusInGoSource`, `TestNoLegacyStatusInTSSource`, `TestGoVetPasses`, `TestStaticcheckInstalled` | PASS |
| 9 | Outbox same-tx | `scenarios/outbox_same_tx_test.go` — `TestOutbox_ApprovalInstanceInsertHasGovernanceEvent`, `TestOutbox_RollbackOmitsEvent`, `TestOutbox_DedupeKey` | PASS WITH NOTES |
| 10 | E2E happy path | `scenarios/e2e_happy_test.go` — `TestE2E_HappyPath_HTTP` | PASS |

All 10 invariants have at least one test. No invariant is uncovered.

---

## Assessment by Dimension

### Anti-flake controls

The seed logging requirement is satisfied: `concurrency_test.go:24–28` declares `const concurrencyTestSeed = 0xDEADBEEF` and logs it at test start via `t.Logf("testSeed=0x%X", concurrencyTestSeed)`. This fulfils the logged-seed requirement.

The N=50 stress loop at `concurrency_test.go:46–54` runs exactly 50 sub-tests (`OCC_Race_N50/iter_01` through `iter_50`), each asserting exactly one winner and one loser (two workers). The constant `occRaceWorkers = 2` is hardcoded. The `INTEGRATION_STRESS_N` environment variable set in `.github/workflows/test-nightly.yml:34` (`INTEGRATION_STRESS_N: "500"`) is never read by the test code. This means the nightly env var has no effect on concurrency pressure; the stress level is always N=50 iterations with 2 workers, not 500.

Barrier usage is appropriate for most tests: a `chan struct{}` (`start`) is closed to release goroutines simultaneously, avoiding sleep-based synchronisation.

The `-race` flag is present in both `test-full.yml:33` and `test-nightly.yml:35`, but absent from `test-smoke.yml`. This is an acceptable tier decision, though it means data-race regressions are not caught at PR time.

### Test isolation

The `testdb.Open` function creates a per-test random schema (`metaldocs_test_<5-byte-hex>`), applies all migrations to it via `ApplyMigrations`, and drops it in `t.Cleanup`. Tests that use `testdb.Open` are properly isolated.

However, several test files bypass `testdb.Open` entirely and write directly to the hardcoded `metaldocs` schema using `openDirectDB`:

- All four tests in `schema_lockdown_test.go`
- All tests in `concurrency_test.go`
- All tests in `obsolete_cascade_test.go`
- All tests in `idempotency_test.go`
- Both tests in `outbox_same_tx_test.go` (except `TestOutbox_DedupeKey` which uses a skip guard)

These tests share the same physical `metaldocs` schema and rely on `t.Cleanup` DELETEs to restore state. If a test panics mid-run or a DELETE is incomplete, data leaks into subsequent tests. This is a deliberate tradeoff (production schema validates real constraints), but it means tests are not fully isolated from each other.

`idempotency_test.go` uses hardcoded UUID strings (`11111111-1111-1111-1111-111111111181` through `...184`) rather than `testdb.DeterministicID`. These are stable across runs, which is acceptable for cleanup-guarded tests, but they could collide if two concurrent test runs share the same database.

The `DATABASE_URL` skip guards are present in `testdb.DSN` (`testdb/db.go:35–43`) and `openDirectDB` (`concurrency_test.go:457–476`). Any test lacking a database URL skips cleanly rather than failing, satisfying the isolation requirement for CI environments without a live database.

### CI tier wiring

Three distinct workflow files are present:

- `test-smoke.yml` — triggers on `pull_request` to `main`/`develop`; 2-minute timeout; runs a named subset via `-run "TestTriggerBypass|TestMembership|TestSchemaLockdown|TestLegacy|TestE2E"`.
- `test-full.yml` — triggers on `push` to `main`; 10-minute timeout; runs `./tests/integration/...` with `-count=1 -race`.
- `test-nightly.yml` — runs on `cron: '0 2 * * *'`; 60-minute timeout; runs `./tests/integration/...` with `-count=1 -race -timeout 3600s`; opens a GitHub issue on failure.

This matches the three-tier architecture required. The `timeout-minutes` values (2, 10, 60) align exactly with the specification.

One gap: the smoke filter uses `TestLegacy` which matches `TestNoLegacyStatusInGoSource` and `TestNoLegacyStatusInTSSource` but those tests invoke `go vet` and walk source trees, which is heavier than the filter implies. `TestGoVetPasses` and `TestStaticcheckInstalled` in `legacy_absent_test.go` are also matched by this filter and add meaningful latency to the smoke tier.

---

## Issues Found

### Critical

None.

### Important

**I1 — SKIP LOCKED barrier is a deadlock in disguise**

File: `tests/integration/scenarios/concurrency_test.go`, lines 179–231.

`selected.Done()` is registered as a `defer` at line 189, which means it fires when the goroutine function returns. `selected.Wait()` is called at line 231 inside the same goroutine body, before any return. All three goroutines reach `selected.Wait()` simultaneously and block, waiting for three calls to `selected.Done()`. Since `Done()` is deferred (fires only on return), and each goroutine is blocked at `selected.Wait()` (which prevents return), the three goroutines deadlock each other permanently.

In practice this test will always time out rather than passing or failing. The correct pattern is to call `selected.Done()` immediately after closing the result cursor (line 229) rather than as a defer, so each goroutine signals that it has completed its SELECT phase before waiting for the others.

**I2 — `INTEGRATION_STRESS_N` env var is declared but never consumed**

File: `.github/workflows/test-nightly.yml:34` sets `INTEGRATION_STRESS_N: "500"`. No Go test code reads this variable. The nightly stress run therefore executes the same N=50 / 2-worker scenario as the full gate, providing no additional load beyond what the full workflow already covers. The variable name implies a configurable iteration count; the test should read `os.Getenv("INTEGRATION_STRESS_N")` and convert it to an integer to drive either the loop count or worker count.

**I3 — Outbox test covers only `doc.submitted`; other state transitions unverified**

File: `tests/integration/scenarios/outbox_same_tx_test.go`.

`TestOutbox_ApprovalInstanceInsertHasGovernanceEvent` only exercises the `doc.submitted` event. Invariant 9 requires governance events to be paired per state transition. State transitions such as `stage.completed`, `doc.approved`, `doc.published`, and `doc.cancelled` are not tested. If these event types are produced by separate code paths (triggers or application-layer inserts), their outbox pairing is untested.

### Minor

**M1 — `TestTriggerBypassBlocked` bypass path seeds user+doc but not tenant**

File: `tests/integration/scenarios/trigger_bypass_test.go`, lines 33–37.

When `SET LOCAL session_replication_role = 'replica'` succeeds, the test calls `fixtures.SeedUser` and `fixtures.SeedDocument` but not `fixtures.SeedTenant`. `SeedTenant` is a no-op in the current fixture implementation (`fixtures/seed.go:17–23`), so this causes no actual failure today. However, if `SeedTenant` is ever given substance (e.g. to insert into a tenants registry table), the bypass-path branch will fail unexpectedly. The call should be added for consistency.

**M2 — `idempotency_test.go` uses hardcoded UUIDs instead of `testdb.DeterministicID`**

File: `tests/integration/scenarios/idempotency_test.go`, lines 18, 84, 142, 218.

Four tests use manually constructed UUIDs (`11111111-1111-1111-1111-111111111181` through `...184`). These are functionally stable but differ from the convention used in all other scenario files, which use `testdb.DeterministicID(t, ...)`. The divergence creates a risk of UUID collision if parallel test runs share the same database or if additional tests are added with similar patterns. Migrating to `DeterministicID` is low-risk and improves consistency.

**M3 — `schema_lockdown_test.go` skips rather than fails on non-privilege errors**

File: `tests/integration/scenarios/schema_lockdown_test.go`, lines 64–75 (`require42501OrSkip`).

When a DDL operation returns an error that is not SQLSTATE 42501 and does not contain "permission denied" (e.g. a syntax error, missing object, or other DB-level error), the helper calls `t.Skip` instead of `t.Fatal`. This means a misconfigured database or a typo in the table name silently skips the test rather than surfacing a real error. The helper should distinguish between "operation succeeded" (skip) and "operation failed for non-privilege reason" (fatal).

**M4 — Smoke workflow includes `go vet` and source-tree walk tests**

File: `.github/workflows/test-smoke.yml:33`.

The `-run "TestLegacy"` filter matches `TestGoVetPasses` and `TestNoLegacyStatusInGoSource`, both of which walk the entire `internal/` directory tree and invoke the `go vet` subprocess. These operations are heavier than the DB-backed tests and may exceed the 2-minute timeout on cold CI runners with a large codebase. Consider moving these to the full gate tier or adding a separate `//go:build` tag to classify them as static-analysis tests.

---

## Sign-off

**APPROVED WITH CONDITIONS**

Conditions:

1. The `selected.Done()` defer deadlock in `testSkipLockedNoDuplicateProcessing` (I1) must be fixed before this test is relied upon in CI. As written, the test will always hang to timeout. Move `selected.Done()` from `defer` to an explicit call immediately after `rows.Close()` at line 229.

2. Either remove `INTEGRATION_STRESS_N` from the nightly workflow env block (if unused) or implement the read in the test code (I2). Leaving a declared-but-ignored env var creates misleading documentation.

3. Outbox invariant coverage for non-`doc.submitted` transitions should be tracked as a follow-up task (I3). It does not block this phase but should be addressed before Phase 12 E2E hardening.
