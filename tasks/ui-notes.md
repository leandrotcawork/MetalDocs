# tasks/ui-notes.md
# Non-durable UI notes and preferences.
#
# Purpose:
# - Capture UI taste decisions (spacing tweaks, text tweaks, visual alignment opinions).
# - This file is allowed to be edited, reorganized, and cleaned up over time.
#
# Non-goals:
# - Do not put architecture, correctness, or boundary rules here. Those belong in `tasks/lessons.md`.
#
# Template:
# Date: YYYY-MM-DD
# Context: <screen/component>
# Decision: <what changed and why>
# Follow-up: <optional>

Date: 2026-03-22
Context: Admin Center layout
Decision: Removed padding from `AdminCenterView` grid wrapper (`.grid`) to match desired spacing from parent shell/card.

Date: 2026-03-22
Context: Admin Center panels (Usuarios online / Ultimas atividades)
Decision: Left-aligned item text by removing `space-between` behavior for panel rows and pushing the right column via `margin-left: auto`.

Date: 2026-03-22
Context: Documents Hub overview
Decision: Increased hub padding and redesigned area cards with left color stripe + progress bar tied to area share.

Date: 2026-03-22
Context: Documents Hub overview
Decision: Applied the same stripe + progress pattern to document type cards.

Date: 2026-03-22
Context: Documents Hub overview (Areas)
Decision: Show all areas regardless of count, align card layout with left stripe + compact progress bar, and surface area descriptions alongside the meta.

Date: 2026-03-22
Context: Documents Hub overview (Tipos de documento)
Decision: Badge prefers short alias (<=3), then short code (<=3); otherwise falls back to initials from name for future dynamic profiles.

Date: 2026-03-22
Context: Documents Hub overview (Tipos de documento)
Decision: Type cards now mirror the Areas card layout with stripe + progress bar + right-side description.

Date: 2026-03-22
Context: Documents Hub overview (Tipos de documento)
Decision: Unified type progress bar thickness with Areas and fixed vertical alignment for badge/title/meta when titles wrap to one or two lines.

Date: 2026-03-22
Context: Documents Hub overview (Tipos de documento)
Decision: Centered profile title text vertically inside the fixed title slot so one-line and two-line titles align consistently.

Date: 2026-03-22
Context: Documents Hub overview (Areas)
Decision: Applied the same fixed title slot and vertical centering model used in type cards to keep one-line and two-line area names visually aligned.

Date: 2026-03-22
Context: Documents Hub overview (Abertos recentemente)
Decision: Wrapped recent rows in a single container card for clearer grouping and cleaner presentation.
