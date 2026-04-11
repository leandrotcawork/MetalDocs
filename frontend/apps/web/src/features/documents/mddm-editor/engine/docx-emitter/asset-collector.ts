import type { MDDMEnvelope, MDDMBlock } from "../../adapter";

function isMDDMBlock(child: unknown): child is MDDMBlock {
  return (
    child !== null &&
    typeof child === "object" &&
    typeof (child as MDDMBlock).type === "string" &&
    !("text" in (child as Record<string, unknown>))
  );
}

function walkBlock(block: MDDMBlock, urls: Set<string>): void {
  if (block.type === "image") {
    const src = (block.props as { src?: unknown }).src;
    if (typeof src === "string" && src.length > 0) {
      urls.add(src);
    }
  }
  const children = block.children ?? [];
  for (const child of children) {
    if (isMDDMBlock(child)) {
      walkBlock(child, urls);
    }
  }
}

/**
 * Walk an MDDM envelope and return a deduplicated list of image URLs.
 * Order matches depth-first walk order for deterministic golden testing.
 */
export function collectImageUrls(envelope: MDDMEnvelope): string[] {
  const urls = new Set<string>();
  for (const block of envelope.blocks ?? []) {
    walkBlock(block, urls);
  }
  return Array.from(urls);
}
