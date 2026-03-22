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
