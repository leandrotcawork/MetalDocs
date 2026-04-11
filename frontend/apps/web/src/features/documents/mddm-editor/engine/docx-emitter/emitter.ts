import { Document, Packer } from "docx";
import type { MDDMEnvelope, MDDMBlock } from "../../adapter";
import type { LayoutTokens } from "../layout-ir";
import { mmToTwip } from "../helpers/units";
import { emitParagraph } from "./emitters/paragraph";
import { emitHeading } from "./emitters/heading";
import { emitSection } from "./emitters/section";
import { emitField } from "./emitters/field";
import { emitFieldGroup } from "./emitters/field-group";

const DOCX_MIME =
  "application/vnd.openxmlformats-officedocument.wordprocessingml.document";

export class MissingEmitterError extends Error {
  constructor(public readonly blockType: string) {
    super(`No DOCX emitter registered for block type "${blockType}"`);
    this.name = "MissingEmitterError";
  }
}

type Emitter = (block: MDDMBlock, tokens: LayoutTokens) => unknown[];

const emitters: Record<string, Emitter> = {
  paragraph: emitParagraph,
  heading: emitHeading,
  section: emitSection,
  field: emitField,
  fieldGroup: emitFieldGroup,
};

export async function mddmToDocx(
  envelope: MDDMEnvelope,
  tokens: LayoutTokens,
): Promise<Blob> {
  const blocks = envelope.blocks ?? [];
  const children: unknown[] = [];

  for (const block of blocks) {
    const emit = emitters[block.type];
    if (!emit) {
      throw new MissingEmitterError(block.type);
    }
    const out = emit(block, tokens);
    children.push(...out);
  }

  const doc = new Document({
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
        children: children as never,
      },
    ],
  });

  // Use Packer.toBuffer so the result is an environment-agnostic byte array.
  // Packer.toBlob relies on a browser Blob implementation that jsdom does not
  // fully provide (no arrayBuffer()), so we construct the Blob ourselves with
  // the correct DOCX MIME type.
  const buffer = (await Packer.toBuffer(doc)) as Uint8Array;
  // Copy into a fresh ArrayBuffer-backed Uint8Array so the Blob constructor
  // gets a concrete BlobPart regardless of whether docx.js returned a Node
  // Buffer (Uint8Array<ArrayBufferLike>) or a plain Uint8Array.
  const bytes = new Uint8Array(buffer.byteLength);
  bytes.set(buffer);
  return new Blob([bytes], { type: DOCX_MIME });
}
