# ADR-0014: Document Profile Bundle Endpoint

## Status
Accepted

## Context
The document create flow needs profile, schema, governance, and taxonomy data. Today the frontend performs multiple round-trips when the user switches a profile. This increases latency and makes the UI feel sluggish, especially in local or constrained environments.

## Decision
Add a dedicated endpoint to fetch a profile bundle in a single request:
`GET /document-profiles/{profileCode}/bundle`

The response includes:
- `profile` (document profile metadata)
- `schema` (active profile schema)
- `governance` (active governance rules)
- `taxonomy` (process areas, document departments, and subjects)

The endpoint is additive and does not replace existing APIs.

## Consequences
- Positive:
  - Reduces round-trips for profile selection and initial load.
  - Enables faster UI updates with fewer network calls.
  - Keeps existing endpoints for backward compatibility.
- Negative:
  - Larger payload per request.
  - Requires keeping bundle response in sync with individual endpoints.

## Alternatives Considered
- Option A: Keep individual endpoints and rely on frontend caching only.
- Option B: Preload everything at bootstrap (increases initial payload and cold start time).
