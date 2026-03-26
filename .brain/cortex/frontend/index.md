---
id: cortex-frontend-index
title: Frontend Cortex Index
region: cortex/frontend
tags: ["react", "typescript", "vite", "zustand", "content-builder", "editor", "pdf"]
weight: 0.80
created_at: "2026-03-26T10:00:00Z"
updated_at: "2026-03-26T11:00:00Z"
---

# Frontend Cortex

## Stack

- **React 18** + **TypeScript** + **Vite** + **react-router-dom v7**
- **Zustand** for global UI state
- **react-pdf** (pdfjs-dist) for PDF rendering
- **DM Sans** + **DM Mono** fonts

## Key Component Areas

### content-builder/
Overleaf-style live document editor. Main entry: `ContentBuilderView.tsx`.

| File | Purpose |
|------|---------|
| `ContentBuilderView.tsx` | Main editor shell — useReducer state, ResizableSplitPane, auto-save status |
| `useAutoSave.ts` | Debounced (3s) auto-save hook with JSON dedup + AbortController |
| `ResizableSplitPane.tsx` | Pointer-event drag handle, localStorage persistence |
| `preview/PreviewPanel.tsx` | Tabbed: "Ao vivo" HTML preview + "PDF" Carbone tab |
| `preview/DocumentPreviewRenderer.tsx` | Renders profile template from contentDraft |
| `preview/PreviewDocumentPage.tsx` | A4 page wrapper (header, footer, content area) |
| `preview/PreviewFieldRenderer.tsx` | Dispatches to field widget by type |
| `preview/templates/templateRegistry.ts` | Maps profileCode → template component |

### DocumentWorkspaceShell
Main app chrome component. **Uses CSS Modules** (`DocumentWorkspaceShell.module.css`) — not global `styles.css`.

| Responsibility | Detail |
|----------------|--------|
| Top navbar | Brand, search, user menu, refresh, notifications |
| Sidebar | Collapsible (44px ↔ 240px) via `sidebarCollapsed` useState + chevron toggle |
| Layout | `workspace-shell` → `workspace-topbar` + `workspace-layout` (sidebar + main) |
| Nav sections | Grouped nav items, profile accordions, secondary sections |

### create/
Document creation wizard. Contains `PdfPreview.tsx` (multi-page react-pdf viewer).

### dashboard/
Document workspace listing, filters, stats.

## State Patterns

- Complex editors: `useReducer` with typed `BuilderAction` union
- Global UI: Zustand stores
- Form field values: live in `contentDraft` (partial `DocumentContent`)

## CSS

- **content-builder/**: Global `styles.css` — design tokens as CSS custom properties, BEM-like class names
- **DocumentWorkspaceShell**: CSS Modules (`DocumentWorkspaceShell.module.css`) — `styles["class-name"]` pattern
- Key tokens: `--surface`, `--accent`, `--border`, `--shadow`, `--text-muted`

## Key Hooks

- `useAutoSave` — debounced save, JSON dedup, AbortController, returns `{ isSaving, lastSavedPdfUrl, lastSavedAt, error, saveNow }`

## Routing

- `react-router-dom v7` — client-side routing
- Editor route: `/documents/:id/edit`
