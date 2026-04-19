# CK5 Plan C — Golden Parity Log

Tracks IR-era goldens that are retired (no CK5 equivalent) and
pending parity tests (blocked on Phase 4 `emitDocxFromExportTree`).

## Retired Goldens

| Golden | Reason |
|--------|--------|
| `asset-collector` | Internal utility test (`collectImageUrls`). Not a document node emitter. No HTML/ExportNode equivalent needed — the new asset resolution path is tested independently in `inline-asset-rewriter.test.ts` and will be tested in the render-docx route tests (Phase 4). |
| `divider` | CK5 has no divider block. The `ExportNode` schema does not include a `divider` kind. HR-style separators in CK5 output are handled via section borders and table styling, not a dedicated node. |
| `field-group` | In CK5, fields are inline `<span class="mddm-field">` nodes emitted as `kind: "field"` in `ExportNode`. There is no field-group container concept in the CK5 HTML output. Field grouping in DOCX is handled at the section/table layout level by the new emitter. |

## Pending Parity Tests (Phase 4 gate)

These tests exist in `src/__tests__/golden-parity.test.ts` as `test.todo` entries.
They will be implemented after `emitDocxFromExportTree(nodes: ExportNode[])` is available (Task 11).

| Golden | HTML Fixture | Status |
|--------|-------------|--------|
| `paragraph` | any `<p>` HTML | pending |
| `heading` | `__fixtures__/nested-formatting.html` | pending |
| `inline-content` | `__fixtures__/nested-formatting.html` | pending |
| `section` | `__fixtures__/section-with-fields.html` | pending |
| `field` | `__fixtures__/section-with-fields.html` | pending |
| `repeatable` | `__fixtures__/repeatable.html` | pending |
| `repeatable-item` | `__fixtures__/repeatable.html` | pending |
| `data-table (fixed)` | `__fixtures__/table-fixed.html` | pending |
| `data-table (dynamic)` | `__fixtures__/table-dynamic.html` | pending |
| `rich-block` | `__fixtures__/rich-block.html` | pending |
| `bullet-list` | needs `__fixtures__/bullet-list.html` | pending |
| `numbered-list` | needs `__fixtures__/numbered-list.html` | pending |
| `blockquote/quote` | needs `__fixtures__/blockquote.html` | pending |
| `image` | needs `__fixtures__/image.html` | pending |
| `emitter (integration)` | needs full-document fixture | pending |

## Implementation Notes

When implementing pending tests (Phase 4+):
- Use `Packer.toString(doc)` to get XML string for comparison
- Scrub RSID and timestamp fields before comparing: `/rsid[A-Z0-9]+="[^"]+"/g`
- Compare per-section: don't assert exact byte equality (timestamps differ)
- Create missing HTML fixtures in `apps/ck5-export/src/__fixtures__/`
