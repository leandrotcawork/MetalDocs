# Spec 2 Plan — Backlog Follow-ups (from Phase 12 Opus review)

Non-blocking hardening tasks identified during phase-end coverage cross-check. Size: ≤1 day each. Target: post-P1-release sprint.

- [ ] **G1** CI lint: signoff routes declare `@signature: password_reauth` in OpenAPI.
- [ ] **G2** Contract test: cascade returns `ErrCascadeTooLarge` at node N+1 (limit 1000).
- [ ] **G3** Handler lint: every mutating route returns `ETag` header + accepts `If-Match`.
- [ ] **G4** Migration lint: reject `timestamp` without tz; require `timestamptz`.
- [ ] **G5** ESLint rule: forbid raw `fetch()` for non-GET in `features/approval/**`.
- [ ] **G6** TS check: `StateBadge` + `SignoffDialog` + `RegistryDetailPanel` import policy from `lib/approvalPolicy.ts` only.
- [ ] **G7** E2E `offline_retry.spec.ts`: IndexedDB queue drains on reconnect; single governance event.
- [ ] **G8** E2E `permission_degradation.spec.ts`: 401 re-auth modal; 403 CTA hidden.
- [ ] **G9** E2E SignoffDialog error classes: network timeout, 429, 5xx, 412, 423.
- [ ] **G10** E2E `integrity_panel.spec.ts`: copy content_hash/version/ETag; drift warning.

---

## Phase 11-12 additions (from Task 12.12 Opus cross-check)

- [ ] **G11** E2E: stuck-instance watchdog auto-cancel (requires clock advance >7d — staging only)
- [ ] **G12** E2E: area RBAC negative case (user without area role cannot see doc)
- [ ] **G13** E2E: 401 redirect via session-expiry injection endpoint
- [ ] **G14** Canary: wire Prometheus to fetchRecentSamples (blocks prod canary) — HIGH
- [ ] **G15** Canary: replace httpGet stub with net/http.Get (blocks prod canary smoke) — HIGH
- [ ] **G16** Supply-chain: cosign signing on release tags + verification in canary controller
