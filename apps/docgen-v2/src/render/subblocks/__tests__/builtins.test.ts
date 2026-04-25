import { describe, expect, test } from "vitest";
import { SubBlockRegistry } from "../registry";
import { registerV1Builtins } from "../builtins";

describe("registerV1Builtins", () => {
  test("registers exactly the 5 v1 sub-block keys", () => {
    const reg = new SubBlockRegistry();
    registerV1Builtins(reg);

    const keys = reg.keys().sort();
    expect(keys).toEqual(
      [
        "approval_signatures_block",
        "doc_header_standard",
        "footer_controlled_copy_notice",
        "footer_page_numbers",
        "revision_box",
      ].sort(),
    );
  });
});
