import { ImageRun, Paragraph } from "docx";
import type { MDDMBlock } from "./types.js";

export type ImageFetcher = (src: string) => Promise<{ bytes: Uint8Array; mime: string }>;

export async function renderImage(block: MDDMBlock, fetcher: ImageFetcher): Promise<Paragraph> {
  const src = block.props.src as string;
  try {
    const { bytes } = await fetcher(src);
    return new Paragraph({
      children: [
        new ImageRun({
          data: bytes,
          transformation: {
            width: 400,
            height: 300,
          },
        } as any),
      ],
    });
  } catch {
    return new Paragraph({
      children: [],
    });
  }
}
