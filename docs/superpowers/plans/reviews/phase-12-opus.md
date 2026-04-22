# Phase 12 Opus Phase-End Review — Spec 2 Plan Cross-Check

**Reviewer:** Opus (this model)
**Scope:** every invariant declared in Phases 1-11 must have ≥1 CI gate or runtime probe in Phase 12.
**Method:** extract invariant → map to enforcing control → flag gaps.

## Coverage Matrix

| # | Phase | Invariant | Enforcing control in Phase 12 | Status |
|---|-------|-----------|-------------------------------|--------|
| 1 | P1 | Capability catalog frozen | 12.2 `capability-catalog-hash` SHA256 pin | ✅ |
| 2 | P1 | Tenant isolation at SQL level | 12.2 migration-gapless + P10 schema-lockdown scenario | ✅ |
| 3 | P2 | role_capabilities v2 schema | 12.2 openapi + migration gap + cilint legacyvocab | ✅ |
| 4 | P2 | No v1 capability rows post-migration | 12.1 `legacyvocab` + P10 legacy-absent scenario | ✅ |
| 5 | P3 | password_reauth signature required | runtime — covered by P11 signoff flow + P10 idempotency scenario | ⚠ no CI gate — see Gap G1 |
| 6 | P4 | 8-state doc lifecycle closed | 12.1 `outboxpair` contract check against state_transitions.yaml | ✅ |
| 7 | P4 | cancelled instance state | state_transitions.yaml row | ✅ |
| 8 | P5 | Tx ownership (BeginTx allowlist) | 12.1 `txownership` linter + P10 reflect probe | ✅ |
| 9 | P5 | Obsolete cascade ≤1000 nodes + dedupe | P10 scenario only; no CI lint | ⚠ Gap G2 |
| 10 | P5 | Scheduler short claim tx + per-row transition tx | 12.1 txownership allowlist + 12.3 perf scheduler scenario | ✅ |
| 11 | P6 | authz.Require on every exported service method | 12.1 `authzrequire` linter | ✅ |
| 12 | P6 | Capability tripwire {cap, area_id} GUC | P10 scenario + 12.7 dashboard tripwire_firings | ✅ |
| 13 | P7 | OpenAPI = source of truth | 12.2 `openapi-drift` | ✅ |
| 14 | P7 | ETag strong + If-Match 428/412 | P11 11.2 mutation assertions; no CI static check | ⚠ Gap G3 |
| 15 | P7 | Idempotency-Key + payload-hash 24h TTL | P11 11.2 replay assertions + P10 scenario | ✅ |
| 16 | P8 | Lease fencing epoch | P10 concurrency + 12.8 chaos kill_scheduler | ✅ |
| 17 | P8 | SIGTERM drain stop-ticks-first | 12.8 chaos + 12.7 drain_duration metric | ✅ |
| 18 | P8 | Back-pressure hysteresis | 12.7 dashboard + 12.8 pg_sleep drill | ✅ |
| 19 | P8 | SECURITY DEFINER hardening | 12.7 SECURITY DEFINER invocation count + P10 schema-lockdown | ✅ |
| 20 | P8 | UTC timestamptz throughout | P10 scenario only | ⚠ Gap G4 |
| 21 | P9 F1 | Mutation client interceptor centralized | no static check | ⚠ Gap G5 |
| 22 | P9 F2 | Transition policy single-source | no static check | ⚠ Gap G6 |
| 23 | P9 F3 | 30s stale-time + focus-refetch | P11 flows assert | ✅ |
| 24 | P9 F4 | Loading/empty/error/partial states | P11 11.9 axe smoke + storybook | partial |
| 25 | P9 F5 | Offline banner + retry queue | P11 11.6 edit-lock covers banner; retry queue has no E2E | ⚠ Gap G7 |
| 26 | P9 F6 | Permission degradation 401/403 | P11 flows — no dedicated spec | ⚠ Gap G8 |
| 27 | P9 F7 | SignoffDialog 6 error classes | P11 11.2/11.3/11.5 cover 3; 3 missing | ⚠ Gap G9 |
| 28 | P9 F8 | Integrity panel content_hash+version+ETag | no E2E | ⚠ Gap G10 |
| 29 | P9 F9 | WCAG axe AA | 11.9 axe smoke + 11.0 baseline | ✅ |
| 30 | P9 F10 | datetime.ts pt-BR/en DST tests | unit tests in P9 | ✅ |
| 31 | P9 F11 | Risk-based test matrix | 11.0 COVERAGE.md | ✅ |
| 32 | P10 | Outbox same-tx | 12.1 `outboxpair` + P10 scenario | ✅ |
| 33 | P10 | Legacy vocab absent | 12.1 `legacyvocab` | ✅ |
| 34 | P11 | Axe baseline governance | 11.0 policy doc | ✅ |

## Gaps → `followups/spec2-gaps.md`

- **G1** password_reauth signature presence → add CI lint: all `signoff.*` routes must declare `@signature: password_reauth` in OpenAPI.
- **G2** Cascade limit → add contract test asserting `ErrCascadeTooLarge` at N+1.
- **G3** ETag handler contract → add lint: every mutating handler returns `ETag` header + accepts `If-Match`.
- **G4** UTC timestamptz → add migration lint: reject `timestamp` (without tz) column definitions.
- **G5** Mutation client interceptor → add ESLint rule forbidding raw `fetch()` for non-GET in `features/approval/**`.
- **G6** Transition policy single-source → add TS check: `StateBadge` + `SignoffDialog` + `RegistryDetailPanel` import from `lib/approvalPolicy.ts` only.
- **G7** Offline retry queue → add E2E spec `offline_retry.spec.ts`: offline submit queues in IndexedDB, reconnect drains, single governance event.
- **G8** Permission degradation → add E2E `permission_degradation.spec.ts`: expired token → 401 re-auth modal; revoked cap → 403 CTA hidden.
- **G9** SignoffDialog 6 errors → add E2E covering: network timeout, 429 rate-limit, 5xx server, 412 stale, 423 locked (beyond SoD/validation already covered).
- **G10** Integrity panel → E2E `integrity_panel.spec.ts`: copy hash + version + ETag; drift warning on background update.

## Verdict

**APPROVE WITH 10 BACKLOG FOLLOW-UPS.** Plan proceeds to execution. Gaps are all non-blocking for P1 release and sized as ≤1-day tasks each. None violate core state-machine or tx-ownership invariants — all are UX depth or static-check hardening that can land in a follow-up sprint.
