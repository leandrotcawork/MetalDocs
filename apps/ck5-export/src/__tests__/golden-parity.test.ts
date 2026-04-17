/**
 * Dual-path golden parity: IR-era emitters vs CK5-HTML-era emitters.
 *
 * Status: pending Phase 4 (`emitDocxFromExportTree` function does not exist yet).
 * Active tests will be filled in when Task 11 (POST /render/docx) lands.
 *
 * Retired goldens are documented in:
 *   docs/superpowers/plans/2026-04-16-ck5-plan-c-production-readiness-parity-log.md
 */
import { describe, it } from "vitest"

describe("golden-parity: IR-era vs CK5-HTML-era DOCX output", () => {
  // ---------------------------------------------------------------------------
  // RETIRED — no CK5/ExportNode equivalent exists
  // ---------------------------------------------------------------------------

  it.todo(
    "retired: asset-collector — internal utility, not a document node (see parity log)",
  )

  it.todo(
    "retired: divider — HR node removed from ExportNode schema; CK5 has no divider block (see parity log)",
  )

  it.todo(
    "retired: field-group — CK5 encodes fields as inline <span class=mddm-field>; field-group container not emitted (see parity log)",
  )

  // ---------------------------------------------------------------------------
  // PENDING Phase 4 — needs emitDocxFromExportTree(ExportNode[])
  // HTML fixtures exist or are noted below
  // ---------------------------------------------------------------------------

  it.todo(
    "parity: paragraph — fixture: any <p> HTML; compare mddmToDocx(IR) vs emitDocxFromExportTree(htmlToExportTree(html))",
  )

  it.todo(
    "parity: heading — fixture: nested-formatting.html; compare mddmToDocx(IR) vs emitDocxFromExportTree(htmlToExportTree(html))",
  )

  it.todo(
    "parity: inline-content (bold/italic/underline/strike) — fixture: nested-formatting.html",
  )

  it.todo(
    "parity: section — fixture: section-with-fields.html; compare mddmToDocx(IR) vs emitDocxFromExportTree(htmlToExportTree(html))",
  )

  it.todo(
    "parity: field — fixture: section-with-fields.html (mddm-field spans); compare mddmToDocx(IR) vs emitDocxFromExportTree(htmlToExportTree(html))",
  )

  it.todo(
    "parity: repeatable — fixture: repeatable.html (<ol class=mddm-repeatable>); compare mddmToDocx(IR) vs emitDocxFromExportTree(htmlToExportTree(html))",
  )

  it.todo(
    "parity: repeatable-item — fixture: repeatable.html (li inside mddm-repeatable); compare mddmToDocx(IR) vs emitDocxFromExportTree(htmlToExportTree(html))",
  )

  it.todo(
    "parity: data-table (fixed) — fixture: table-fixed.html; compare mddmToDocx(IR) vs emitDocxFromExportTree(htmlToExportTree(html))",
  )

  it.todo(
    "parity: data-table (dynamic) — fixture: table-dynamic.html; compare mddmToDocx(IR) vs emitDocxFromExportTree(htmlToExportTree(html))",
  )

  it.todo(
    "parity: rich-block — fixture: rich-block.html; compare mddmToDocx(IR) vs emitDocxFromExportTree(htmlToExportTree(html))",
  )

  it.todo(
    "parity: bullet-list — fixture: <ul><li> HTML (needs creation); compare mddmToDocx(IR) vs emitDocxFromExportTree(htmlToExportTree(html))",
  )

  it.todo(
    "parity: numbered-list — fixture: <ol><li> HTML (needs creation, non-repeatable); compare mddmToDocx(IR) vs emitDocxFromExportTree(htmlToExportTree(html))",
  )

  it.todo(
    "parity: blockquote/quote — fixture: <blockquote> HTML (needs creation); compare mddmToDocx(IR) vs emitDocxFromExportTree(htmlToExportTree(html))",
  )

  it.todo(
    "parity: image — fixture: <img> HTML with resolved asset (needs creation); compare mddmToDocx(IR) vs emitDocxFromExportTree(htmlToExportTree(html))",
  )

  it.todo(
    "parity: emitter (integration) — fixture: full document HTML covering all node types; compare mddmToDocx(IR) vs emitDocxFromExportTree(htmlToExportTree(html))",
  )
})
