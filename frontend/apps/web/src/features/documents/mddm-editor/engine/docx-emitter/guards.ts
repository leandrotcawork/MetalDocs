import type { MDDMBlock } from "../../adapter";

/**
 * Type guard that distinguishes an MDDMBlock child node from an MDDMTextRun.
 * Text runs have a `text` property; block nodes have a `type` string and do not.
 */
export function isMDDMBlock(child: unknown): child is MDDMBlock {
  return (
    child !== null &&
    typeof child === "object" &&
    typeof (child as MDDMBlock).type === "string" &&
    !("text" in (child as Record<string, unknown>))
  );
}
