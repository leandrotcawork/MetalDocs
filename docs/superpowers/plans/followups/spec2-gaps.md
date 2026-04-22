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
