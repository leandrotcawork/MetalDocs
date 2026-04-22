# E2E Flows

## Required Environment Variables
- `E2E_BASE_URL` (optional; defaults to `http://localhost:8080`)
- `METALDOCS_E2E=1`
- `DATABASE_URL`

## Run Approval Flows (Parallel)
```bash
pnpm exec playwright test e2e/flows/ --project=parallel-flows
```

## Run Scheduled/Clock Flows (Serial)
```bash
pnpm exec playwright test e2e/flows/ --project=serial-clock
```

## View Playwright Report
```bash
pnpm exec playwright show-report
```
