# tasks/lessons.md
# Read at the start of EVERY session before touching any code.
# Add new lessons after every correction. Never delete existing lessons.

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
