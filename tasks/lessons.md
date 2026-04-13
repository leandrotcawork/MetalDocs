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

## Lesson 7 - Repeatable item numbering must search only repeatable siblings
Date: 2026-04-12 | Trigger: correction
Wrong:   `findItemIndex` matched any block children and used `1` as the recursive miss value, so helper blocks and empty branches produced incorrect nested repeatable numbering.
Correct: `findItemIndex` now matches only `repeatableItem` siblings inside `repeatable` blocks, recurses deeper with `0` as the miss sentinel, and falls back to `1` only at the call site.
Rule:    Recursive numbering helpers must scope sibling matching to the owning container type and use a distinct not-found sentinel during traversal.
Layer:   application

## Lesson 8 - Custom table blocks need a DOM-backed node view
Date: 2026-04-13 | Trigger: correction
Wrong:   `dataTable` replaced BlockNote's generated PM node with a custom `TiptapNode` but relied on `renderHTML`, which mounted table rows under a `<div>` and produced invalid table markup in-editor.
Correct: Custom PM nodes that own table-like content must provide `addNodeView()` with a real table `contentDOM` so ProseMirror mounts `<tr>` children into valid table structure.
Rule:    When a BlockNote block swaps in a custom ProseMirror node for structured content, it must also provide the DOM container that matches that content model instead of relying on fallback HTML rendering.
Layer:   application

## Lesson 9 - Structural table upgrades need neutral filler cells
Date: 2026-04-13 | Trigger: correction
Wrong:   The `fieldGroup -> table` import path filled an odd trailing slot in 2-column layouts with a bold header-style empty label cell and emitted value cells with `styles: undefined`.
Correct: Adapter upgrades that synthesize table rows now emit neutral empty cells for missing positions and omit undefined style payloads so imported structure matches real user-editable table content.
Rule:    When upgrading sparse structured content into native table cells, placeholders must be semantically empty rather than styled sentinel cells.
Layer:   application

## Lesson 10 - Native editor upgrades need reversible metadata on save
Date: 2026-04-13 | Trigger: correction
Wrong:   `fieldGroup` blocks were upgraded into native BlockNote `table` blocks on load, but the save path treated them as unsupported plain tables and lost the original structured `fieldGroup/field` model.
Correct: Native editor upgrades that temporarily project structured MDDM blocks into BlockNote-native blocks must carry parseable metadata and a dedicated reverse adapter so save reconstructs the canonical storage shape.
Rule:    Any adapter that maps canonical structured content into a native editor block type must implement an explicit reverse conversion before persistence.
Layer:   application

## Lesson 11 - Schema removals must update editor integration fixtures
Date: 2026-04-13 | Trigger: correction
Wrong:   `runtime-token-export.integration.test.tsx` still exported a `field` block after the BlockNote schema removed `field/fieldGroup`, causing BlockNote HTML serialization to read an undefined block spec.
Correct: When a custom block type is removed from `mddmSchemaBlockSpecs`, editor integration tests and fixtures must be rewritten to use only the remaining registered block types.
Rule:    Any schema-level block removal must be reflected immediately in editor integration fixtures so export/runtime tests only exercise registered block specs.
Layer:   process

## Lesson 12 - Native table adapters must accept both BlockNote cell encodings
Date: 2026-04-13 | Trigger: correction
Wrong:   The `fieldGroup -> table` upgrade assumed every table cell used BlockNote's simplified `InlineContent[]` encoding, so adding label-cell props broke reverse conversion and edit guards that compare table content.
Correct: Native table adapters and guards must normalize both simplified array cells and full `{ type: "tableCell", props, content }` cells before reading or comparing structured table data.
Rule:    Whenever a BlockNote table feature starts using cell-level props, all reverse adapters and block-level guards must handle both cell encodings explicitly.
Layer:   application

## Lesson 13 - Editor DOM guards must live at the active editing surface
Date: 2026-04-13 | Trigger: correction
Wrong:   `MDDMEditor` tried to reject label-cell edits via BlockNote `onBeforeChange`, after ProseMirror had already applied text mutations inside `<th>` cells.
Correct: Header-cell edit locks now attach to the Tiptap view DOM and force `<th>` nodes to `contentEditable="false"` on mount and after DOM mutations.
Rule:    When text editing happens in a lower editor layer, enforcement must be attached at that layer's live DOM or transaction surface rather than a higher-level block diff hook.
Layer:   application
