import { Document, Packer } from "docx";
import type { MDDMEnvelope, MDDMBlock } from "../../adapter";
import type { LayoutTokens } from "../layout-ir";
import type { ResolvedAsset } from "../asset-resolver";
import { mmToTwip } from "../helpers/units";

import { emitParagraph } from "./emitters/paragraph";
import { emitHeading } from "./emitters/heading";
import { emitSection } from "./emitters/section";
import { emitField } from "./emitters/field";
import { emitFieldGroup } from "./emitters/field-group";
import { emitBulletListItem } from "./emitters/bullet-list-item";
import { emitNumberedListItem, MDDM_NUMBERING_REF } from "./emitters/numbered-list-item";
import { emitImage } from "./emitters/image";
import { emitQuote } from "./emitters/quote";
import { emitDivider } from "./emitters/divider";
import { emitDataTable } from "./emitters/data-table";
import { emitDataTableRow } from "./emitters/data-table-row";
import { emitDataTableCell } from "./emitters/data-table-cell";
import { emitRepeatable } from "./emitters/repeatable";
import { emitRepeatableItem } from "./emitters/repeatable-item";
import { emitRichBlock } from "./emitters/rich-block";

const DOCX_MIME = "application/vnd.openxmlformats-officedocument.wordprocessingml.document";

export class MissingEmitterError extends Error {
  constructor(public readonly blockType: string) {
    super(`No DOCX emitter registered for block type "${blockType}"`);
    this.name = "MissingEmitterError";
  }
}

export type EmitContext = {
  tokens: LayoutTokens;
  assetMap: ReadonlyMap<string, ResolvedAsset>;
};

type Emitter = (block: MDDMBlock, ctx: EmitContext) => unknown[];

function makeRegistry(ctx: EmitContext): Record<string, Emitter> {
  // renderChild is captured by closure so structural emitters can recurse
  // through the registry without an import cycle.
  const renderChild = (child: MDDMBlock): unknown[] => {
    const emit = registry[child.type];
    if (!emit) throw new MissingEmitterError(child.type);
    return emit(child, ctx);
  };

  const registry: Record<string, Emitter> = {
    paragraph: (b, c) => emitParagraph(b, c.tokens),
    heading:   (b, c) => emitHeading(b, c.tokens),
    section:   (b, c) => emitSection(b, c.tokens),
    field:     (b, c) => emitField(b, c.tokens),
    fieldGroup: (b, c) => emitFieldGroup(b, c.tokens),

    bulletListItem:   (b, c) => emitBulletListItem(b, c.tokens),
    numberedListItem: (b, c) => emitNumberedListItem(b, c.tokens),
    image:            (b, c) => emitImage(b, c.tokens, c.assetMap),
    quote:            (b, c) => emitQuote(b, c.tokens),
    divider:          (b, c) => emitDivider(b, c.tokens),

    dataTable:     (b, c) => emitDataTable(b, c.tokens),
    dataTableRow:  (b, c) => [emitDataTableRow(b, c.tokens)],
    dataTableCell: (b, c) => [emitDataTableCell(b, c.tokens)],

    repeatable:     (b, c) => emitRepeatable(b, c.tokens, renderChild),
    repeatableItem: (b, c) => emitRepeatableItem(b, c.tokens, renderChild),
    richBlock:      (b, c) => emitRichBlock(b, c.tokens, renderChild),
  };
  return registry;
}

export const REGISTERED_EMITTER_TYPES: readonly string[] = [
  "paragraph", "heading", "section", "field", "fieldGroup",
  "bulletListItem", "numberedListItem", "image", "quote", "divider",
  "dataTable", "dataTableRow", "dataTableCell",
  "repeatable", "repeatableItem", "richBlock",
];

export async function mddmToDocx(
  envelope: MDDMEnvelope,
  tokens: LayoutTokens,
  assetMap: ReadonlyMap<string, ResolvedAsset> = new Map(),
): Promise<Blob> {
  const ctx: EmitContext = { tokens, assetMap };
  const registry = makeRegistry(ctx);

  const blocks = envelope.blocks ?? [];
  const children: unknown[] = [];

  for (const block of blocks) {
    const emit = registry[block.type];
    if (!emit) {
      throw new MissingEmitterError(block.type);
    }
    children.push(...emit(block, ctx));
  }

  const doc = new Document({
    numbering: {
      config: [
        {
          reference: MDDM_NUMBERING_REF,
          levels: [
            {
              level: 0,
              format: "decimal" as any,
              text: "%1.",
              alignment: "left" as any,
            },
          ],
        },
      ],
    },
    sections: [
      {
        properties: {
          page: {
            size: {
              width: mmToTwip(tokens.page.widthMm),
              height: mmToTwip(tokens.page.heightMm),
            },
            margin: {
              top: mmToTwip(tokens.page.marginTopMm),
              right: mmToTwip(tokens.page.marginRightMm),
              bottom: mmToTwip(tokens.page.marginBottomMm),
              left: mmToTwip(tokens.page.marginLeftMm),
            },
          },
        },
        children: children as any,
      },
    ],
  });

  // Packer.toBuffer works in Node/jsdom (Vitest) but JSZip does not support
  // "nodebuffer" in real browsers, so it throws there. Packer.toBlob is the
  // browser-native path but jsdom's Blob lacks arrayBuffer(), making it
  // unreliable in tests. Strategy: try toBuffer first; if it throws the
  // "nodebuffer is not supported" error, fall back to Packer.toBlob.
  try {
    const buffer = (await Packer.toBuffer(doc)) as Uint8Array;
    const bytes = new Uint8Array(buffer.byteLength);
    bytes.set(buffer);
    return new Blob([bytes], { type: DOCX_MIME });
  } catch (err) {
    if (
      err instanceof Error &&
      err.message.includes("nodebuffer is not supported")
    ) {
      const blob = await Packer.toBlob(doc);
      if (blob.type === DOCX_MIME) return blob;
      const raw = await blob.arrayBuffer();
      return new Blob([raw], { type: DOCX_MIME });
    }
    throw err;
  }
}
