import { describe, expect, test } from "vitest";

import { placeholderToRun, type PlaceholderRun } from "../eigenpal-template-mode";

function makePlaceholder(
  placeholderType: PlaceholderRun["placeholderType"],
  options?: string[],
): PlaceholderRun {
  return {
    type: "placeholder",
    id: `${placeholderType ?? "default"}-id`,
    label: `${placeholderType ?? "default"}-label`,
    placeholderType,
    options,
  };
}

describe("placeholderToRun sdt type mapping", () => {
  test("maps placeholder types to expected sdtType", () => {
    expect(placeholderToRun(makePlaceholder("date")).properties.sdtType).toBe("date");
    expect(placeholderToRun(makePlaceholder("select")).properties.sdtType).toBe("dropdown");
    expect(placeholderToRun(makePlaceholder("picture")).properties.sdtType).toBe("picture");
    expect(placeholderToRun(makePlaceholder("computed")).properties.sdtType).toBe("plainText");
    expect(placeholderToRun(makePlaceholder("text")).properties.sdtType).toBe("richText");
    expect(placeholderToRun(makePlaceholder(undefined)).properties.sdtType).toBe("richText");
  });

  test("maps select options to listItems", () => {
    const run = placeholderToRun(makePlaceholder("select", ["One", "Two"]));
    expect(run.properties.listItems).toEqual([
      { displayText: "One", value: "One" },
      { displayText: "Two", value: "Two" },
    ]);
  });

  test("computed placeholders lock content", () => {
    const run = placeholderToRun(makePlaceholder("computed"));
    expect(run.properties.lock).toBe("sdtContentLocked");
  });
});
