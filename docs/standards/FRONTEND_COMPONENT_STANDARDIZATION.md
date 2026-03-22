# Frontend Component Standardization (MetalDocs)

## Scope
- `frontend/apps/web/src/components/*`
- `frontend/apps/web/src/features/*`

## Objective
- Stop visual drift between repeated controls.
- Reuse the same React components for repeated field patterns.

## Form controls (required)
- Text boxes must use `TextFieldBox` from `frontend/apps/web/src/components/ui/FormFieldBox.tsx`.
- Dropdowns must use `DropdownFieldBox` (which uses `FilterDropdown`/`SelectMenu` spotlight pattern).
- Do not introduce new native `<select>` in feature screens that already use spotlight dropdowns.

## Dropdown behavior
- Selection must close the menu in single-select mode.
- Search is enabled only when options exceed `searchThreshold`.
- Visual style must remain the spotlight style from `SelectMenu`.

## Layout rules for repeated cards
- For sibling cards in the same row, equalize height at grid level.
- Prefer `grid-auto-rows` + card `height: 100%` over fixed height per card class.
- Avoid clipping content by hard-coding different heights per card.

## Migration rule
- Any screen touching repeated fields (`nome`, `email`, `username`, search, profile/department/process selectors) must migrate to shared field components in the same change.
