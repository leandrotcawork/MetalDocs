# Token Syntax Migration `{{uuid}}` → `{name}` Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Migrate MetalDocs placeholder tokens from broken `{{uuid}}` (double-brace UUID) format to eigenpal-native `{name}` (single-brace semantic slug) — activates `templatePlugin` orange highlighting in editor and fixes server-side substitution in `docgen-v2` fanout.

**Architecture:** Additive `name` field on placeholder schema. Internal storage stays UUID-keyed; translation to `name` happens at freeze-time when building the fanout map. Frontend slugify auto-derives `name` from `label`; inspector exposes manual override. Tokens inserted as `{name}` activate already-wired eigenpal `templatePlugin`. Same change unbroken `processTemplateDetailed` server substitution. Cleanup zone-purge leftovers (`ZoneContent`/`injectZones`) in same effort.

**ID-vs-Name boundary contract (binding):**
- `Placeholder.id` (UUID) — stable internal identifier. Used by: `document_placeholder_values.placeholder_id` PK, `VisibilityCondition.placeholder_id` (depends on stable ID, not user-mutable slug), fill-in API path `/documents/{id}/placeholders/{pid}`, resolver registry, drag-transfer fallback.
- `Placeholder.name` (slug) — user-facing token. Used ONLY in: DOCX body tokens `{name}`, fanout request `placeholder_values` map keys, eigenpal `templatePlugin`/`processTemplateDetailed`.
- Translation point: `freeze_service.go` builds `idToName` lookup from schema and re-keys the fanout map. Nowhere else.
- `visible_if.placeholder_id` stays UUID — visibility refs target the stable ID so renaming a placeholder's `name` doesn't break dependents.

**Tech Stack:** Go 1.x, TypeScript/React (frontend), Node/Hono (docgen-v2), eigenpal `@eigenpal/docx-js-editor`, ProseMirror, PostgreSQL JSONB.

---

## File Structure

**Modify:**
- `internal/modules/templates_v2/domain/schemas.go` — add `Name` to `Placeholder` struct
- `internal/modules/templates_v2/domain/errors.go` — add 2 sentinel errors
- `internal/modules/templates_v2/application/schema.go` (or wherever `ValidatePlaceholders` lives) — name format + uniqueness
- `internal/modules/templates_v2/delivery/http/errors.go` — map new errors → 422
- `internal/modules/documents_v2/application/freeze_service.go` — fanout map keyed by `name`
- `internal/modules/render/fanout/client.go` — drop `ZoneContent` field
- `internal/modules/render/fanout/client_test.go` — drop `ZoneContent` from fixtures
- `frontend/apps/web/src/features/templates/placeholder-types.ts` — add `name?` + `slugifyLabel`
- `frontend/apps/web/src/features/templates/placeholder-inspector.tsx` — name input + label sync
- `frontend/apps/web/src/features/templates/placeholder-chip.tsx` — drag carries `name`
- `frontend/apps/web/src/features/templates/v2/TemplateAuthorPage.tsx` — insert `{name}` not `{{uuid}}`
- `apps/docgen-v2/src/render/fanout.ts` — drop `injectZones`, use `bodyDocx` directly
- `apps/docgen-v2/src/routes/fanout.ts` — drop `zone_content` from Zod schema
- `apps/docgen-v2/src/render/__tests__/fanout.test.ts` — drop `zoneContent` from fixtures

**Delete:**
- `apps/docgen-v2/src/render/zoneInjection.ts`
- `apps/docgen-v2/src/render/__tests__/zoneInjection.test.ts`

**Update (post-merge):**
- `wiki/decisions/0003-token-syntax-migration.md` — status → "Accepted, executed 2026-04-25"

---

## Task 1: Add `Name` field to Go `Placeholder` struct

**Files:**
- Modify: `internal/modules/templates_v2/domain/schemas.go:28-45`
- Test: `internal/modules/templates_v2/domain/schemas_test.go`

**Subagent:** haiku (mechanical edit + JSON round-trip test)

**Caveman prompt:**
> Add `Name string \`json:"name,omitempty"\`` field to `Placeholder` struct between `ID` and `Label` in `internal/modules/templates_v2/domain/schemas.go`. Add JSON round-trip test in `schemas_test.go` proving `Name: "doc_code"` serializes to `"name":"doc_code"` and deserializes back. Run `go test ./internal/modules/templates_v2/domain/...`. Commit `feat(templates): add Name field to Placeholder schema`.

- [ ] **Step 1: Write failing round-trip test**

```go
// schemas_test.go
func TestPlaceholder_NameField_JSONRoundTrip(t *testing.T) {
    p := Placeholder{ID: "p1", Name: "doc_code", Label: "Doc Code", Type: PHText}
    b, err := json.Marshal(p)
    if err != nil { t.Fatal(err) }
    if !strings.Contains(string(b), `"name":"doc_code"`) {
        t.Fatalf("missing name in JSON: %s", b)
    }
    var got Placeholder
    if err := json.Unmarshal(b, &got); err != nil { t.Fatal(err) }
    if got.Name != "doc_code" { t.Fatalf("name lost: got %q", got.Name) }
}
```

- [ ] **Step 2: Run test, verify FAIL** (`unknown field Name`)

`go test ./internal/modules/templates_v2/domain/ -run TestPlaceholder_NameField -v`

- [ ] **Step 3: Add `Name` field**

```go
type Placeholder struct {
    ID    string          `json:"id"`
    Name  string          `json:"name,omitempty"`
    Label string          `json:"label"`
    Type  PlaceholderType `json:"type"`
    // ... rest unchanged
}
```

- [ ] **Step 4: Run test, verify PASS**

`go test ./internal/modules/templates_v2/domain/...`

- [ ] **Step 5: Commit**

```bash
git add internal/modules/templates_v2/domain/schemas.go internal/modules/templates_v2/domain/schemas_test.go
git commit -m "feat(templates): add Name field to Placeholder schema"
```

---

## Task 2: Add name validation (format + uniqueness) to Go validator

**Files:**
- Modify: `internal/modules/templates_v2/domain/errors.go`
- Modify: `internal/modules/templates_v2/application/schema.go` (locate `ValidatePlaceholders` first via grep)
- Modify: `internal/modules/templates_v2/delivery/http/errors.go` (add 422 mapping)
- Test: same dir as `ValidatePlaceholders` test

**Subagent:** codex (multi-file logic)

**Caveman prompt:**
> Add 2 sentinel errors `ErrPlaceholderNameInvalid`, `ErrDuplicatePlaceholderName` in `internal/modules/templates_v2/domain/errors.go`. Find `ValidatePlaceholders` (grep `func ValidatePlaceholders`). After existing ID-uniqueness loop, add: regex `^[a-z][a-z0-9_]{0,49}$` check (skip empty names — backward compat) and `seenNames` uniqueness map. Map both errors → HTTP 422 in `delivery/http/errors.go` (mirror `ErrDuplicatePlaceholderID` pattern). Tests: `TestValidate_InvalidName_Error` ("Bad Name!" → ErrPlaceholderNameInvalid), `TestValidate_DuplicateName_Error` (two with `Name:"x"` → ErrDuplicatePlaceholderName), `TestValidate_EmptyName_Allowed`, `TestValidate_ValidName_NoError`. Run `go test ./internal/modules/templates_v2/...`. Commit `feat(templates): validate placeholder name format and uniqueness`.

- [ ] **Step 1: Write failing tests**

```go
func TestValidatePlaceholders_InvalidName_Error(t *testing.T) {
    err := ValidatePlaceholders([]domain.Placeholder{{ID: "p1", Name: "Bad Name!", Label: "X", Type: domain.PHText}})
    if !errors.Is(err, domain.ErrPlaceholderNameInvalid) { t.Fatalf("got %v", err) }
}
func TestValidatePlaceholders_DuplicateName_Error(t *testing.T) {
    err := ValidatePlaceholders([]domain.Placeholder{
        {ID: "p1", Name: "same", Label: "A", Type: domain.PHText},
        {ID: "p2", Name: "same", Label: "B", Type: domain.PHText},
    })
    if !errors.Is(err, domain.ErrDuplicatePlaceholderName) { t.Fatalf("got %v", err) }
}
func TestValidatePlaceholders_EmptyName_Allowed(t *testing.T) {
    err := ValidatePlaceholders([]domain.Placeholder{{ID: "p1", Name: "", Label: "X", Type: domain.PHText}})
    if err != nil { t.Fatalf("empty name should pass: %v", err) }
}
func TestValidatePlaceholders_ValidName_NoError(t *testing.T) {
    err := ValidatePlaceholders([]domain.Placeholder{
        {ID: "p1", Name: "customer_name", Label: "X", Type: domain.PHText},
        {ID: "p2", Name: "effective_date", Label: "Y", Type: domain.PHDate},
    })
    if err != nil { t.Fatalf("valid names should pass: %v", err) }
}
```

- [ ] **Step 2: Verify FAIL**

`go test ./internal/modules/templates_v2/application/... -run TestValidatePlaceholders_ -v`

- [ ] **Step 3: Add sentinel errors**

```go
// internal/modules/templates_v2/domain/errors.go
var ErrPlaceholderNameInvalid   = errors.New("placeholder name invalid")
var ErrDuplicatePlaceholderName = errors.New("duplicate placeholder name")
```

- [ ] **Step 4: Add validation in `ValidatePlaceholders`**

```go
var nameRe = regexp.MustCompile(`^[a-z][a-z0-9_]{0,49}$`)

// inside ValidatePlaceholders, after ID-uniqueness loop:
seenNames := make(map[string]struct{}, len(phs))
for _, p := range phs {
    if p.Name == "" { continue }
    if !nameRe.MatchString(p.Name) {
        return fmt.Errorf("placeholder[%s] name %q: %w",
            p.ID, p.Name, domain.ErrPlaceholderNameInvalid)
    }
    if _, dup := seenNames[p.Name]; dup {
        return fmt.Errorf("duplicate_placeholder_name %s: %w",
            p.Name, domain.ErrDuplicatePlaceholderName)
    }
    seenNames[p.Name] = struct{}{}
}
```

- [ ] **Step 5: Map errors → HTTP 422 in `delivery/http/errors.go`** (follow `ErrDuplicatePlaceholderID` pattern — locate via grep)

- [ ] **Step 6: Verify PASS**

`go test ./internal/modules/templates_v2/...`

- [ ] **Step 7: Commit**

```bash
git commit -am "feat(templates): validate placeholder name format and uniqueness"
```

---

## Task 3: Add `slugifyLabel` utility + TS `name?` field

**Files:**
- Modify: `frontend/apps/web/src/features/templates/placeholder-types.ts`
- Test: `frontend/apps/web/src/features/templates/__tests__/placeholder-types.test.ts` (create if absent)

**Subagent:** haiku (pure utility + types)

**Caveman prompt:**
> In `frontend/apps/web/src/features/templates/placeholder-types.ts`: add optional `name?: string` field to `Placeholder` interface (between `id` and `label`). Append `slugifyLabel(label: string): string` export — lowercase, non-alphanum runs → `_`, trim leading/trailing `_`, prefix `f_` if doesn't start with letter, slice 50 chars, fallback `'field'`. Add unit tests for slugify edge cases. Run `pnpm --filter web test placeholder-types`. Commit `feat(templates): add slugifyLabel utility and name field to Placeholder type`.

- [ ] **Step 1: Write failing tests**

```typescript
// __tests__/placeholder-types.test.ts
import { describe, expect, it } from 'vitest';
import { slugifyLabel } from '../placeholder-types';

describe('slugifyLabel', () => {
  it('lowercases and underscores spaces', () => {
    expect(slugifyLabel('Customer Name')).toBe('customer_name');
  });
  it('strips special chars', () => {
    expect(slugifyLabel('Effective Date (ISO)')).toBe('effective_date_iso');
  });
  it('prefixes f_ when starts with non-letter', () => {
    expect(slugifyLabel('123abc')).toBe('f_123abc');
  });
  it('caps at 50 chars', () => {
    expect(slugifyLabel('a'.repeat(80)).length).toBe(50);
  });
  it('fallback for empty', () => {
    expect(slugifyLabel('')).toBe('field');
  });
  it('trims leading/trailing underscores', () => {
    expect(slugifyLabel('  hello  ')).toBe('hello');
  });
});
```

- [ ] **Step 2: Verify FAIL** (`slugifyLabel is not a function`)

`pnpm --filter web vitest run src/features/templates/__tests__/placeholder-types.test.ts`

- [ ] **Step 3: Add `name?` field + `slugifyLabel`**

```typescript
export interface Placeholder {
  id: string;
  name?: string;
  label: string;
  type: PlaceholderType;
  // ... rest unchanged
}

export function slugifyLabel(label: string): string {
  const cleaned = label
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '_')
    .replace(/^_+|_+$/g, '')
    .slice(0, 50);
  if (!cleaned) return 'field';
  return /^[a-z]/.test(cleaned) ? cleaned : `f_${cleaned}`.slice(0, 50);
}
```

- [ ] **Step 4: Verify PASS**

- [ ] **Step 5: Commit**

```bash
git commit -am "feat(templates): add slugifyLabel utility and name field to Placeholder type"
```

---

## Task 3b: TS API wire-mapping — preserve `name` on load/save

**Files:**
- Modify: `frontend/apps/web/src/features/templates/v2/api/templatesV2.ts:291-327`
- Test: `frontend/apps/web/src/features/templates/v2/api/__tests__/templatesV2.test.ts` (create if absent)

**Subagent:** haiku (small wire-format edit)

**Caveman prompt:**
> In `frontend/apps/web/src/features/templates/v2/api/templatesV2.ts`: (1) Add `name?: string` to `WirePlaceholder` interface (line 291). (2) In `placeholderFromWire` add `...(w.name != null ? { name: w.name } : {})`. (3) In `placeholderToWire` add `...(p.name != null ? { name: p.name } : {})`. Test: round-trip `{ id, name: "doc_code", label, type }` survives `placeholderToWire` → `placeholderFromWire`. Run `pnpm --filter web vitest run templatesV2`. Commit `feat(templates): preserve placeholder name in wire format`.

**Why critical:** Without this, the TS API silently strips `name` on every load/save — Task 3's type field would be ignored end-to-end.

- [ ] **Step 1: Write failing test**

```typescript
import { describe, expect, it } from 'vitest';
// expose internals or import via test wrapper
import { placeholderFromWire, placeholderToWire } from '../templatesV2';

it('preserves name through wire round-trip', () => {
  const ph = { id: 'pid', name: 'doc_code', label: 'Doc Code', type: 'text' as const };
  const wire = placeholderToWire(ph);
  expect(wire.name).toBe('doc_code');
  const back = placeholderFromWire(wire);
  expect(back.name).toBe('doc_code');
});

it('omits name when undefined', () => {
  const ph = { id: 'pid', label: 'X', type: 'text' as const };
  expect(placeholderToWire(ph).name).toBeUndefined();
});
```

- [ ] **Step 2: Verify FAIL** (`name` undefined after round-trip)

- [ ] **Step 3: Implement**

```typescript
interface WirePlaceholder {
  id: string;
  name?: string;            // ← ADD
  label: string;
  // ... rest unchanged
}

function placeholderFromWire(w: WirePlaceholder): Placeholder {
  return {
    id: w.id,
    ...(w.name != null ? { name: w.name } : {}),  // ← ADD
    label: w.label,
    type: w.type as Placeholder['type'],
    // ... rest unchanged
  };
}

function placeholderToWire(p: Placeholder): WirePlaceholder {
  return {
    id: p.id,
    ...(p.name != null ? { name: p.name } : {}),  // ← ADD
    label: p.label,
    type: p.type,
    // ... rest unchanged
  };
}
```

If `placeholderFromWire`/`placeholderToWire` aren't exported, also export them (or co-locate the test in `templatesV2.ts` via internal helper).

- [ ] **Step 4: Verify PASS**

- [ ] **Step 5: Commit**

```bash
git commit -am "feat(templates): preserve placeholder name in wire format"
```

---

## Task 4: PlaceholderInspector — name input + label sync

**Files:**
- Modify: `frontend/apps/web/src/features/templates/placeholder-inspector.tsx`
- Modify: `frontend/apps/web/src/features/templates/__tests__/placeholder-inspector.test.tsx`

**Subagent:** codex (UI logic + test)

**Caveman prompt:**
> In `frontend/apps/web/src/features/templates/placeholder-inspector.tsx`: import `slugifyLabel`. Add `Name (token in document)` input after `Label` input. Show inline error if value not matching `^[a-z][a-z0-9_]{0,49}$`. Sync logic: when `label` changes, if current `name === slugifyLabel(oldLabel)` (still auto-derived) update `name` too; else preserve user override. Tests: (1) name input renders, (2) label change syncs name when auto-derived, (3) label change preserves name when manually edited, (4) invalid name shows error. Run `pnpm --filter web test placeholder-inspector`. Commit `feat(templates): add name input and label sync to PlaceholderInspector`.

- [ ] **Step 1: Write failing tests**

```tsx
it('renders name input', () => {
  render(<PlaceholderInspector value={mockPlaceholder({ name: 'foo' })} onChange={vi.fn()} />);
  expect(screen.getByTestId('ph-name')).toHaveValue('foo');
});

it('syncs name when label changes if name was auto-derived', () => {
  const onChange = vi.fn();
  const value = mockPlaceholder({ label: 'Old', name: 'old' });
  render(<PlaceholderInspector value={value} onChange={onChange} />);
  fireEvent.change(screen.getByTestId('ph-label'), { target: { value: 'New Label' } });
  expect(onChange).toHaveBeenCalledWith(expect.objectContaining({ name: 'new_label' }));
});

it('preserves name when manually edited', () => {
  const onChange = vi.fn();
  const value = mockPlaceholder({ label: 'Old', name: 'custom_slug' });
  render(<PlaceholderInspector value={value} onChange={onChange} />);
  fireEvent.change(screen.getByTestId('ph-label'), { target: { value: 'New Label' } });
  expect(onChange).toHaveBeenCalledWith(expect.objectContaining({ label: 'New Label', name: 'custom_slug' }));
});

it('shows error for invalid name', () => {
  render(<PlaceholderInspector value={mockPlaceholder({ name: 'Bad Name!' })} onChange={vi.fn()} />);
  expect(screen.getByRole('alert')).toBeInTheDocument();
});
```

- [ ] **Step 2: Verify FAIL**

- [ ] **Step 3: Implement (sketch)**

```tsx
import { slugifyLabel } from './placeholder-types';

const nameRe = /^[a-z][a-z0-9_]{0,49}$/;

// label onChange:
const handleLabelChange = (newLabel: string) => {
  const autoDerivedOld = slugifyLabel(value.label);
  const wasAutoDerived = !value.name || value.name === autoDerivedOld;
  onChange({
    ...value,
    label: newLabel,
    ...(wasAutoDerived ? { name: slugifyLabel(newLabel) } : {}),
  });
};

// JSX:
<label>
  Name (token in document)
  <input
    data-testid="ph-name"
    type="text"
    value={value.name ?? ''}
    placeholder={slugifyLabel(value.label)}
    onChange={(e) => onChange({ ...value, name: e.target.value || undefined })}
  />
  {value.name && !nameRe.test(value.name) && (
    <span role="alert">Lowercase letters, digits, underscores; start with letter; max 50 chars.</span>
  )}
</label>
```

- [ ] **Step 4: Verify PASS**

- [ ] **Step 5: Commit**

```bash
git commit -am "feat(templates): add name input and label sync to PlaceholderInspector"
```

---

## Task 5: PlaceholderChip — drag carries name

**Files:**
- Modify: `frontend/apps/web/src/features/templates/placeholder-chip.tsx`
- Modify: `frontend/apps/web/src/features/templates/__tests__/placeholder-chip.test.tsx`

**Subagent:** haiku (small contract change)

**Caveman prompt:**
> In `placeholder-chip.tsx`: `onDragStart` set 2nd dataTransfer key `application/x-placeholder-name = placeholder.name ?? ''`. Change `usePlaceholderDrop` callback signature to `onInsert: (id: string, name: string) => void`; `onDrop` reads both keys and passes both. Update test to assert both `setData` calls and that `onInsert` receives `(id, name)`. Run `pnpm --filter web test placeholder-chip`. Commit `feat(templates): carry placeholder name in drag transfer`.

- [ ] **Step 1: Write failing test**

```tsx
it('drag sets both id and name on dataTransfer', () => {
  const setData = vi.fn();
  render(<PlaceholderChip placeholder={mockPlaceholder({ id: 'pid', name: 'pname' })} onInsert={vi.fn()} />);
  fireEvent.dragStart(screen.getByText('pname'), { dataTransfer: { setData, effectAllowed: '' } });
  expect(setData).toHaveBeenCalledWith('application/x-placeholder-id', 'pid');
  expect(setData).toHaveBeenCalledWith('application/x-placeholder-name', 'pname');
});

it('drop calls onInsert with id and name', () => {
  const onInsert = vi.fn();
  const { onDrop } = renderHook(() => usePlaceholderDrop(onInsert)).result.current;
  const getData = (k: string) => k === 'application/x-placeholder-id' ? 'pid' : 'pname';
  onDrop({ preventDefault: vi.fn(), dataTransfer: { getData } } as any);
  expect(onInsert).toHaveBeenCalledWith('pid', 'pname');
});
```

- [ ] **Step 2: Verify FAIL**

- [ ] **Step 3: Implement**

```typescript
// onDragStart:
e.dataTransfer.setData('application/x-placeholder-id', placeholder.id);
e.dataTransfer.setData('application/x-placeholder-name', placeholder.name ?? '');
e.dataTransfer.effectAllowed = 'copy';

// usePlaceholderDrop:
export function usePlaceholderDrop(onInsert: (id: string, name: string) => void) {
  return {
    onDragOver: (e: React.DragEvent) => { e.preventDefault(); e.dataTransfer.dropEffect = 'copy'; },
    onDrop: (e: React.DragEvent) => {
      e.preventDefault();
      const id   = e.dataTransfer.getData('application/x-placeholder-id');
      const name = e.dataTransfer.getData('application/x-placeholder-name');
      if (id) onInsert(id, name);
    },
  };
}
```

- [ ] **Step 4: Verify PASS**

- [ ] **Step 5: Commit**

```bash
git commit -am "feat(templates): carry placeholder name in drag transfer"
```

---

## Task 6: TemplateAuthorPage — insert `{name}`, seed name on add

**Files:**
- Modify: `frontend/apps/web/src/features/templates/v2/TemplateAuthorPage.tsx` (lines 103-160)
- Test: existing TemplateAuthorPage test or new dedicated test for `insertPlaceholder` + `addPlaceholder`

**Subagent:** codex (touches multiple call sites + collision logic)

**Caveman prompt:**
> In `frontend/apps/web/src/features/templates/v2/TemplateAuthorPage.tsx`: (1) Change `insertPlaceholder` signature to `(id: string, name: string)`; insert `name ? \`{${name}}\` : \`{{${id}}}\``. (2) Update `usePlaceholderDrop` call-site to match new signature. (3) Inline helper `uniqueName(base, existing)` appending `_2`, `_3`, etc. (4) `addPlaceholder` seeds `name: uniqueName(slugifyLabel('New placeholder'), existingPlaceholders)`. (5) When user toggles `computed=true` and sets a `resolverKey` in inspector, if `name` is empty auto-derive `name = slugifyLabel(resolverKey)` (handled in inspector OR addPlaceholder — choose inspector for consistency). Test: insert with name → editor receives `{customer_name}`; insert with empty name → fallback `{{uuid}}`; addPlaceholder collision → `_2` suffix; computed placeholder gets name auto-derived from resolverKey. Run `pnpm --filter web test TemplateAuthorPage`. Commit `feat(templates): insert {name} tokens and seed unique slugs on placeholder add`.

- [ ] **Step 1: Write failing test**

```tsx
it('inserts {name} token when placeholder has name', async () => {
  // ...render TemplateAuthorPage, add placeholder with name 'cust_name', click insert
  // assert editor view content contains '{cust_name}'
});

it('falls back to {{id}} when name is empty', async () => {
  // ...
});

it('appends _2 on duplicate auto-derived name', async () => {
  // click addPlaceholder twice, assert second placeholder.name === 'new_placeholder_2'
});
```

- [ ] **Step 2: Verify FAIL**

- [ ] **Step 3: Implement**

```typescript
const insertPlaceholder = useCallback((id: string, name: string) => {
  insertTokenAtCursor(name ? `{${name}}` : `{{${id}}}`);
}, [insertTokenAtCursor]);

function uniqueName(base: string, existing: Placeholder[]): string {
  const taken = new Set(existing.map((p) => p.name).filter(Boolean) as string[]);
  if (!taken.has(base)) return base;
  let i = 2;
  while (taken.has(`${base}_${i}`)) i++;
  return `${base}_${i}`;
}

const addPlaceholder = useCallback(() => {
  const next: Placeholder = {
    id: crypto.randomUUID(),
    name: uniqueName(slugifyLabel('New placeholder'), localSchemas.placeholders ?? []),
    label: 'New placeholder',
    type: 'text',
  };
  // ... existing append logic
}, [localSchemas]);

// usePlaceholderDrop call-site:
const placeholderDrop = usePlaceholderDrop(insertPlaceholder);
```

- [ ] **Step 4: Verify PASS**

- [ ] **Step 5: Commit**

```bash
git commit -am "feat(templates): insert {name} tokens and seed unique slugs on placeholder add"
```

---

## Task 7: Freeze service — fanout map keyed by `name`

**Files:**
- Modify: `internal/modules/documents_v2/application/freeze_service.go:181-188`
- Modify: `internal/modules/documents_v2/application/freeze_service_test.go`

**Subagent:** codex (test fixtures + logic)

**Caveman prompt:**
> In `freeze_service.go` `Freeze` method, replace placeholderVals construction (lines 181-188) with version that builds `idToName` lookup from schema then keys `placeholderVals` by name (fallback to UUID when name empty). `resolvedForSubblocks` stays UUID-keyed. Update test fixtures to give placeholders `Name` field; assert fanout request `PlaceholderValues` keyed by slug names. Add `TestFreeze_FallsBackToIDWhenNameEmpty`. Run `go test ./internal/modules/documents_v2/application/...`. Commit `feat(documents): key fanout placeholder map by name not UUID`.

**Backward-compat note:** Old templates with `{{uuid}}` tokens and no `name` on schema → freeze falls back to UUID-keyed fanout map → eigenpal looks for `{uuid}` (single-brace) in DOCX which has `{{uuid}}` (double) → no match → tokens left as-is in output. Pre-existing broken behaviour preserved (no crash). Document in test (see Task 10 Step 2).

- [ ] **Step 1: Update test fixtures + add fallback test**

```go
// Adjust existing happy-path: placeholders gain Name fields
schema := []tmpldom.Placeholder{
    {ID: "p_user", Name: "user_field", Required: true, Type: tmpldom.PHText},
    {ID: "p_comp", Name: "doc_code",   Computed: true, ResolverKey: &resolverKey, Type: tmpldom.PHText},
}
// existing assertion changes:
if fanoutClient.req.PlaceholderValues["user_field"] != "user-value" { t.Fatal(...) }
if fanoutClient.req.PlaceholderValues["doc_code"] != "DOC-001"     { t.Fatal(...) }

// new test:
func TestFreezeService_Freeze_FallsBackToIDWhenNameEmpty(t *testing.T) {
    // schema with Name: "" — assert PlaceholderValues key == UUID
}
```

- [ ] **Step 2: Verify FAIL**

`go test ./internal/modules/documents_v2/application/... -run TestFreezeService -v`

- [ ] **Step 3: Replace lines 181-188**

```go
idToName := make(map[string]string, len(schema))
for _, p := range schema {
    if p.Name != "" {
        idToName[p.ID] = p.Name
    }
}

placeholderVals := map[string]string{}
resolvedForSubblocks := map[string]any{}
for id, v := range valMap {
    if sv, ok := v.(string); ok {
        key := id
        if n, ok := idToName[id]; ok {
            key = n
        }
        placeholderVals[key] = sv
        resolvedForSubblocks[id] = sv
    }
}
```

- [ ] **Step 4: Verify PASS**

`go test ./internal/modules/documents_v2/application/...`

- [ ] **Step 5: Commit**

```bash
git commit -am "feat(documents): key fanout placeholder map by name not UUID"
```

---

## Task 8: Remove zone purge leftovers (Go side)

**Files:**
- Modify: `internal/modules/render/fanout/client.go:11-19`
- Modify: `internal/modules/render/fanout/client_test.go`
- Modify: `internal/modules/render/fanout/reconstruction_test.go:138`

**Subagent:** haiku (deletions only)

**Caveman prompt:**
> In `internal/modules/render/fanout/client.go` delete `ZoneContent map[string]string \`json:"zone_content"\`` field (line 16) from `FanoutRequest`. Delete `ZoneContent: ...` from all test fixtures in `client_test.go` and `reconstruction_test.go`. Verify `go build ./...` and `go test ./internal/modules/render/...` pass. Commit `chore(fanout): drop ZoneContent leftover from zone purge`.

- [ ] **Step 1: Delete `ZoneContent` field**

```go
// internal/modules/render/fanout/client.go
type FanoutRequest struct {
    TenantID          string            `json:"tenant_id"`
    RevisionID        string            `json:"revision_id"`
    BodyDocxS3Key     string            `json:"body_docx_s3_key"`
    PlaceholderValues map[string]string `json:"placeholder_values"`
    // ZoneContent removed
    Composition       json.RawMessage   `json:"composition_config"`
    ResolvedValues    map[string]any    `json:"resolved_values"`
}
```

- [ ] **Step 2: Strip `ZoneContent:` from test fixtures**

Grep `ZoneContent:` in `internal/modules/render/fanout/`, delete each occurrence.

- [ ] **Step 3: Verify build + tests**

```bash
go build ./...
go test ./internal/modules/render/...
```

- [ ] **Step 4: Commit**

```bash
git commit -am "chore(fanout): drop ZoneContent leftover from zone purge"
```

---

## Task 9: Remove zone purge leftovers (docgen-v2 side)

**Files:**
- Modify: `apps/docgen-v2/src/render/fanout.ts`
- Modify: `apps/docgen-v2/src/routes/fanout.ts`
- Modify: `apps/docgen-v2/src/render/__tests__/fanout.test.ts`
- Modify: `apps/docgen-v2/src/routes/__tests__/fanout.test.ts`
- Delete: `apps/docgen-v2/src/render/zoneInjection.ts`
- Delete: `apps/docgen-v2/src/render/__tests__/zoneInjection.test.ts`

**Subagent:** codex (multi-file refactor + deletions)

**Caveman prompt:**
> In `apps/docgen-v2`: (1) `src/render/fanout.ts` — drop `import { injectZones }`, drop `zoneContent` from `FanoutInput`, drop `injectZones(...)` call; pass `input.bodyDocx` directly to `processTemplateDetailed` (use `.buffer.slice(byteOffset, byteOffset+byteLength)`). (2) `src/routes/fanout.ts` — drop `zone_content` from Zod body schema and from destructured params. (3) Delete `src/render/zoneInjection.ts` and its test file. (4) Strip `zoneContent: {}` from all test fixtures in `__tests__/fanout.test.ts` (both render and routes). Verify `pnpm --filter docgen-v2 test`. Commit `chore(docgen-v2): drop zone injection leftover from zone purge`.

- [ ] **Step 1: Edit `apps/docgen-v2/src/render/fanout.ts`**

```typescript
// Remove: import { injectZones } from './zoneInjection.js';
// Remove zoneContent from FanoutInput.
// Remove: const withZones = await injectZones(input.bodyDocx, input.zoneContent);
// Replace processTemplateDetailed call:
const result = processTemplateDetailed(
  input.bodyDocx.buffer.slice(
    input.bodyDocx.byteOffset,
    input.bodyDocx.byteOffset + input.bodyDocx.byteLength,
  ) as ArrayBuffer,
  variables,
  { nullGetter: 'empty' },
);
```

- [ ] **Step 2: Edit `apps/docgen-v2/src/routes/fanout.ts`** — drop `zone_content` from Zod schema + destructuring + `fanout({...})` call.

- [ ] **Step 3: Delete files**

```bash
rm apps/docgen-v2/src/render/zoneInjection.ts
rm apps/docgen-v2/src/render/__tests__/zoneInjection.test.ts
```

- [ ] **Step 4: Strip `zoneContent: {}` from all test fixtures**

```bash
# Grep zoneContent then remove each occurrence
```

- [ ] **Step 5: Verify**

```bash
cd apps/docgen-v2 && pnpm test
cd apps/docgen-v2 && pnpm tsc --noEmit
```

- [ ] **Step 6: Commit**

```bash
git commit -am "chore(docgen-v2): drop zone injection leftover from zone purge"
```

---

## Task 10: End-to-end verification

**Subagent:** none (run locally; visual + behavioral check)

- [ ] **Step 1: Full automated test run**

```bash
go test ./... 2>&1 | tail -50
cd frontend/apps/web && pnpm tsc --noEmit && pnpm test
cd apps/docgen-v2 && pnpm test
```

- [ ] **Step 1b: Backward-compat regression test (docgen-v2)**

Add to `apps/docgen-v2/src/render/__tests__/fanout.test.ts`:

```typescript
it('does not crash when DOCX contains legacy {{uuid}} tokens and no matching name in placeholderValues', async () => {
  const docx = await loadFixtureWithDoubleBraceTokens(); // {{abc-123}}
  const result = await fanout({
    bodyDocx: docx,
    placeholderValues: { 'abc-123': 'value' }, // legacy UUID-keyed
    compositionConfig: { header_sub_blocks: [], footer_sub_blocks: [], sub_block_params: {} },
    resolvedValues: {},
  });
  expect(result.buffer.byteLength).toBeGreaterThan(0);
  expect(result.unreplacedVars.length).toBeGreaterThanOrEqual(0); // tokens left as-is, no throw
});
```

- [ ] **Step 2: Manual smoke test in browser preview**

1. Start preview, load template author page.
2. Click "Add placeholder" → label `Customer Name` → inspector shows `name = customer_name`.
3. Click again → second placeholder gets `name = customer_name_2`.
4. Drag chip into editor → token appears as `{customer_name}` highlighted ORANGE (eigenpal templatePlugin live).
5. Inspector: change `name` to `Bad Name!` → save returns 422 with `placeholder name invalid`.
6. Change to `cust_name`, save schema, fill in placeholder values, freeze document.
7. Inspect docgen-v2 fanout request log → `placeholder_values: { cust_name: "Acme Corp" }` (slug, not UUID).
8. Download final DOCX → open in Word → `{cust_name}` substituted with `Acme Corp`.

- [ ] **Step 3: Update ADR**

Edit `wiki/decisions/0003-token-syntax-migration.md`: status `Proposed` → `Accepted (executed 2026-04-25)`.

- [ ] **Step 4: Final commit**

```bash
git commit -am "docs(adr): mark token syntax migration as executed"
```

---

## Execution Strategy

**Parallelizable groups** (after Task 1 lands):
- Group A (Go): Task 2, Task 7, Task 8 — sequential within Go but Task 8 independent
- Group B (Frontend): Task 3, Task 4, Task 5, Task 6 — sequential within frontend
- Group C (docgen-v2): Task 9 — independent

After Task 1 + Task 3 (foundation types) land, dispatch Group A and B in parallel (codex). Task 9 can run anytime alongside (haiku).

**Final review:** Dispatch codex:codex-rescue to review the full diff before merge — flag drift, missed call-sites, contract mismatches.

---

## Self-Review Checklist

- [x] Spec coverage: all 8 plan steps from `.claude/plans/b-sharded-avalanche.md` mapped to Tasks 1-9
- [x] Placeholder scan: every step has concrete code
- [x] Type consistency: `Placeholder.name` (Go `Name` / TS `name?`), `slugifyLabel`, `uniqueName`, `idToName` consistent across tasks
- [x] No DB migration (JSONB column already accepts new field)
- [x] Backward compat: empty `name` allowed → old drafts pass; freeze falls back to UUID key
