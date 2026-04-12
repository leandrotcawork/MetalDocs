type UnknownRecord = Record<string, unknown>;

export type MDDMMark = {
  type: string;
};

export type MDDMDocumentRef = {
  target_document_id: string;
  target_revision_label?: string;
};

export type MDDMTextRun = {
  type?: "text";
  text: string;
  marks?: MDDMMark[];
  link?: {
    href: string;
    title?: string;
  };
  document_ref?: MDDMDocumentRef;
};

export type MDDMBlock = {
  id: string;
  template_block_id?: string;
  type: string;
  props: UnknownRecord;
  content?: unknown;
  children?: MDDMBlock[] | MDDMTextRun[];
};

export type MDDMEnvelope = {
  mddm_version: number;
  template_ref: unknown;
  blocks: MDDMBlock[];
};

type BlockNoteText = {
  type: "text";
  text: string;
  styles?: Record<string, boolean>;
  __mddm_document_ref?: MDDMDocumentRef;
  __mddm_link_title?: string;
};

type BlockNoteLink = {
  type: "link";
  href: string;
  content: BlockNoteText[];
  __mddm_document_ref?: MDDMDocumentRef;
  __mddm_link_title?: string;
};

type BlockNoteInline = BlockNoteText | BlockNoteLink;

type TableCellContent = { type: "text"; text: string; styles?: Record<string, boolean> };
type TableRow = { cells: TableCellContent[][] };
type TableContent = {
  type: "tableContent";
  columnWidths: (number | null)[];
  headerRows: number;
  rows: TableRow[];
};

type BlockNoteBlock = {
  id?: string;
  type: string;
  props?: UnknownRecord;
  content?: BlockNoteInline[] | TableContent;
  children?: BlockNoteBlock[];
};

type EnvelopeMeta = {
  mddm_version: number;
  template_ref: unknown;
};

type BlockArrayWithMeta = BlockNoteBlock[] & {
  __mddm_envelope_meta__?: EnvelopeMeta;
};

const ENVELOPE_META_KEY = "__mddm_envelope_meta__";

const ALLOWED_MDDM_TYPES = new Set<string>([
  "section",
  "fieldGroup",
  "field",
  "repeatable",
  "repeatableItem",
  "dataTable",
  "richBlock",
  "paragraph",
  "heading",
  "bulletListItem",
  "numberedListItem",
  "image",
  "quote",
  "code",
  "divider",
]);

const INLINE_BLOCK_TYPES = new Set<string>([
  "paragraph",
  "heading",
  "bulletListItem",
  "numberedListItem",
  "field",
  "code",
]);

const LEAF_BLOCK_TYPES = new Set<string>(["image", "divider"]);

const MARK_ORDER = ["bold", "code", "italic", "strike", "underline"];

function normalizeInt(value: unknown, fallback: number): number {
  if (value === null || value === undefined || value === "") {
    return fallback;
  }
  const n = Number(value);
  return Number.isFinite(n) ? Math.trunc(n) : fallback;
}

function newUUID(): string {
  const cryptoValue = (globalThis as any)?.crypto;
  if (cryptoValue && typeof cryptoValue.randomUUID === "function") {
    return cryptoValue.randomUUID();
  }

  // Fallback: RFC4122-ish v4 UUID.
  let out = "";
  for (let i = 0; i < 36; i += 1) {
    if (i === 8 || i === 13 || i === 18 || i === 23) {
      out += "-";
      continue;
    }
    if (i === 14) {
      out += "4";
      continue;
    }
    const r = Math.floor(Math.random() * 16);
    if (i === 19) {
      out += ((r & 0x3) | 0x8).toString(16);
      continue;
    }
    out += r.toString(16);
  }
  return out;
}

export function mddmToBlockNote(envelope: MDDMEnvelope): BlockNoteBlock[] {
  const blocks = envelope.blocks.map(toBlockNoteBlock) as BlockArrayWithMeta;
  blocks[ENVELOPE_META_KEY] = {
    mddm_version: envelope.mddm_version,
    template_ref: envelope.template_ref ?? null,
  };
  return blocks;
}

// Convert old dataTable (with dataTableRow/dataTableCell children) to new tableContent format.
// Returns null if block is already in new format or is not a dataTable.
function migrateOldDataTable(block: MDDMBlock): TableContent | null {
  if (block.type !== "dataTable") return null;
  const children = block.children ?? [];
  if (!Array.isArray(children) || children.length === 0) return null;
  const firstChild = (children as unknown[])[0];
  if (!isRecord(firstChild) || (firstChild as MDDMBlock).type !== "dataTableRow") return null;

  const columns = parseColumns(block.props?.columns ?? block.props?.columnsJson) as Array<{ key: string; label: string }>;

  // Header row: one cell per column with column label
  const headerRow: TableRow = {
    cells: columns.map((col) => [{ type: "text" as const, text: col.label ?? "" }]),
  };

  // Data rows from dataTableRow children
  const dataRows: TableRow[] = (children as MDDMBlock[]).map((row) => {
    const rowCells = (row.children ?? []) as MDDMBlock[];
    const cells: TableCellContent[][] = columns.map((col) => {
      const cell = rowCells.find((c) => asString(c.props?.columnKey) === col.key);
      if (!cell) return [{ type: "text" as const, text: "" }];
      const runs = (cell.children ?? []) as Array<{ text?: string }>;
      const text = runs.map((r) => asString(r.text)).join("");
      return [{ type: "text" as const, text }];
    });
    return { cells };
  });

  return {
    type: "tableContent",
    columnWidths: columns.map(() => null),
    headerRows: 1,
    rows: [headerRow, ...dataRows],
  };
}

function toBlockNoteBlock(block: MDDMBlock): BlockNoteBlock {
  const props = cloneRecord(block.props);
  if (block.template_block_id) {
    props.__template_block_id = block.template_block_id;
  }

  const output: BlockNoteBlock = {
    id: block.id,
    type: toBlockNoteType(block.type),
    props: toBlockNoteProps(block.type, props),
  };

  // DataTable: migrate old children format to tableContent, or pass through new format
  if (block.type === "dataTable") {
    const migrated = migrateOldDataTable(block);
    if (migrated) {
      // Old format detected — convert to tableContent
      output.content = migrated;
      output.children = [];
      return output;
    }
    // New format: block.content is already tableContent
    if (isRecord(block.content) && (block.content as any).type === "tableContent") {
      output.content = block.content as TableContent;
      output.children = [];
      return output;
    }
    // Empty table (new doc from template without rows)
    output.content = {
      type: "tableContent",
      columnWidths: [],
      headerRows: 1,
      rows: [{ cells: [] }],
    };
    output.children = [];
    return output;
  }

  if (block.type === "quote") {
    const quoteChildren = Array.isArray(block.children)
      ? block.children
      : [];
    output.props = output.props ?? {};
    output.props.__quote_children_json = JSON.stringify(quoteChildren);
    output.content = toBlockNoteInline(quoteToInline(quoteChildren));
    return output;
  }

  // multiParagraph fields store their value as child blocks, not inline runs
  if (block.type === "field" && asString(block.props?.valueMode) === "multiParagraph") {
    if (Array.isArray(block.children)) {
      output.children = (block.children as MDDMBlock[]).map(toBlockNoteBlock);
    }
    return output;
  }

  if (INLINE_BLOCK_TYPES.has(block.type)) {
    output.content = toBlockNoteInline(
      Array.isArray(block.children) ? (block.children as MDDMTextRun[]) : [],
    );
    return output;
  }

  if (Array.isArray(block.children)) {
    output.children = (block.children as MDDMBlock[]).map(toBlockNoteBlock);
  }

  return output;
}

export function blockNoteToMDDM(
  blocks: BlockNoteBlock[],
  metaOverride?: Partial<EnvelopeMeta>,
): MDDMEnvelope {
  const meta = resolveEnvelopeMeta(blocks, metaOverride);

  return {
    mddm_version: meta.mddm_version,
    template_ref: meta.template_ref,
    blocks: blocks.map(toMDDMBlock),
  };
}

function toMDDMBlock(block: BlockNoteBlock): MDDMBlock {
  const mddmType = toMDDMType(block.type);
  const rawProps = cloneRecord(block.props);
  const templateBlockID = asOptionalString(rawProps.__template_block_id);
  delete rawProps.__template_block_id;

  const props = toMDDMProps(mddmType, rawProps);

  const output: MDDMBlock = {
    id: asString(block.id),
    type: mddmType,
    props,
  };

  if (templateBlockID) {
    output.template_block_id = templateBlockID;
  }

  // DataTable: store tableContent in content field, not children
  if (mddmType === "dataTable") {
    const tableContent = block.content;
    if (isRecord(tableContent) && (tableContent as any).type === "tableContent") {
      output.content = tableContent;
    }
    output.children = [];
    return output;
  }

  if (mddmType === "quote") {
    output.children = quoteFromBlockNote(block.content, rawProps);
    return output;
  }

  // multiParagraph fields store their value as child blocks, not inline runs
  if (mddmType === "field" && asString(block.props?.valueMode) === "multiParagraph") {
    output.children = Array.isArray(block.children)
      ? block.children.map(toMDDMBlock)
      : [];
    return output;
  }

  if (INLINE_BLOCK_TYPES.has(mddmType)) {
    const inlineRuns = fromBlockNoteInline(block.content);
    output.children =
      mddmType === "code"
        ? inlineRuns.map((run) => ({ ...run, type: "text" as const }))
        : inlineRuns;
    return output;
  }

  if (!LEAF_BLOCK_TYPES.has(mddmType)) {
    output.children = Array.isArray(block.children)
      ? block.children.map(toMDDMBlock)
      : [];
  }

  return output;
}

function toBlockNoteType(mddmType: string): string {
  if (!ALLOWED_MDDM_TYPES.has(mddmType)) {
    throw new Error(`unsupported block type: ${mddmType}`);
  }
  if (mddmType === "code") {
    return "codeBlock";
  }
  return mddmType;
}

function toMDDMType(blockNoteType: string): string {
  const mapped = blockNoteType === "codeBlock" ? "code" : blockNoteType;

  if (!ALLOWED_MDDM_TYPES.has(mapped)) {
    throw new Error(`unsupported block type: ${blockNoteType}`);
  }

  return mapped;
}

function toBlockNoteProps(type: string, props: UnknownRecord): UnknownRecord {
  if (type === "image") {
    const next = cloneRecord(props);
    next.url = asString(props.src);
    next.name = asString(props.alt);
    delete next.src;
    delete next.alt;
    return next;
  }

  if (type === "dataTable") {
    const next = cloneRecord(props);
    // Remove columnsJson — columns are now stored as the header row in tableContent
    delete next.columnsJson;
    delete next.columns;
    return next;
  }

  return props;
}

function toMDDMProps(type: string, props: UnknownRecord): UnknownRecord {
  // BlockNote tends to carry default styling props; MDDM export must be whitelist-only
  // so we don't leak editor defaults into persisted payloads.
  const next = cloneRecord(props);

  switch (type) {
    case "paragraph":
    case "quote":
    case "divider":
      return {};

    case "heading": {
      const level = Number(next.level);
      const normalized =
        Number.isFinite(level) && level >= 1 && level <= 3 ? Math.trunc(level) : 1;
      return { level: normalized };
    }

    case "bulletListItem":
    case "numberedListItem": {
      const level = Number(next.level);
      const normalized = Number.isFinite(level) && level >= 0 ? Math.trunc(level) : 0;
      return { level: normalized };
    }

    case "code":
      return { language: asString(next.language) };

    case "image":
      return {
        src: asString(next.url),
        alt: asString(next.name),
        caption: asString(next.caption),
      };

    case "dataTable":
      return {
        label: asString(next.label),
        locked: Boolean(next.locked),
        density: asString(next.density) || "normal",
      };

    case "field": {
      const valueMode = asOptionalString(next.valueMode);
      if (valueMode && valueMode !== "inline" && valueMode !== "multiParagraph") {
        throw new Error(`unsupported field valueMode: ${valueMode}`);
      }
      const props: UnknownRecord = {
        label: asString(next.label),
        valueMode: valueMode || "inline",
        locked: Boolean(next.locked),
      };
      const hint = asOptionalString(next.hint);
      if (hint) {
        props.hint = hint;
      }
      props.layout = asString(next.layout) || "grid";
      return props;
    }

    case "fieldGroup": {
      const columns = Number(next.columns);
      const normalized = columns === 2 ? 2 : 1;
      return { columns: normalized, locked: Boolean(next.locked) };
    }

    case "section":
      return {
        title: asString(next.title),
        color: asString(next.color),
        locked: Boolean(next.locked),
        ...(next.optional === true ? { optional: true } : {}),
        variant: asString(next.variant) || "bar",
      };

    case "repeatable":
      return {
        label: asString(next.label),
        itemPrefix: asString(next.itemPrefix),
        locked: Boolean(next.locked),
        minItems: normalizeInt(next.minItems, 0),
        maxItems: normalizeInt(next.maxItems, 100),
      };

    case "repeatableItem":
      return {
        title: asString(next.title),
        style: asString(next.style) || "bordered",
      };

    case "richBlock":
      return {
        label: asString(next.label),
        locked: Boolean(next.locked),
        chrome: asString(next.chrome) || "labeled",
      };

    default:
      break;
  }

  // Should be unreachable because `toMDDMType` fail-closes.
  return {};
}

function toBlockNoteInline(runs: MDDMTextRun[]): BlockNoteInline[] {
  return runs.map((run) => {
    const styles = marksToStyles(run.marks);
    const textNode: BlockNoteText = {
      type: "text",
      text: run.text ?? "",
      styles,
    };

    if (run.document_ref) {
      textNode.__mddm_document_ref = run.document_ref;
    }
    if (run.link?.title) {
      textNode.__mddm_link_title = run.link.title;
    }

    if (run.link?.href) {
      const linkNode: BlockNoteLink = {
        type: "link",
        href: run.link.href,
        content: [textNode],
      };
      if (run.document_ref) {
        linkNode.__mddm_document_ref = run.document_ref;
      }
      if (run.link.title) {
        linkNode.__mddm_link_title = run.link.title;
      }
      return linkNode;
    }

    return textNode;
  });
}

function fromBlockNoteInline(content: unknown): MDDMTextRun[] {
  if (!Array.isArray(content)) {
    return [];
  }

  const runs: MDDMTextRun[] = [];
  for (const item of content) {
    if (!isRecord(item)) {
      continue;
    }

    if (item.type === "link" && isString(item.href)) {
      const textNodes = Array.isArray(item.content)
        ? item.content
        : [];
      for (const textNode of textNodes) {
        if (!isRecord(textNode) || textNode.type !== "text") {
          continue;
        }

        const run: MDDMTextRun = {
          text: asString(textNode.text),
        };

        const marks = stylesToMarks(textNode.styles);
        if (marks.length > 0) {
          run.marks = marks;
        }

        run.link = { href: item.href };
        const title = asOptionalString(
          item.__mddm_link_title ?? textNode.__mddm_link_title,
        );
        if (title) {
          run.link.title = title;
        }

        const documentRef = toDocumentRef(
          item.__mddm_document_ref ?? textNode.__mddm_document_ref,
        );
        if (documentRef) {
          run.document_ref = documentRef;
        }

        runs.push(run);
      }
      continue;
    }

    if (item.type === "text") {
      const run: MDDMTextRun = {
        text: asString(item.text),
      };

      const marks = stylesToMarks(item.styles);
      if (marks.length > 0) {
        run.marks = marks;
      }

      const documentRef = toDocumentRef(item.__mddm_document_ref);
      if (documentRef) {
        run.document_ref = documentRef;
      }

      runs.push(run);
    }
  }

  return runs;
}

function quoteToInline(children: unknown[]): MDDMTextRun[] {
  const runs: MDDMTextRun[] = [];
  for (const child of children) {
    if (!isRecord(child) || child.type !== "paragraph") {
      continue;
    }
    if (!Array.isArray(child.children)) {
      continue;
    }
    runs.push(...(child.children as MDDMTextRun[]));
  }
  return runs;
}

function quoteFromBlockNote(
  content: unknown,
  rawProps: UnknownRecord,
): MDDMBlock[] {
  // Source of truth is the live editor content. Metadata is only used to preserve
  // the original paragraph ID when possible, never to preserve stale text.
  const fromMetadata = parseQuoteChildren(rawProps.__quote_children_json);
  const metadataID =
    Array.isArray(fromMetadata) &&
    fromMetadata.length > 0 &&
    isRecord(fromMetadata[0])
      ? asOptionalString((fromMetadata[0] as any).id)
      : undefined;
  const paragraphID =
    metadataID ?? newUUID();

  return [
    {
      id: paragraphID,
      type: "paragraph",
      props: {},
      children: fromBlockNoteInline(content),
    },
  ];
}

function parseQuoteChildren(value: unknown): MDDMBlock[] | undefined {
  if (!isString(value)) {
    return undefined;
  }

  try {
    const parsed = JSON.parse(value);
    return Array.isArray(parsed) ? (parsed as MDDMBlock[]) : undefined;
  } catch {
    return undefined;
  }
}

function parseColumns(value: unknown): unknown[] {
  if (Array.isArray(value)) {
    return value;
  }
  if (!isString(value)) {
    return [];
  }
  try {
    const parsed = JSON.parse(value);
    return Array.isArray(parsed) ? parsed : [];
  } catch {
    return [];
  }
}

function resolveEnvelopeMeta(
  blocks: BlockNoteBlock[],
  metaOverride?: Partial<EnvelopeMeta>,
): EnvelopeMeta {
  const metaFromBlocks = (blocks as BlockArrayWithMeta)[ENVELOPE_META_KEY];
  return {
    mddm_version: Number(
      metaOverride?.mddm_version ??
        metaFromBlocks?.mddm_version ??
        1,
    ),
    template_ref:
      metaOverride?.template_ref ??
      metaFromBlocks?.template_ref ??
      null,
  };
}

function marksToStyles(marks: MDDMMark[] | undefined): Record<string, boolean> {
  const styles: Record<string, boolean> = {};
  for (const mark of marks ?? []) {
    if (!isString(mark?.type)) {
      continue;
    }
    styles[mark.type] = true;
  }
  return styles;
}

function stylesToMarks(stylesValue: unknown): MDDMMark[] {
  if (!isRecord(stylesValue)) {
    return [];
  }

  const marks = Object.entries(stylesValue)
    .filter(([, enabled]) => enabled === true)
    .map(([type]) => ({ type }));

  return marks.sort(compareMarks);
}

function compareMarks(a: MDDMMark, b: MDDMMark): number {
  const left = MARK_ORDER.indexOf(a.type);
  const right = MARK_ORDER.indexOf(b.type);

  if (left === -1 && right === -1) {
    return a.type.localeCompare(b.type);
  }
  if (left === -1) {
    return 1;
  }
  if (right === -1) {
    return -1;
  }

  return left - right;
}

function toDocumentRef(value: unknown): MDDMDocumentRef | undefined {
  if (!isRecord(value)) {
    return undefined;
  }
  if (!isString(value.target_document_id) || value.target_document_id === "") {
    return undefined;
  }
  const result: MDDMDocumentRef = {
    target_document_id: value.target_document_id,
  };
  const revisionLabel = asOptionalString(value.target_revision_label);
  if (revisionLabel) {
    result.target_revision_label = revisionLabel;
  }
  return result;
}

function cloneRecord(value: unknown): UnknownRecord {
  if (!isRecord(value)) {
    return {};
  }
  return { ...value };
}

function asString(value: unknown): string {
  return isString(value) ? value : "";
}

function asOptionalString(value: unknown): string | undefined {
  return isString(value) && value.trim() !== "" ? value : undefined;
}

function isRecord(value: unknown): value is UnknownRecord {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function isString(value: unknown): value is string {
  return typeof value === "string";
}
