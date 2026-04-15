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

## Lesson 14 - Scrollbar fixes must target the true scroll owner
Date: 2026-04-13 | Trigger: correction
Wrong:   The content-builder scrollbar bug investigation focused on BlockNote/editor wrappers before confirming which element actually owned `scrollTop` and `scrollHeight`.
Correct: Instrument the live page first and apply scrollbar behavior changes only on the confirmed scroll owner (`DocumentWorkspaceShell.workspace-main`), leaving non-scrolling editor descendants untouched.
Rule:    Visual scrolling bugs must be fixed on the element that truly owns scrolling, not on descendant content surfaces that only move with their parent.
Layer:   process

## Lesson 15 - Locked editor guards must preserve node identity, not counts
Date: 2026-04-13 | Trigger: correction
Wrong:   `MDDMEditor` accepted transactions that replaced locked template blocks as long as the total count of `locked=true` nodes stayed the same.
Correct: The transaction guard now compares locked-node identities across `state.doc` and `tr.doc` and rejects any mutation that removes a previously locked node identity.
Rule:    Lock enforcement for structured editors must validate protected-node identity continuity, not just aggregate counts.
Layer:   application

## Lesson 15 - Static toolbar buttons must not steal table-cell focus
Date: 2026-04-13 | Trigger: correction
Wrong:   The MDDM formatting toolbar let `mousedown`/`pointerdown` on interactive controls blur the active ProseMirror table cell before the BlockNote button logic ran.
Correct: The toolbar now captures pointer/mouse down on interactive controls and prevents default so the table-cell selection survives until the click handler applies the formatting.
Rule:    In ProseMirror table editing, preserve cell selection through toolbar interaction or formatting state will desynchronize from the visible cursor.
Layer:   delivery

## Lesson 16 - Table alignment state must come from the active cell
Date: 2026-04-13 | Trigger: correction
Wrong:   The default BlockNote alignment button path in MDDM read table alignment from `rows[0].cells[0]` and then moved the caret to the table start after updates, which desynced toolbar state and cursor position for non-first cells.
Correct: MDDM now uses table-aware alignment buttons that read alignment from the currently selected cell and reapply selection after table updates instead of forcing cursor placement at table start.
Rule:    For table formatting controls, derive UI state and post-update cursor behavior from the active cell selection, not from a fixed cell.
Layer:   delivery

## Lesson 17 - Table alignment must restore text cursor, not cell selection
Date: 2026-04-13 | Trigger: correction
Wrong:   After table alignment updates, the editor restored a `CellSelection`, leaving the cell highlighted and causing next typing to replace the whole cell content.
Correct: When alignment is triggered from a collapsed text cursor, the post-update restore now converts selection back to `TextSelection` inside the same cell.
Rule:    In rich-text tables, style actions triggered from a caret must return to caret mode after structural updates to preserve typing semantics.
Layer:   delivery

## Lesson 18 - Locked-node identity checks cannot use document position fallback
Date: 2026-04-13 | Trigger: correction
Wrong:   The lock guard used `pos:${pos}` as fallback identity for locked nodes without `attrs.id`, causing normal edits to be rejected when node positions shifted.
Correct: Lock guard now enforces stable `attrs.id` continuity for locked nodes and uses count-only fallback for legacy anonymous locked nodes to avoid false rejections.
Rule:    Transaction guards must never use mutable position as identity when edits can reorder or reflow the same protected nodes.
Layer:   application

## Lesson 19 - PO template blocks must stay schema-compatible
Date: 2026-04-13 | Trigger: correction
Wrong:   `po_template.go` emitted unsupported block props (`field.layout`, `repeatableItem.locked`) and an invalid stage seed shape (empty Etapa item and invalid image placeholder), causing MDDM schema validation to fail.
Correct: PO template builders now emit only schema-allowed props, include the required `Etapas` rich content seed, and keep optional diagram content in a schema-valid placeholder block.
Rule:    Canonical template fixtures must be generated strictly from the active MDDM schema contract, including nested child block shapes.
Layer:   domain

## Lesson 20 - RBAC denials need a dedicated forbidden sentinel
Date: 2026-04-14 | Trigger: correction
Wrong:   Template RBAC denials returned `domain.ErrDocumentNotFound`, conflating authorization failures with real not-found conditions.
Correct: Template RBAC denials now return `domain.ErrForbidden`, and delivery maps it to HTTP 404 with debug-only logging for security-through-obscurity.
Rule:    Authorization denials must use a dedicated domain sentinel and only be masked at the delivery boundary.
Layer:   application

## Lesson 21 - Lifecycle selection must read and filter template status
Date: 2026-04-14 | Trigger: correction
Wrong:   `ListDocumentTemplateVersions` did not scan `status`, and lifecycle "latest published" selection used max version by key without `published` status filtering.
Correct: `ListDocumentTemplateVersions` now selects/scans `status`, and lifecycle selectors accept only `published` (or empty legacy status) before choosing the max version.
Rule:    Any lifecycle decision that depends on version state must load status from persistence and filter status explicitly before version ordering.
Layer:   application

## Lesson 22 - Template publish must persist version+draft cleanup atomically
Date: 2026-04-14 | Trigger: correction
Wrong:   `PublishAuthorized` called `InsertTemplateVersion` then `DeleteTemplateDraft` separately and swallowed draft-delete failures, allowing published version + orphan draft divergence.
Correct: Template publish now uses `PublishTemplateAtomic` in the repository to insert the published version and delete the draft in one transaction/critical section.
Rule:    Lifecycle transitions that move state across tables must be atomic and must never swallow cleanup failures.
Layer:   infrastructure

## Lesson 23 - Template admin clients must follow handler contracts exactly
Date: 2026-04-14 | Trigger: correction
Wrong:   The frontend template client used stale routes and payloads (`DELETE /draft`, `POST /discard`, clone body `{name}`, multipart import) that no longer matched `template_admin_handler.go`.
Correct: Template admin clients now call the exact handler contract for path, method, payload, and body shape before any browser-level validation is trusted.
Rule:    Admin workflow verification is meaningless until the UI client is aligned byte-for-byte with the active backend contract.
Layer:   delivery

## Lesson 24 - Publish must persist live editor state before lifecycle transition
Date: 2026-04-14 | Trigger: correction
Wrong:   The template editor publish action validated the current BlockNote document but published the last saved draft, allowing stale content to be promoted.
Correct: Template publish now saves the current editor blocks first, advances `lockVersion`, and only then calls the publish endpoint.
Rule:    Any lifecycle action that promotes editable state must persist the in-memory editor document before the transition request is sent.
Layer:   application

## Lesson 25 - Client strict validation must normalize sparse codec storage
Date: 2026-04-14 | Trigger: correction
Wrong:   `validate-template.ts` validated only legacy `props.style` / `props.caps` objects and treated missing capability fields as errors, even though the editor persists sparse `styleJson` / `capabilitiesJson` with codec defaults implied.
Correct: The client publish validator now reads persisted JSON fields, preserves unknown-field checks, and merges codec defaults before strict capability validation.
Rule:    Strict client validation must validate the effective codec state, not reject valid sparse editor storage that relies on explicit defaults.
Layer:   application

## Lesson 26 - Browser validation needs stable selectors at workflow boundaries
Date: 2026-04-14 | Trigger: correction
Wrong:   The template authoring sidebar exposed critical controls only through unlabeled sibling inputs and tab text, forcing brittle browser tests to depend on DOM order.
Correct: Template authoring controls now expose stable `data-testid` hooks for tabs and editable property/style/capability fields used by the validation workflow.
Rule:    Any user-critical workflow that must be browser-verified needs stable selectors on its actionable controls before the e2e suite can be trusted.
Layer:   process

## Lesson 27 - Strict codecs must reject present optional fields with wrong types
Date: 2026-04-14 | Trigger: correction
Wrong:   The React strict codecs only rejected unknown style keys, but silently accepted present style fields with the wrong type by coercing them to `undefined`.
Correct: Optional style fields now remain optional, but any present non-string value raises `CodecStrictError` so the client publish gate matches the intended strict contract.
Rule:    In a strict validator, optional means omit-or-valid; it never means accept an invalid present value by dropping it.
Layer:   domain

## Lesson 28 - Duplicated draft state must update both local and shared stores
Date: 2026-04-14 | Trigger: correction
Wrong:   The stripped-fields acknowledgement flow updated the shared template store but left `useTemplateDraft`'s local draft stale, so the banner and lockVersion could remain outdated.
Correct: Server-returned draft replacements now update both local hook state and the shared store through a single `replaceDraft` path.
Rule:    When a feature keeps mirrored local and global state, every server-authoritative replacement must flow through one shared updater or the UI will drift.
Layer:   application

## Lesson 29 - Client validators must track the actual editor schema, not a subset guess
Date: 2026-04-14 | Trigger: correction
Wrong:   The template publish validator only recognized custom template blocks, so valid built-in BlockNote blocks like `paragraph` were flagged as unknown during real authoring flows.
Correct: The client validator now passes through built-in schema block types and keeps strict codec checks only for custom MDDM blocks.
Rule:    Any validator attached to editor output must be derived from the editor schema surface, or browser workflows will fail on valid runtime-generated blocks.
Layer:   domain

## Lesson 30 - Block-tree validation must distinguish child blocks from inline runs
Date: 2026-04-14 | Trigger: correction
Wrong:   The template block validator recursively treated every `children` entry as another block, so inline text runs under built-in blocks like `paragraph` were reported as unknown block types.
Correct: Recursive validation now descends only into real block nodes and ignores inline text children.
Rule:    Any traversal over BlockNote/MDDM trees must separate block children from inline content nodes before applying block-level rules.
Layer:   domain

## Lesson 31 - Playwright route globs must cover nested REST action paths
Date: 2026-04-14 | Trigger: correction
Wrong:   The template admin browser stub matched only `/templates/:key`, so nested action routes like `/templates/:key/clone` and `/deprecate` leaked past the harness.
Correct: Browser route interception now uses nested path globs that explicitly cover action endpoints under `/templates/**`.
Rule:    When stubbing REST endpoints in Playwright, match the full action path depth or the harness will silently test the wrong backend surface.
Layer:   process
## Lesson 32 - Publish validation must preserve structured failure details
Date: 2026-04-14 | Trigger: correction
Wrong:   Template publish failures collapsed to the sentinel `ErrTemplatePublishValidation`, so the HTTP layer returned a generic 422 without the field-level `errors[]` payload the admin client expects.
Correct: Template publish now returns a wrapped validation error carrying structured `PublishError[]`, and the handler serializes that exact payload in the 422 response.
Rule:    When the UI depends on structured validation feedback, domain errors must preserve the machine-readable details all the way through the delivery response.
Layer:   delivery
## Lesson 33 - E2E login checks must target stable shell indicators
Date: 2026-04-14 | Trigger: correction
Wrong:   The template-admin Playwright helper treated post-login success as a specific `button` role named "Todos Documentos", which broke when the shell rendered the same destination as navigation text instead of that exact role.
Correct: The login helper now waits for stable authenticated shell text (`Todos Documentos` or `Painel documental`) rather than a brittle role-specific widget.
Rule:    Browser helpers should assert authentication against stable app-shell landmarks, not transient component roles.
Layer:   process
## Lesson 34 - Shared seeded e2e suites need an explicit single-worker entrypoint
Date: 2026-04-14 | Trigger: correction
Wrong:   The new template-admin Playwright specs were valid but relied on operators to remember a single-worker invocation, so the default parallel run introduced cross-test interference against the shared seeded workspace.
Correct: The frontend package now exposes a dedicated `e2e:template-admin` script that runs the admin validation suite with `--workers=1`.
Rule:    If an e2e suite depends on shared seeded state, its canonical command must encode the required execution mode instead of leaving it implicit.
Layer:   process
## Lesson 35 - Browser smoke specs must assert stable contracts, not volatile renderer internals
Date: 2026-04-14 | Trigger: correction
Wrong:   Legacy smoke tests hard-coded assumptions about immediate repeatable-item DOM rendering and inline editor-save state transitions that no longer hold in the current browser editor flow.
Correct: The specs now validate stable user-visible scaffolding and deterministic API-level save rejection semantics, with bundle parsing that tolerates empty draft bodies by falling back to template snapshot blocks.
Rule:    E2E smoke tests should anchor on stable product contracts and deterministic state transitions, not incidental editor implementation details.
Layer:   process
## Lesson 36 - Harness temp files must use platform temp-path APIs
Date: 2026-04-14 | Trigger: correction
Wrong:   The DOCX harness used `$env:TEMP` directly, which resolved to an invalid short path in this Windows profile and caused false failures when checking output file existence.
Correct: The harness now resolves temp storage via `[System.IO.Path]::GetTempPath()` before writing/reading `docgen-harness.docx`.
Rule:    Verification scripts must use runtime platform path APIs for temp files instead of raw env-path assumptions.
Layer:   process
## Lesson 37 - Gotenberg Chromium in Docker needs explicit shared memory sizing
Date: 2026-04-14 | Trigger: correction
Wrong:   The compose `gotenberg` service ran with default shared memory, causing Chromium startup timeouts (`websocket url timeout reached`) and 500s on `/forms/chromium/convert/html`.
Correct: `deploy/compose/docker-compose.yml` now sets `shm_size: 1gb` for `gotenberg`, restoring stable HTML->PDF conversion.
Rule:    Any Chromium-based render service in Docker must reserve adequate `/dev/shm` or parity and conversion tests will fail nondeterministically.
Layer:   infrastructure

## Lesson 38 - E2E smoke commands over shared seeded state must encode worker constraints
Date: 2026-04-14 | Trigger: correction
Wrong:   The default `e2e:smoke` script allowed parallel workers, producing intermittent timeouts and navigation race failures in shared seeded MDDM scenarios.
Correct: `frontend/apps/web/package.json` now defines `e2e:smoke` with `--workers=1`.
Rule:    If e2e scenarios share seeded backend state, enforce serial execution in the canonical command rather than relying on caller discipline.
Layer:   process

## Lesson 39 - Editor-ready state must be reactive for dependent template controls
Date: 2026-04-14 | Trigger: correction
Wrong:   `TemplateEditorView` stored the BlockNote instance only in a ref, so `BlockPalette`/`PropertySidebar` could keep receiving `null` until an unrelated rerender, breaking blank-draft insertion with `Editor nao esta pronto`.
Correct: `TemplateEditorView` now mirrors `onEditorReady` into component state and passes that reactive instance to template controls.
Rule:    Any UI control that depends on async editor initialization must consume reactive ready-state, not a non-rendering ref-only value.
Layer:   application

## Lesson 40 - Template palette must only expose block types registered in editor schema
Date: 2026-04-14 | Trigger: correction
Wrong:   The template palette exposed `field`, but the MDDM BlockNote schema no longer registers a `field` block, causing runtime insertion errors (`Cannot read properties of undefined (reading 'isInGroup')`).
Correct: Palette block rules now include only schema-registered template blocks and insertion context resolves from selected block state.
Rule:    Any authoring palette must be derived from the active editor schema; exposing non-registered block types turns basic insert actions into runtime failures.
Layer:   application

## Lesson 41 - DataTable insertion must seed valid tableContent
Date: 2026-04-14 | Trigger: correction
Wrong:   The template palette inserted `dataTable` with `children: []`, but the registered ProseMirror node requires `tableRow+` content, causing runtime rejection (`Invalid content for node dataTable: <>`).
Correct: New `dataTable` inserts now seed minimal valid `tableContent` with one empty cell row.
Rule:    Any block backed by non-inline ProseMirror content constraints must be inserted with schema-valid initial content, not generic empty children.
Layer:   application

## Lesson 42 - Layout guard tests must assert composition, not only element existence
Date: 2026-04-14 | Trigger: correction
Wrong:   The initial `TemplateEditorView` layout test only checked that test IDs existed, allowing structure regressions (wrong nesting/placement) to pass.
Correct: The layout test now asserts containment relationships and that mocked palette/sidebar/editor mount inside the expected layout containers.
Rule:    UI composition regression tests should verify structural contracts, not just selector presence.
Layer:   process

## Lesson 43 - E2E assertions should avoid CSS-module class selectors
Date: 2026-04-14 | Trigger: correction
Wrong:   The scroll-ownership e2e check queried `main.workspace-main`, which is unstable under CSS modules and silently weakened the regression guard.
Correct: The test now targets stable `data-testid` anchors and validates real scroll behavior on the intended scroll host.
Rule:    Browser regression tests should anchor on stable test selectors and behavior signals, not generated CSS-module class names.
Layer:   process

## Lesson 44 - Layout density browser checks need a seeded editor response
Date: 2026-04-14 | Trigger: correction
Wrong:   The new paper-density e2e check navigated to a template key without stubbing the template GET response, so the editor never mounted and the assertion failed on missing chrome.
Correct: The density test now seeds the template draft response before opening the editor, then measures the rendered paper stack metrics.
Rule:    Browser layout assertions must first establish the editor state they are measuring, or they will fail for setup reasons instead of real regressions.
Layer:   process

## Lesson 45 - Plan-locked UX polish values must be implemented literally
Date: 2026-04-14 | Trigger: correction
Wrong:   Task 3 shipped spacing tweaks that were close to the plan but not exact (`toolbarWrapper`, `editorRoot`, and `--mddm-block-gap` fallback), causing spec-compliance failure after implementation.
Correct: Plan-locked UX polish now uses the exact declared values for toolbar chrome, paper width/padding/border, and block-gap fallback before being considered complete.
Rule:    When a plan specifies concrete CSS constants as acceptance criteria, implement them literally and treat near-equivalents as non-compliant.
Layer:   process

## Lesson 46 - Visual density tests must assert centering, not just spacing caps
Date: 2026-04-14 | Trigger: correction
Wrong:   The paper-density browser test only checked padding/gap upper bounds and did not assert left/right inset symmetry, allowing centering regressions to pass.
Correct: The test now compares left/right insets and enforces a tight symmetry threshold alongside spacing caps.
Rule:    Layout-density checks for centered canvases must include an explicit centering assertion, not only spacing constraints.
Layer:   process

## Lesson 47 - Density checks must exercise desktop rules and cap horizontal inset
Date: 2026-04-14 | Trigger: correction
Wrong:   The density test ran at default viewport (hitting responsive fallback) and did not cap horizontal inset, so oversized centered margins could still pass.
Correct: The test now forces a desktop viewport and asserts maximum left inset while keeping inset symmetry checks.
Rule:    When validating primary desktop layout behavior, browser tests must set a desktop viewport and assert both symmetry and absolute margin bounds.
Layer:   process

## Lesson 48 - UX layout tests must assert real controls, not marker attributes alone
Date: 2026-04-14 | Trigger: correction
Wrong:   The metadata compactness test only asserted `data-density=\"compact\"`, so CTA regressions could pass while the marker remained unchanged.
Correct: The test now validates the real metadata action controls (preview/discard/save/publish) in addition to the compact density marker.
Rule:    UI regression tests should anchor on real user-facing controls and contracts, not only synthetic marker attributes.
Layer:   process

## Lesson 49 - Contrast checks must mount real side panels, not mocks
Date: 2026-04-14 | Trigger: correction
Wrong:   The template editor layout test mocked `BlockPalette` and `PropertySidebar`, which prevented asserting real contrast attributes and panel wiring.
Correct: The test now mounts real side panels and verifies `data-contrast=\"high\"` on both palette and sidebar.
Rule:    UI contrast regressions should be validated against real rendered panels whenever the component contract is visual and attribute-based.
Layer:   process

## Lesson 50 - New editor e2e scenarios must seed their template draft explicitly
Date: 2026-04-14 | Trigger: correction
Wrong:   A new page-stack Playwright scenario opened a template key without stubbing `GET /api/v1/templates/{key}`, causing setup failure before assertions.
Correct: The scenario now seeds the draft response for its key before opening the template editor route.
Rule:    Every isolated template-admin e2e scenario must provide explicit API seed stubs for the template key it opens.
Layer:   process

## Lesson 51 - Non-print layout spacers must be explicitly neutralized in print mode
Date: 2026-04-14 | Trigger: correction
Wrong:   A page-stack spacer pseudo-element used for editor workspace geometry remained active in `@media print`, adding trailing space in print/export output.
Correct: The print stylesheet now disables the pseudo-element (`content: none; width: 0; height: 0`) while keeping the screen layout hook intact.
Rule:    Any screen-only layout spacer introduced in editor chrome must be explicitly disabled in print styles.
Layer:   process

## Lesson 52 - First section spacing must not add top gap on empty page
Date: 2026-04-14 | Trigger: correction
Wrong:   `.bn-block-content[data-content-type="section"]` applied `margin-top` to all section blocks, including the first inserted section.
Correct: Section top spacing now applies only to `.bn-block-content[data-content-type="section"]:not(:first-child)`, keeping the first section flush to the page top.
Rule:    In editor page flow, inter-section spacing must target subsequent blocks only and never push the first content block down.
Layer:   delivery

## Lesson 53 - Browser editor wrappers must not own scroll or page padding
Date: 2026-04-14 | Trigger: correction
Wrong:   `BrowserDocumentEditorView.module.css` applied inner wrapper padding (`.surface`, `.editorViewport`) and clipped viewport overflow, creating an extra scroll/padding layer around the MDDM editor.
Correct: Browser editor wrappers now use zero inner padding and visible viewport overflow so scrolling stays owned by the page/main shell instead of an inner editor div.
Rule:    Scroll ownership for the browser editor must remain on the outer workspace/page container, while editor wrappers stay size-bound and non-scrolling.
Layer:   delivery

## Lesson 54 - Error-state breathing room must be targeted, not global editor padding
Date: 2026-04-14 | Trigger: correction
Wrong:   Removing `.surface` padding to enforce scroll ownership left `.errorBanner` visually edge-crowded against the shell border.
Correct: Keep zero padding/scroll ownership on `.surface` and `.editorViewport`, but add scoped `.errorBanner` margins to preserve alert spacing without restoring editor-wrapper padding.
Rule:    When removing global layout padding for scroll correctness, reintroduce visual spacing only on the specific transient state component that needs it.
Layer:   delivery

## Lesson 55 - Blank template section insertion must replace BlockNote placeholder paragraphs
Date: 2026-04-14 | Trigger: correction
Wrong:   Template palette treated BlockNote's auto-created empty root paragraph as real content, so inserting `section` on a blank draft appended after the placeholder and left a false first-line gap.
Correct: Section creation now treats root documents made only of empty paragraphs as blank and replaces those placeholders with the first section.
Rule:    When BlockNote seeds placeholder paragraphs in an otherwise blank template, root-level insertion logic must collapse them before applying author-visible structure.
Layer:   delivery

## Lesson 56 - Section blocks must not add their own first-block top margin
Date: 2026-04-14 | Trigger: correction
Wrong:   `blocks/Section.module.css` gave every section block `margin-top`, so even the first real section started below the paper's first writable line.
Correct: Section blocks now rely on container-level spacing only, with no intrinsic top margin on the section wrapper.
Rule:    When first-block spacing matters, keep inter-block spacing at the container layer and do not duplicate it inside the block component itself.
Layer:   delivery

## Lesson 57 - Template editor route needs pane-local scroll ownership
Date: 2026-04-14 | Trigger: correction
Wrong:   Template editor moved scrolling to `workspace-content`, which made whole workspace column slide while side chrome stayed visually misaligned.
Correct: Template editor route now clips `workspace-content` and delegates vertical scrolling only to `TemplateEditorView.documentPane`, while inner MDDM wrappers remain non-scrolling.
Rule:    For split-pane authoring layouts, keep scroll ownership on central document pane rather than generic workspace wrappers or nested editor internals.
Layer:   delivery

## Lesson 58 - Structural BlockNote wrappers can add hidden first-line offset
Date: 2026-04-14 | Trigger: correction
Wrong:   Section-gap debugging stopped at block wrapper margins, while BlockNote's `.bn-block-content[data-content-type=\"section\"]` still injected `padding-top: 3px` and pushed the first section below the page start.
Correct: MDDM editor globals now explicitly zero top padding on structural `bn-block-content` wrappers so the first section aligns with the paper padding edge.
Rule:    When pixel alignment matters in BlockNote, inspect wrapper padding as well as custom block CSS because framework chrome can add hidden offsets.
Layer:   delivery

## Lesson 59 - Margin normalizers must use field-specific non-finite fallbacks
Date: 2026-04-14 | Trigger: correction
Wrong:   Generic margin clamp fallback always used top-margin default, causing invalid right/left/bottom inputs to silently become top defaults.
Correct: Clamp/read/write margin helpers now pass explicit fallback per margin field and tests assert non-finite inputs map to each field's own default.
Rule:    Shared normalizers for multi-field layout settings must take field-specific fallback values instead of a single hardcoded default.
Layer:   application

## Lesson 60 - Runtime token wiring needs a draft-meta integration guard
Date: 2026-04-14 | Trigger: correction
Wrong:   Layout tests validated template editor structure but did not assert `draft.meta.page` parsing reached `MDDMEditor`, allowing silent drift in runtime margin wiring.
Correct: `TemplateEditorView.layout` now captures mocked `MDDMEditor` props and asserts parsed page settings are forwarded from draft meta.
Rule:    Any view-level runtime settings path must have an integration-style regression test at the handoff boundary, not only isolated unit tests.
Layer:   application

## Lesson 61 - Draft save payloads must carry current template meta
Date: 2026-04-14 | Trigger: correction
Wrong:   `useTemplateDraft` sent `apiSaveDraft` payloads with only `blocks` and `lockVersion`, so local margin/meta edits could be dropped on save/publish pre-save.
Correct: `saveDraft` and publish pre-save now include `meta: current.meta`, and regression tests assert both paths send the current meta payload.
Rule:    Any save operation that persists editor state must include all mutable draft surfaces (blocks + meta) to avoid silent state loss.
Layer:   application

## Lesson 62 - Paper padding must be driven by page margin tokens
Date: 2026-04-14 | Trigger: correction
Wrong:   `MDDMEditor.module.css` used hardcoded paper padding values, so sidebar margin controls changed state but not visible canvas padding.
Correct: Paper padding now binds to `--mddm-margin-top/right/bottom/left`, and Playwright verifies live padding updates after control changes.
Rule:    Any user-editable layout setting must be wired from tokens/CSS vars to rendered geometry, with a real-browser assertion on computed styles.
Layer:   delivery

## Lesson 63 - Print layout must keep configured page margins
Date: 2026-04-14 | Trigger: correction
Wrong:   `@media print` in `MDDMEditor.module.css` reset `.editorRoot` padding to `0`, dropping user-configured margin settings in print output.
Correct: Print styles now use the same `--mddm-margin-top/right/bottom/left` variables as screen layout so configured margins render consistently.
Rule:    Print styles for editable page geometry must reuse the same margin tokens as screen mode instead of hard resets.
Layer:   delivery

## Lesson 64 - Persistence e2e must re-enter editor via routed navigation, not raw reload
Date: 2026-04-14 | Trigger: correction
Wrong:   The new margin-persistence test used `page.reload()` and expected editor chrome immediately, causing flaky failures where the editor route shell was not re-established deterministically.
Correct: The test now reopens the editor using the same routed navigation flow (`/#/registry` + `openTemplateEditor(...)`) and verifies persisted margin fields after a fresh GET.
Rule:    For SPA editor persistence checks, always re-enter through explicit route navigation helpers instead of relying on raw reload semantics.
Layer:   process
