import { describe, expect, test } from "vitest";

import { placeholderToRun, type PlaceholderRun } from "../eigenpal-template-mode";

describe("inline SDT round-trip", () => {
  test("date placeholder maps to sdtType=date, preserves tag and alias", () => {
    const p: PlaceholderRun = { type: "placeholder", id: "p1", label: "Effective Date", placeholderType: "date" };
    const node = placeholderToRun(p);
    expect(node.type).toBe("inlineSdt");
    expect(node.properties.sdtType).toBe("date");
    expect(node.properties.tag).toBe("placeholder:p1");
    expect(node.properties.alias).toBe("Effective Date");
  });

  test("select placeholder maps to sdtType=dropdown and preserves listItems", () => {
    const p: PlaceholderRun = {
      type: "placeholder",
      id: "status",
      label: "Status",
      placeholderType: "select",
      options: ["Draft", "Active"],
    };
    const node = placeholderToRun(p);
    expect(node.properties.sdtType).toBe("dropdown");
    expect(node.properties.listItems).toEqual([
      { displayText: "Draft", value: "Draft" },
      { displayText: "Active", value: "Active" },
    ]);
  });

  test("computed placeholder maps to sdtType=plainText and lock=sdtContentLocked", () => {
    const p: PlaceholderRun = { type: "placeholder", id: "rev", label: "Rev No", placeholderType: "computed" };
    const node = placeholderToRun(p);
    expect(node.properties.sdtType).toBe("plainText");
    expect(node.properties.lock).toBe("sdtContentLocked");
  });

  test("text placeholder maps to sdtType=richText with no lock", () => {
    const p: PlaceholderRun = { type: "placeholder", id: "desc", label: "Description", placeholderType: "text" };
    const node = placeholderToRun(p);
    expect(node.properties.sdtType).toBe("richText");
    expect(node.properties.lock).toBeUndefined();
  });

  test("picture placeholder maps to sdtType=picture", () => {
    const p: PlaceholderRun = { type: "placeholder", id: "logo", label: "Logo", placeholderType: "picture" };
    const node = placeholderToRun(p);
    expect(node.properties.sdtType).toBe("picture");
  });

  test("tag uses placeholder: prefix and alias matches label exactly", () => {
    const p: PlaceholderRun = { type: "placeholder", id: "my-id", label: "My Label" };
    const node = placeholderToRun(p);
    expect(node.properties.tag).toBe("placeholder:my-id");
    expect(node.properties.alias).toBe("My Label");
  });
});
