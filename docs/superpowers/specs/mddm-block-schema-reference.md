# MDDM Block Schema Reference

This is the developer-facing reference for the 17 MDDM block types.

See `docs/superpowers/specs/2026-04-07-mddm-foundational-design.md` for the architectural rationale.

## Block categories

### Structural blocks (template skeleton, may have `template_block_id`)

| Block | Children | Purpose |
|-------|----------|---------|
| Section | mixed | Top-level organizational unit |
| FieldGroup | Field[] | Form-style label/value table |
| Field | InlineContent OR Block[] | Single labelled field |
| Repeatable | RepeatableItem[] | User-extensible list |
| DataTable | DataTableRow[] | User-extensible structured table |
| RichBlock | Block[] | Labelled long-form content area |

### Content blocks (user-fillable, no `template_block_id`)

| Block | Children | Purpose |
|-------|----------|---------|
| RepeatableItem | Block[] | One item in a Repeatable |
| DataTableRow | DataTableCell[] | One row in a DataTable |
| DataTableCell | InlineContent | One cell in a row |
| Paragraph | InlineContent | Plain paragraph |
| Heading | InlineContent | Sub-heading (level 1-3) |
| BulletListItem | InlineContent | Bullet list item with `level` prop |
| NumberedListItem | InlineContent | Numbered list item with `level` prop |
| Image | (leaf) | Embedded image |
| Quote | Paragraph[] | Blockquote |
| Code | text-only | Code block |
| Divider | (leaf) | Horizontal rule |

## Identity

Every block has:
- `id`: UUID v4, document-local, immutable for the lifetime of the version
- `template_block_id` (structural blocks only): UUID inherited from the template at instantiation, used for lock enforcement

## Validation layers

- **Layer 1 (JSON Schema)**: structural - types, required fields, prop shapes, parent->children grammar
- **Layer 2 (Go business rules)**: locked-block immutability, minItems/maxItems, ID uniqueness, image existence/auth, cross-doc reference validation, size limits

Both layers run on every save server-side.
