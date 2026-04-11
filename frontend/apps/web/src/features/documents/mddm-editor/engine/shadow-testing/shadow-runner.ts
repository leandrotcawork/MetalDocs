import type { MDDMEnvelope } from "../../adapter";
import type { RendererPin } from "../../../../../lib.types";

const SHADOW_TIMEOUT_MS = 30_000;
const MIN_DEVICE_MEMORY_GB = 4;

export type ShadowResult =
  | { ok: true; xml: string; durationMs: number }
  | { ok: false; error: string; durationMs: number }
  | { ok: false; error: "skipped_low_memory"; durationMs: 0 }
  | { ok: false; error: "timeout"; durationMs: number };

/**
 * Run the new client-side exportDocx in a dedicated Worker and return
 * either the normalized document.xml string or a clearly-marked failure.
 * This function NEVER throws.
 */
export async function runShadowExport(
  envelope: MDDMEnvelope,
  rendererPin: RendererPin | null,
): Promise<ShadowResult> {
  const deviceMemory = (navigator as unknown as { deviceMemory?: number }).deviceMemory;
  if (typeof deviceMemory === "number" && deviceMemory > 0 && deviceMemory < MIN_DEVICE_MEMORY_GB) {
    return { ok: false, error: "skipped_low_memory", durationMs: 0 };
  }

  const worker = new Worker(new URL("./shadow.worker.ts", import.meta.url), { type: "module" });

  return new Promise<ShadowResult>((resolve) => {
    const start = performance.now();
    const timer = setTimeout(() => {
      worker.terminate();
      resolve({ ok: false, error: "timeout", durationMs: Math.round(performance.now() - start) });
    }, SHADOW_TIMEOUT_MS);

    worker.addEventListener("message", (event: MessageEvent) => {
      clearTimeout(timer);
      worker.terminate();
      resolve(event.data as ShadowResult);
    });

    worker.addEventListener("error", (event) => {
      clearTimeout(timer);
      worker.terminate();
      resolve({
        ok: false,
        error: String(event.message ?? "worker error"),
        durationMs: Math.round(performance.now() - start),
      });
    });

    worker.postMessage({ envelope, rendererPin });
  });
}
