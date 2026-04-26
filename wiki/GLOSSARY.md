# Glossary

> **Last verified:** 2026-04-25
> **Scope:** Terms used across MetalDocs codebase, docs, and PRs.

## A

**ADR** — Architecture Decision Record. Short doc capturing a decision + reasoning. Lives in `wiki/decisions/`.

**Approval Route** — Configurable sequence of stages a template/document goes through before publish. Stored in `approval_routes` + `approval_route_stages` tables.

**Area** — Organizational unit (department-like). Used for scoping permissions. See `concepts/area-membership` (TBD).

## C

**Capability** — Permission unit (e.g., `doc:edit:draft`, `doc:view:published`). Granted per role + area. See `modules/iam-rbac.md`.

**CD / Controlled Document** — A document instance bound to a profile + identified by an auto-generated code (e.g., `E2E-0001`). Lives in `controlled_documents` table.

**Compose** — `docker compose -f deploy/compose/docker-compose.yml` — local dev stack (Postgres, MinIO, Gotenberg, api, web).

**Content Hash** — SHA256 of frozen DOCX body. Stored on document at freeze time. Immutable proof of artifact identity.

## D

**Docxtemplater** — Template substitution library. Native syntax: `{var}`, `{#section}`, `{^inverted}`, `{@raw}`. Used by both eigenpal client-side and MetalDocs server fanout.

**Draft** — Initial state of a template version or document instance. Editable. Status enum value.

## E

**Eigenpal** — `@eigenpal/docx-js-editor` — DOCX WYSIWYG editor library MetalDocs uses. ProseMirror-based. See `modules/editor-ui-eigenpal.md`.

**Editable Zone** — DEPRECATED. Removed 2026-04-25 in commit `chore/purge-editable-zones`. See `decisions/0002-zone-purge.md`.

## F

**Fanout** — Server-side service that takes a frozen template + placeholder values and renders the final DOCX + PDF. Lives in `internal/modules/render/fanout/`.

**Fill-in** — Process of supplying placeholder values to a document instance during draft state.

**Freeze** — Operation that locks a document at approval time: computes content/values/schema hashes, persists immutable snapshot, triggers fanout. See `workflows/freeze-and-fanout.md`.

## G

**Gotenberg** — Open-source DOCX → PDF conversion service. Runs as compose service. Called by fanout.

## I

**ISO Segregation** — Workflow rule: the user who submits cannot approve. Enforced by approval module. Error: `templates_v2: iso_segregation_violation`.

## M

**MinIO** — S3-compatible object storage. Stores template DOCX bodies, document final artifacts. Compose service.

## P

**Placeholder** — Variable in a template DOCX that gets substituted at fill-in time. Currently MetalDocs uses `{{uuid}}` token format (legacy); eigenpal-native is `{name}`. See `concepts/placeholders.md`.

**ProseMirror** — Rich-text editor framework eigenpal is built on. We rarely interact with it directly — eigenpal abstracts it.

**Profile / Document Profile** — Type of controlled document (e.g., "Quality Manual", "SOP"). Binds to a default template version + sequence counter for code generation.

## S

**Schema** — JSON definition of a template's variables (placeholders). Stored on `templates_v2_template_version.placeholder_schema_snapshot`.

**Snapshot** — Immutable copy of template schema at the moment a document was created. Lives on `documents.placeholder_schema_snapshot`. Decouples document from later template edits.

## T

**templatePlugin** — Eigenpal's native ProseMirror plugin for placeholder detection + highlighting. Currently wired but **not effective** because token format mismatch. See `concepts/placeholders.md`.

**Tenant** — Multi-tenancy boundary. Currently single dev tenant `ffffffff-...`. Stored on most rows.

## V

**Values Hash** — SHA256 of all placeholder values at freeze time. Together with content_hash + schema_hash, proves what was rendered.

## Z

**Zone** — DEPRECATED (see Editable Zone).
