# ADR 0002: Purge editable zones

> **Last verified:** 2026-04-25
> **Status:** Accepted, executed (PR #7)
> **Date:** 2026-04-25
> **Scope:** Removal of "editable zones" from frontend, backend, DB.

## Context

"Editable zones" were a MetalDocs-custom layer above eigenpal — `[[id]]` tokens designating regions reviewers could edit during fill-in/approval. They originated as a workaround for eigenpal's failed T1 (restricted editing via Word `<w:permStart/End>` XML — eigenpal ignores it).

After zones were built, the team realized:
- Eigenpal-spike never tested zones
- Zones add a parallel concept on top of placeholders without filling a real gap (placeholders + read-only doc body = same effect)
- Maintenance burden across frontend (UI, drag-drop, content policy), backend (routes, service, freeze, repository), DB (snapshot column + content table)
- "If eigenpal handles it natively, we shouldn't carry it" (RESULTS.md note)

## Decision

**Remove editable zones end-to-end.** Rely on placeholders only. Document body outside placeholders is a fixed template — non-editable at fill-in time.

## Scope of removal

**Frontend:**
- `frontend/apps/web/src/features/templates/zone-chip.tsx` (deleted)
- `frontend/apps/web/src/features/templates/zone-inspector.tsx` (deleted)
- ZoneChip/ZoneInspector imports + JSX in `TemplateAuthorPage.tsx`
- Zone fields in schema hooks, fill-in loader, editor adapter (`eigenpal-template-mode.ts`)
- Zone tests (round-trip, bookmark spike)

**Backend:**
- Routes: `GET /documents/{id}/zones`, `PUT /documents/{id}/zones/{zid}` in `fillin_handler.go`
- Service: `SetZoneContent`, `LoadZonesSchema` in `fillin_service.go`
- Freeze: `zoneMap` in `freeze_service.go`
- Repository: `UpsertZoneContent`, `LoadZoneContents`, `ZoneContent` in `fillin_repository.go` + `snapshot_repository.go`
- templates_v2: `EditableZones` field in `UpdateSchemasCmd`, repo + domain
- Domain: `ZonesSchemaJSON` field on snapshot

**Database (migration `0157_drop_editable_zones.sql`):**
- DROP COLUMN `editable_zones_schema_snapshot` from `templates_v2_template_version`
- DROP COLUMN `editable_zones_schema` from `templates_v2_template_version` and `documents`
- DROP COLUMN `editable_zones` from `templates_v2_template_version`
- DROP TABLE `document_editable_zone_content`

**Misc:**
- `/zones` permission entries in `permissions.go`
- Zone seeding in `cmd/seed-test-document/main.go`
- Zone steps in e2e spec

## Trade-offs accepted

- **Reviewers can no longer edit document body during approval.** All variability must be expressed as placeholders. If reviewer needs to edit narrative text, treat it as a `text` placeholder (with `maxLength` if needed) — author marks the section as a placeholder during template authoring.
- **Migration:** No legacy data exists in production yet — no migration risk. If zones existed in dev/test data, the DB drop is destructive (`document_editable_zone_content` deleted entirely).

## Consequences

- Smaller surface area: ~500 LOC removed across frontend + backend
- Test plan simplified: no Stage 8 zone fill-in
- Path opens to fully eigenpal-native template authoring (next: ADR 0003)
- Future "rich body editing" use case requires reopening this — design alternative (e.g., per-section "free text" placeholder type) before resurrecting zones

## Verification

- Branch: `chore/purge-editable-zones`
- PR: https://github.com/leandrotcawork/MetalDocs/pull/7
- Build: `go build ./...` clean, `go test ./internal/...` green, `pnpm tsc --noEmit` no new errors
- Smoke: Variables panel shows only PLACEHOLDERS section, no console errors
- Migration applied locally: `0157_drop_editable_zones.sql`

## Cross-refs

- [concepts/placeholders.md](../concepts/placeholders.md)
- [decisions/0001-eigenpal-adoption.md](0001-eigenpal-adoption.md)
- [decisions/0003-token-syntax-migration.md](0003-token-syntax-migration.md)
- Plan: `docs/superpowers/plans/2026-04-25-purge-editable-zones.md`
