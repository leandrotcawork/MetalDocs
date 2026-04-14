import { createCanvas } from "@napi-rs/canvas";
import { PNG } from "pngjs";
import pixelmatch from "pixelmatch";
import * as pdfjsLib from "pdfjs-dist/legacy/build/pdf.mjs";

/** Render the first page of a PDF Buffer to a PNG Buffer. */
export async function rasterizePdfFirstPageToPng(pdfBytes: Uint8Array): Promise<Buffer> {
  const task = (pdfjsLib as any).getDocument({
    data: pdfBytes,
    disableWorker: true,
  });

  const pdf = await task.promise;
  const firstPage = await pdf.getPage(1);
  const viewport = firstPage.getViewport({ scale: 2 });

  const canvas = createCanvas(Math.ceil(viewport.width), Math.ceil(viewport.height));
  const ctx = canvas.getContext("2d");
  await firstPage.render({
    canvasContext: ctx as any,
    viewport,
  }).promise;

  return Buffer.from(canvas.toBuffer("image/png"));
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
