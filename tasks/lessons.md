## Lesson 1 — Preserve MDDM Variant Props on Save
Date: 2026-04-10 | Trigger: correction
Wrong:   `toMDDMProps` dropped `section.variant`, `field.layout`, `repeatableItem.style`, `richBlock.chrome`, and `dataTable.density` during save.
Correct: `toMDDMProps` explicitly persists those props with safe defaults for each block type.
Rule:    Adapter save mappings must preserve all renderer-relevant block props to avoid silent style regressions.
Layer:   application

## Lesson 6 - Ready callback must observe fully initialized editor state
Date: 2026-04-12 | Trigger: review
Wrong:   `onEditorReady` fired before `setEditorTokens(editor, tokens)` and tests only asserted token state after callback completion.
Correct: `MDDMEditor` now sets runtime tokens before calling `onEditorReady`, and callback-time test assertions verify tokens are already attached when the callback runs.
Rule:    Any readiness callback must expose a fully initialized object, and tests must assert state at callback time to catch ordering regressions.
Layer:   application

## Lesson 5 - Ignore local build/runtime artifacts at repo root
Date: 2026-04-10 | Trigger: correction
Wrong:   Root-local artifacts (`.gocache-build/`, `node_modules/`, `metaldocs-api.exe`) remained unignored and polluted `git status`.
Correct: Add explicit root ignore rules for local cache/dependency/build artifacts and keep them out of versioned scope.
Rule:    Repository root must ignore machine-local generated artifacts so `main` stays reviewable and operationally clean.
Layer:   process

## Lesson 4 - Keep tests aligned with active integration contracts
Date: 2026-04-10 | Trigger: correction
Wrong:   Tests still asserted browser export through `/generate-browser` and stale-row overwrite semantics after runtime moved to `/render/mddm-docx` and idempotent no-conflict seed insert.
Correct: Update tests to assert the current docgen route/payload contract and validate canonical insert idempotence from an empty seed state.
Rule:    When integration contracts change intentionally, adjust regression tests to the new source-of-truth behavior instead of preserving obsolete expectations.
Layer:   process

## Lesson 3 - Include Evidence Provenance Metadata
Date: 2026-04-10 | Trigger: correction
Wrong:   `docs/superpowers/reports/2026-04-10-mddm-verification-evidence.md` recorded check outcomes without commit/branch/runner metadata.
Correct: Verification evidence now includes `Git commit`, `Branch`, `Runner host`, and `Runner user` fields.
Rule:    Verification artifacts must include provenance metadata so results are reproducible and auditable.
Layer:   process

## Lesson 2 — Reject Invalid Data Table Columns
Date: 2026-04-10 | Trigger: correction
Wrong:   `parseDataTableColumns` accepted objects with empty `key`/`label` values and allowed duplicate column keys.
Correct: `parseDataTableColumns` only returns array entries with non-empty trimmed `key`/`label` values and skips repeated keys after the first valid occurrence.
Rule:    Parsers must enforce structural invariants at the boundary so downstream render code can assume valid, unique column definitions.
Layer:   application

## Lesson 7 - External HTML exporters must read runtime editor tokens
Date: 2026-04-12 | Trigger: correction
Wrong:   MDDM block `toExternalHTML` implementations closed over `defaultLayoutTokens`, and `RepeatableItem` export hardcoded `itemNumber={1}`.
Correct: All MDDM block exporters now call `getEditorTokens(editor)`, while section and repeatable item numbering derive from `editor.document`.
Rule:    Export renderers must consume runtime editor state instead of static defaults whenever layout tokens or derived numbering can vary at runtime.
Layer:   application
