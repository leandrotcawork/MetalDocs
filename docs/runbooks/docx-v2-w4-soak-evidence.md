# W4 Exports Soak Evidence

> Fill this file in after completing the dogfood soak described in
> `docx-v2-w4-dogfood.md`. Commit it to the feature branch before
> merging W5 cutover.

## Soak Period

- Start: _____________
- End: _____________
- Total PDF exports performed: _____________
- Participants: _____________

## Pass Criteria Results

| # | Criterion | Target | Actual | Pass? |
|---|---|---|---|---|
| 1 | PDF exports succeed without errors | ≥ 18/20 | | |
| 2 | Cache hit rate | ≥ 80% | | |
| 3 | P95 cold-miss latency | < 20 s | | |
| 4 | P95 warm-hit latency | < 2 s | | |
| 5 | 429 triggers and clears correctly | 100% | | |
| 6 | DOCX download opens without corruption | 100% | | |
| 7 | No 5xx in audit log | 0 | | |
| 8 | No system-error export failures | 0 | | |

## Monitoring Queries Output

```
-- Paste output of cache miss ratio query here
```

```
-- Paste output of export volume query here
```

## Issues Found

<!-- List any bugs found during soak with status (fixed/deferred/wontfix) -->

## Sign-Off

Soak lead: ________________  Date: ________________

Tech lead: ________________  Date: ________________

## Gate Status

<!-- Set to PASS or FAIL before merging W5 -->
**GATE: PENDING**
