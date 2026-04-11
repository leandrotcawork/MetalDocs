import type { LayoutTokens } from "../layout-ir";
import type { RendererPin } from "../../../../../lib.types";

export type LoadedRenderer = {
  rendererVersion: string;
  layoutIRHash: string;
  tokens: LayoutTokens;
  mddmToDocx: typeof import("../docx-emitter").mddmToDocx;
  wrapInPrintDocument: typeof import("../print-stylesheet").wrapInPrintDocument;
  printStylesheet: string;
};

export class RendererBundleNotFoundError extends Error {
  constructor(public readonly rendererVersion: string) {
    super(`No renderer bundle registered for version "${rendererVersion}"`);
    this.name = "RendererBundleNotFoundError";
  }
}

async function fromBundle(bundle: typeof import("./v1.0.0/index")): Promise<LoadedRenderer> {
  return {
    rendererVersion: bundle.BUNDLE_RENDERER_VERSION,
    layoutIRHash: bundle.BUNDLE_LAYOUT_IR_HASH,
    tokens: bundle.defaultLayoutTokens,
    mddmToDocx: bundle.mddmToDocx,
    wrapInPrintDocument: bundle.wrapInPrintDocument,
    printStylesheet: bundle.PRINT_STYLESHEET,
  };
}

export async function loadCurrentRenderer(): Promise<LoadedRenderer> {
  const bundle = await import("./current");
  return fromBundle(bundle as unknown as typeof import("./v1.0.0/index"));
}

export async function loadPinnedRenderer(pin: RendererPin): Promise<LoadedRenderer> {
  switch (pin.renderer_version) {
    case "1.0.0":
      return fromBundle(await import("./v1.0.0/index"));
    default:
      throw new RendererBundleNotFoundError(pin.renderer_version);
  }
}
