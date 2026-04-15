# CKEditor 5 Studio Spike Design

**Date:** 2026-04-15
**Status:** Proposed
**Owner:** Codex

## 1. Goal

Create standalone visual spike inside MetalDocs to evaluate CKEditor 5 as future replacement for current BlockNote-based template editor. Spike must not depend on MetalDocs backend, canonical MDDM contract, or production frontend routes.

Purpose is narrow:
- validate CKEditor 5 authoring quality
- validate native CKEditor toolbar quality
- validate editorial shell styling against provided maroon reference
- validate left block library + center editor + right properties shell composition

This spike is explicitly disposable.

## 2. Scope

Included:
- new isolated app at `apps/ck5-studio/`
- React + Vite setup
- one main screen matching editorial reference direction
- top nav shell
- left library/palette
- center paper-like editable canvas
- right properties panel
- native CKEditor 5 toolbar mounted in shell
- block insertion buttons for:
  - Text
  - Heading
  - Section
  - Table
  - Image
- local-only state
- optional `localStorage` draft restore for convenience

Excluded:
- MetalDocs auth
- MetalDocs API calls
- MDDM persistence
- template publish workflow
- DOCX/PDF export
- collaboration
- exact pagination parity with Word/Google Docs
- backend integration
- production route wiring

## 3. App Placement

Spike lives in `apps/ck5-studio/`.

Reason:
- isolated from production app
- fast to delete later
- no accidental coupling with `frontend/apps/web`
- clean evaluation of CKEditor integration cost

## 4. UX Shape

Single screen:
- top editorial navigation bar
- left panel with “Library” and insert actions
- centered paper workspace over pale background
- right panel with block/properties presentation

Visual direction follows provided “Editorial Architect” reference:
- maroon primary accent
- pale blue side surfaces
- white paper center
- Newsreader for editorial titles
- Inter for UI chrome
- soft radius, no harsh borders, tonal layering over line-heavy layout

Toolbar strategy:
- use native CKEditor 5 toolbar, not custom fake toolbar
- place it visibly in layout so user evaluates real CKEditor command surface

## 5. Editor Behavior

Editor is CKEditor 5 decoupled editor.

Why decoupled:
- toolbar placement flexible
- closest fit to document-style shell
- best match for future template-builder experiment

Initial editing model:
- free writing inside document body
- library buttons insert starter content/snippets
- no custom schema/widget system in phase 1

Insertion behavior:
- `Text`: insert paragraph at selection
- `Heading`: insert heading block
- `Section`: insert styled section header snippet
- `Table`: insert starter 2-column table
- `Image`: trigger local image insert/upload path supported by editor

“Section” in this spike is presentational insertion, not yet true MetalDocs semantic block.

## 6. Right Panel

Right properties panel is visual-first, not full engine.

Phase 1 responsibilities:
- show selected element summary if detectable
- show curated controls matching reference tone
- allow simple local visual changes where low-cost
- tolerate partial mock behavior when full binding is not worth cost

Panel exists to test shell feel, not to prove backend-ready block modeling.

## 7. State Model

State stays local to spike app.

Recommended structure:
- editor instance state
- inserted content state owned by CKEditor
- UI shell state for active library item / selected element summary
- optional `localStorage` save/restore for convenience only

No cross-app store reuse from production frontend.

## 8. Styling Strategy

Styling built from scratch for spike.

Rules:
- do not import production MetalDocs layout CSS
- keep CSS isolated under `apps/ck5-studio/`
- use CSS variables for palette, spacing, radii, shadows
- reproduce reference feel, not pixel-perfect clone

Important:
- native CKEditor classes will need overrides
- toolbar and editable area should still feel like one editorial product
- editor content CSS and shell CSS must both be tuned

## 9. Success Criteria

Spike successful if:
- app runs independently
- shell visibly resembles editorial reference
- native CKEditor toolbar is working
- typing feels strong
- image handling works locally
- table insertion/editing works
- left palette insertion works
- app is easy to remove later without touching production flows

## 10. Non-Goals For This Spike

This spike does not answer:
- whether CKEditor should become canonical persisted format
- whether CKEditor can replace MDDM with no adapter cost
- whether current DOCX pipeline should be replaced
- whether premium CKEditor features justify licensing cost
- whether paged editing can fully match Word

Those become later decision inputs after visual spike succeeds.

## 11. Risks

- premium/demo features may require license or may not exist in free self-hosted setup
- native toolbar may need substantial CSS tuning to fit shell
- section-like insertion may look good but still not prove semantic block fit
- visual success may hide later migration cost into MDDM/export/governance stack

## 12. Recommendation

Build this spike before any backend migration decision.

Reason:
- cheapest way to validate ceiling
- avoids premature rewrite of template stack
- gives concrete evidence for future architecture decision

## 13. Implementation Direction

Next step after spec approval:
- write implementation plan for isolated CKEditor spike
- then build app in small safe tasks
- then run browser verification on real Chromium

