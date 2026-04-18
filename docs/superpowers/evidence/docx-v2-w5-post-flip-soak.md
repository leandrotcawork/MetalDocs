# W5 Post-Flip Soak Log

Flag flip applied (UTC): YYYY-MM-DDTHH:MM:SSZ (env-var rollout — see Task 1 evidence)
Soak window:            5 business days, zero P0 incidents.
Target audience:        100% of tenants (all API/worker/frontend replicas report `METALDOCS_DOCX_V2_ENABLED=true` in `GET /api/v1/feature-flags`).

## Monitored SLOs (rolling 24h windows)
- /api/v2/documents POST availability ≥ 99.9%
- /api/v2/export/pdf p95 < 5000ms
- cached=false ratio < 50% (target 30% after day 2)
- 429 rate per user < 5/day max
- Gotenberg OOM events: 0

## Participants
- Admin on-call: @handle
- SRE: @handle
- Product manager: @handle

## Daily results

### Day 1 — YYYY-MM-DD
- /api/v2/documents availability: __.__%
- /api/v2/export/pdf p95: __ms
- cached=false ratio: __%
- 429 rate per user (max): __/day
- Gotenberg OOM events: __
- Tenants exercising /api/v2: __/__ (active/total)
- Incidents: none | P0 | P1 | P2 (describe)
- Sign-off: @admin @sre

### Day 2 — YYYY-MM-DD
- /api/v2/documents availability: __.__%
- /api/v2/export/pdf p95: __ms
- cached=false ratio: __%
- 429 rate per user (max): __/day
- Gotenberg OOM events: __
- Tenants exercising /api/v2: __/__ (active/total)
- Incidents: none | P0 | P1 | P2 (describe)
- Sign-off: @admin @sre

### Day 3 — YYYY-MM-DD
- /api/v2/documents availability: __.__%
- /api/v2/export/pdf p95: __ms
- cached=false ratio: __%
- 429 rate per user (max): __/day
- Gotenberg OOM events: __
- Tenants exercising /api/v2: __/__ (active/total)
- Incidents: none | P0 | P1 | P2 (describe)
- Sign-off: @admin @sre

### Day 4 — YYYY-MM-DD
- /api/v2/documents availability: __.__%
- /api/v2/export/pdf p95: __ms
- cached=false ratio: __%
- 429 rate per user (max): __/day
- Gotenberg OOM events: __
- Tenants exercising /api/v2: __/__ (active/total)
- Incidents: none | P0 | P1 | P2 (describe)
- Sign-off: @admin @sre

### Day 5 — YYYY-MM-DD
- /api/v2/documents availability: __.__%
- /api/v2/export/pdf p95: __ms
- cached=false ratio: __%
- 429 rate per user (max): __/day
- Gotenberg OOM events: __
- Tenants exercising /api/v2: __/__ (active/total)
- Incidents: none | P0 | P1 | P2 (describe)
- Sign-off: @admin @sre

## Exit decision

- [ ] All 5 days logged with within-SLO values.
- [ ] Zero P0 incidents.
- [ ] Tenants-exercising fraction reached ≥ 80% on at least one day (proves real traffic, not dark launch).
- [ ] Admin sign-off: @handle — YYYY-MM-DD
- [ ] Product manager sign-off: @handle — YYYY-MM-DD
- [ ] SRE sign-off: @handle — YYYY-MM-DD

**Decision:** GO / NO-GO → W5 destruction (Task 5 + Task 6).
