/// <reference lib="webworker" />
import { exportDocx } from "../export";
import { unzipDocxDocumentXml } from "../golden/golden-helpers";
import type { MDDMEnvelope } from "../../adapter";
import type { RendererPin } from "../../../../../lib.types";

type ShadowRequest = {
  envelope: MDDMEnvelope;
  rendererPin: RendererPin | null;
};

type ShadowResponse =
  | { ok: true; xml: string; durationMs: number }
  | { ok: false; error: string; durationMs: number };

self.addEventListener("message", async (event: MessageEvent<ShadowRequest>) => {
  const start = performance.now();
  try {
    const blob = await exportDocx(event.data.envelope, { rendererPin: event.data.rendererPin });
    const xml = await unzipDocxDocumentXml(blob);
    const durationMs = Math.round(performance.now() - start);
    const response: ShadowResponse = { ok: true, xml, durationMs };
    (self as unknown as Worker).postMessage(response);
  } catch (err) {
    const durationMs = Math.round(performance.now() - start);
    const response: ShadowResponse = {
      ok: false,
      error: err instanceof Error ? err.message : String(err),
      durationMs,
    };
    (self as unknown as Worker).postMessage(response);
  }
});
