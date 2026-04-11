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

export const REGISTERED_EMITTER_TYPES: readonly string[] = Object.keys(emitters);

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
