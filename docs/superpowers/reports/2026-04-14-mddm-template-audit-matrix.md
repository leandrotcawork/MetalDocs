# MDDM Template Validation Audit Matrix (2026-04-14)

Source of truth:
- Phase 1 engine spec: `docs/superpowers/specs/2026-04-13-mddm-template-engine-design.md`
- Phase 2 admin plan: `C:/Users/leandro.theodoro.MN-NTB-LEANDROT/.claude/plans/vivid-inventing-flurry.md`

Status scale:
- `implemented`: requirement is implemented with direct code + test evidence
- `partial`: implementation exists, but missing one or more acceptance details or gate is red
- `missing`: not found in codebase

## Requirement Matrix

| Requirement bucket | Status | Implementation evidence | Test evidence |
|---|---|---|---|
| Frontend template API contract alignment (`delete`, `discard-draft`, `clone {newName}`, import raw body) | implemented | `frontend/apps/web/src/api/templates.ts` | `frontend/apps/web/src/api/__tests__/templates.test.ts` |
| Published edit flow must call `/templates/{key}/edit` before editor | implemented | `frontend/apps/web/src/components/templates/TemplateListPanel.tsx`, `frontend/apps/web/src/features/templates/useTemplateDraft.ts` | `frontend/apps/web/src/components/templates/__tests__/TemplateListPanel.behavior.test.tsx`, `frontend/apps/web/src/features/templates/__tests__/useTemplateDraft.test.tsx` |
| Typed codecs for block style/capabilities | implemented | `frontend/apps/web/src/features/documents/mddm-editor/engine/codecs/*.ts` | `frontend/apps/web/src/features/documents/mddm-editor/engine/codecs/__tests__/*.test.ts` |
| Strict client validation (unknown fields + wrong types + schema-aligned block traversal) | implemented | `frontend/apps/web/src/features/documents/mddm-editor/engine/codecs/validate-template.ts`, `codec-utils.ts` | `frontend/apps/web/src/features/documents/mddm-editor/engine/codecs/__tests__/strict.test.ts` |
| Layout interpreters + ViewModels as parity contract | implemented | `frontend/apps/web/src/features/documents/mddm-editor/engine/layout-interpreter/*` | `frontend/apps/web/src/features/documents/mddm-editor/engine/layout-interpreter/__tests__/*` |
| React emitters use interpreter output | implemented | `frontend/apps/web/src/features/documents/mddm-editor/blocks/*.tsx` | interpreter + block tests under `engine/layout-interpreter` and `mddm-editor/__tests__` |
| DOCX emitters use interpreter output | implemented | `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/emitters/*.ts` | `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/__tests__/*.test.ts`, golden tests |
| Template lifecycle service (create/save/publish/edit/deprecate/clone/delete/discard/import/acknowledge/export) | implemented | `internal/modules/documents/application/service_template_lifecycle.go` | `internal/modules/documents/application/service_template_lifecycle_test.go` |
| Atomic publish (insert version + delete draft) | implemented | `internal/modules/documents/infrastructure/postgres/template_drafts_repo.go` (`PublishTemplateAtomic`) | `internal/modules/documents/application/service_template_lifecycle_test.go` |
| Stripped-fields import + acknowledge gate before publish | implemented | `service_template_lifecycle.go`, `template_admin_handler.go`, `TemplateEditorView.tsx` | `service_template_lifecycle_test.go`, `playwright/e2e/template-admin-editor.spec.ts` |
| Publish validation response shape must include structured `errors[]` | implemented | `internal/modules/documents/domain/template_lifecycle.go`, `service_template_lifecycle.go`, `delivery/http/handler.go` | `internal/modules/documents/delivery/http/template_admin_handler_test.go`, `service_template_lifecycle_test.go` |
| Admin authoring UI workflow (create/save/publish, validation panel, list actions) | implemented | `frontend/apps/web/src/features/templates/*`, `src/components/templates/*` | `frontend/apps/web/playwright/e2e/template-admin-editor.spec.ts`, `template-admin-list-actions.spec.ts` |
| RBAC capability model (`template.view/edit/publish/export`) | implemented | `internal/modules/documents/domain/model.go`, `service_template_lifecycle.go`, `service_policies.go` | `service_template_lifecycle_test.go`, handler tests |
| Template audit log writes on mutation | implemented | `service_template_lifecycle.go` + `infrastructure/*/template_drafts_repo.go` | `service_template_lifecycle_test.go`, `internal/modules/documents/infrastructure/memory/template_drafts_repo_test.go` |
| Render parity gate (`mddm-visual-parity.spec.ts`) | implemented | `frontend/apps/web/e2e/mddm-visual-parity.spec.ts`, `src/test-harness/MDDMTestHarness.tsx`, `deploy/compose/docker-compose.yml` (`shm_size`) | `npx playwright test e2e/mddm-visual-parity.spec.ts --project=mddm-visual-parity --workers=1` passes |
| Legacy MDDM smoke specs in baseline command | implemented | Existing specs present and executable | `tests/e2e/mddm-create-from-template.spec.ts` and `tests/e2e/mddm-validation-rejection.spec.ts` pass with canonical single-worker smoke command |
| Phase-2 strict codec implementation detail from plan (`domain/mddm/codec_schemas/*.json` + `ParseBlocksStrict/Lenient`) | partial | Strict validation is implemented via `mddm.ValidateMDDMBytes` + business-rule checks in `service_template_lifecycle.go` | Service/handler tests pass, but named JSON schema file structure from plan is not present as described |

## Gate Verdict

- Gate 1 (contract audit): `implemented` for admin-template contracts, with one noted `partial` plan-vs-implementation detail on strict-codec file structure.
- Gate 2 (headless checks): `implemented` for template-focused backend/frontend suites.
- Gate 3 (browser authoring workflow): `implemented` via dedicated Playwright suites and canonical single-worker commands.
- Gate 4 (render parity evidence): `implemented` in this session.

Overall validation state: `implemented` for executable gates, with one documented plan-detail `partial` on strict-codec file layout naming.
