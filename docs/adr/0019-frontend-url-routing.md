# ADR-0019: Frontend URL Routing (Hash Router)

## Status
Accepted

## Context
Today the MetalDocs frontend navigates only via client state (`activeView` in the UI store).
That means browser back/forward does not reflect app navigation and often exits the domain.
We need URL-driven navigation so history works and deep links can be shared.

The backend currently serves the SPA without guaranteeing a fallback rewrite for arbitrary paths.
A Hash Router avoids server rewrites and works under any hosting setup.

## Decision
- Adopt `HashRouter` for the initial URL routing implementation.
- Map workspace views to URLs (e.g. `/#/documents`, `/#/create`).
- Map Documents Hub internal views to URLs:
  - Overview: `/#/documents` (and `/#/documents/mine`, `/#/documents/recent`)
  - Collection: `/#/documents/area/:areaCode`, `/#/documents/type/:profileCode`
  - Detail: `/#/documents/doc/:documentId`
- Use query params for hub state:
  - `status=all|draft|review|approved`
  - `mode=card|list`
  - `q=<search>`
- Browser history rules:
  - Push for overview -> collection -> detail
  - Replace for tab/mode/search changes within the same collection

## Consequences
- Positive:
  - Browser back/forward works inside the app.
  - Deep links are stable without server configuration.
  - Clear path to future BrowserRouter migration.
- Negative:
  - URLs include `/#/` (less clean).
  - A future migration will need hosting support for SPA rewrites.

## Acceptance test
- `cd frontend/apps/web; npm.cmd run build`
- Manual: `/#/documents` -> collection -> detail, then use browser back/forward without leaving the domain.
