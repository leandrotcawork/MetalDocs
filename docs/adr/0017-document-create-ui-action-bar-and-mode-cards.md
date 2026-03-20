# ADR-0017: Document Create UI Action Bar and Mode Cards

## Status
Accepted

## Context
The document creation step includes an action footer and content-mode cards.
Current UI does not match the approved HTML reference: the action bar feels oversized/out of theme and the mode cards miss title hierarchy and spacing, reducing clarity and polish.

## Decision
Standardize the document creation UI to a single layout pattern:
- Action bar uses the shared button system, compact sizing, and a fixed bar with border-top consistent with the reference.
- Mode cards follow a three-tier hierarchy: icon, title, description, and an optional "Recomendado" badge with deliberate spacing.

## Consequences
- Positive:
  - Consistent, professional UI aligned with the reference.
  - Better affordance for mode selection.
- Negative:
  - Requires small markup and CSS changes in the create flow.

## Alternatives Considered
- Keep current layout and only adjust spacing.
- Use an entirely new card component unrelated to the reference.
