# MDDM Native DOCX Export Rollout Runbook

## Status ladder

| Phase | `MDDM_NATIVE_EXPORT_ROLLOUT_PCT` | Who is in the rollout |
|-------|----------------------------------|----------------------|
| Phase 1 (shadow) | 0 | Nobody sees the new path; everyone sends shadow telemetry |
| Phase 2 (canary) | 5 | ~5% of users export via the new path |
| Phase 2 (expanded) | 25 | ~25% of users |
| Phase 2 (half) | 50 | ~50% of users |
| Phase 3 (full) | 100 | All users |

## Promoting a phase

1. Run aggregate query on `metaldocs.mddm_shadow_diff_events` to confirm drift is acceptable:
   ```sql
   SELECT
     COUNT(*) FILTER (WHERE current_xml_hash = shadow_xml_hash) AS identical,
     COUNT(*) FILTER (WHERE current_xml_hash <> shadow_xml_hash) AS different,
     COUNT(*) FILTER (WHERE shadow_error <> '') AS failed,
     COUNT(*) AS total
   FROM metaldocs.mddm_shadow_diff_events
   WHERE recorded_at > NOW() - INTERVAL '7 days';
   ```
2. Acceptance thresholds:
   - `different / total < 5%`
   - `failed / total < 1%`
   - No `shadow_error` values that repeat more than 3 times
3. If thresholds are met, update the deployment env var:
   ```bash
   METALDOCS_MDDM_NATIVE_EXPORT_ROLLOUT_PCT=5
   ```
4. Redeploy (or restart the API process).
5. Verify the new percentage is active by loading the app and checking:
   ```bash
   curl -s http://localhost:8080/api/v1/feature-flags | jq .
   ```

## Monitoring during canary

Watch these indicators for 24 hours after each promotion:
- Application error rate from `mddm-engine:export-docx` scope (frontend telemetry)
- Support channel mentions of "DOCX export broken" or similar
- Docgen service latency (should drop as the percentage rises)
- DOCX generation time from the new path (should be < 3s p95)

## Rollback

If any indicator spikes:
```bash
METALDOCS_MDDM_NATIVE_EXPORT_ROLLOUT_PCT=0
```
and redeploy. Plan 1's docgen backend path is still active — no code revert is required.

## Decommission (Phase 4)

Only begin Phase 4 (Part 7+ of Plan 4) after two full weeks at 100% with no regressions.
The two-week safety window is not optional.
