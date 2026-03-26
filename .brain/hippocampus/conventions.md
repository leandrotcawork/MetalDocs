---
id: hippocampus-conventions
title: MetalDocs Conventions
region: hippocampus
tags: ["conventions", "naming", "style", "patterns"]
weight: 0.95
created_at: "2026-03-26T10:00:00Z"
updated_at: "2026-03-26T10:00:00Z"
---

# MetalDocs Conventions

## Frontend

### File Naming
- React components: `PascalCase.tsx` (e.g., `ContentBuilderView.tsx`)
- Hooks: `camelCase.ts` with `use` prefix (e.g., `useAutoSave.ts`)
- Types/interfaces: `camelCase.ts` (e.g., `contentSchemaTypes.ts`)
- Feature directories: `kebab-case/` (e.g., `content-builder/`)

### Component Patterns
- Named exports only (no default exports)
- Props type defined inline as `type ComponentNameProps = { ... }`
- State managed via `useReducer` for complex components (`ContentBuilderView`)
- Simple local state with `useState`

### CSS Conventions
- Single global `styles.css` — NO CSS modules, NO Tailwind
- Design tokens as CSS custom properties: `--surface`, `--accent`, `--border`, `--shadow`
- BEM-like class names: `.content-builder-sections-nav.is-collapsed`
- State modifier classes: `.is-active`, `.is-collapsed`, `.is-dirty`
- Portuguese UI text throughout (Brazilian Portuguese)

### API Layer
- All API calls go through `src/api/` module functions
- Pattern: `api.saveDocumentContentNative(documentId, payload)`
- No direct fetch calls in components

## Backend (Go)

### Package Layout
- `internal/modules/<domain>/` — domain handlers, services, models
- `internal/platform/` — shared db, storage, middleware
- `api/` — HTTP route registration

### Naming
- Handler functions: `HandleVerbNoun` (e.g., `HandleCreateDocument`)
- SQL files: numbered `NNNN_description.sql`
- No ORM — raw SQL via pgx

## Database

- PostgreSQL only
- Migrations: sequential numbered SQL files in `migrations/`
- No rollback scripts — forward-only migrations

## Git

- Commit message format: `type(scope): description`
  - Types: `feat`, `fix`, `refactor`, `docs`, `chore`
  - Scopes: `frontend`, `backend`, `frontend-editor`, `frontend-dashboard`, `infra`
- Branch: `main` only (no feature branches observed)

## Language

- UI text: Brazilian Portuguese (`Salvar`, `Editando...`, `Carregando PDF...`)
- Code: English identifiers, English comments
- Error messages: Brazilian Portuguese in UI, English in logs
