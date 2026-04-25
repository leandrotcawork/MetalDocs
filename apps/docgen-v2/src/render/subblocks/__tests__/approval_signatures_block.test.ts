import { describe, expect, test } from "vitest";
import { ApprovalSignaturesBlock } from "../approval_signatures_block";

describe("ApprovalSignaturesBlock", () => {
  test("renders one row per approver", async () => {
    const ooxml = await ApprovalSignaturesBlock.render({
      params: {},
      values: {
        approvers: [
          { user_id: "u-1", display_name: "Alice Smith", signed_at: "2026-04-20T14:00:00Z" },
          { user_id: "u-2", display_name: "Bob Lee", signed_at: "2026-04-21T09:30:00Z" },
        ],
      },
    });

    expect(ooxml).toContain("<w:tbl>");
    expect(ooxml).toContain("Alice Smith");
    expect(ooxml).toContain("Bob Lee");
    expect(ooxml).toContain("2026-04-20T14:00:00Z");
    expect(ooxml.match(/<w:tr>/g)?.length).toBe(3);
  });

  test("empty approvers renders header only", async () => {
    const ooxml = await ApprovalSignaturesBlock.render({
      params: {},
      values: { approvers: [] },
    });
    expect(ooxml.match(/<w:tr>/g)?.length).toBe(1);
  });

  test("missing approvers treated as empty", async () => {
    const ooxml = await ApprovalSignaturesBlock.render({ params: {}, values: {} });
    expect(ooxml).toContain("<w:tbl>");
    expect(ooxml.match(/<w:tr>/g)?.length).toBe(1);
  });
});
