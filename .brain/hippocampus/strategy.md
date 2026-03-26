---
id: hippocampus-strategy
title: MetalDocs Product Strategy
region: hippocampus
tags: ["strategy", "product", "goals"]
weight: 0.85
created_at: "2026-03-26T10:00:00Z"
updated_at: "2026-03-26T10:00:00Z"
---

# MetalDocs Product Strategy

## Product Goal

MetalDocs is a controlled document management system targeting metalworking / industrial companies that must maintain ISO 9001 compliance. The product replaces Word/Google Docs-based workflows with a structured, audit-ready document editor.

## Core Value Propositions

1. **Form-driven document creation** — Profiles (PO, IT, RG, FM) enforce structure without requiring Word skills
2. **Live editing experience** — Overleaf-style split-pane: instant HTML preview + debounced Carbone PDF rendering
3. **Version control** — Every save creates a traceable revision with audit trail
4. **Controlled distribution** — Role-based access, document approval workflows

## Current Development Focus (Q1 2026)

- **Live document editor**: Overleaf-style ContentBuilderView with ResizableSplitPane, PreviewPanel (live HTML + PDF tabs), useAutoSave
- **Dashboard UX**: Premium layout, workspace shells, document management
- **Retractable sidebar**: Navigation sidebar collapses to 36px strip for more editor space

## Technical Bets

- Carbone for PDF rendering (DOCX template fidelity over programmatic PDF)
- React + Vite SPA with server-side API (no Next.js/SSR complexity)
- PostgreSQL raw SQL (no ORM — full control over queries)
- MinIO for PDF storage (S3-compatible, self-hosted)

## Design Principles

- Brazilian Portuguese UI throughout
- Single global stylesheet (design tokens, no CSS-in-JS)
- Named exports, no default exports in React
- Keep editor feel like a document tool, not a form
