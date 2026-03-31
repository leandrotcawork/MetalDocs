# Docgen Hybrid Architecture Design

Date: 2026-03-31
Status: Draft for user review
Scope: Replace Carbone as the document generation path with a Go-owned content lifecycle and a separate Node.js docgen renderer.

## Problem

MetalDocs currently couples editable content shape to Carbone template rendering. That works for flat, structured merge data, but it does not scale to richer editorial content or future document layouts that require programmatic DOCX composition.

The architecture must change from:

- Frontend edits content shaped for Carbone
- Go persists document content and calls Carbone
- Carbone renders the final artifact

To:

- Frontend edits canonical content shaped for editing
- Go persists canonical content and owns version rules
- Go assembles a normalized render payload
- `apps/docgen` renders `.docx` from that payload

## Goals

- Remove Carbone from the active document generation path.
- Keep the Go backend as the source of truth for document content, permissions, audit, and version lifecycle.
- Introduce a separate `apps/docgen` service that is stateless and rendering-only.
- Allow the content model to support both structured fields and rich block-based sections.
- Keep export routing and authorization consistent with the current backend-owned flow.

## Non-Goals

- Rewriting historical mock/archive data.
- Changing approval workflow semantics.
- Moving business rules into the frontend or into docgen.
- Making docgen read Postgres directly.
- Keeping the persisted content model optimized for Carbone.

## Architectural Decision

Adopt a hybrid architecture with clear ownership boundaries:

- React frontend owns editing UX only.
- Go `documents` module owns canonical content, editable draft rules, immutable non-draft rules, authorization, audit, events, and export orchestration.
- `apps/docgen` owns DOCX composition only.

The Go backend will assemble the full `DocumentPayload` and send it to docgen. Docgen will not fetch document data on its own.

## Ownership Boundaries

### Frontend

- Render the content builder UI.
- Continue using schema-driven widgets for structured fields.
- Add rich editors for fields/sections that require block-based editing.
- Autosave to Go APIs only.
- Trigger export through Go APIs only.

### Go Backend

- Authenticate and authorize every request.
- Load the active editable document version.
- Enforce version mutability rules.
- Persist canonical content to Postgres.
- Record audit entries for content changes.
- Publish content update events with idempotency keys.
- Project canonical content into the docgen render contract.
- Proxy the docgen export response back to the client.

### apps/docgen

- Expose `POST /generate`.
- Accept a normalized `DocumentPayload`.
- Generate and stream a valid `.docx`.
- Contain no auth logic, no workflow logic, no DB access, and no document lifecycle logic.

## Canonical Content Model

The canonical persisted content remains Go-owned and version-aware.

Recommendation:

- Keep canonical content in `document_versions.native_content`.
- Evolve that JSON shape to support both:
  - structured fields
  - rich block arrays

This avoids splitting the source of truth across multiple persistence models. The persisted content should be shaped for editing and domain ownership, not for Carbone merge convenience.

Rich sections should be represented as typed JSON substructures inside canonical content. The exact editor block schema can evolve independently from the docgen render projection as long as Go can normalize it.

## Version Lifecycle

Version rules:

- `DRAFT` versions are editable in place.
- Non-draft versions are immutable.
- If a user edits an immutable version, Go first creates a new draft version from that version, then applies the edits to the new draft.

Implications:

- Existing or future approved versions are never mutated.
- Export never becomes the source of truth.
- The canonical persisted draft content is always the basis for rendering.

## Editing Flow

1. Frontend loads the current editable content from Go.
2. User edits structured fields and rich content in the same content builder experience.
3. Frontend sends updates to Go.
4. Go checks auth, permission, and version mutability.
5. If the target version is immutable, Go forks a new draft version first.
6. Go persists the canonical content update.
7. Go records audit and publishes domain events.
8. Frontend receives the updated draft/version identity as needed.

## Export Flow

1. Frontend clicks export through a Go-owned route.
2. Go loads the active content snapshot to export.
3. Go assembles a normalized `DocumentPayload`.
4. Go calls `apps/docgen` `POST /generate`.
5. Docgen returns `.docx`.
6. Go streams the result back to the client.

This preserves one backend-owned route for auth, permission checks, logging, and observability.

## API Strategy

Approved direction: Approach 1.

That means:

- Keep one canonical content API in Go.
- Add targeted rich-field endpoints only where the editor UX benefits from narrower updates or autosave behavior.
- Do not let public APIs mirror docgen internals.

Recommended contract split:

- Editing API:
  - optimized for content editing and autosave
  - version-aware
  - Go-owned
- Render contract:
  - optimized for docgen rendering
  - internal service-to-service payload
  - Go assembles it from canonical content

These two contracts are intentionally different.

## Data Contract Layers

### Canonical persisted content

- Stored by Go in Postgres.
- Editing-oriented.
- Source of truth.

### Render payload

- Derived by Go from canonical content.
- Normalized for docgen.
- Not the persistence model.

This separation prevents DOCX layout concerns from leaking back into the database schema and editor UX.

## Service Boundaries and Rationale

The key boundary is:

- Go owns document meaning.
- Docgen owns document appearance.

Why this is the right split:

- Authorization belongs with the system of record.
- Version decisions belong with the system of record.
- Audit and event publication belong with the system of record.
- Rendering complexity belongs in the specialized rendering service.

This boundary also makes future evolution safer:

- New rich sections do not require another renderer rewrite.
- New layout rules do not require database redesign.
- Frontend editing can evolve without coupling directly to DOCX primitives.

## Risks

- Some current content schema paths are still implicitly shaped by Carbone and will require normalization in Go.
- If rich content is introduced ad hoc, profile schemas can drift without a clear normalization layer.
- Export parity with current visual expectations requires dedicated docgen rendering tests.
- If Go leaks render-specific assumptions into canonical storage, the same coupling problem will reappear in a different form.
  Mitigation: the Go application layer must contain an explicit projection function that maps canonical content to DocumentPayload — this function is the firewall between storage shape and render shape, and it must be the only place that knows both.

## Testing Strategy

The implementation plan must include:

- Go unit tests for version mutability and draft forking behavior.
- Go tests for canonical content projection into `DocumentPayload`.
- Contract tests for the Go-to-docgen request/response boundary.
- Docgen tests that validate `.docx` output generation for representative document payloads.
- End-to-end export flow verification through the Go API.

## Docgen Test Harness (Minimal)

This harness is intentionally minimal and standalone. It is not wired into any
existing test runner or Makefile target.

Success criteria:

- `tsc --noEmit` in `apps/docgen` passes with zero errors.
- `node dist/index.js` starts without crashing.
- A single `curl -X POST http://localhost:3001/generate` with a sample payload
  returns a valid `.docx` binary (non-zero bytes) and the response
  `Content-Type` is `application/vnd.openxmlformats-officedocument.wordprocessingml.document`.

This harness exists to validate docgen boot + payload handling only. It is not a
replacement for future unit or integration test suites.

## Deferred Decisions

These are intentionally deferred to implementation planning:

- Exact shape of the canonical rich block schema per profile.
  Constraint: The block schema must be defined as a typed Go struct in the domain layer before any persistence or API code is written. The schema is not inferred from Tiptap's output format — Tiptap's JSON is an editor concern and must be adapted at the frontend boundary before reaching Go.
- Whether any targeted rich-field endpoint is needed immediately, or whether the first slice can ship with canonical whole-content save only.
- Exact proxy route names and transport details between Go and docgen.
- Deployment/container orchestration details for `apps/docgen`.

## Acceptance Direction

This design is successful if MetalDocs reaches the following state:

- Carbone is no longer required for the active export path.
- The frontend edits canonical Go-owned content rather than Carbone-shaped payloads.
- Go remains the single owner of document lifecycle and business rules.
- `apps/docgen` is a pure rendering service.
- Exported `.docx` files are generated through the Go -> docgen flow.
