# packages/docx-editor (reserved)

Empty scaffold for a future subtree fork of `@eigenpal/docx-js-editor`.

**Fork trigger:** first confirmed blocker requiring library internals
(restricted-cell editing, custom node schemas, paginator override).
Until then, `@metaldocs/editor-ui` depends on the upstream package
pinned at `0.0.34` exact.

To fork:

```bash
git subtree add --prefix=packages/docx-editor \
  https://github.com/eigenpal/docx-js-editor v0.0.34 --squash
```
