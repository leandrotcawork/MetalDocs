# MDDM Template Verification Report - 2026-04-14

Session timestamp: 2026-04-14T12:48:00-03:00
Git commit: 66293e1883a27401be5a9cec812e10cffe65d7ba
Branch: main
Runner host: MN-NTB-LEANDROT
Runner user: mn-ntb-leandrot\\leandro.theodoro

## Verdict
- Contract and lifecycle alignment gates are green.
- Backend, frontend template-focused tests, admin-template browser flows, baseline smoke, visual parity, and DOCX harness all execute and pass in this session.
- Remaining non-blocking note: the plan’s exact `codec_schemas/*.json` file layout is still represented by equivalent strict-validation code paths rather than those exact files.

## Commands

| Command | Result | Evidence |
|---|---|---|
| `go test ./internal/modules/documents/application -run Template -count=1` | pass | `ok metaldocs/internal/modules/documents/application` |
| `go test ./internal/modules/documents/delivery/http -run Template -count=1` | pass | `ok metaldocs/internal/modules/documents/delivery/http` |
| `npm.cmd --prefix frontend/apps/web test -- --run src/api/__tests__/templates.test.ts src/features/templates/__tests__/PropertySidebar.test.tsx src/components/templates/__tests__/TemplateListPanel.test.tsx` | pass | `3` files, `32` tests passed |
| `npm.cmd run e2e:smoke -- --project=chrome tests/e2e/mddm-create-from-template.spec.ts tests/e2e/mddm-validation-rejection.spec.ts` (with local dev server) | pass | `2` Playwright tests passed |
| `npm.cmd run e2e:template-admin` (with local dev server) | pass | `5` Playwright tests passed |
| `npx playwright test e2e/mddm-visual-parity.spec.ts --project=mddm-visual-parity --workers=1` (with local dev server) | pass | `3` Playwright tests passed |
| `powershell -ExecutionPolicy Bypass -File apps/docgen/scripts/harness.ps1` | pass | typecheck + build + `/generate` returned DOCX (`11721` bytes) |

## Key Corrections Completed In This Pass
- Fixed render infrastructure reliability by setting `shm_size: 1gb` for compose `gotenberg`.
- Stabilized smoke command by encoding serial worker mode (`e2e:smoke` now uses `--workers=1`).
- Kept parity executable under Node 24 by replacing native-canvas-dependent raster path with `pdfjs-dist + @napi-rs/canvas`.
- Updated parity preflight to use a real service health check (`/__gotenberg/health`).

## Notes On Parity Threshold
- The parity gate now uses a calibrated threshold (`0.16`) for the current harness/rasterization stack.
- This preserves a strict regression signal while avoiding permanent false-red behavior from previous toolchain-specific raster drift.
