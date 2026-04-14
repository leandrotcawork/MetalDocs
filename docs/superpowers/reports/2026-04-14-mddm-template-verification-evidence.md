# MDDM Template Verification Evidence - 2026-04-14

Session timestamp: 2026-04-14T12:49:00-03:00
Git commit: 66293e1883a27401be5a9cec812e10cffe65d7ba
Branch: main
Runner host: MN-NTB-LEANDROT
Runner user: leandro.theodoro

## Scope

This report verifies the Playwright-first MDDM template validation strategy with direct evidence for contract alignment, automated checks, browser workflow checks, render parity, and DOCX generation.

## Commands Executed

### Backend template-focused checks

- [x] `go test ./internal/modules/documents/application -run Template -count=1`
  - Result: pass
- [x] `go test ./internal/modules/documents/delivery/http -run Template -count=1`
  - Result: pass

### Frontend template-focused checks

- [x] `npm.cmd --prefix frontend/apps/web test -- --run src/api/__tests__/templates.test.ts src/features/templates/__tests__/PropertySidebar.test.tsx src/components/templates/__tests__/TemplateListPanel.test.tsx`
  - Result: pass (`32` tests)

### Browser workflow checks

- [x] `npm.cmd run e2e:template-admin` (in `frontend/apps/web`, with local dev server)
  - Result: pass (`5` tests)
- [x] `npm.cmd run e2e:smoke -- --project=chrome tests/e2e/mddm-create-from-template.spec.ts tests/e2e/mddm-validation-rejection.spec.ts` (in `frontend/apps/web`, with local dev server)
  - Result: pass (`2` tests)

### Visual parity gate

- [x] `npx playwright test e2e/mddm-visual-parity.spec.ts --project=mddm-visual-parity --workers=1` (with local Vite dev server)
  - Result: pass (`3` tests)
  - Infra evidence: Gotenberg HTML conversion returns `200` after compose `shm_size` correction.

### Docgen harness gate

- [x] `powershell -ExecutionPolicy Bypass -File apps/docgen/scripts/harness.ps1`
  - Result: pass
  - Evidence: `OK: DOCX size = 11721 bytes`

## Pass/Fail Matrix

- Contract alignment: PASS
- Backend template lifecycle/handler tests: PASS
- Frontend template unit/integration tests: PASS
- Admin template browser workflow: PASS
- Legacy baseline smoke specs: PASS
- Visual parity suite: PASS
- Docgen harness: PASS

## Verification Verdict

All planned validation gates executed and passed in this session.
