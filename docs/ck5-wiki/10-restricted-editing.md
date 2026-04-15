---
title: Restricted editing
status: draft
area: editing
priority: HIGH
---

# 10 — Restricted editing

CKEditor 5's restricted editing feature provides two cooperating plugins, `StandardEditingMode` (author flow) and `RestrictedEditingMode` (fill flow), connected through a shared marker type (`restrictedEditingException`). Authors mark regions as editable; end users in restricted mode can only type inside those regions.

This page resolves the long-standing question of whether exception regions are **inline only** or may also be **block level**, and documents the APIs, commands, and markers relevant to MetalDocs' Author/Fill split.

---

## Critical finding — exception element type

**Answer: CKEditor 5 supports BOTH inline and block exceptions.** The older MetalDocs memory stating "only `<span>` works, not `<p>`/`<div>`" is **obsolete**. The feature exposes two separate commands and two separate downcast paths.

### Evidence from the official docs

The restricted-editing feature docs explicitly describe **two** kinds of exception fields:

> "Inline editable fields … only allow content editing with features enabled in the restricted mode. … no splitting paragraphs (striking the Enter key) is allowed. Tables or block images cannot be added in this field, either."
>
> "Block editable fields … enable all content editing features loaded in the editor. Content inside the block can be anything, including lists, tables, images etc."
>
> — [Restricted editing: Inline and block editable fields](https://ckeditor.com/docs/ckeditor5/latest/features/restricted-editing.html#inline-and-block-editable-fields)

The API surface backs this up. The `@ckeditor/ckeditor5-restricted-editing` module registers three "exception" commands:

- `restrictedEditingException` — `RestrictedEditingExceptionCommand` (inline / selection-wrapping)
- `restrictedEditingExceptionBlock` — `RestrictedEditingExceptionBlockCommand`
- `restrictedEditingExceptionAuto` — picks inline or block based on the current selection shape

The block command's documented behaviour:

> "The command that toggles exception blocks for the restricted editing. … Wraps or unwraps the selected blocks with non-restricted area. … If blocks which are supposed to be unwrapped are in the middle of an exception, start it or end it, then the exception will be split (if needed) and the blocks will be moved out of it."
>
> — [`RestrictedEditingExceptionBlockCommand`](https://ckeditor.com/docs/ckeditor5/latest/api/module_restricted-editing_restrictededitingexceptionblockcommand-RestrictedEditingExceptionBlockCommand.html)

### What element carries `class="restricted-editing-exception"`?

- **Inline exceptions** round-trip as `<span class="restricted-editing-exception">…</span>`.
- **Block exceptions** round-trip as a wrapper element around one or more block children. The docs do not pin the exact wrapper tag in prose, but the published HTML data format produced by `RestrictedEditingExceptionBlockCommand` uses a block wrapper (in practice `<div class="restricted-editing-exception">` around the contained `<p>` / `<ul>` / `<table>` / etc.). Inside the block exception, the wrapped block elements remain their normal tags (`<p>`, `<h2>`, `<table>`); they are **not** re-parented into spans.

### Constraints on block exceptions

- `restrictedEditing.allowedCommands` only affects **inline** fields. The feature docs warn:
  > "This setting only applies to inline editing fields, where only inline content inserting or editing commands are allowed. Block content commands such as `insertTable` or `enter` cannot be allowed via this setting, as they are only available in block editing fields."
- Block exceptions automatically enable **all** editing commands loaded in the editor while the caret is inside them. You cannot narrow them down the way you narrow inline exceptions.
- The block command operates on whole blocks; partial-paragraph selections are expanded to block boundaries.

### Confidence

**High** for the existence of block exceptions and the two-command split. **Medium** for the exact wrapper tag name (`<div>`) in the data view — the API page for the block command describes behaviour but does not quote the tag literally. If MetalDocs depends on the exact tag (e.g. for DOCX export mapping), run an empirical round-trip on the pinned v48 build and record the output HTML.

### Recommended empirical check

In `apps/ck5-studio`, load `StandardEditingMode` + `RestrictedEditingMode`, execute `editor.execute('restrictedEditingExceptionBlock')` over a multi-paragraph selection, then `editor.getData()`. Record the exact wrapper element and class list in this wiki.

---

## StandardEditingMode — the Author flow

- Plugin: `StandardEditingMode` (`@ckeditor/ckeditor5-restricted-editing`).
- Purpose: authors create the template. They can edit freely **and** mark arbitrary regions as exceptions (future editable zones).
- UI: toolbar buttons for `restrictedEditingException` (inline toggle) and — when added — `restrictedEditingExceptionBlock`.
- When to use in MetalDocs: the "Author" / template-design editor.

Both modes share the same data model marker (`restrictedEditingException`), so a document saved by a Standard-mode editor opens with the same exception regions in a Restricted-mode editor.

---

## RestrictedEditingMode — the Fill flow

- Plugin: `RestrictedEditingMode`, which pulls in `RestrictedEditingModeEditing` + `RestrictedEditingModeUI`.
- Purpose: end users fill the template. Editing is only possible inside marked exceptions; everything else is read-only.
- Inside an inline exception: only commands listed in `restrictedEditing.allowedCommands` plus the always-on typing commands are enabled.
- Inside a block exception: all editor commands are enabled.
- Outside exceptions: a small always-on allow-list (see below) — everything else is disabled.
- When to use in MetalDocs: the end-user "Fill" editor.

Per [`RestrictedEditingModeEditing`](https://ckeditor.com/docs/ckeditor5/latest/api/module_restricted-editing_restrictededitingmodeediting-RestrictedEditingModeEditing.html), the plugin tracks two command sets internally:

- `_alwaysEnabled` — commands enabled **outside** exceptions (navigation, undo/redo, etc.).
- `_allowedInException` — typing commands (`input`, `insertText`, `delete`, `deleteForward`) plus whatever the consumer added via `allowedCommands`.

The plugin toggles command `isEnabled` as the selection crosses exception boundaries (`_checkCommands()` → `_enableCommands()` / `_disableCommands()`).

---

## You cannot run both modes at once

`StandardEditingMode` and `RestrictedEditingMode` are mutually exclusive. The feature docs state:

> "The restricted editing feature introduces two modes: the standard editing mode and the restricted editing mode."

They are listed as separate plugins that a consumer picks **one** of at editor creation. The public docs do **not** expose a supported runtime swap: to switch an open document from Author to Fill (or vice versa), **destroy and re-initialise the editor** with the other plugin list. The underlying marker data survives because it lives in the serialised HTML/model, not in the plugin instance.

(If we ever want live toggling we would need to build a custom wrapper that destroys and recreates the editor while preserving selection — out of scope for the default feature.)

---

## The `restrictedEditingException` marker

- Marker group name: `restrictedEditingException`.
- Created and managed by the exception commands (`RestrictedEditingExceptionCommand`, `RestrictedEditingExceptionBlockCommand`, `RestrictedEditingExceptionAutoCommand`).
- Round-trips through data:
  - **Inline**: downcast to `<span class="restricted-editing-exception">`; upcast reads the same element back into a marker.
  - **Block**: downcast to a block wrapper carrying `class="restricted-editing-exception"` around the covered blocks; upcast restores the marker over the contained block range.
- The marker is the single source of truth — both `StandardEditingMode` (for UI toggling) and `RestrictedEditingMode` (for gating commands) read it.

---

## Commands

Registered by the package:

| Command | Class | Purpose |
|---|---|---|
| `restrictedEditingException` | `RestrictedEditingExceptionCommand` | Toggle an inline exception around the selection. |
| `restrictedEditingExceptionBlock` | `RestrictedEditingExceptionBlockCommand` | Toggle a block exception around the selected blocks. |
| `restrictedEditingExceptionAuto` | `RestrictedEditingExceptionAutoCommand` | Chooses inline vs block based on selection shape. |
| `goToNextRestrictedEditingException` | `RestrictedEditingModeNavigationCommand` | Move caret to the next exception region (Fill mode navigation). |
| `goToPreviousRestrictedEditingException` | `RestrictedEditingModeNavigationCommand` | Move caret to the previous exception region. |

The navigation commands are registered by `RestrictedEditingModeEditing`, so they are only available when `RestrictedEditingMode` is loaded. In Fill mode they are the canonical way to hop between fields (bind to Tab / Shift+Tab in the UI layer).

---

## Configuration: `restrictedEditing.allowedCommands`

Example from the feature docs:

> "the following configuration allows the users to type, delete but also to bold text … `restrictedEditing: { allowedCommands: [ 'bold' ] }`"

Rules:

- The list augments the built-in typing allow-list (`input`, `insertText`, `delete`, `deleteForward`). You do **not** need to add those.
- Applies **only inside inline exceptions**. Block exceptions ignore it.
- Block-structural commands (`insertTable`, `enter`, `insertImage`, etc.) cannot be enabled through this setting — quoting the docs, they "are only available in block editing fields." If you need tables inside a fillable region, use a block exception containing a table (or a table cell as exception — see below).

---

## Nested exceptions

Not documented as supported. The feature docs do not describe nesting, and the block command's `_removeException` splits exceptions rather than nesting them. Treat nesting as **unsupported**; flatten overlapping regions in the author UI.

Status: `⚠ uncertain` — docs silent. If you need this, test and document behaviour explicitly.

---

## Exceptions inside widgets / nested editables

The public docs do not address widget interaction directly. Practically:

- Exceptions live on the model via markers, so they can span content inside widget nested editables **if** the widget's nested editable accepts the corresponding model elements.
- For MetalDocs' `Section` widget: an inline exception inside `Section`'s editable body should work; a block exception wrapping several blocks inside the editable should also work, because the block command wraps `$block` children.
- Exceptions **across** widget boundaries (starting outside the widget and ending inside its editable) are almost certainly not supported — the marker range would cross a non-editable boundary.

Status: `⚠ uncertain` — verify empirically for the specific Section widget schema.

---

## Table cells as exceptions

Not documented as a first-class element. Two workable patterns:

1. **Content-level exception inside a cell** — mark the cell's paragraph as an inline or block exception. This is the normal path and works because table cells contain standard block content.
2. **Whole cell as the exception** — not directly supported; there is no "cell exception" command. Emulate by placing a block exception around the cell's contents.

Status: `⚠ uncertain` for whole-cell semantics. Model-level behaviour (caret navigation, Tab between cells vs Tab between exceptions) needs empirical verification.

---

## Styling hooks (editing view)

- Inline exception region: `<span class="restricted-editing-exception">` (same class in editing and data view).
- Block exception region: block wrapper carrying `class="restricted-editing-exception"`.
- In **Restricted** mode, the editor also marks the root as non-editable where it is outside exceptions; CKEditor applies its standard `ck-restricted-editing_mode_restricted` / `ck-restricted-editing_mode_standard` classes on the editable root (confirm exact class names against your v48 build when writing CSS).

Style these from `apps/ck5-studio/src/styles/app.css` to give fillable regions a distinctive background in Fill mode.

---

## Summary of resolved vs open questions

Resolved:

- Block `<div class="restricted-editing-exception">` **does** work on modern CK5 via `RestrictedEditingExceptionBlockCommand`. The old "span-only" memory is outdated.
- Standard and Restricted modes are mutually exclusive; toggle by re-initialising the editor.
- The marker is `restrictedEditingException`, shared across both modes.
- Navigation commands are `goToNextRestrictedEditingException` / `goToPreviousRestrictedEditingException`.
- `allowedCommands` only affects inline exceptions; block exceptions always enable all commands.

Still `⚠ uncertain` (docs silent — verify empirically on pinned v48):

- Exact data-view tag emitted by the block command (almost certainly `<div>`, not quoted literally in docs).
- Nested exceptions.
- Whole table cell as an exception.
- Exception boundaries that cross a widget's nested-editable boundary.

---

## Sources

- https://ckeditor.com/docs/ckeditor5/latest/features/restricted-editing.html
- https://ckeditor.com/docs/ckeditor5/latest/features/restricted-editing.html#inline-and-block-editable-fields
- https://ckeditor.com/docs/ckeditor5/latest/api/restricted-editing.html
- https://ckeditor.com/docs/ckeditor5/latest/api/module_restricted-editing_restrictededitingmodeediting-RestrictedEditingModeEditing.html
- https://ckeditor.com/docs/ckeditor5/latest/api/module_restricted-editing_restrictededitingexceptionblockcommand-RestrictedEditingExceptionBlockCommand.html
- https://ckeditor.com/blog/ckeditor-drupal-modules-footnotes-restricted-editing/
