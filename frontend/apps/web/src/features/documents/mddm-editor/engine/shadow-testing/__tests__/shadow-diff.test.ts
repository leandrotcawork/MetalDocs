import { describe, expect, it } from "vitest";
import { computeShadowDiff, hashNormalizedXml } from "../shadow-diff";

describe("computeShadowDiff", () => {
  it("reports zero drift for identical XML", () => {
    const xml = `<w:document><w:body><w:p><w:r><w:t>hello</w:t></w:r></w:p></w:body></w:document>`;
    const diff = computeShadowDiff(xml, xml);
    expect(diff.current_xml_hash).toBe(diff.shadow_xml_hash);
    expect(diff.diff_summary.identical).toBe(true);
  });

  it("reports drift for different XML", () => {
    const a = `<w:document><w:body><w:p><w:r><w:t>A</w:t></w:r></w:p></w:body></w:document>`;
    const b = `<w:document><w:body><w:p><w:r><w:t>B</w:t></w:r></w:p></w:body></w:document>`;
    const diff = computeShadowDiff(a, b);
    expect(diff.current_xml_hash).not.toBe(diff.shadow_xml_hash);
    expect(diff.diff_summary.identical).toBe(false);
  });

  it("strips rsid attributes before hashing", () => {
    const a = `<w:p w:rsidR="1234" w:rsidRDefault="5678"><w:r><w:t>x</w:t></w:r></w:p>`;
    const b = `<w:p w:rsidR="abcd" w:rsidRDefault="efgh"><w:r><w:t>x</w:t></w:r></w:p>`;
    const diff = computeShadowDiff(a, b);
    expect(diff.current_xml_hash).toBe(diff.shadow_xml_hash);
    expect(diff.diff_summary.identical).toBe(true);
  });

  it("hashNormalizedXml is deterministic", async () => {
    const xml = `<w:p><w:r><w:t>same</w:t></w:r></w:p>`;
    const h1 = await hashNormalizedXml(xml);
    const h2 = await hashNormalizedXml(xml);
    expect(h1).toBe(h2);
    expect(h1).toMatch(/^[0-9a-f]{64}$/);
  });
});
