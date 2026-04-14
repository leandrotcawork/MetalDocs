import { describe, it, expect } from "vitest";
import { CodecStrictError } from "../codec-utils";
import { parseSectionStyleStrict, parseSectionCapsStrict } from "../section-codec";
import { parseDataTableStyleStrict, parseDataTableCapsStrict } from "../data-table-codec";
import { parseRepeatableStyleStrict, parseRepeatableCapsStrict } from "../repeatable-codec";
import { parseRepeatableItemStyleStrict, parseRepeatableItemCapsStrict } from "../repeatable-item-codec";
import { parseRichBlockStyleStrict, parseRichBlockCapsStrict } from "../rich-block-codec";
import { validateTemplate } from "../validate-template";

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function expectStrictError(fn: () => unknown, fieldFragment: string) {
  let caught: unknown;
  try {
    fn();
  } catch (err) {
    caught = err;
  }
  expect(caught).toBeInstanceOf(CodecStrictError);
  expect((caught as CodecStrictError).field).toContain(fieldFragment);
}

// ---------------------------------------------------------------------------
// section-codec strict
// ---------------------------------------------------------------------------

describe("parseSectionStyleStrict", () => {
  it("passes with a valid style object", () => {
    const result = parseSectionStyleStrict({
      headerHeight: "48px",
      headerBackground: "#fff",
      headerColor: "#000",
      headerFontSize: "16px",
      headerFontWeight: "bold",
    });
    expect(result.headerHeight).toBe("48px");
  });

  it("passes with an empty object (all fields optional)", () => {
    expect(() => parseSectionStyleStrict({})).not.toThrow();
  });

  it("throws CodecStrictError for an unknown field", () => {
    expectStrictError(() => parseSectionStyleStrict({ unknownField: "x" }), "unknownField");
  });
});

describe("parseSectionCapsStrict", () => {
  const validCaps = { locked: true, removable: false, reorderable: false };

  it("passes with valid caps", () => {
    const result = parseSectionCapsStrict(validCaps);
    expect(result.locked).toBe(true);
  });

  it("throws CodecStrictError for missing required field", () => {
    expectStrictError(() => parseSectionCapsStrict({ locked: true, removable: false }), "reorderable");
  });

  it("throws CodecStrictError for unknown field", () => {
    expectStrictError(
      () => parseSectionCapsStrict({ ...validCaps, extraField: true }),
      "extraField",
    );
  });

  it("throws CodecStrictError when boolean field has wrong type", () => {
    expectStrictError(
      () => parseSectionCapsStrict({ locked: "yes", removable: false, reorderable: false }),
      "locked",
    );
  });
});

// ---------------------------------------------------------------------------
// data-table-codec strict
// ---------------------------------------------------------------------------

describe("parseDataTableStyleStrict", () => {
  it("passes with valid style", () => {
    const result = parseDataTableStyleStrict({ headerBackground: "#eee", density: "compact" });
    expect(result.density).toBe("compact");
  });

  it("throws for unknown field", () => {
    expectStrictError(() => parseDataTableStyleStrict({ badField: "x" }), "badField");
  });

  it("throws for invalid density value", () => {
    expectStrictError(() => parseDataTableStyleStrict({ density: "ultra" }), "density");
  });
});

describe("parseDataTableCapsStrict", () => {
  const validCaps = {
    locked: false,
    removable: false,
    mode: "dynamic",
    editableZones: ["cells"],
    addRows: true,
    removeRows: true,
    addColumns: false,
    removeColumns: false,
    resizeColumns: false,
    headerLocked: true,
    maxRows: 100,
  };

  it("passes with valid caps", () => {
    const result = parseDataTableCapsStrict(validCaps);
    expect(result.mode).toBe("dynamic");
    expect(result.maxRows).toBe(100);
  });

  it("throws for unknown field", () => {
    expectStrictError(() => parseDataTableCapsStrict({ ...validCaps, mystery: true }), "mystery");
  });

  it("throws for invalid mode value", () => {
    expectStrictError(() => parseDataTableCapsStrict({ ...validCaps, mode: "turbo" }), "mode");
  });

  it("throws for missing required field (maxRows)", () => {
    const { maxRows: _omit, ...rest } = validCaps;
    expectStrictError(() => parseDataTableCapsStrict(rest), "maxRows");
  });
});

// ---------------------------------------------------------------------------
// repeatable-codec strict
// ---------------------------------------------------------------------------

describe("parseRepeatableStyleStrict", () => {
  it("passes with empty object", () => {
    expect(() => parseRepeatableStyleStrict({})).not.toThrow();
  });

  it("throws for unknown field", () => {
    expectStrictError(() => parseRepeatableStyleStrict({ rogue: "x" }), "rogue");
  });
});

describe("parseRepeatableCapsStrict", () => {
  const validCaps = { locked: true, removable: false, addItems: true, removeItems: true, maxItems: 100, minItems: 0 };

  it("passes with valid caps", () => {
    const result = parseRepeatableCapsStrict(validCaps);
    expect(result.maxItems).toBe(100);
  });

  it("throws for unknown field", () => {
    expectStrictError(() => parseRepeatableCapsStrict({ ...validCaps, extra: 1 }), "extra");
  });

  it("throws for missing minItems", () => {
    const { minItems: _omit, ...rest } = validCaps;
    expectStrictError(() => parseRepeatableCapsStrict(rest), "minItems");
  });
});

// ---------------------------------------------------------------------------
// repeatable-item-codec strict
// ---------------------------------------------------------------------------

describe("parseRepeatableItemStyleStrict", () => {
  it("passes with valid style", () => {
    const result = parseRepeatableItemStyleStrict({ accentBorderColor: "#f00" });
    expect(result.accentBorderColor).toBe("#f00");
  });

  it("throws for unknown field", () => {
    expectStrictError(() => parseRepeatableItemStyleStrict({ shadow: "none" }), "shadow");
  });
});

describe("parseRepeatableItemCapsStrict", () => {
  const validCaps = { locked: false, removable: true, editableZones: ["content"] };

  it("passes with valid caps", () => {
    const result = parseRepeatableItemCapsStrict(validCaps);
    expect(result.editableZones).toEqual(["content"]);
  });

  it("throws for unknown field", () => {
    expectStrictError(() => parseRepeatableItemCapsStrict({ ...validCaps, hidden: true }), "hidden");
  });

  it("throws for non-array editableZones", () => {
    expectStrictError(
      () => parseRepeatableItemCapsStrict({ ...validCaps, editableZones: "all" }),
      "editableZones",
    );
  });
});

// ---------------------------------------------------------------------------
// rich-block-codec strict
// ---------------------------------------------------------------------------

describe("parseRichBlockStyleStrict", () => {
  it("passes with empty object", () => {
    expect(() => parseRichBlockStyleStrict({})).not.toThrow();
  });

  it("throws for unknown field", () => {
    expectStrictError(() => parseRichBlockStyleStrict({ ghost: "true" }), "ghost");
  });
});

describe("parseRichBlockCapsStrict", () => {
  const validCaps = { locked: true, removable: false, editableZones: ["content"] };

  it("passes with valid caps", () => {
    const result = parseRichBlockCapsStrict(validCaps);
    expect(result.locked).toBe(true);
  });

  it("throws for unknown field", () => {
    expectStrictError(() => parseRichBlockCapsStrict({ ...validCaps, canDelete: false }), "canDelete");
  });
});

// ---------------------------------------------------------------------------
// validateTemplate
// ---------------------------------------------------------------------------

describe("validateTemplate", () => {
  it("returns [] for a valid section block", () => {
    const blocks = [
      {
        id: "block-1",
        type: "section",
        props: {
          style: { headerHeight: "48px" },
          caps: { locked: true, removable: false, reorderable: false },
        },
      },
    ];
    expect(validateTemplate(blocks)).toEqual([]);
  });

  it("returns error for unknown block type", () => {
    const blocks = [{ id: "b1", type: "unknownType", props: {} }];
    const errors = validateTemplate(blocks);
    expect(errors).toHaveLength(1);
    expect(errors[0].blockType).toBe("unknownType");
    expect(errors[0].field).toBe("type");
  });

  it("returns error for unknown style field in section", () => {
    const blocks = [
      {
        id: "b2",
        type: "section",
        props: {
          style: { headerHeight: "48px", rogueField: "oops" },
          caps: { locked: true, removable: false, reorderable: false },
        },
      },
    ];
    const errors = validateTemplate(blocks);
    expect(errors).toHaveLength(1);
    expect(errors[0].field).toContain("rogueField");
  });

  it("recurses into children and collects errors", () => {
    const blocks = [
      {
        id: "parent",
        type: "section",
        props: {
          style: {},
          caps: { locked: true, removable: false, reorderable: false },
        },
        children: [
          { id: "child", type: "INVALID_TYPE", props: {} },
        ],
      },
    ];
    const errors = validateTemplate(blocks);
    expect(errors.some((e) => e.blockId === "child")).toBe(true);
  });

  it("returns [] for valid dataTable block", () => {
    const blocks = [
      {
        id: "dt-1",
        type: "dataTable",
        props: {
          style: { density: "normal" },
          caps: {
            locked: false,
            removable: false,
            mode: "dynamic",
            editableZones: ["cells"],
            addRows: true,
            removeRows: true,
            addColumns: false,
            removeColumns: false,
            resizeColumns: false,
            headerLocked: true,
            maxRows: 50,
          },
        },
      },
    ];
    expect(validateTemplate(blocks)).toEqual([]);
  });

  it("returns [] for valid repeatable block", () => {
    const blocks = [
      {
        id: "rep-1",
        type: "repeatable",
        props: {
          style: {},
          caps: { locked: true, removable: false, addItems: true, removeItems: true, maxItems: 10, minItems: 1 },
        },
      },
    ];
    expect(validateTemplate(blocks)).toEqual([]);
  });
});
