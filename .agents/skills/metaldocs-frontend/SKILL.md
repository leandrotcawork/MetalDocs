---
name: metaldocs-frontend
description: Implement any MetalDocs frontend feature following professional architecture. Covers where state lives (Zustand stores), how API calls are structured (domain-split), how components are scoped (CSS Modules), and how to keep components focused (no God Components, no prop drilling). Use whenever implementing any React feature, page, or component.
---

# MetalDocs Frontend

## Before writing any code
Read `tasks/lessons.md`. Apply every lesson in this task.

## Where does this code belong?

Before creating any file, answer:

**Is it state shared across features?**
→ `src/store/<domain>.store.ts` — Zustand slice

**Is it an API call?**
→ `src/api/<domain>.ts` — one file per backend domain

**Is it a full page/view?**
→ `src/features/<domain>/<FeatureName>.tsx` + `<FeatureName>.module.css`

**Is it a hook that composes store + API?**
→ `src/features/<domain>/use<FeatureName>.ts`

**Is it a primitive used in 3+ features?**
→ `src/components/ui/<Name>/<Name>.tsx` + `<Name>.module.css`

**Is it a base style (token, reset, body)?**
→ `src/styles/tokens.css` or `src/styles/base.css`

## State — Zustand, one store per domain

```ts
// src/store/documents.store.ts
import { create } from 'zustand'

interface DocumentsStore {
  documents: SearchDocumentItem[]
  selectedDocument: DocumentListItem | null
  isLoading: boolean
  setDocuments: (docs: SearchDocumentItem[]) => void
  setSelectedDocument: (doc: DocumentListItem | null) => void
  setLoading: (v: boolean) => void
}

export const useDocumentsStore = create<DocumentsStore>((set) => ({
  documents: [],
  selectedDocument: null,
  isLoading: false,
  setDocuments: (documents) => set({ documents }),
  setSelectedDocument: (selectedDocument) => set({ selectedDocument }),
  setLoading: (isLoading) => set({ isLoading }),
}))
```

Feature components read from their store directly — no prop drilling:
```tsx
function DocumentsWorkspace() {
  const { documents, selectedDocument, isLoading } = useDocumentsStore()
  // no props needed for data this component owns
}
```

**Rule:** If a component receives more than 5 props, ask whether some should come from a store instead.

## API — one file per backend domain

```ts
// src/api/client.ts — transport only, no business logic
export async function request<T>(path: string, init?: RequestInit): Promise<T>
export async function requestBlob(path: string, init?: RequestInit): Promise<Blob>

// src/api/documents.ts — endpoints + normalizers for this domain only
import { request } from './client'
export async function searchDocuments(params: URLSearchParams) {
  const res = await request<{ items: SearchDocumentItem[] }>(`/search/documents?${params}`)
  return { items: res.items.map(normalizeSearchDocument) }
}
function normalizeSearchDocument(v: SearchDocumentItem): SearchDocumentItem { ... }
```

Domain split: `auth.ts` | `documents.ts` | `iam.ts` | `notifications.ts` | `registry.ts` | `workflow.ts`

## Components — CSS Modules, no global CSS

```tsx
// src/features/documents/DocumentsWorkspace.tsx
import styles from './DocumentsWorkspace.module.css'

export function DocumentsWorkspace() {
  const { documents } = useDocumentsStore()
  return <div className={styles.shell}>...</div>
}
```

```css
/* DocumentsWorkspace.module.css */
.shell { display: grid; gap: 0; }
.toolbar { background: var(--surface); border-bottom: 1px solid var(--border); }
```

**Rules:**
- Every component has its own `.module.css` — never add to `styles.css`
- Only `styles/tokens.css`, `styles/reset.css`, `styles/base.css` are global
- Always `var(--token)` — never hardcoded hex, px font-size, or spacing
- No `style={{ }}` for layout or typography

## Component size rule

A component is too big if it:
- Has more than 3 `useState` calls → extract a custom hook or split the component
- Has more than 5 props → some data should come from a store
- Handles fetch + renders UI → split into a hook + a presentational component
- Is longer than ~150 lines → look for a natural split point

## App.tsx role — router only

App.tsx reads from stores and renders features. No useState for domain data. No fetch calls. No handlers.

```tsx
function App() {
  const { authState, user } = useAuthStore()
  if (authState === 'loading') return <LoadingScreen />
  if (!user) return <AuthShell />
  if (user.mustChangePassword) return <PasswordChangePanel />
  return <WorkspaceShell />
}
```

## After task
1. `tsc --noEmit` passes
2. No console errors in browser
3. Real data visible
4. `git commit -m "feat(frontend): <what>"`

## References
- `references/css-tokens.md` — full design token list
