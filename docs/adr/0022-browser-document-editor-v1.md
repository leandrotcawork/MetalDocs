# ADR-0022: Browser Document Editor V1

## Status
Accepted

## Context
The governed-canvas pilot introduced template snapshots and draft-token concurrency on top of the legacy split-pane content builder and MetalDocs native content envelope. The approved browser-editor v1 replaces that pilot path with a single browser-native editing surface backed by versioned templates and HTML body persistence.

The first delivery must keep scope narrow:
- self-hosted browser editor only
- canonical template and draft bodies persisted as HTML
- DOCX and PDF remain server-owned derived artifacts
- additive contracts alongside the existing native editor flow until cutover is verified

## Decision
Adopt self-hosted CKEditor 5 for browser document editing in v1.

Persist browser template snapshots and draft revision bodies as HTML. The backend remains the source of truth for template assignment, template version snapshotting, and draft concurrency through `draftToken`.

Keep DOCX and PDF generation server-owned. The browser editor does not own export conversion; it only edits canonical HTML content that the server later transforms into DOCX and PDF artifacts.

Expose additive API contracts for:
- `GET /documents/{documentId}/browser-editor-bundle`
- `POST /documents/{documentId}/content/browser`

## Consequences
- Template HTML and revision HTML become first-class persisted content.
- The governed-canvas pilot path becomes transitional legacy behavior while browser-editor routes are introduced in parallel.
- The frontend must bootstrap CKEditor 5 and a license-key environment variable before UI cutover work can proceed.
- Export fidelity remains centralized on the server, which avoids client-side DOCX/PDF divergence.

## Acceptance test
```bash
cd frontend/apps/web
npm.cmd install
npm.cmd run build
```
