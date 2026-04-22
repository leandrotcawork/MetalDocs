# Axe Baseline Policy

**Owner:** frontend-lead  
**Last updated:** 2026-04-22  
**Scope:** All approval E2E flows (`e2e/flows/`)

---

## Purpose

Tracks known axe-core violations that are accepted as baseline. Net-new violations introduced by PRs fail the E2E coverage gate unconditionally.

---

## Baseline Update Process

1. Author opens a PR that updates `e2e/axe-baseline.json`.
2. PR description must include a `reason` field explaining why the violation is acceptable (e.g., third-party component, known upstream bug with fix ETA).
3. At least **1 reviewer** from the frontend-lead list must approve the baseline change before merge.
4. Changes to baseline are never bundled with feature changes — dedicated PR only.

---

## Axe Configuration (Deterministic Runs)

To prevent false-positive diffs from dynamic content:

| Setting | Value | Reason |
|---------|-------|--------|
| Viewport | 1280×800 | Fixed across all workers |
| Locale | `pt-BR` | Primary language target |
| `page.clock.install()` | Freeze at test start | Eliminates relative-time churn in timeline/lock-badge |
| `AxeBuilder.exclude('.timeline-timestamp')` | Yes | Timestamps are dynamic; content tested separately |
| `AxeBuilder.exclude('.lock-badge-relative-time')` | Yes | Relative time; semantic role tested in unit tests |
| WCAG tags | `['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa']` | WCAG 2.1 AA compliance target |

---

## Severity Levels

| axe Impact | Treatment |
|-----------|-----------|
| `critical` | Always fails build, even if in baseline (must be fixed) |
| `serious` | Fails build; can be baselined with justification |
| `moderate` | Fails build; can be baselined with justification |
| `minor` | Logged only; never fails build |

---

## Baseline File Format

```json
[
  {
    "id": "color-contrast",
    "impact": "serious",
    "nodes": ["button.secondary-action"],
    "reason": "Upstream design token bug; tracked in design-system#142; fix ETA 2026-06-01",
    "approved_by": "frontend-lead",
    "approved_at": "2026-04-22"
  }
]
```

---

## Running Axe Locally

```bash
# Run single flow
pnpm playwright test e2e/flows/happy_path.spec.ts --project=parallel-flows

# Generate axe diff vs baseline
node scripts/axe-diff.mjs --report test-results/axe-report.json --baseline e2e/axe-baseline.json
```

---

## CI Integration

`.github/workflows/e2e-coverage-gate.yml` runs `axe-diff.mjs` after each E2E suite. Added violations fail the gate. Removed violations are logged (baseline can be trimmed).
