type Json = any;

const MARK_ORDER = ["bold", "code", "italic", "strike", "underline"];

function sortKeys(obj: Json): Json {
  if (Array.isArray(obj)) return obj.map(sortKeys);
  if (obj === null || typeof obj !== "object") return obj;
  const sorted: Record<string, Json> = {};
  for (const key of Object.keys(obj).sort()) {
    sorted[key] = sortKeys(obj[key]);
  }
  return sorted;
}

function nfc(s: string): string {
  return s.normalize("NFC");
}

function sortMarks(marks: Json[] | undefined): Json[] | undefined {
  if (!marks) return undefined;
  return [...marks].sort((a, b) => {
    return MARK_ORDER.indexOf(a.type) - MARK_ORDER.indexOf(b.type);
  });
}

function marksEqual(a: Json[] | undefined, b: Json[] | undefined): boolean {
  if (!a && !b) return true;
  if (!a || !b) return false;
  if (a.length !== b.length) return false;
  for (let i = 0; i < a.length; i++) {
    if (a[i].type !== b[i].type) return false;
  }
  return true;
}

function canonicalizeInlineContent(runs: Json[]): Json[] {
  // Sort marks within each run
  const sorted = runs.map((r) => ({
    ...r,
    text: nfc(r.text),
    marks: sortMarks(r.marks),
  }));
  // Merge adjacent runs with identical marks/links/document_refs
  const merged: Json[] = [];
  for (const run of sorted) {
    const last = merged[merged.length - 1];
    if (
      last &&
      marksEqual(last.marks, run.marks) &&
      JSON.stringify(last.link) === JSON.stringify(run.link) &&
      JSON.stringify(last.document_ref) === JSON.stringify(run.document_ref)
    ) {
      last.text += run.text;
    } else {
      merged.push({ ...run });
    }
  }
  // Strip undefined props
  return merged.map((r) => {
    const out: Json = { text: r.text };
    if (r.marks) out.marks = r.marks;
    if (r.link) out.link = r.link;
    if (r.document_ref) out.document_ref = r.document_ref;
    return out;
  });
}

function canonicalizeBlock(block: Json): Json {
  const result: Json = { ...block };

  // For inline-content children (paragraph, heading, listItems, dataTableCell)
  const inlineParents = new Set([
    "paragraph",
    "heading",
    "bulletListItem",
    "numberedListItem",
    "dataTableCell",
  ]);

  if (inlineParents.has(block.type) && Array.isArray(block.children)) {
    result.children = canonicalizeInlineContent(block.children);
  } else if (Array.isArray(block.children)) {
    result.children = block.children.map(canonicalizeBlock);
  }

  // NFC string fields except code blocks
  if (block.type !== "code" && block.props) {
    const titleNfc = block.props.title ? { title: nfc(block.props.title) } : {};
    const labelNfc = block.props.label ? { label: nfc(block.props.label) } : {};
    if (block.props.title || block.props.label) {
      result.props = { ...block.props, ...titleNfc, ...labelNfc };
    }
  }

  return result;
}

export function canonicalizeMDDM(envelope: Json): Json {
  const out: Json = {
    mddm_version: envelope.mddm_version,
    blocks: (envelope.blocks ?? []).map(canonicalizeBlock),
    template_ref: envelope.template_ref ?? null,
  };
  return sortKeys(out);
}
