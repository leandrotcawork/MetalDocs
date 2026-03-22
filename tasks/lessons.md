# tasks/lessons.md
# Read at the start of EVERY session before touching any code.
#
# Scope:
# - This file is for durable engineering lessons: architecture, boundaries, correctness, reliability, and repeatable frontend patterns.
# - Avoid logging pure preference ("make it prettier", pixel nudges) unless it reveals a reusable rule.
# - UI taste/polish notes go to `tasks/ui-notes.md` (editable/overwrite OK).
#
# Hygiene:
# - Prefer 1 lesson per reusable pattern; do not spam per micro-change.
# - When in doubt: can this prevent a future regression? If no, it is a UI note.

---

## Lesson BB - Scroll lists still need inner spacing parity
Date: 2026-03-22 | Trigger: correction
Wrong:   Leaving `Base de usuarios` list flush to edges while sibling panels keep internal padding
Correct: Add padding on the scrollable `ul` so list content follows the same card spacing rhythm
Rule:    Scroll containers must preserve the same internal spacing standards as non-scroll panels.
Layer:   frontend

## Lesson BC - Matching visual patterns requires matching DOM structure
Date: 2026-03-22 | Trigger: correction
Wrong:   Keeping `Base de usuarios` as an `article` with ad-hoc sections while sibling dashboard cards use panel-style `div` structure
Correct: Rebuild the card with the same header/actions/list container pattern as the audit and online panels
Rule:    When cards are meant to share a UI pattern, align both CSS and DOM structure instead of styling around mismatched markup.
Layer:   frontend

## Lesson BD - Apply spacing at the grid container, not each panel
Date: 2026-03-22 | Trigger: correction
Wrong:   Relying on per-panel padding alone, leaving the grid edges flush to the parent
Correct: Add padding to the grid wrapper so all child panels inherit the same outer breathing room
Rule:    Use container padding for consistent outer spacing across a grid of panels.
Layer:   frontend

## Lesson BE - Matching panel padding requires root-level spacing
Date: 2026-03-22 | Trigger: correction
Wrong:   Applying padding only to inner rows (`header/search/list`) while the card root remains unpadded
Correct: Place spacing on the card root and reduce inner paddings to keep a single spacing source
Rule:    When matching panel primitives, keep padding ownership at the container level.
Layer:   frontend

## Lesson BF - Use one spacing model across sibling cards
Date: 2026-03-22 | Trigger: correction
Wrong:   Base card uses root-level padding while Create/Edit still rely on mixed inner paddings
Correct: Apply the same root-level spacing model to Create, Base and Edit cards
Rule:    Sibling cards in the same section should share the same spacing ownership model.
Layer:   frontend

## Lesson BG - Sibling cards should share semantic element type
Date: 2026-03-22 | Trigger: correction
Wrong:   Mixing `div` and `article` for cards with the same role in one section
Correct: Use a single semantic tag across sibling cards (`article` in this case)
Rule:    Keep semantic structure consistent when components represent the same UI primitive.
Layer:   frontend
