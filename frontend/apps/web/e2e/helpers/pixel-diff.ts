import { PNG } from "pngjs";
import pixelmatch from "pixelmatch";
// pdf-img-convert wraps pdfjs-dist with a Node canvas backend so we don't
// have to configure a canvas factory manually.
// eslint-disable-next-line @typescript-eslint/ban-ts-comment
// @ts-expect-error — pdf-img-convert ships without types
import * as pdfImgConvert from "pdf-img-convert";

/** Render the first page of a PDF Buffer to a PNG Buffer. */
export async function rasterizePdfFirstPageToPng(pdfBytes: Uint8Array): Promise<Buffer> {
  // pdf-img-convert returns an array of PNG Uint8Arrays, one per page.
  const images = (await pdfImgConvert.convert(Buffer.from(pdfBytes), {
    page_numbers: [1],
    scale: 2.0, // render at 2x for better diff resolution
  })) as Uint8Array[];

  if (!images || images.length === 0) {
    throw new Error("pdf-img-convert returned no images");
  }
  return Buffer.from(images[0]!);
}

/** Compare two PNG buffers; returns the fraction of differing pixels (0..1).
 *  Buffers are resampled to the smaller dimensions before comparison. */
export function pngDiffPercent(a: Buffer, b: Buffer): number {
  const left = PNG.sync.read(a);
  const right = PNG.sync.read(b);
  const width = Math.min(left.width, right.width);
  const height = Math.min(left.height, right.height);
  if (width === 0 || height === 0) {
    throw new Error("pngDiffPercent: empty image");
  }

  // If dimensions differ, crop both to the shared region before diffing.
  function crop(png: PNG, w: number, h: number): Buffer {
    if (png.width === w && png.height === h) return png.data as unknown as Buffer;
    const cropped = new PNG({ width: w, height: h });
    PNG.bitblt(png, cropped, 0, 0, w, h, 0, 0);
    return cropped.data as unknown as Buffer;
  }

  const leftData = crop(left, width, height);
  const rightData = crop(right, width, height);

  const diff = new PNG({ width, height });
  const numDiff = pixelmatch(leftData, rightData, diff.data, width, height, {
    threshold: 0.1,
  });
  return numDiff / (width * height);
}
