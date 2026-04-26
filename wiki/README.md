# MetalDocs Wiki

> **Last verified:** 2026-04-26
> **Purpose:** Single source of truth for codebase knowledge. Read this first — drill into folders only after.

## How to use this wiki

- **Humans:** Browse by folder.
- **AI agents:** Read this index, then `Glob wiki/**/*.md` to discover. Each doc has `Last verified:` + `Key files:` block at the top — use those file:line anchors instead of re-grepping.
- **Drift policy:** When changing code referenced by a doc, update the doc's `Last verified` stamp. Stale stamps = trust nothing in that doc until verified.

---

## Index

### Vision
- [vision/product-vision.md](vision/product-vision.md) — what MetalDocs is, problem it solves
- [vision/target-users.md](vision/target-users.md) — quality engineers, ISO-bound orgs, document control roles

### Architecture
- [architecture/system-overview.md](architecture/system-overview.md) — services, ports, data flow at a glance
- [architecture/data-model.md](architecture/data-model.md) — Postgres tables, key relationships
- [architecture/tech-stack.md](architecture/tech-stack.md) — Go, React, Postgres, MinIO, Gotenberg, eigenpal
- [architecture/deployment.md](architecture/deployment.md) — Docker compose, env vars, dev setup

### Modules (one per backend module / frontend feature)
- [modules/templates-v2.md](modules/templates-v2.md) — template authoring, schemas, versioning, approval
- [modules/documents-v2.md](modules/documents-v2.md) — document instances, fill-in, freeze, view
- [modules/taxonomy.md](modules/taxonomy.md) — document profiles, areas, departments, subjects
- [modules/approval.md](modules/approval.md) — approval routes, signoffs, ISO segregation
- [modules/render-fanout.md](modules/render-fanout.md) — DOCX → PDF rendering, substitution engine
- [modules/iam-rbac.md](modules/iam-rbac.md) — capabilities, role checks, area-scoped permissions
- [modules/editor-ui-eigenpal.md](modules/editor-ui-eigenpal.md) — eigenpal integration layer, plugin wiring (Last verified: 2026-04-26)

### Concepts (cross-cutting)
- [concepts/placeholders.md](concepts/placeholders.md) — **CRITICAL:** fixed 7-token catalog, substitution at freeze (Last verified: 2026-04-26)
- [concepts/token-syntax.md](concepts/token-syntax.md) — `{name}` vs `{{uuid}}` — why it matters
- [concepts/controlled-documents.md](concepts/controlled-documents.md) — code generation, profile binding, sequence counters
- [concepts/iso-segregation.md](concepts/iso-segregation.md) — why submitter cannot approve own submit
- [concepts/freeze-and-hashing.md](concepts/freeze-and-hashing.md) — content_hash, values_hash, schema_hash, immutability

### Workflows (end-to-end flows)
- [workflows/template-authoring.md](workflows/template-authoring.md) — create → edit schema → submit → approve
- [workflows/document-fillin.md](workflows/document-fillin.md) — pick CD → wizard → editor → fill placeholders
- [workflows/approval.md](workflows/approval.md) — submit, route, signoffs, idempotency
- [workflows/freeze-and-fanout.md](workflows/freeze-and-fanout.md) — approve → freeze → fanout → PDF artifact

### Decisions (ADRs)
- [decisions/0001-eigenpal-adoption.md](decisions/0001-eigenpal-adoption.md) — why we picked eigenpal over CKEditor/BlockNote
- [decisions/0002-zone-purge.md](decisions/0002-zone-purge.md) — why we removed editable zones (2026-04-25)
- [decisions/0003-token-syntax-migration.md](decisions/0003-token-syntax-migration.md) — plan to move from `{{uuid}}` → `{name}`
- [decisions/0008-placeholder-fixed-catalog.md](decisions/0008-placeholder-fixed-catalog.md) — replace user-fill placeholders with fixed 7-token computed catalog (2026-04-26)

### References
- [references/eigenpal-spike.md](references/eigenpal-spike.md) — pointer to spike repo + key findings (T1–T8)
- [references/environment-setup.md](references/environment-setup.md) — local dev: compose, migrations, seed
- [references/how-to-run-tests.md](references/how-to-run-tests.md) — Go tests, frontend vitest, e2e playwright
- [references/local-dev-startup.md](references/local-dev-startup.md) — **START HERE** — PS script, port, credentials, common mistakes
- [references/local-dev-credentials.md](references/local-dev-credentials.md) — admin login details, DB access

### Glossary
- [GLOSSARY.md](GLOSSARY.md) — placeholder, zone (deprecated), fanout, freeze, eigenpal, controlled doc, profile, etc.

---

## Conventions

**Filename:** kebab-case, descriptive. ADRs prefix with 4-digit number.

**File header (every doc):**
```markdown
# Title

> **Last verified:** YYYY-MM-DD
> **Scope:** what this covers
> **Out of scope:** what it doesn't (link to where it does)
> **Key files:**
> - `path/to/file.go:42` — anchor description
> - `path/to/other.tsx:115` — anchor description
```

**Cross-refs:** Use full path + line numbers. Example: `internal/modules/templates_v2/application/schema.go:42`.

**Length:** Hard cap ~300 lines. Split if longer.

**Code blocks:** Always with language tag for highlighting.
