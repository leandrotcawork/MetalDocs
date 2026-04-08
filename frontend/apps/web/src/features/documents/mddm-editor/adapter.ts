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

type BlockNoteBlock = {
  id?: string;
  type: string;
  props?: UnknownRecord;
  content?: BlockNoteInline[];
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

const INLINE_BLOCK_TYPES = new Set<string>([
  "paragraph",
  "heading",
  "bulletListItem",
  "numberedListItem",
  "dataTableCell",
  "code",
]);

const LEAF_BLOCK_TYPES = new Set<string>(["image", "divider"]);

const MARK_ORDER = ["bold", "code", "italic", "strike", "underline"];

export function mddmToBlockNote(envelope: MDDMEnvelope): BlockNoteBlock[] {
  const blocks = envelope.blocks.map(toBlockNoteBlock) as BlockArrayWithMeta;
  blocks[ENVELOPE_META_KEY] = {
    mddm_version: envelope.mddm_version,
    template_ref: envelope.template_ref ?? null,
  };
  return blocks;
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

  if (block.type === "quote") {
    const quoteChildren = Array.isArray(block.children)
      ? block.children
      : [];
    output.props = output.props ?? {};
    output.props.__quote_children_json = JSON.stringify(quoteChildren);
    output.content = toBlockNoteInline(quoteToInline(quoteChildren));
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

  if (mddmType === "quote") {
    output.children = quoteFromBlockNote(block.content, rawProps);
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
  if (mddmType === "code") {
    return "codeBlock";
  }
  return mddmType;
}

function toMDDMType(blockNoteType: string): string {
  if (blockNoteType === "codeBlock") {
    return "code";
  }
  return blockNoteType;
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
    if (Array.isArray(next.columns)) {
      next.columnsJson = JSON.stringify(next.columns);
      delete next.columns;
    } else if (!isString(next.columnsJson)) {
      next.columnsJson = "[]";
    }
    return next;
  }

  return props;
}

function toMDDMProps(type: string, props: UnknownRecord): UnknownRecord {
  const next = cloneRecord(props);

  if (type === "image") {
    next.src = asString(next.url);
    next.alt = asString(next.alt ?? next.name);
    next.caption = asString(next.caption);
    delete next.url;
    delete next.name;
    delete next.showPreview;
    delete next.previewWidth;
  }

  if (type === "dataTable") {
    next.columns = parseColumns(next.columnsJson);
    delete next.columnsJson;
  }

  if (type === "quote") {
    delete next.__quote_children_json;
  }

  return next;
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
  const fromMetadata = parseQuoteChildren(rawProps.__quote_children_json);
  if (fromMetadata) {
    return fromMetadata;
  }

  return [
    {
      id: "",
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
