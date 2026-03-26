---
id: hippocampus-architecture
title: MetalDocs Architecture
region: hippocampus
tags: ["architecture", "stack", "system-design"]
weight: 0.95
created_at: "2026-03-26T10:00:00Z"
updated_at: "2026-03-26T10:00:00Z"
---

# MetalDocs Architecture

## System Overview

MetalDocs is a web-based ISO document management system for metalworking companies. It allows users to create, edit, version, and distribute controlled documents (Procedures, Work Instructions, Records, Forms) using a form-driven editor that renders to PDF via Carbone.

## Stack

| Layer | Technology |
|-------|-----------|
| Backend | Go 1.24, net/http, pgx/v5 |
| Frontend | React 18, TypeScript, Vite, Zustand, react-router-dom v7 |
| Database | PostgreSQL (pgx driver, SQL migrations) |
| Document Rendering | Carbone (DOCX templates → PDF via Node.js server) |
| Object Storage | MinIO (S3-compatible, PDF storage) |
| Infra | Docker Compose, nginx reverse proxy |
| PDF Preview | react-pdf (pdfjs-dist) |

## Module Layout

```
metaldocs/
├── api/              # HTTP handlers (Go)
├── internal/
│   ├── modules/      # Domain modules (documents, iam, audit)
│   └── platform/     # Shared infra (db, storage, middleware)
├── migrations/       # PostgreSQL migrations (numbered SQL files)
├── frontend/apps/web/
│   └── src/
│       ├── components/
│       │   ├── content-builder/   # Live document editor (Overleaf-style)
│       │   ├── create/            # Document creation flow
│       │   └── dashboard/         # Document workspace
│       ├── pages/
│       └── styles.css             # Single global stylesheet (design tokens)
├── carbone/          # Carbone server + DOCX templates
├── deploy/           # Docker Compose, nginx config
└── scripts/          # DB build scripts, brain utilities
```

## Document Profiles

Each profile maps to a Carbone DOCX template:
- **PO** (Procedimento Operacional) → `template-po.docx`
- **IT** (Instrução de Trabalho) → `template-it.docx`
- **RG** (Registro) → `template-rg.docx`
- **FM** (Formulário) → `template-fm.docx`

## Key Flows

### Document Editing
1. User opens document in `ContentBuilderView` (split-pane editor)
2. Form fields driven by profile JSON schema
3. Auto-save fires after 3s debounce via `useAutoSave`
4. Save calls `api.saveDocumentContentNative` → triggers Carbone render
5. PDF URL returned → preview updates in right pane
6. Live HTML preview updates instantly on every keystroke

### Authentication
- IAM module with RBAC (roles + permissions)
- Audit events tracked per user action

## Architectural Decisions

- Single global `styles.css` — no CSS modules, no Tailwind
- Zustand for local UI state in complex components
- No ORM — raw SQL via pgx
- Carbone chosen for DOCX template fidelity (no code-level PDF generation)
