# Targeted Editor Shell Cleanup Design

## Goal

Stabilize the MetalDocs browser document editor by cleaning up only the editor shell around BlockNote, without rewriting the workspace shell or changing backend/API behavior.

The immediate objective is to remove the class of bugs where editor chrome escapes its intended area:

- transient gray browser scrollbar behavior during editor scrolling
- floating table/editor controls appearing in the workspace top bar area
- visual "phantom space" below the editor canvas

This is a containment and ownership redesign for the editor surface, not a product redesign.

### Success criteria

- Editor chrome never renders above or inside the global workspace top bar.
- Floating BlockNote table controls remain visually contained to the editor viewport.
- The editor page has one explicit visual composition owner and one explicit editor viewport boundary.
- Manual scrolling no longer reveals chrome escaping outside the editor shell.
- No API contracts, document schemas, or backend behavior change.

### Non-goals

- Rewriting `DocumentWorkspaceShell` or the workspace navigation shell.
- Changing MDDM block schema, BlockNote schema registration, or document persistence.
- Reworking the full content-builder flow or create flow.
- Fixing unrelated TypeScript errors in `RichSlot.tsx` / `RichField.tsx`.
- General UI polish unrelated to containment and scroll/chrome ownership.

## Problem statement

The browser editor currently stacks three distinct chrome layers:

1. Global workspace top bar in `DocumentWorkspaceShell`
2. Editor page top bar in `BrowserDocumentEditorView`
3. BlockNote formatting chrome in `MDDMEditor`

At the same time, the editor subtree mixes:

- sticky positioning
- large artificial paper spacing
- multiple wrappers with `min-height`, `overflow: clip`, and grid/flex sizing rules
- BlockNote-native floating table chrome plus project-level overrides

This combination makes containment brittle. The strongest symptom is that table/editor chrome can appear in the workspace top bar region, which indicates the editor chrome is not anchored to a single reliable boundary.

## Current structure

Today the browser editor renders through this chain:

```text
DocumentWorkspaceShell
  workspace-topbar
  workspace-main (content-builder mode)
    BrowserDocumentEditorView.root
      BrowserDocumentEditorView.topbar
      BrowserDocumentEditorView.metaBar
      BrowserDocumentEditorView.surface
        BrowserDocumentEditorView.editorShell
          MDDMEditor.pageShell
            MDDMEditor.toolbarWrapper
            MDDMEditor.editorRoot
              BlockNoteView / BlockNoteViewEditor
```

This tree gives us too many layout owners for one surface:

- `BrowserDocumentEditorView` acts like a page shell
- `MDDMEditor` also acts like a page shell
- BlockNote still injects its own floating chrome inside that subtree

The result is a shell-inside-shell model where containment rules are implicit instead of deliberate.

## Root-cause hypothesis

The primary failure is not "bad div names" or a general need to rewrite the whole workspace. The primary failure is:

**editor chrome ownership and containment are split across multiple wrappers with competing positioning and overflow rules.**

More specifically:

- the editor page shell owns top-level composition
- the BlockNote adapter also owns visual shell behavior
- sticky and floating chrome are allowed to resolve against overly broad ancestors
- project CSS overrides hide or reshape BlockNote table chrome without defining one authoritative boundary for where that chrome may render

That makes the system vulnerable to floating controls, sticky toolbars, and scroll artifacts escaping into the wrong visual layer.

## Options considered

### Option 1 - CSS-only containment patch

Adjust `overflow`, `z-index`, `position`, and `height` in the current structure without changing the component tree.

**Pros**

- smallest diff
- fastest to try

**Cons**

- keeps split ownership between `BrowserDocumentEditorView` and `MDDMEditor`
- likely masks symptoms without simplifying the system
- future BlockNote chrome changes remain high-risk

### Option 2 - Targeted editor-shell cleanup

Keep the workspace shell intact, but make the browser editor page the only visual shell owner and reduce `MDDMEditor` to an editor adapter with a strict containment boundary.

**Pros**

- fixes the structural problem locally
- reduces hidden layout coupling
- keeps scope focused on the failing surface

**Cons**

- moderate refactor, not a one-line CSS tweak
- requires updating layout tests/manual verification

### Option 3 - Workspace/content-builder rewrite

Rework the broader workspace shell and content-builder layout end-to-end.

**Pros**

- maximum long-term control

**Cons**

- far wider scope than current evidence supports
- high regression risk
- mixes diagnosis with speculative redesign

### Recommendation

Choose **Option 2: targeted editor-shell cleanup**.

It addresses the actual boundary failure without turning a local editor issue into a workspace-wide rewrite.

## Proposed design

### 1. Single visual shell owner

`BrowserDocumentEditorView` becomes the only component that owns page-level editor composition.

It is responsible for:

- editor page top bar
- document metadata bar
- editor viewport boundary
- footer/status area

`MDDMEditor` stops acting like a page shell and becomes a narrower BlockNote adapter.

### 2. Explicit editor viewport boundary

Introduce one dedicated viewport wrapper inside `BrowserDocumentEditorView` that is the containment boundary for all editor visuals.

Conceptual structure:

```text
BrowserDocumentEditorView.root
  header
  meta
  surface
    editorViewport
      MDDMEditor.root
        formattingToolbar
        paperSurface
          BlockNoteViewEditor
  footer
```

The `editorViewport` boundary is responsible for:

- positioning context for editor chrome
- clipping/containment for floating visuals
- explicit min-height behavior
- isolating editor chrome from the workspace top bar

This boundary must be deliberate, not incidental.

### 3. Reduce `MDDMEditor` responsibility

`MDDMEditor` should own:

- BlockNote initialization
- schema/hooks/plugins
- BlockNote formatting toolbar placement inside its local root
- paper/document surface

`MDDMEditor` should not own:

- outer page framing
- broader page composition
- secondary shell semantics already handled by `BrowserDocumentEditorView`

In practice, this means removing the current "page shell" role from `MDDMEditor` and keeping only a local editor root plus paper surface.

### 4. Chrome layering rules

The layering contract becomes:

- `DocumentWorkspaceShell` top bar is always the highest workspace chrome.
- `BrowserDocumentEditorView` top bar stays within the editor page shell only.
- BlockNote formatting/table chrome must render inside the editor viewport boundary only.
- No editor chrome may visually overlap the workspace top bar region.

Implementation-wise, the design does not assume whether a given BlockNote control is `sticky`, absolutely positioned, or internally floated. The contract is simply that all editor chrome must resolve inside the local editor boundary.

If a BlockNote control is rendered via a portal-like pattern or global floating root, the implementation must explicitly re-scope or suppress that behavior.

### 5. Scroll ownership principle

This cleanup does not require rewriting the entire workspace scroll model.

Instead, it establishes a narrower rule:

- the browser editor page may continue to use the current workspace scroll owner
- but editor chrome must not depend on global page/root coordinates
- all editor-local sticky/floating behavior must resolve within the editor viewport boundary

This avoids over-claiming "root page scroll is the cause" when current evidence does not prove that.

### 6. CSS model

The cleanup should move toward these layout properties:

- `BrowserDocumentEditorView.root`: explicit full-height page frame, `min-height: 0`
- `BrowserDocumentEditorView.surface`: explicit content shell, `min-height: 0`
- `BrowserDocumentEditorView.editorViewport`: `position: relative`, `min-height: 0`, `isolation: isolate`
- `MDDMEditor.root`: local editor boundary, not a second page shell
- `MDDMEditor.paperSurface`: document paper styling only

The important rule is not a specific CSS trick. The important rule is **one page shell owner, one editor viewport boundary, one local editor root**.

## Files in scope

Primary targets:

- `frontend/apps/web/src/features/documents/browser-editor/BrowserDocumentEditorView.tsx`
- `frontend/apps/web/src/features/documents/browser-editor/BrowserDocumentEditorView.module.css`
- `frontend/apps/web/src/features/documents/mddm-editor/MDDMEditor.tsx`
- `frontend/apps/web/src/features/documents/mddm-editor/MDDMEditor.module.css`
- `frontend/apps/web/src/features/documents/mddm-editor/mddm-editor-global.css`

Secondary/optional touch:

- `frontend/apps/web/src/components/content-builder/ContentBuilderView.tsx`

Explicitly out of scope unless evidence later forces it:

- `frontend/apps/web/src/components/DocumentWorkspaceShell.tsx`
- `frontend/apps/web/src/components/DocumentWorkspaceShell.module.css`
- root app shell sizing in `styles.css` / `styles/base.css`

## Risks

### Risk 1 - BlockNote floating controls use DOM patterns we have not fully constrained

Mitigation:

- inspect actual rendered DOM during implementation
- anchor containment rules to observed selectors, not guesses
- avoid broad global CSS that changes unrelated BlockNote behavior

### Risk 2 - Sticky toolbar behavior regresses

Mitigation:

- preserve one local sticky toolbar behavior inside the editor root
- test toolbar visibility during long scroll sessions

### Risk 3 - Table chrome becomes hidden rather than correctly contained

Mitigation:

- prefer containment and re-scoping over blanket `display: none`
- manually validate table interactions after each structural step

## Validation strategy

### Automated

- add structural tests that assert the browser editor tree has a dedicated editor viewport boundary
- add CSS/DOM contract tests for the new containment classes
- if practical, add a Playwright assertion that editor floating chrome bounding boxes do not overlap the workspace top bar area during table interaction

### Manual

Validate the exact user-reported flow:

1. Open browser editor
2. Scroll repeatedly up/down
3. Interact with native table controls
4. Confirm no white floating controls appear in the workspace top bar
5. Confirm no visual phantom area appears below the editor canvas

## Deferred items

These are intentionally deferred unless the targeted cleanup fails to resolve the issue:

- locking `html/body/#root` to fixed-height hidden overflow
- replacing `100vh` with `100dvh` across the workspace shell
- broader content-builder rewrite
- full workspace shell redesign

## Implementation boundary

This design is Level 1 only:

- restructure the browser editor shell
- establish clear containment ownership
- validate that editor chrome stays inside the editor surface

Deferred:

- broader workspace architecture changes
- generic viewport hardening unrelated to the editor shell
- cross-feature cleanup outside the browser editor surface
