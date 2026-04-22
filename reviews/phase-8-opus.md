# Phase 8 Review — Opus

**Verdict**: PASS_WITH_NOTES

## Summary

Phase 8 delivers a functionally complete, lease-based distributed scheduler with fencing tokens, four background jobs, and graceful shutdown. Both critical issues flagged in the OPERATIONS round-1 review (epoch monotonicity, backpressure state lag) are confirmed applied and correct in the current codebase. One important structural gap remains in the integration tests: TestProbe6 verifies the pre-0149 delete behavior, not the post-0149 expire-in-place behavior, creating a test that would silently pass against a regressed implementation. All other dimensions are solid.

---

## Dimension Assessments

### 1. Lease SQL Correctness

**Rating: PASS**

The four SQL functions implement a correct distributed lease protocol.

`acquire_lease` (`migrations/0148_job_leases.sql:20-93`) uses `FOR UPDATE SKIP LOCKED` on the existing row, then branches on three states: (a) no row exists — insert with `epoch = 1`; (b) row exists and expired — UPDATE with `epoch + 1`; (c) row exists, not expired, same leader — renew heartbeat and return same epoch; (d) row exists, not expired, different leader — return `false, -1`. The `EXCEPTION WHEN unique_violation` block at lines 59-63 correctly handles the race window between SKIP LOCKED returning NOT FOUND and INSERT.

One semantic note on the NOT FOUND branch (lines 36-44): the function re-checks with a plain SELECT (without SKIP LOCKED) to determine whether the row exists but was locked by another session. If that plain SELECT also finds nothing it proceeds to INSERT. This is correct. If it finds a row (meaning another session holds the `FOR UPDATE` lock), it returns `false, -1`. The behavior is sound, though this two-step dance is slightly fragile: between the `FOR UPDATE SKIP LOCKED` returning nothing and the re-check SELECT, another session could insert and commit, causing the re-check SELECT to find the row and return `false`, while the original session's intent was to acquire a brand-new lease. In practice this is not a correctness bug — the caller will retry on the next tick — but it is worth noting.

`heartbeat_lease` correctly validates all three columns (job, leader, epoch) before updating. The hardcoded `interval '5 minutes'` at line 109 is a design choice that bypasses the caller's TTL, making the heartbeat renewal window implicit rather than parameterized. This is acceptable for a single-TTL design but should be documented.

`release_lease` — after migration 0149 (`migrations/0149_job_leases_epoch_monotonic.sql`) — now expires the row in place (`SET expires_at = now() - interval '1 second'`) instead of deleting it. This correctly preserves `lease_epoch` monotonicity for the job_name key. The epoch will only increase on the next `acquire_lease` call which hits the expired-branch UPDATE.

`assert_lease_epoch` raises SQLSTATE P0001 with a recognizable sentinel string `ErrLeaseEpochStale`. Notably it only checks `job_name + epoch`, not `leader_id`. This is sufficient for the fencing use-case (the caller already holds the epoch and can compare it) but it means a stale leader who somehow retains an old epoch value and calls the function will get a pass if no takeover has occurred. The current usage pattern (assert inside the job's own transaction with an epoch it already verified at acquire time) is safe.

### 2. Scheduler Runtime

**Rating: PASS**

The scheduler is well-structured. Key observations:

**Ticker management**: Each job loop creates its own ticker, registers it in `s.tickers`, and defers both `Stop()` and `unregisterTicker()`. `stopAllTickers()` takes a snapshot of the ticker slice under lock before stopping, avoiding modification-during-iteration. This is correct.

**Heartbeat goroutine**: The heartbeat is launched in a goroutine that receives a `stop` channel and a `cancel` func. On a heartbeat failure (lease stolen), it calls `cancel()` and returns, which correctly propagates cancellation into the job context. The job's `jobBaseCtx` is derived via `context.WithoutCancel(ctx)` (line 196), isolating the job context from the parent shutdown signal — the job only sees cancellation via the heartbeat's `cancel()`. This is an intentional design: a job should not be aborted merely because the process is shutting down; the drain sequence handles that separately. This is correct.

**Drain sequence**: `Start()` calls `stopAllTickers()` then `drain()` after `ctx.Done()`. The drain sequence: (1) wait up to `drainWait` (30s) for in-flight to reach zero; (2) if timeout, cancel all in-flight job contexts and wait `forceWait` (5s); (3) if still stuck, force-release leases via `context.Background()`. The `waitForInFlight` implementation polls at 10ms intervals (line 377), which is a tight busy-loop for a 30-second window. This works correctly but a channel-based notification (closing `inFlight.done`) would be more efficient. Not a defect for the current scale.

**Backpressure hysteresis** (IMP-001 fix confirmed): `probePressure` now returns `s.inPressure` from inside the lock (line 306), reflecting the current-tick decision. The `current` captured at line 263 is used only as the fallback for DB probe failure (line 274), which is correct — on a DB error, fall back to the last known state.

**Metrics accounting**: The `incRun` call at line 220 runs unconditionally after the job function returns, even when the job errored. This means `RunsTotal` counts both successful and failed executions, while `ErrorsTotal` is a subset. This is a reasonable counter semantics choice but is implicit — it would benefit from a comment clarifying that `RunsTotal` = total executions (success + failure).

**Log at job completion** (line 221): `"scheduler_job_completed"` is logged at INFO regardless of whether the job errored. This can be misleading in operational dashboards. A more accurate approach would log at WARN or ERROR on failure. This is a minor polish item.

### 3. Job Implementations

**Rating: PASS**

**`effective_date_publisher`**: Clean delegation to `svc.RunDuePublishes`. The backlog detection at line 35 (warn when `result.Processed >= DefaultBatchSize`) is correct and operational. The job's own DB handle is passed to `RunDuePublishes` — the service controls transaction scope internally. No issues.

**`stuck_instance_watchdog`**: Correctly bounded by `BatchSize = 50`. The GUC bypass pattern is implemented in `listStuckInstances` (line 101) by setting `metaldocs.bypass_authz` inside a transaction, which is necessary to allow the watchdog to query across tenant boundaries.

One structural concern: `setBypassAuthz` (lines 140-151) opens its own transaction just to set a GUC and commits it, then `cancelSvc.CancelInstance` opens another connection from the pool where the GUC is no longer set (GUC `set_config` with `is_local = true` only persists for the current transaction). The `set_config` call with `true` as the third argument is transaction-local. So when `CancelInstance` obtains a new connection from the pool, the `metaldocs.bypass_authz` GUC will not be set in that connection. The bypass is effectively a no-op for the cancel path unless `CancelInstance` itself sets it internally.

This is a correctness concern for the auto-cancel path. If the authorization check inside `CancelInstance` requires the GUC and it is not set, the cancel may fail with an authorization error rather than succeeding. The fact that tests apparently pass may reflect that bypass-authz is not enforced in the test environment or that `CancelInstance` has an internal bypass. Needs confirmation from the cancel service implementation, but should be flagged.

**`idempotency_janitor`**: Correctly bounded by `MaxIterations = 10` with early exit when `n == 0`. The query uses `ctid` for stable-row identity within the batch, which is a correct and efficient PostgreSQL pattern for bounded-batch deletes without a secondary index scan on every iteration. No issues.

**`lease_reaper`**: The DELETE uses a subquery with `FOR UPDATE SKIP LOCKED` to avoid contention with `acquire_lease` in flight. Deletes rows expired more than 10 minutes ago — this threshold (10 minutes beyond `expires_at`) is conservative by design so the reaper does not race with heartbeat renewal. Governance event emission per reclaimed row is correct and atomic within the single transaction.

Note: The `lease_reaper` hard-deletes rows (including their epoch history), which would reset monotonicity on the next insert for that job_name. However, the reaper only runs on leases expired for 10+ minutes and sets `expires_at = now() - 1s` is done by `release_lease`. In practice a reaped lease is one that was never properly released — the reaper reclaims truly dead leases. On the next `acquire_lease` call the NOT FOUND path would insert with `epoch = 1`, losing epoch history. For jobs that are reaped and then re-registered (e.g., after a long server outage), this means epoch monotonicity restarts from 1. This is an acceptable tradeoff documented in the spec (epoch is opaque across restarts), but the interaction between the reaper (hard delete) and the 0149 fix (soft expire via release_lease) creates a subtle asymmetry. The reaper bypasses the 0149 monotonicity guarantee. This is minor but worth noting for future auditors.

### 4. App Wiring

**Rating: PASS**

All four jobs are individually env-gated via `jobEnabled()` (lines 174, 182, 190, 198 of `main.go`). The `jobEnabled` function returns `true` by default (unless the env var is explicitly `"false"`), which is a safe-default-on pattern suitable for the intended deployment model.

`schedulerLeaderID()` returns `hostname:pid` (lines 310-316). This is stable across heartbeats for a given process (hostname does not change, pid does not change within a process lifetime), and is unique across pods/containers because hostname differs per pod in Kubernetes. The fallback to `"unknown"` if hostname cannot be obtained is safe — but if two instances both fail to resolve hostname, their leader IDs would collide as `unknown:<pid>`, and pid is not guaranteed unique across hosts. This is an edge case in broken environments, not a production concern, but worth documenting.

The scheduler is started in a goroutine with its own `sync.WaitGroup` (lines 207-212). `schedulerWG.Wait()` is called both in the server-error path (line 249) and in the shutdown path (line 259). This ensures the scheduler drains before the process exits. The `server.Shutdown(shutdownCtx)` uses a 15-second timeout, and the scheduler has a 30s drain + 5s force window = 35s total potential wait — if both are concurrent this could extend shutdown beyond the 15s HTTP drain window. This is acceptable since the HTTP server drains first (connections) and the scheduler can continue its drain in parallel.

### 5. Integration Tests

**Rating: PASS_WITH_NOTES**

The 10 probes cover the happy path and negative paths of all four SQL functions:

- TestProbe1: new-job acquire, epoch=1 check, table row verification
- TestProbe2: expired-lease takeover, epoch increment to 2
- TestProbe3: reentrant acquire by same leader, same epoch returned
- TestProbe4: heartbeat extends `expires_at`
- TestProbe5: heartbeat with stale epoch returns false
- TestProbe6: release with correct epoch — currently asserts `count = 0` (row deleted)
- TestProbe7: release with wrong epoch — row remains
- TestProbe8: assert_lease_epoch with valid epoch passes
- TestProbe9: assert_lease_epoch with stale epoch returns SQLSTATE P0001 + sentinel string
- TestProbe10: lease reaper deletes expired-11m-ago lease and emits governance event

**TestProbe6 is stale after the 0149 migration.** The test asserts `count = 0` (row deleted, lines 292-296). After migration 0149, `release_lease` expires the row in place instead of deleting it, so `count` will be 1, not 0. This test will fail against the current schema, making the probe a false negative. The probe must be updated to assert: (a) the row still exists (`count = 1`), and (b) `expires_at < now()`.

Beyond this, the following scenarios lack explicit coverage:

- **Epoch monotonicity across release+reacquire**: No probe verifies that after release_lease (expire) + acquire_lease, the new epoch is strictly `old_epoch + 1`. This is the exact invariant that CRIT-001 and 0149 protect. A TestProbe_EpochMonotonic test should be added.
- **Crash recovery** (leader dies without releasing): No probe simulates a process crash (lease expires naturally) followed by a new leader taking over at `epoch + 1`. TestProbe2 tests takeover of an artificially expired lease, but does not verify that a crashed-leader's epoch is not reusable.
- **Double-leader startup race**: No probe tests two concurrent `acquire_lease` calls for the same new job_name to confirm only one succeeds (the `EXCEPTION WHEN unique_violation` path).
- **Scheduler-level integration** (start/stop lifecycle, backpressure, job execution): There are no Go-level scheduler tests. All probes are SQL-function tests. The scheduler runtime is tested only at the SQL layer, not at the Go `Scheduler` struct layer. Adding even one smoke test for `Scheduler.Start` + job execution would improve confidence in the wiring.

The test harness helper `testDB` uses `LIKE 'test-%'` cleanup (line 37), which is correct and safe.

### 6. Operational Concerns

**Rating: PASS_WITH_NOTES**

**DB unavailable**: If the DB is down, `acquireLease` will return an error. The job loop logs the error and continues (line 188). Backpressure probing also fails and falls back to `currentPressure()` (line 274). This is correct — jobs are skipped until the DB recovers. No deadlock or hang risk. The heartbeat loop on an active job would also fail, and after a heartbeat error it logs and `continue`s (line 251-252) rather than cancelling the job — the job context is not cancelled on a transient heartbeat error, only on a `!ok` (lease stolen) response. A persistent DB outage would leave the heartbeat continuously failing and logging errors while the job continues running. This is acceptable (the job may complete before the DB comes back, and the lease will eventually expire), but the operator experience is noisy.

**Double-leader startup**: If two instances start simultaneously and both attempt `acquire_lease` for the same new job_name, the `unique_violation` exception block (lines 59-63) ensures only one succeeds. The other returns `false, -1` and waits for the next tick. This is correct.

**Lease TTL mismatch**: `acquire_lease` is called with `'5 minutes'` (scheduler.go line 319) and `heartbeat_lease` hardcodes `interval '5 minutes'` (0148 line 109). The heartbeat fires every `heartbeatEvery = time.Minute`. This gives a 5-minute lease renewed every 1 minute — a 5x safety margin before expiry. Healthy. However, `acquire_lease`'s TTL parameter and `heartbeat_lease`'s hardcoded TTL are separate and could diverge. If acquire is changed to a shorter TTL without updating heartbeat, the lease could expire between heartbeats.

**Reaper monotonicity bypass**: As noted in Dimension 3, the reaper hard-deletes rows rather than soft-expiring them. For a reaped job that later restarts, `epoch` resets to 1. Old assert tokens from a prior run would pass `assert_lease_epoch` if the new epoch happens to be 1 again. This is a narrow but real epoch reuse window for reaped-then-restarted jobs.

---

## Issues Found

### Critical

None.

### Important

**IMP-001 (new): TestProbe6 assertion is wrong after migration 0149.**

- File: `internal/modules/jobs/scheduler/integration_test.go`, lines 291-297
- The probe calls `release_lease` then checks `count(*) = 0`, asserting the row was deleted. After 0149, `release_lease` expires the row in place — the row remains. The test will fail. This makes the probe a regression detector that fires on correct behavior. It must be updated to:
  - Assert `count(*) = 1` (row exists)
  - Assert `expires_at < now()` (row is expired)

**IMP-002: Epoch monotonicity invariant has no test coverage.**

- File: `internal/modules/jobs/scheduler/integration_test.go`
- No probe verifies that after `release_lease` + `acquire_lease` (expire-and-reacquire), the new epoch equals `old_epoch + 1`. This is the invariant introduced by 0149. Without this probe, a regression to the delete-and-reinsert behavior would not be caught. A `TestProbe_EpochMonotonicAcrossRelease` probe should be added.

**IMP-003: GUC bypass in `setBypassAuthz` is ineffective for the `CancelInstance` call.**

- File: `internal/modules/jobs/stuck_instance_watchdog/job.go`, lines 140-151
- `set_config('metaldocs.bypass_authz', ..., true)` is transaction-local. The transaction is committed at line 150, and the GUC value is discarded. When `cancelSvc.CancelInstance` runs (line 59), it uses a fresh connection from the pool where the GUC is not set. If `CancelInstance` enforces authz via this GUC, the auto-cancel will fail silently (error is logged and `continue`d at line 68). Recommend: either pass the bypass signal via the `CancelInput` struct, or set the GUC inside the same transaction that `CancelInstance` uses.

### Minor

**MIN-001: `"scheduler_job_completed"` logged at INFO on error.**

- File: `internal/modules/jobs/scheduler/scheduler.go`, line 221
- The log line is emitted regardless of whether `err != nil`. In an operational dashboard, an INFO "completed" log immediately after an ERROR "failed" log is confusing. Consider gating the INFO log on `err == nil` and logging at WARN or WARN+err on failure.

**MIN-002: `RunsTotal` counter includes failed runs without documentation.**

- File: `internal/modules/jobs/scheduler/scheduler.go`, line 220
- `incRun` is called after the job function returns regardless of error. This means `RunsTotal` = successes + failures. A comment at the counter definition (`metrics.go` or inline) clarifying this would prevent future misinterpretation.

**MIN-003: Reaper hard-delete bypasses 0149 epoch monotonicity for reaped-then-restarted jobs.**

- File: `internal/modules/jobs/scheduler/lease_reaper.go`, lines 18-26
- The reaper `DELETE`s rows rather than soft-expiring them. For a job reaped due to a long outage that then restarts, `acquire_lease` inserts with `epoch = 1`. A stale token holding `epoch = 1` from a prior run would pass `assert_lease_epoch`. The window is narrow (requires a stale token surviving across an outage + reaper cycle), but the interaction with 0149 is asymmetric. Consider having the reaper UPDATE `expires_at` (soft expire) rather than DELETE, or accept this as documented behavior and add a comment.

**MIN-004: Heartbeat TTL is hardcoded in `heartbeat_lease`, not derived from the acquire TTL.**

- File: `migrations/0148_job_leases.sql`, line 109
- The heartbeat extends expiry by a hardcoded `interval '5 minutes'` rather than accepting a TTL parameter. If the acquire TTL is ever tuned, the heartbeat TTL must be updated separately. Recommend parameterizing or at least adding a comment coupling the two.

---

## Phase 8 Sign-off

**APPROVED WITH CONDITIONS**

Conditions:

1. **Fix TestProbe6** (`integration_test.go`) — update the assertion from `count = 0` (row deleted) to `count = 1` + `expires_at < now()` to match the post-0149 behavior. This is the only test that is currently wrong against the shipped code.

2. **Add TestProbe_EpochMonotonicAcrossRelease** — verify that `release_lease` followed by `acquire_lease` on the same job yields `new_epoch = old_epoch + 1`. This locks in the CRIT-001 / 0149 invariant against regression.

3. **Investigate IMP-003** (GUC bypass in watchdog auto-cancel path) — confirm whether `CancelInstance` applies its own bypass or relies on the GUC being set in-connection. If the GUC is relied upon, fix the call site.

Conditions 1 and 2 are test-layer fixes that do not require migration or production deployment changes. Condition 3 requires code investigation before the next phase merges the watchdog into a production-monitored environment.
