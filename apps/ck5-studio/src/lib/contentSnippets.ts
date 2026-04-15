import type { LibraryItemKey } from "../types";

export const DEFAULT_EDITORIAL_HTML = `
<h1>Untitled Editorial Concept</h1>
<p><em>Oct 24, 2023 &bull; <span class="restricted-editing-exception">Senior Editor</span> &bull; Draft</em></p>
<table>
  <thead>
    <tr><th>Framework</th><th>Outcome</th></tr>
  </thead>
  <tbody>
    <tr><td><em>Minimalist Cohesion</em></td><td><span class="restricted-editing-exception">Reduced cognitive load for deep focus writing.</span></td></tr>
    <tr><td><em>Typographic Authority</em></td><td><span class="restricted-editing-exception">Establishing trust through classic editorial stems.</span></td></tr>
  </tbody>
</table>
<p>[ Image placeholder ]</p>
<p><em><span class="restricted-editing-exception">Fig 1.1: The Interplay of Light and Intellectual Space</span></em></p>
`;

const SNIPPETS: Record<Exclude<LibraryItemKey, "image">, string> = {
  text: "<p>Start writing here.</p>",
  heading: "<h2>Section Heading</h2>",
  section: "<h2>New Section</h2><p>Describe this section.</p>",
  table: "<table><thead><tr><th>Column</th><th>Column</th></tr></thead><tbody><tr><td></td><td></td></tr></tbody></table>",
  note: `
<section class="template-note-block">
  <h3>Template Note Block</h3>
  <p><strong>Policy (locked):</strong> Keep approval statement unchanged.</p>
  <p><strong>Owner Note (editable):</strong> <span class="restricted-editing-exception">Type owner-specific instructions...</span></p>
  <p><strong>Action Date (editable):</strong> <span class="restricted-editing-exception">YYYY-MM-DD</span></p>
  <p><strong>Compliance Tag (locked):</strong> CK5-TEMPLATE-BLOCK-NOTE</p>
</section>
`,
  mixed: `
<h2>Section 2 - Locked Header</h2>
<div class="restricted-editing-exception">
  <p><strong>Editable body:</strong> Write narrative, add tables, and insert media here in Fill mode.</p>
  <p>This is a full editing region under a locked section title.</p>
</div>
`,
};

export function snippetFor(key: Exclude<LibraryItemKey, "image">): string {
  return SNIPPETS[key];
}
