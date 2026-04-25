import { describe, expect, test } from "vitest";
import { FooterControlledCopyNotice } from "../footer_controlled_copy_notice";

describe("FooterControlledCopyNotice", () => {
  test("uses default notice when params.notice_text missing", async () => {
    const ooxml = await FooterControlledCopyNotice.render({ params: {}, values: {} });
    expect(ooxml).toContain("CONTROLLED COPY — WHEN PRINTED");
  });

  test("uses tenant notice_text override", async () => {
    const ooxml = await FooterControlledCopyNotice.render({
      params: { notice_text: "INTERNAL USE ONLY" },
      values: {},
    });
    expect(ooxml).toContain("INTERNAL USE ONLY");
    expect(ooxml).not.toContain("CONTROLLED COPY");
  });

  test("empty string override falls back to default", async () => {
    const ooxml = await FooterControlledCopyNotice.render({
      params: { notice_text: "" },
      values: {},
    });
    expect(ooxml).toContain("CONTROLLED COPY — WHEN PRINTED");
  });
});
