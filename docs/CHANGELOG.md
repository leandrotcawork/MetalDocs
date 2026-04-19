# MetalDocs Changelog

## 2026-04-18 — docx-editor platform cutover (W5)

### Added
- `@eigenpal/docx-js-editor`-based editor under `/api/v2/*`.
- Content-addressed revision + export pipeline (docgen-v2 + Gotenberg).
- Pessimistic-lock editor sessions + autosave-crash recovery.
- Per-route rate-limit middleware.
- W4 dogfood + W5 post-flip soak evidence artifacts.
- Destructive migration 0113 drops CK5/MDDM/legacy-docgen tables.

### Removed
- CKEditor 5 (ck5-export, ck5-studio apps).
- MDDM block rendering pipeline (mddm_* tables, EtapaBody, RichBlock, RendererPin).
- Legacy docgen client (replaced by docgen-v2 + Gotenberg).
- `METALDOCS_DOCX_V2_ENABLED` feature flag (docx-editor is now the only path).

### Changed
- OpenAPI partials renamed: `documents-v2.yaml` → `documents.yaml`, `templates-v2.yaml` → `templates.yaml`.
- CI workflow renamed: `docx-v2-ci.yml` → `ci.yml`.
- Templates-v2 and documents-v2 nav items always visible (no feature flag guard).
