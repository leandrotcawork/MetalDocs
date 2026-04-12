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

// Single source of truth for registered block types. TypeScript enforces that
// makeRegistry's Record uses exactly these keys — adding a type here without
// wiring the emitter (or vice-versa) is a compile error.
const KNOWN_BLOCK_TYPES = [
  "paragraph", "heading", "section", "field", "fieldGroup",
  "bulletListItem", "numberedListItem", "image", "quote", "divider",
  "dataTable",
  "repeatable", "repeatableItem", "richBlock",
] as const;

type KnownBlockType = typeof KNOWN_BLOCK_TYPES[number];

export const REGISTERED_EMITTER_TYPES: readonly string[] = KNOWN_BLOCK_TYPES;

function makeRegistry(ctx: EmitContext): Record<KnownBlockType, Emitter> {
  // renderChild is captured by closure so structural emitters can recurse
  // through the registry without an import cycle.
  const renderChild = (child: MDDMBlock): unknown[] => {
    const emit = registry[child.type as KnownBlockType];
    if (!emit) throw new MissingEmitterError(child.type);
    return emit(child, ctx);
  };

  const registry: Record<KnownBlockType, Emitter> = {
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

    dataTable: (b, c) => emitDataTable(b, c.tokens),

    repeatable:     (b, c) => emitRepeatable(b, c.tokens, renderChild),
    repeatableItem: (b, c) => emitRepeatableItem(b, c.tokens, renderChild),
    richBlock:      (b, c) => emitRichBlock(b, c.tokens, renderChild),
  };
  return registry;
}

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
    const emit = (registry as Record<string, Emitter | undefined>)[block.type];
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

  const blob = await Packer.toBlob(doc);
  // Re-wrap with explicit DOCX MIME. In some jsdom versions, Blob.arrayBuffer()
  // is absent — fall back to returning the blob directly since Packer.toBlob
  // already sets the correct MIME type.
  if (typeof blob.arrayBuffer !== "function") return blob;
  return new Blob([await blob.arrayBuffer()], { type: DOCX_MIME });
}
