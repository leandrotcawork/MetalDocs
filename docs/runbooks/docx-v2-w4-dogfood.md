# W4 Exports Dogfood Soak

## Goal

Validate W4 export functionality in a real environment before enabling for all users.
Gate criteria must all be met before proceeding to W5 cutover.

## Participants

- At least 2 engineers with writer-role accounts
- At least 1 admin account for RBAC verification

## Duration

Minimum 2 business days of active use. At least 20 PDF exports total.

---

## Soak Scenarios

### 1. Cold-miss PDF export
1. Open any finalized document.
2. Click **Export PDF**.
3. Verify PDF downloads and opens correctly.
4. Check `data-export-cached="false"` in the UI status badge.

### 2. Warm-hit PDF export (cache)
1. Without modifying the document, click **Export PDF** again.
2. Verify response is noticeably faster (< 2 s).
3. Check `data-export-cached="true"` in the UI status badge.

### 3. DOCX download
1. Click **Download .docx**.
2. Verify the file opens in Word / LibreOffice without errors.
3. Mergefields should be collapsed (all filled values present).

### 4. Rate-limit behaviour
1. Click **Export PDF** rapidly more than 20 times in under 1 minute.
2. Verify the UI shows "Rate limited — retry in Xs" alert.
3. Wait for the retry window and confirm export succeeds again.

### 5. Cross-user isolation
1. User A exports document D.
2. User B (no access to D) attempts `POST /api/v2/documents/{D}/export/pdf`.
3. Verify 403 returned.

### 6. Large document
1. Export a document with ≥ 20 pages.
2. Verify no timeout (HTTP 504 / 502) within 30 s.
3. Verify PDF page count matches document page count.

---

## Pass Criteria

| # | Criterion | Target |
|---|---|---|
| 1 | PDF exports succeed without errors | ≥ 18/20 |
| 2 | Cache hit rate (repeat exports) | ≥ 80% |
| 3 | P95 cold-miss latency | < 20 s |
| 4 | P95 warm-hit latency | < 2 s |
| 5 | 429 triggers after 20 req/min and clears correctly | 100% |
| 6 | DOCX download opens without corruption | 100% |
| 7 | No 5xx errors in audit log during soak | 0 |
| 8 | No data-export-status="error" due to system error (not user error) | 0 |

---

## Monitoring During Soak

```sql
-- Cache miss ratio (should be < 20% after warm-up)
SELECT payload->>'cached', count(*)
FROM audit_log
WHERE event_type = 'export.pdf_generated'
  AND created_at > now() - interval '2 days'
GROUP BY 1;

-- Error events
SELECT payload->>'error', count(*)
FROM audit_log
WHERE event_type = 'export.pdf_generated'
  AND payload->>'error' IS NOT NULL
GROUP BY 1;

-- Export volume
SELECT date_trunc('hour', created_at), count(*)
FROM document_exports
WHERE created_at > now() - interval '2 days'
GROUP BY 1 ORDER BY 1;
```

---

## Sign-Off

Once all pass criteria are met, record evidence in
`docs/runbooks/docx-v2-w4-soak-evidence.md` and obtain sign-off from
the tech lead before merging W5 cutover branch.

Soak lead: ________________  Date: ________________

Tech lead sign-off: ________________  Date: ________________
