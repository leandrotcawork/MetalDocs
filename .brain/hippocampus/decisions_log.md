---
id: hippocampus-decisions-log
title: MetalDocs Architecture Decision Log
region: hippocampus
tags: ["decisions", "adr", "architecture"]
weight: 0.75
created_at: "2026-03-26T10:00:00Z"
updated_at: "2026-03-26T11:00:00Z"
---

# Architecture Decision Log

## ADR-001: Hybrid Preview Approach (2026-03-25)

**Decision**: Use instant client-side HTML preview for real-time feedback + debounced Carbone PDF rendering in background.

**Context**: Server-side Carbone rendering takes 2-5s. Pure client-side rendering can't match Carbone DOCX output.

**Consequence**: Two preview tabs — "Ao vivo" (HTML approximation) and "PDF" (Carbone output). HTML preview labeled "Rascunho visual" to set expectations.

---

## ADR-002: No library for ResizableSplitPane (2026-03-25)

**Decision**: Implement resizable split pane with pure pointer events + CSS (no `react-split-pane` or similar library).

**Context**: Avoid adding a dependency for a simple interaction. Pointer events API handles all drag cases cleanly.

**Consequence**: `ResizableSplitPane.tsx` — 97 lines. localStorage persistence for user preference.

---

## ADR-003: Auto-save with JSON dedup (2026-03-25)

**Decision**: Skip save if `JSON.stringify(contentDraft)` hasn't changed since last successful save.

**Context**: Prevent duplicate server-side Carbone renders when user hasn't changed content.

**Consequence**: `useAutoSave` compares JSON strings as a lightweight content hash. No SHA dependency needed.

---

## ADR-005: ContentBuilderView uses useReducer for all UI state (2026-03-25)

**Decision**: All `ContentBuilderView` state lives in a single `useReducer` (`BuilderState`) with typed `BuilderAction` union — including `contentDraft`, `pdfUrl`, `sidebarCollapsed`, `previewMode`, and auto-save status.

**Context**: Editor has multiple interdependent state slices. `useState` per slice causes stale closure bugs when callbacks read old state. `useReducer` dispatches are stable references.

**Consequence**: Every state change is an explicit typed action. `set_sidebar`, `set_pdf`, `set_preview_mode` etc. Easier to trace state flows and add new slices without refactoring callbacks.

---

## ADR-004: Retractable sidebar state in reducer (2026-03-26)

**Decision**: Add `sidebarCollapsed: boolean` to `BuilderState` (useReducer) rather than a separate useState.

**Context**: `ContentBuilderView` already uses useReducer for content state — keeps all UI state in one place.

**Consequence**: `set_sidebar` action dispatched by toggle button. Collapsed sidebar = 36px strip.
