---
title: Schema
status: draft
area: core
---

# 04 — Schema

The **schema** is the rulebook for CKEditor 5's model. Every element, attribute,
and nesting relationship that the editor tolerates is declared here. Commands
consult it to decide whether they may run; converters consult it to decide what
to emit; the engine's post-fixers consult it to auto-repair invalid trees.

Access it via `editor.model.schema`.

---

## 1. Registering an element

The canonical call is `schema.register(name, definition)`:

```js
editor.model.schema.register( 'simpleBox', {
    inheritAllFrom: '$blockObject'
} );

editor.model.schema.register( 'simpleBoxTitle', {
    isLimit: true,
    allowIn: 'simpleBox',
    allowContentOf: '$block'
} );

editor.model.schema.register( 'simpleBoxDescription', {
    isLimit: true,
    allowIn: 'simpleBox',
    allowContentOf: '$root'
} );
```
> Source: Implementing a block widget tutorial.

Docs are explicit that `register()` "can be used only once for a given item
name which ensures that only a single editing feature can introduce this
item." To layer additional rules (typical when extending a vendor plugin),
use `schema.extend(name, definition)` — it "can only be used for defined
items."

## 2. Generic presets (inherit-from)

CKEditor ships a small set of pseudo-items that exist purely so your
definitions can inherit from them. `inheritAllFrom: '<preset>'` copies every
flag and allow-rule.

| Preset           | Flags set                          | Use for                                                 |
| ---------------- | ---------------------------------- | ------------------------------------------------------- |
| `$root`          | `isLimit: true`                    | Editable root. You rarely register your own.            |
| `$block`         | `isBlock: true`                    | A paragraph-like block that may contain `$text`.        |
| `$container`     | block, accepts other blocks        | Block that nests other blocks (e.g. blockquote).        |
| `$blockObject`   | `isBlock + isObject + isSelectable`| Self-contained block widget (image-like, our sections). |
| `$inlineObject`  | `isInline + isObject + isSelectable`| Inline widget (mention chip, inline placeholder).      |
| `$text`          | `isInline + isContent`             | You never register this; you allow it inside blocks.    |

Rule of thumb for MetalDocs:

- `mddmSection`, `mddmRepeatable`, `mddmDataTable` → inherit from
  `$blockObject` (they are widgets).
- `mddmSectionBody` → inherit from `$container` (holds paragraphs, tables, …).
- `mddmSectionHeader` → custom: `allowContentOf: '$block'` so it accepts
  inline formatting but not nested blocks.
- `mddmRepeatableItem` → `$container` (rich content inside each row).

## 3. Allow rules

Quoted semantics from the Schema deep-dive:

| Property            | Meaning                                                                 |
| ------------------- | ----------------------------------------------------------------------- |
| `allowIn`           | "Specifies parent elements where an item can exist."                    |
| `allowChildren`     | "Declares which nodes are permitted inside."                            |
| `allowContentOf`    | "Inherits content permissions from another item." (copies its children) |
| `allowWhere`        | "Places item wherever another can go." (copies its parents)             |
| `allowAttributes`   | Whitelists attribute names on this element.                             |
| `allowAttributesOf` | Inherits the allowed-attribute list from another item.                  |
| `inheritAllFrom`    | Shortcut for `allowContentOf` + `allowWhere` + `allowAttributesOf`.     |

`allowIn` and `allowChildren` are mirror images — define one side; the engine
infers the other. Prefer `allowIn` when *you* own the child element (you know
its parents). Prefer `allowChildren` when *you* own the parent container.

## 4. Behavioural flags

These booleans change how the engine treats the element during selection,
deletion, and output:

| Flag           | Behaviour                                                           | When MetalDocs wants it                                  |
| -------------- | ------------------------------------------------------------------- | -------------------------------------------------------- |
| `isBlock`      | Treated as block-level in flow. Gets its own line box.              | Every `mddm*` container.                                 |
| `isInline`     | Treated as inline. `$text`, soft breaks, inline widgets.            | Only inline mddm widgets (none yet — ⚠ uncertain).       |
| `isObject`     | "Self-contained." Selected as a unit; delete removes the whole thing.| Widgets whose internals are managed (section, datatable).|
| `isSelectable` | Enables whole-element selection (click-to-select).                  | Implied by `isObject`; set explicitly on limit wrappers. |
| `isLimit`      | "Creates boundaries preventing selection/deletion crossing."        | Section header, datatable cells — lock structure.        |
| `isContent`    | "Appears in editor output; skipped when empty." (prevents pruning). | Set on leaf widgets that must persist even when empty.   |

## 5. Schema checks (used by commands)

Commands read the schema to compute their `isEnabled` state:

- `schema.checkChild( parent, child )` → `true` if `child` may be inserted
  into `parent`. "Verifies whether a child can exist within a parent
  context."
- `schema.checkAttribute( context, attributeName )` → `true` if the attribute
  is legal on the element at the end of `context`. "Determines if an
  attribute is permitted on an element."

Both accept a *position*, a *selection*, or a simple *element*. Our custom
commands (e.g. `InsertMddmSection`) call `checkChild` on the current
selection's parent before enabling themselves.

## 6. Disallow rules

`disallowChildren`, `disallowIn`, `disallowAttributes` subtract from the
allow-set. Documented precedence (highest first):

1. `disallowChildren` / `disallowIn` from **own** definition
2. `allowChildren` / `allowIn` from **own** definition
3. `disallowChildren` / `disallowIn` from **inherited** definition
4. `allowChildren` / `allowIn` from **inherited** definition

Translation: an element's own disallow always wins over its own allow, and
its own rules always beat whatever it inherited. This matters when we inherit
from `$container` but want to *forbid* specific block types inside
(`mddmRepeatable` forbidding nested `mddmRepeatable`).

## 7. Dynamic rules

Static definitions can't express every constraint. For those, register
callbacks:

```js
schema.addChildCheck( ( context, childDef ) => {
    // Disallow nesting a repeatable inside another repeatable.
    if ( context.endsWith( 'mddmRepeatableItem' )
         && childDef.name === 'mddmRepeatable' ) {
        return false;
    }
} );

schema.addAttributeCheck( ( context, attributeName ) => {
    if ( context.endsWith( 'mddmSectionHeader' )
         && attributeName === 'fontSize' ) {
        return false; // header typography is template-controlled
    }
} );
```

> "Callbacks return `true` (allow), `false` (disallow), or `undefined` (check
> further rules)."

Return `undefined` to defer to the static definitions — never `false` unless
you are sure you want to override them.

## 8. MetalDocs patterns

### 8.1 `mddmSection` — exactly one header + one body

```js
schema.register( 'mddmSection', {
    inheritAllFrom: '$blockObject',
    allowChildren: [ 'mddmSectionHeader', 'mddmSectionBody' ],
    allowAttributes: [ 'sectionId', 'sectionKind' ]
} );

schema.register( 'mddmSectionHeader', {
    isLimit: true,
    allowIn: 'mddmSection',
    allowContentOf: '$block'
} );

schema.register( 'mddmSectionBody', {
    isLimit: true,
    allowIn: 'mddmSection',
    allowContentOf: '$root'      // rich: paragraphs, tables, lists, …
} );
```

"Exactly one of each" cannot be expressed statically — enforce it in a
post-fixer or via `addChildCheck` that rejects a second header/body.

### 8.2 `mddmRepeatable` — only `mddmRepeatableItem` children

```js
schema.register( 'mddmRepeatable', {
    inheritAllFrom: '$blockObject',
    allowChildren: [ 'mddmRepeatableItem' ]
} );

schema.register( 'mddmRepeatableItem', {
    inheritAllFrom: '$container',
    allowIn: 'mddmRepeatable',
    isLimit: true                // item boundary is hard
} );
```

### 8.3 Fixed `mddmDataTable` — locked row count

```js
schema.register( 'mddmDataTable', {
    inheritAllFrom: '$blockObject',
    isLimit: true,               // selection cannot leak out
    allowChildren: [ 'mddmDataRow' ],
    allowAttributes: [ 'tableId', 'rowCount' ]
} );

schema.register( 'mddmDataRow', {
    isLimit: true,
    allowIn: 'mddmDataTable',
    allowChildren: [ 'mddmDataCell' ]
} );
```

Row-count is enforced by **not registering** an `insertRow` command, not by
the schema. The schema says "only `mddmDataRow` is allowed" — which commands
you expose decides whether the user can add more.

⚠ uncertain: whether `isLimit` on the outer table alone is enough to stop
CK5's default table plugin from attempting row insertion. If the default
`Table` plugin is loaded, its commands may still fire; safest path is to not
register the default table feature for this element at all.

## 9. Schema vs restricted editing

These two concerns are **orthogonal**:

- **Schema** answers "is this element legal at this position?" It is static
  structural truth and applies to every edit, undo, paste, and conversion.
- **Restricted editing** answers "is the user allowed to change this right
  now?" It gates **commands**, not the model structure.

A section body can be schema-valid (may contain a paragraph) yet
restricted-editing-locked (the user cannot type into it). Don't try to
express lock state through `disallowChildren` — you'll break programmatic
insertion (template hydration, paste handlers). Keep locking in the
restricted-editing layer; keep structural truth in the schema.

---

## Open questions

- ⚠ uncertain: does `allowContentOf: '$root'` transitively include future
  plugins registered after our section? (Appears yes — rules resolve
  lazily — but needs a test.)
- ⚠ uncertain: best pattern for "at most N children" — post-fixer vs
  `addChildCheck`. Post-fixer is more forgiving during paste; addChildCheck
  is stricter. Recommend post-fixer for MetalDocs until we have a hard
  counter-example.

## Sources

- https://ckeditor.com/docs/ckeditor5/latest/framework/deep-dive/schema.html
- https://ckeditor.com/docs/ckeditor5/latest/framework/architecture/intro.html
- https://ckeditor.com/docs/ckeditor5/latest/framework/tutorials/widgets/implementing-a-block-widget.html
