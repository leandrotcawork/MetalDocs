# MDDM Template Engine Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace hardcoded Go templates with a declarative template engine. Templates are MDDM blocks with typed codecs, capabilities, and style overrides. Shared Layout Interpreters produce ViewModels consumed by both React and DOCX emitters.

**Architecture:** Codec → Interpreter → ViewModel → Emitter (React/DOCX/HTML). Each block type gets a typed codec (parse/validate/normalize), an enhanced interpreter (merge style overrides with Layout IR defaults), and refactored emitters that consume ViewModels instead of raw props.

**Tech Stack:** BlockNote v0.47.3, TypeScript, Vitest, docx.js

**Execution order:** Tasks 1-6 (codecs, ViewModels, component rules) → Task 8 (schema props) → Tasks 7, 9 (Section interpreter + React cutover) → Tasks 14-17 (remaining block interpreters + React cutover) → Task 18 (DOCX emitter migration) → Tasks 10-11 (template types + PO template) → Tasks 12-13 (IR hash + full test suite). Task 8 MUST precede interpreter work so schemas include styleJson/capabilitiesJson before interpreters parse them.

---

## File Structure

### New Files

```
engine/codecs/
├── codec-utils.ts            — safeParse, expectString, expectBoolean, stripUndefined, resolveThemeRef
├── section-codec.ts          — SectionStyle, SectionCapabilities, SectionCodec
├── data-table-codec.ts       — DataTableStyle, DataTableCapabilities, DataTableCodec
├── repeatable-codec.ts       — RepeatableStyle, RepeatableCapabilities, RepeatableCodec
├── repeatable-item-codec.ts  — RepeatableItemStyle, RepeatableItemCapabilities, RepeatableItemCodec
├── rich-block-codec.ts       — RichBlockStyle, RichBlockCapabilities, RichBlockCodec
├── index.ts                  — Re-export all codecs
└── __tests__/
    ├── codec-utils.test.ts
    ├── section-codec.test.ts
    ├── data-table-codec.test.ts
    ├── repeatable-codec.test.ts
    ├── repeatable-item-codec.test.ts
    └── rich-block-codec.test.ts

engine/template/
├── types.ts                  — TemplateDefinition, TemplateBlock, TemplateRef, TemplateStatus
├── validate.ts               — validateTemplate()
├── instantiate.ts            — instantiateTemplate()
├── index.ts                  — Re-export
└── __tests__/
    ├── validate.test.ts
    └── instantiate.test.ts

engine/layout-interpreter/
├── view-models.ts            — All ViewModel type definitions (NEW)
└── __tests__/
    ├── section-interpreter.test.ts   (NEW)
    ├── data-table-interpreter.test.ts (NEW)
    ├── repeatable-interpreter.test.ts (enhanced)
    └── rich-block-interpreter.test.ts (NEW)

templates/
└── po-standard.ts            — PO template defined as JSON
```

### Modified Files

```
engine/codecs/              — NEW directory
engine/template/            — NEW directory
engine/layout-interpreter/
├── types.ts                — Add ViewModel types import
├── section-interpreter.ts  — Refactor to use SectionCodec
├── repeatable-interpreter.ts — Refactor to use RepeatableCodec
├── field-interpreter.ts    — Refactor to use codec pattern
└── data-table-interpreter.ts — NEW (or refactor if exists)

blocks/
├── Section.tsx             — render() calls interpretSection(), uses ViewModel
├── DataTable.tsx           — render() calls interpretDataTable(), uses ViewModel
├── Repeatable.tsx          — render() calls interpretRepeatable(), uses ViewModel
├── RepeatableItem.tsx      — render() calls interpretRepeatableItem(), uses ViewModel
└── RichBlock.tsx           — render() calls interpretRichBlock(), uses ViewModel

engine/docx-emitter/emitters/
├── section.ts              — Refactor to call interpretSection() for ViewModel
├── data-table.ts           — Refactor to call interpretDataTable() for ViewModel
├── repeatable.ts           — Refactor to call interpretRepeatable() for ViewModel
├── repeatable-item.ts      — Refactor to call interpretRepeatableItem() for ViewModel
└── rich-block.ts           — Refactor to call interpretRichBlock() for ViewModel

schema.ts                   — Add styleJson, capabilitiesJson to propSchemas
engine/layout-ir/components.ts — Add DataTable, Repeatable, RichBlock component rules
engine/ir-hash/recorded-hash.ts — Update hash after Layout IR changes
```

---

### Task 1: Codec Utilities

**Files:**
- Create: `engine/codecs/codec-utils.ts`
- Test: `engine/codecs/__tests__/codec-utils.test.ts`

- [ ] **Step 1: Write the failing tests**

```typescript
// engine/codecs/__tests__/codec-utils.test.ts
import { describe, it, expect } from "vitest";
import {
  safeParse,
  expectString,
  expectBoolean,
  expectNumber,
  stripUndefined,
  resolveThemeRef,
} from "../codec-utils";

describe("safeParse", () => {
  it("parses valid JSON", () => {
    expect(safeParse('{"a":1}', {})).toEqual({ a: 1 });
  });

  it("returns fallback on invalid JSON", () => {
    expect(safeParse("not json", { fallback: true })).toEqual({ fallback: true });
  });

  it("returns fallback on empty string", () => {
    expect(safeParse("", {})).toEqual({});
  });
});

describe("expectString", () => {
  it("returns string values", () => {
    expect(expectString("hello")).toBe("hello");
  });

  it("returns undefined for non-strings", () => {
    expect(expectString(42)).toBeUndefined();
    expect(expectString(null)).toBeUndefined();
    expect(expectString(undefined)).toBeUndefined();
  });
});

describe("expectBoolean", () => {
  it("returns boolean values", () => {
    expect(expectBoolean(true, false)).toBe(true);
  });

  it("returns default for non-booleans", () => {
    expect(expectBoolean("yes", false)).toBe(false);
    expect(expectBoolean(undefined, true)).toBe(true);
  });
});

describe("expectNumber", () => {
  it("returns number values", () => {
    expect(expectNumber(42, 0)).toBe(42);
  });

  it("returns default for non-numbers", () => {
    expect(expectNumber("42", 0)).toBe(0);
    expect(expectNumber(undefined, 10)).toBe(10);
  });
});

describe("stripUndefined", () => {
  it("removes undefined keys", () => {
    expect(stripUndefined({ a: 1, b: undefined, c: "x" })).toEqual({ a: 1, c: "x" });
  });
});

describe("resolveThemeRef", () => {
  const theme = { accent: "#6b1f2a", accentLight: "#f9f3f3", accentDark: "#3e1018", accentBorder: "#dfc8c8" };

  it("resolves theme.accent", () => {
    expect(resolveThemeRef("theme.accent", theme)).toBe("#6b1f2a");
  });

  it("resolves theme.accentLight", () => {
    expect(resolveThemeRef("theme.accentLight", theme)).toBe("#f9f3f3");
  });

  it("returns non-theme strings as-is", () => {
    expect(resolveThemeRef("#ff0000", theme)).toBe("#ff0000");
  });

  it("returns undefined for undefined input", () => {
    expect(resolveThemeRef(undefined, theme)).toBeUndefined();
  });
});
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd frontend/apps/web && rtk vitest run src/features/documents/mddm-editor/engine/codecs/__tests__/codec-utils.test.ts`
Expected: FAIL — module not found

- [ ] **Step 3: Implement codec-utils**

```typescript
// engine/codecs/codec-utils.ts

export function safeParse(json: string, fallback: Record<string, unknown>): Record<string, unknown> {
  if (!json || json === "") return fallback;
  try {
    const parsed = JSON.parse(json);
    if (typeof parsed !== "object" || parsed === null) return fallback;
    return parsed as Record<string, unknown>;
  } catch {
    return fallback;
  }
}

export function expectString(value: unknown): string | undefined {
  return typeof value === "string" ? value : undefined;
}

export function expectBoolean(value: unknown, defaultValue: boolean): boolean {
  return typeof value === "boolean" ? value : defaultValue;
}

export function expectNumber(value: unknown, defaultValue: number): number {
  return typeof value === "number" ? value : defaultValue;
}

export function stripUndefined<T extends Record<string, unknown>>(obj: T): Partial<T> {
  const result: Record<string, unknown> = {};
  for (const [key, val] of Object.entries(obj)) {
    if (val !== undefined) result[key] = val;
  }
  return result as Partial<T>;
}

type ThemeColors = {
  accent: string;
  accentLight: string;
  accentDark: string;
  accentBorder: string;
};

export function resolveThemeRef(value: string | undefined, theme: ThemeColors): string | undefined {
  if (value === undefined) return undefined;
  if (value.startsWith("theme.")) {
    const key = value.slice(6) as keyof ThemeColors;
    return theme[key] ?? value;
  }
  return value;
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd frontend/apps/web && rtk vitest run src/features/documents/mddm-editor/engine/codecs/__tests__/codec-utils.test.ts`
Expected: PASS — all tests green

- [ ] **Step 5: Commit**

```bash
rtk git add frontend/apps/web/src/features/documents/mddm-editor/engine/codecs/codec-utils.ts frontend/apps/web/src/features/documents/mddm-editor/engine/codecs/__tests__/codec-utils.test.ts
rtk git commit -m "feat(mddm): add codec utility functions (safeParse, expectString, resolveThemeRef, etc.)"
```

---

### Task 2: Section Codec

**Files:**
- Create: `engine/codecs/section-codec.ts`
- Test: `engine/codecs/__tests__/section-codec.test.ts`

- [ ] **Step 1: Write the failing tests**

```typescript
// engine/codecs/__tests__/section-codec.test.ts
import { describe, it, expect } from "vitest";
import { SectionCodec } from "../section-codec";

describe("SectionCodec.parseStyle", () => {
  it("parses valid style JSON", () => {
    const style = SectionCodec.parseStyle('{"headerHeight":"12mm","headerBackground":"#ff0000"}');
    expect(style.headerHeight).toBe("12mm");
    expect(style.headerBackground).toBe("#ff0000");
  });

  it("returns empty style for empty JSON", () => {
    const style = SectionCodec.parseStyle("{}");
    expect(style.headerHeight).toBeUndefined();
    expect(style.headerBackground).toBeUndefined();
  });

  it("strips unknown fields", () => {
    const style = SectionCodec.parseStyle('{"headerHeight":"12mm","unknownField":"value"}');
    expect(style.headerHeight).toBe("12mm");
    expect((style as any).unknownField).toBeUndefined();
  });

  it("ignores non-string values for string fields", () => {
    const style = SectionCodec.parseStyle('{"headerHeight":42}');
    expect(style.headerHeight).toBeUndefined();
  });

  it("handles malformed JSON gracefully", () => {
    const style = SectionCodec.parseStyle("not json");
    expect(style).toEqual(SectionCodec.defaultStyle());
  });
});

describe("SectionCodec.parseCaps", () => {
  it("parses valid capabilities", () => {
    const caps = SectionCodec.parseCaps('{"locked":false,"removable":true}');
    expect(caps.locked).toBe(false);
    expect(caps.removable).toBe(true);
  });

  it("applies defaults for missing fields", () => {
    const caps = SectionCodec.parseCaps("{}");
    expect(caps.locked).toBe(true);
    expect(caps.removable).toBe(false);
    expect(caps.reorderable).toBe(false);
  });

  it("handles malformed JSON gracefully", () => {
    const caps = SectionCodec.parseCaps("broken");
    expect(caps).toEqual(SectionCodec.defaultCaps());
  });
});

describe("SectionCodec.serializeStyle", () => {
  it("round-trips through parse", () => {
    const original = { headerHeight: "12mm", headerBackground: "#ff0000" };
    const serialized = SectionCodec.serializeStyle(original);
    const parsed = SectionCodec.parseStyle(serialized);
    expect(parsed.headerHeight).toBe("12mm");
    expect(parsed.headerBackground).toBe("#ff0000");
  });

  it("strips undefined values", () => {
    const serialized = SectionCodec.serializeStyle({ headerHeight: "12mm" });
    expect(serialized).not.toContain("undefined");
  });
});
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd frontend/apps/web && rtk vitest run src/features/documents/mddm-editor/engine/codecs/__tests__/section-codec.test.ts`
Expected: FAIL — module not found

- [ ] **Step 3: Implement SectionCodec**

```typescript
// engine/codecs/section-codec.ts
import { safeParse, expectString, expectBoolean, stripUndefined } from "./codec-utils";

export type SectionStyle = {
  headerHeight?: string;
  headerBackground?: string;
  headerColor?: string;
  headerFontSize?: string;
  headerFontWeight?: string;
};

export type SectionCapabilities = {
  locked: boolean;
  removable: boolean;
  reorderable: boolean;
};

export const SectionCodec = {
  parseStyle(json: string): SectionStyle {
    const raw = safeParse(json, {});
    return {
      headerHeight: expectString(raw.headerHeight),
      headerBackground: expectString(raw.headerBackground),
      headerColor: expectString(raw.headerColor),
      headerFontSize: expectString(raw.headerFontSize),
      headerFontWeight: expectString(raw.headerFontWeight),
    };
  },

  parseCaps(json: string): SectionCapabilities {
    const raw = safeParse(json, {});
    return {
      locked: expectBoolean(raw.locked, true),
      removable: expectBoolean(raw.removable, false),
      reorderable: expectBoolean(raw.reorderable, false),
    };
  },

  defaultStyle(): SectionStyle {
    return {};
  },

  defaultCaps(): SectionCapabilities {
    return { locked: true, removable: false, reorderable: false };
  },

  serializeStyle(style: SectionStyle): string {
    return JSON.stringify(stripUndefined(style as Record<string, unknown>));
  },

  serializeCaps(caps: SectionCapabilities): string {
    return JSON.stringify(caps);
  },
};
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd frontend/apps/web && rtk vitest run src/features/documents/mddm-editor/engine/codecs/__tests__/section-codec.test.ts`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
rtk git add frontend/apps/web/src/features/documents/mddm-editor/engine/codecs/section-codec.ts frontend/apps/web/src/features/documents/mddm-editor/engine/codecs/__tests__/section-codec.test.ts
rtk git commit -m "feat(mddm): add SectionCodec with typed parse/serialize/validate"
```

---

### Task 3: DataTable Codec

**Files:**
- Create: `engine/codecs/data-table-codec.ts`
- Test: `engine/codecs/__tests__/data-table-codec.test.ts`

- [ ] **Step 1: Write the failing tests**

```typescript
// engine/codecs/__tests__/data-table-codec.test.ts
import { describe, it, expect } from "vitest";
import { DataTableCodec } from "../data-table-codec";

describe("DataTableCodec.parseStyle", () => {
  it("parses all style fields", () => {
    const style = DataTableCodec.parseStyle(JSON.stringify({
      headerBackground: "#f9f3f3",
      headerColor: "#3e1018",
      headerFontWeight: "bold",
      cellBorderColor: "#dfc8c8",
      cellPadding: "2mm",
      density: "compact",
    }));
    expect(style.headerBackground).toBe("#f9f3f3");
    expect(style.density).toBe("compact");
  });

  it("rejects invalid density values", () => {
    const style = DataTableCodec.parseStyle('{"density":"huge"}');
    expect(style.density).toBeUndefined();
  });
});

describe("DataTableCodec.parseCaps", () => {
  it("parses fixed mode", () => {
    const caps = DataTableCodec.parseCaps(JSON.stringify({
      locked: true,
      mode: "fixed",
      addRows: false,
      removeRows: false,
    }));
    expect(caps.mode).toBe("fixed");
    expect(caps.addRows).toBe(false);
  });

  it("parses dynamic mode with maxRows", () => {
    const caps = DataTableCodec.parseCaps(JSON.stringify({
      mode: "dynamic",
      addRows: true,
      removeRows: true,
      maxRows: 50,
    }));
    expect(caps.mode).toBe("dynamic");
    expect(caps.addRows).toBe(true);
    expect(caps.maxRows).toBe(50);
  });

  it("defaults to dynamic mode", () => {
    const caps = DataTableCodec.parseCaps("{}");
    expect(caps.mode).toBe("dynamic");
    expect(caps.maxRows).toBe(100);
  });

  it("rejects invalid mode values", () => {
    const caps = DataTableCodec.parseCaps('{"mode":"weird"}');
    expect(caps.mode).toBe("dynamic");
  });
});

describe("DataTableCodec round-trip", () => {
  it("serializes and parses back identically", () => {
    const original = { headerBackground: "#ff0000", density: "compact" as const };
    const serialized = DataTableCodec.serializeStyle(original);
    const parsed = DataTableCodec.parseStyle(serialized);
    expect(parsed.headerBackground).toBe("#ff0000");
    expect(parsed.density).toBe("compact");
  });
});
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd frontend/apps/web && rtk vitest run src/features/documents/mddm-editor/engine/codecs/__tests__/data-table-codec.test.ts`
Expected: FAIL

- [ ] **Step 3: Implement DataTableCodec**

```typescript
// engine/codecs/data-table-codec.ts
import { safeParse, expectString, expectBoolean, expectNumber, stripUndefined } from "./codec-utils";

export type DataTableStyle = {
  headerBackground?: string;
  headerColor?: string;
  headerFontWeight?: string;
  cellBorderColor?: string;
  cellPadding?: string;
  density?: "normal" | "compact";
};

export type DataTableCapabilities = {
  locked: boolean;
  removable: boolean;
  mode: "fixed" | "dynamic";
  editableZones: string[];
  addRows: boolean;
  removeRows: boolean;
  addColumns: boolean;
  removeColumns: boolean;
  resizeColumns: boolean;
  headerLocked: boolean;
  maxRows: number;
};

const VALID_DENSITIES = ["normal", "compact"] as const;
const VALID_MODES = ["fixed", "dynamic"] as const;

export const DataTableCodec = {
  parseStyle(json: string): DataTableStyle {
    const raw = safeParse(json, {});
    const density = expectString(raw.density);
    return {
      headerBackground: expectString(raw.headerBackground),
      headerColor: expectString(raw.headerColor),
      headerFontWeight: expectString(raw.headerFontWeight),
      cellBorderColor: expectString(raw.cellBorderColor),
      cellPadding: expectString(raw.cellPadding),
      density: density && VALID_DENSITIES.includes(density as any) ? density as "normal" | "compact" : undefined,
    };
  },

  parseCaps(json: string): DataTableCapabilities {
    const raw = safeParse(json, {});
    const mode = expectString(raw.mode);
    return {
      locked: expectBoolean(raw.locked, false),
      removable: expectBoolean(raw.removable, false),
      mode: mode && VALID_MODES.includes(mode as any) ? mode as "fixed" | "dynamic" : "dynamic",
      editableZones: Array.isArray(raw.editableZones) ? raw.editableZones.filter((z: unknown) => typeof z === "string") : ["cells"],
      addRows: expectBoolean(raw.addRows, true),
      removeRows: expectBoolean(raw.removeRows, true),
      addColumns: expectBoolean(raw.addColumns, false),
      removeColumns: expectBoolean(raw.removeColumns, false),
      resizeColumns: expectBoolean(raw.resizeColumns, false),
      headerLocked: expectBoolean(raw.headerLocked, true),
      maxRows: expectNumber(raw.maxRows, 100),
    };
  },

  defaultStyle(): DataTableStyle { return {}; },
  defaultCaps(): DataTableCapabilities {
    return {
      locked: false, removable: false, mode: "dynamic",
      editableZones: ["cells"], addRows: true, removeRows: true,
      addColumns: false, removeColumns: false, resizeColumns: false,
      headerLocked: true, maxRows: 100,
    };
  },

  serializeStyle(style: DataTableStyle): string {
    return JSON.stringify(stripUndefined(style as Record<string, unknown>));
  },
  serializeCaps(caps: DataTableCapabilities): string {
    return JSON.stringify(caps);
  },
};
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd frontend/apps/web && rtk vitest run src/features/documents/mddm-editor/engine/codecs/__tests__/data-table-codec.test.ts`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
rtk git add frontend/apps/web/src/features/documents/mddm-editor/engine/codecs/data-table-codec.ts frontend/apps/web/src/features/documents/mddm-editor/engine/codecs/__tests__/data-table-codec.test.ts
rtk git commit -m "feat(mddm): add DataTableCodec with fixed/dynamic mode support"
```

---

### Task 4: Repeatable, RepeatableItem, and RichBlock Codecs

**Files:**
- Create: `engine/codecs/repeatable-codec.ts`
- Create: `engine/codecs/repeatable-item-codec.ts`
- Create: `engine/codecs/rich-block-codec.ts`
- Test: `engine/codecs/__tests__/repeatable-codec.test.ts`
- Test: `engine/codecs/__tests__/repeatable-item-codec.test.ts`
- Test: `engine/codecs/__tests__/rich-block-codec.test.ts`

- [ ] **Step 1: Write failing tests for all three codecs**

```typescript
// engine/codecs/__tests__/repeatable-codec.test.ts
import { describe, it, expect } from "vitest";
import { RepeatableCodec } from "../repeatable-codec";

describe("RepeatableCodec.parseCaps", () => {
  it("parses addItems/removeItems/maxItems/minItems", () => {
    const caps = RepeatableCodec.parseCaps(JSON.stringify({
      addItems: true, removeItems: false, maxItems: 20, minItems: 1,
    }));
    expect(caps.addItems).toBe(true);
    expect(caps.removeItems).toBe(false);
    expect(caps.maxItems).toBe(20);
    expect(caps.minItems).toBe(1);
  });

  it("defaults maxItems to 100, minItems to 0", () => {
    const caps = RepeatableCodec.parseCaps("{}");
    expect(caps.maxItems).toBe(100);
    expect(caps.minItems).toBe(0);
  });
});

describe("RepeatableCodec.parseStyle", () => {
  it("parses border and accent styles", () => {
    const style = RepeatableCodec.parseStyle(JSON.stringify({
      borderColor: "#dfc8c8", itemAccentBorder: "#6b1f2a", itemAccentWidth: "3pt",
    }));
    expect(style.borderColor).toBe("#dfc8c8");
    expect(style.itemAccentBorder).toBe("#6b1f2a");
  });
});
```

```typescript
// engine/codecs/__tests__/repeatable-item-codec.test.ts
import { describe, it, expect } from "vitest";
import { RepeatableItemCodec } from "../repeatable-item-codec";

describe("RepeatableItemCodec.parseCaps", () => {
  it("parses editableZones", () => {
    const caps = RepeatableItemCodec.parseCaps(JSON.stringify({ editableZones: ["content"] }));
    expect(caps.editableZones).toEqual(["content"]);
  });

  it("defaults editableZones to ['content']", () => {
    const caps = RepeatableItemCodec.parseCaps("{}");
    expect(caps.editableZones).toEqual(["content"]);
  });
});
```

```typescript
// engine/codecs/__tests__/rich-block-codec.test.ts
import { describe, it, expect } from "vitest";
import { RichBlockCodec } from "../rich-block-codec";

describe("RichBlockCodec.parseCaps", () => {
  it("parses editableZones", () => {
    const caps = RichBlockCodec.parseCaps(JSON.stringify({ editableZones: ["content"] }));
    expect(caps.editableZones).toEqual(["content"]);
  });

  it("defaults locked to true", () => {
    const caps = RichBlockCodec.parseCaps("{}");
    expect(caps.locked).toBe(true);
  });
});

describe("RichBlockCodec.parseStyle", () => {
  it("parses label styling", () => {
    const style = RichBlockCodec.parseStyle(JSON.stringify({
      labelBackground: "#f9f3f3", labelFontSize: "10pt",
    }));
    expect(style.labelBackground).toBe("#f9f3f3");
  });
});
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd frontend/apps/web && rtk vitest run src/features/documents/mddm-editor/engine/codecs/__tests__/repeatable-codec.test.ts src/features/documents/mddm-editor/engine/codecs/__tests__/repeatable-item-codec.test.ts src/features/documents/mddm-editor/engine/codecs/__tests__/rich-block-codec.test.ts`
Expected: FAIL

- [ ] **Step 3: Implement RepeatableCodec**

```typescript
// engine/codecs/repeatable-codec.ts
import { safeParse, expectString, expectBoolean, expectNumber, stripUndefined } from "./codec-utils";

export type RepeatableStyle = {
  borderColor?: string;
  itemAccentBorder?: string;
  itemAccentWidth?: string;
};

export type RepeatableCapabilities = {
  locked: boolean;
  removable: boolean;
  addItems: boolean;
  removeItems: boolean;
  maxItems: number;
  minItems: number;
};

export const RepeatableCodec = {
  parseStyle(json: string): RepeatableStyle {
    const raw = safeParse(json, {});
    return {
      borderColor: expectString(raw.borderColor),
      itemAccentBorder: expectString(raw.itemAccentBorder),
      itemAccentWidth: expectString(raw.itemAccentWidth),
    };
  },

  parseCaps(json: string): RepeatableCapabilities {
    const raw = safeParse(json, {});
    return {
      locked: expectBoolean(raw.locked, true),
      removable: expectBoolean(raw.removable, false),
      addItems: expectBoolean(raw.addItems, true),
      removeItems: expectBoolean(raw.removeItems, true),
      maxItems: expectNumber(raw.maxItems, 100),
      minItems: expectNumber(raw.minItems, 0),
    };
  },

  defaultStyle(): RepeatableStyle { return {}; },
  defaultCaps(): RepeatableCapabilities {
    return { locked: true, removable: false, addItems: true, removeItems: true, maxItems: 100, minItems: 0 };
  },

  serializeStyle(style: RepeatableStyle): string { return JSON.stringify(stripUndefined(style as Record<string, unknown>)); },
  serializeCaps(caps: RepeatableCapabilities): string { return JSON.stringify(caps); },
};
```

- [ ] **Step 4: Implement RepeatableItemCodec**

```typescript
// engine/codecs/repeatable-item-codec.ts
import { safeParse, expectString, expectBoolean, stripUndefined } from "./codec-utils";

export type RepeatableItemStyle = {
  accentBorderColor?: string;
  accentBorderWidth?: string;
};

export type RepeatableItemCapabilities = {
  locked: boolean;
  removable: boolean;
  editableZones: string[];
};

export const RepeatableItemCodec = {
  parseStyle(json: string): RepeatableItemStyle {
    const raw = safeParse(json, {});
    return {
      accentBorderColor: expectString(raw.accentBorderColor),
      accentBorderWidth: expectString(raw.accentBorderWidth),
    };
  },

  parseCaps(json: string): RepeatableItemCapabilities {
    const raw = safeParse(json, {});
    return {
      locked: expectBoolean(raw.locked, false),
      removable: expectBoolean(raw.removable, true),
      editableZones: Array.isArray(raw.editableZones) ? raw.editableZones.filter((z: unknown) => typeof z === "string") : ["content"],
    };
  },

  defaultStyle(): RepeatableItemStyle { return {}; },
  defaultCaps(): RepeatableItemCapabilities {
    return { locked: false, removable: true, editableZones: ["content"] };
  },

  serializeStyle(style: RepeatableItemStyle): string { return JSON.stringify(stripUndefined(style as Record<string, unknown>)); },
  serializeCaps(caps: RepeatableItemCapabilities): string { return JSON.stringify(caps); },
};
```

- [ ] **Step 5: Implement RichBlockCodec**

```typescript
// engine/codecs/rich-block-codec.ts
import { safeParse, expectString, expectBoolean, stripUndefined } from "./codec-utils";

export type RichBlockStyle = {
  labelBackground?: string;
  labelFontSize?: string;
  labelColor?: string;
  borderColor?: string;
};

export type RichBlockCapabilities = {
  locked: boolean;
  removable: boolean;
  editableZones: string[];
};

export const RichBlockCodec = {
  parseStyle(json: string): RichBlockStyle {
    const raw = safeParse(json, {});
    return {
      labelBackground: expectString(raw.labelBackground),
      labelFontSize: expectString(raw.labelFontSize),
      labelColor: expectString(raw.labelColor),
      borderColor: expectString(raw.borderColor),
    };
  },

  parseCaps(json: string): RichBlockCapabilities {
    const raw = safeParse(json, {});
    return {
      locked: expectBoolean(raw.locked, true),
      removable: expectBoolean(raw.removable, false),
      editableZones: Array.isArray(raw.editableZones) ? raw.editableZones.filter((z: unknown) => typeof z === "string") : ["content"],
    };
  },

  defaultStyle(): RichBlockStyle { return {}; },
  defaultCaps(): RichBlockCapabilities {
    return { locked: true, removable: false, editableZones: ["content"] };
  },

  serializeStyle(style: RichBlockStyle): string { return JSON.stringify(stripUndefined(style as Record<string, unknown>)); },
  serializeCaps(caps: RichBlockCapabilities): string { return JSON.stringify(caps); },
};
```

- [ ] **Step 6: Run all codec tests**

Run: `cd frontend/apps/web && rtk vitest run src/features/documents/mddm-editor/engine/codecs/`
Expected: PASS — all codec tests green

- [ ] **Step 7: Create codec index and commit**

```typescript
// engine/codecs/index.ts
export { SectionCodec, type SectionStyle, type SectionCapabilities } from "./section-codec";
export { DataTableCodec, type DataTableStyle, type DataTableCapabilities } from "./data-table-codec";
export { RepeatableCodec, type RepeatableStyle, type RepeatableCapabilities } from "./repeatable-codec";
export { RepeatableItemCodec, type RepeatableItemStyle, type RepeatableItemCapabilities } from "./repeatable-item-codec";
export { RichBlockCodec, type RichBlockStyle, type RichBlockCapabilities } from "./rich-block-codec";
export { safeParse, expectString, expectBoolean, expectNumber, stripUndefined, resolveThemeRef } from "./codec-utils";
```

```bash
rtk git add frontend/apps/web/src/features/documents/mddm-editor/engine/codecs/
rtk git commit -m "feat(mddm): add Repeatable, RepeatableItem, RichBlock codecs + index"
```

---

### Task 5: ViewModel Type Definitions

**Files:**
- Create: `engine/layout-interpreter/view-models.ts`

- [ ] **Step 1: Write ViewModel types**

```typescript
// engine/layout-interpreter/view-models.ts

export type SectionViewModel = {
  number: string;
  title: string;
  optional: boolean;
  headerHeight: string;
  headerBg: string;
  headerColor: string;
  headerFontSize: string;
  headerFontWeight: string;
  locked: boolean;
  removable: boolean;
};

export type DataTableViewModel = {
  label: string;
  mode: "fixed" | "dynamic";
  headerBg: string;
  headerColor: string;
  headerFontWeight: string;
  cellBorderColor: string;
  cellPadding: string;
  density: "normal" | "compact";
  locked: boolean;
  removable: boolean;
  canAddRows: boolean;
  canRemoveRows: boolean;
  canAddColumns: boolean;
  canRemoveColumns: boolean;
  canResizeColumns: boolean;
  headerLocked: boolean;
  maxRows: number;
};

export type RepeatableViewModel = {
  label: string;
  itemPrefix: string;
  borderColor: string;
  itemAccentBorder: string;
  itemAccentWidth: string;
  locked: boolean;
  removable: boolean;
  canAddItems: boolean;
  canRemoveItems: boolean;
  maxItems: number;
  minItems: number;
  currentItemCount: number;
};

export type RepeatableItemViewModel = {
  title: string;
  number: string;
  accentBorderColor: string;
  accentBorderWidth: string;
  locked: boolean;
  removable: boolean;
};

export type RichBlockViewModel = {
  label: string;
  chrome: string;
  labelBackground: string;
  labelFontSize: string;
  labelColor: string;
  borderColor: string;
  locked: boolean;
  removable: boolean;
};
```

- [ ] **Step 2: Commit**

```bash
rtk git add frontend/apps/web/src/features/documents/mddm-editor/engine/layout-interpreter/view-models.ts
rtk git commit -m "feat(mddm): add ViewModel type definitions for all MDDM block types"
```

---

### Task 6: Enhance Component Rules for Missing Block Types

**Files:**
- Modify: `engine/layout-ir/components.ts`

- [ ] **Step 1: Add DataTable, Repeatable, RichBlock rules**

Add to `engine/layout-ir/components.ts` after existing rules:

```typescript
export type DataTableRule = Readonly<{
  headerBackgroundToken: "theme.accentLight";
  headerFontColor: string;
  headerFontWeight: "bold" | "normal";
  cellBorderColorToken: "theme.accentBorder";
  cellPaddingMm: number;
  defaultDensity: "normal" | "compact";
}>;

export type RepeatableRule = Readonly<{
  borderColorToken: "theme.accentBorder";
  itemAccentBorderToken: "theme.accent";
  itemAccentWidthPt: number;
}>;

export type RichBlockRule = Readonly<{
  labelBackgroundToken: "theme.accentLight";
  labelFontSizePt: number;
  labelFontColor: string;
  borderColorToken: "theme.accentBorder";
}>;
```

Update `ComponentRules` type to include them:

```typescript
export type ComponentRules = Readonly<{
  section: SectionRule;
  field: FieldRule;
  fieldGroup: FieldGroupRule;
  dataTable: DataTableRule;
  repeatable: RepeatableRule;
  richBlock: RichBlockRule;
}>;
```

Add defaults to `defaultComponentRules`:

```typescript
dataTable: {
  headerBackgroundToken: "theme.accentLight",
  headerFontColor: "#3e1018",
  headerFontWeight: "bold",
  cellBorderColorToken: "theme.accentBorder",
  cellPaddingMm: 2,
  defaultDensity: "normal",
},
repeatable: {
  borderColorToken: "theme.accentBorder",
  itemAccentBorderToken: "theme.accent",
  itemAccentWidthPt: 3,
},
richBlock: {
  labelBackgroundToken: "theme.accentLight",
  labelFontSizePt: 9,
  labelFontColor: "#3e1018",
  borderColorToken: "theme.accentBorder",
},
```

- [ ] **Step 2: Run existing token-bridge tests to verify no regression**

Run: `cd frontend/apps/web && rtk vitest run src/features/documents/mddm-editor/engine/layout-ir/`
Expected: PASS (existing tests should still pass — we only added new rules)

- [ ] **Step 3: Commit**

```bash
rtk git add frontend/apps/web/src/features/documents/mddm-editor/engine/layout-ir/components.ts
rtk git commit -m "feat(mddm): add DataTable, Repeatable, RichBlock component rules to Layout IR"
```

---

### Task 7: Refactor Section Interpreter to Use Codec

**Files:**
- Modify: `engine/layout-interpreter/section-interpreter.ts`
- Test: `engine/layout-interpreter/__tests__/section-interpreter.test.ts`

- [ ] **Step 1: Write failing tests for enhanced interpreter**

```typescript
// engine/layout-interpreter/__tests__/section-interpreter.test.ts
import { describe, it, expect } from "vitest";
import { interpretSection } from "../section-interpreter";
import { defaultLayoutTokens } from "../../layout-ir";

describe("interpretSection", () => {
  it("uses Layout IR defaults when no style override", () => {
    const block = { props: { title: "OBJETIVO", styleJson: "{}", capabilitiesJson: "{}" } };
    const vm = interpretSection(block as any, defaultLayoutTokens, { sectionIndex: 2 });

    expect(vm.number).toBe("3");
    expect(vm.title).toBe("OBJETIVO");
    expect(vm.headerBg).toBe(defaultLayoutTokens.theme.accent);
    expect(vm.headerHeight).toBe("8mm");
    expect(vm.headerFontSize).toBe("13pt");
    expect(vm.locked).toBe(true);
  });

  it("applies style overrides", () => {
    const block = {
      props: {
        title: "CUSTOM",
        styleJson: JSON.stringify({ headerHeight: "12mm", headerBackground: "#ff0000" }),
        capabilitiesJson: "{}",
      },
    };
    const vm = interpretSection(block as any, defaultLayoutTokens, { sectionIndex: 0 });

    expect(vm.headerHeight).toBe("12mm");
    expect(vm.headerBg).toBe("#ff0000");
    expect(vm.headerColor).toBe("#ffffff"); // default
  });

  it("resolves theme references in style", () => {
    const block = {
      props: {
        title: "THEMED",
        styleJson: JSON.stringify({ headerBackground: "theme.accentDark" }),
        capabilitiesJson: "{}",
      },
    };
    const vm = interpretSection(block as any, defaultLayoutTokens, { sectionIndex: 0 });

    expect(vm.headerBg).toBe(defaultLayoutTokens.theme.accentDark);
  });

  it("reads capabilities from codec", () => {
    const block = {
      props: {
        title: "UNLOCKED",
        styleJson: "{}",
        capabilitiesJson: JSON.stringify({ locked: false, removable: true }),
      },
    };
    const vm = interpretSection(block as any, defaultLayoutTokens, { sectionIndex: 0 });

    expect(vm.locked).toBe(false);
    expect(vm.removable).toBe(true);
  });
});
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd frontend/apps/web && rtk vitest run src/features/documents/mddm-editor/engine/layout-interpreter/__tests__/section-interpreter.test.ts`
Expected: FAIL

- [ ] **Step 3: Refactor section-interpreter.ts**

Replace the entire file:

```typescript
// engine/layout-interpreter/section-interpreter.ts
import type { LayoutTokens } from "../layout-ir";
import { defaultComponentRules } from "../layout-ir";
import { SectionCodec } from "../codecs";
import { resolveThemeRef } from "../codecs/codec-utils";
import type { SectionViewModel } from "./view-models";

type InterpretSectionContext = {
  sectionIndex: number;
};

export function interpretSection(
  block: { props: Record<string, unknown> },
  tokens: LayoutTokens,
  context: InterpretSectionContext,
): SectionViewModel {
  const style = SectionCodec.parseStyle((block.props.styleJson as string) ?? "{}");
  const caps = SectionCodec.parseCaps((block.props.capabilitiesJson as string) ?? "{}");
  const rule = defaultComponentRules.section;

  return {
    number: String(context.sectionIndex + 1),
    title: (block.props.title as string) ?? "",
    optional: (block.props.optional as boolean) ?? false,
    headerHeight: style.headerHeight ?? `${rule.headerHeightMm}mm`,
    headerBg: resolveThemeRef(style.headerBackground, tokens.theme) ?? tokens.theme.accent,
    headerColor: style.headerColor ?? rule.headerFontColor,
    headerFontSize: style.headerFontSize ?? `${rule.headerFontSizePt}pt`,
    headerFontWeight: style.headerFontWeight ?? rule.headerFontWeight,
    locked: caps.locked,
    removable: caps.removable,
  };
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd frontend/apps/web && rtk vitest run src/features/documents/mddm-editor/engine/layout-interpreter/__tests__/section-interpreter.test.ts`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
rtk git add frontend/apps/web/src/features/documents/mddm-editor/engine/layout-interpreter/section-interpreter.ts frontend/apps/web/src/features/documents/mddm-editor/engine/layout-interpreter/__tests__/section-interpreter.test.ts
rtk git commit -m "refactor(mddm): section interpreter uses SectionCodec + produces SectionViewModel"
```

---

### Task 8: Add styleJson/capabilitiesJson Props to Schema

**Files:**
- Modify: `blocks/Section.tsx` — add props to propSchema
- Modify: `blocks/DataTable.tsx` — add props to propSchema
- Modify: `blocks/Repeatable.tsx` — add props to propSchema
- Modify: `blocks/RepeatableItem.tsx` — add props to propSchema
- Modify: `blocks/RichBlock.tsx` — add props to propSchema

- [ ] **Step 1: Add styleJson and capabilitiesJson to Section propSchema**

In `blocks/Section.tsx`, add to propSchema:

```typescript
propSchema: {
  title: { default: "" },
  color: { default: "#6b1f2a" },
  locked: { default: true },
  optional: { default: false },
  variant: { default: "bar" },
  __template_block_id: { default: "" },
  styleJson: { default: "{}" },
  capabilitiesJson: { default: "{}" },
},
```

- [ ] **Step 2: Add to all other block propSchemas**

Same pattern — add `styleJson: { default: "{}" }` and `capabilitiesJson: { default: "{}" }` to DataTable, Repeatable, RepeatableItem, and RichBlock propSchemas.

- [ ] **Step 3: Run existing tests to verify no regression**

Run: `cd frontend/apps/web && rtk vitest run src/features/documents/mddm-editor/`
Expected: PASS — adding new optional props with defaults shouldn't break anything

- [ ] **Step 4: Commit**

```bash
rtk git add frontend/apps/web/src/features/documents/mddm-editor/blocks/
rtk git commit -m "feat(mddm): add styleJson + capabilitiesJson props to all MDDM block schemas"
```

---

### Task 9: Refactor Section React Render to Use Interpreter

**Files:**
- Modify: `blocks/Section.tsx`

- [ ] **Step 1: Refactor render() to call interpretSection()**

```typescript
// blocks/Section.tsx
import { createReactBlockSpec } from "@blocknote/react";
import styles from "./Section.module.css";
import { SectionExternalHTML } from "../engine/external-html";
import { getEditorTokens } from "../engine/editor-tokens";
import { interpretSection } from "../engine/layout-interpreter/section-interpreter";

export const Section = createReactBlockSpec(
  {
    type: "section",
    propSchema: {
      title: { default: "" },
      color: { default: "#6b1f2a" },
      locked: { default: true },
      optional: { default: false },
      variant: { default: "bar" },
      __template_block_id: { default: "" },
      styleJson: { default: "{}" },
      capabilitiesJson: { default: "{}" },
    },
    content: "none",
  },
  {
    render: (props) => {
      const tokens = getEditorTokens(props.editor);
      const sectionIndex = (props.editor.document as any[])
        .filter((b: any) => b.type === "section")
        .findIndex((b: any) => b.id === props.block.id);
      const vm = interpretSection(
        { props: props.block.props as Record<string, unknown> },
        tokens,
        { sectionIndex: sectionIndex >= 0 ? sectionIndex : 0 },
      );

      return (
        <div
          className={styles.section}
          data-mddm-block="section"
          data-locked={vm.locked}
          style={{
            height: vm.headerHeight,
            background: vm.headerBg,
            color: vm.headerColor,
            fontSize: vm.headerFontSize,
            fontWeight: vm.headerFontWeight,
          }}
        >
          <div className={styles.sectionHeader}>
            <span className={styles.sectionNumber}>{vm.number}.</span>
            <span className={styles.sectionTitle}>{vm.title}</span>
            {vm.optional ? (
              <span className={styles.optionalBadge}>Opcional</span>
            ) : null}
          </div>
        </div>
      );
    },
    toExternalHTML: ({ block, editor }) => {
      const tokens = getEditorTokens(editor);
      const sectionIndex = (editor.document as any[])
        .filter((b: any) => b.type === "section")
        .findIndex((b: any) => b.id === block.id);
      const vm = interpretSection(
        { props: block.props as Record<string, unknown> },
        tokens,
        { sectionIndex: sectionIndex >= 0 ? sectionIndex : 0 },
      );

      return (
        <SectionExternalHTML
          title={vm.title}
          tokens={tokens}
          sectionNumber={parseInt(vm.number, 10)}
        />
      );
    },
  },
);
```

- [ ] **Step 2: Build to verify no TypeScript errors**

Run: `cd frontend/apps/web && pnpm exec vite build 2>&1 | tail -5`
Expected: Build succeeds

- [ ] **Step 3: Commit**

```bash
rtk git add frontend/apps/web/src/features/documents/mddm-editor/blocks/Section.tsx
rtk git commit -m "refactor(mddm): Section render() uses interpretSection() + ViewModel"
```

---

### Task 10: Template Schema Types + Validation

**Files:**
- Create: `engine/template/types.ts`
- Create: `engine/template/validate.ts`
- Create: `engine/template/instantiate.ts`
- Create: `engine/template/index.ts`
- Test: `engine/template/__tests__/validate.test.ts`
- Test: `engine/template/__tests__/instantiate.test.ts`

- [ ] **Step 1: Write template types**

```typescript
// engine/template/types.ts

export type TemplateStatus = "draft" | "published" | "deprecated";

export type TemplateMeta = {
  name: string;
  description: string;
  createdAt: string;
  updatedAt: string;
};

export type TemplateTheme = {
  accent: string;
  accentLight: string;
  accentDark: string;
  accentBorder: string;
};

export type TemplateBlock = {
  type: string;
  props: Record<string, unknown>;
  style?: Record<string, unknown>;
  capabilities?: Record<string, unknown>;
  columns?: Array<{ key: string; label: string; width: string; locked?: boolean }>;
  content?: unknown;
  children?: TemplateBlock[];
};

export type TemplateDefinition = {
  templateKey: string;
  version: number;
  profileCode: string;
  status: TemplateStatus;
  meta: TemplateMeta;
  theme: TemplateTheme;
  blocks: TemplateBlock[];
};

export type TemplateRef = {
  templateKey: string;
  templateVersion: number;
  instantiatedAt: string;
};
```

- [ ] **Step 2: Write failing validation tests**

```typescript
// engine/template/__tests__/validate.test.ts
import { describe, it, expect } from "vitest";
import { validateTemplate, type ValidationError } from "../validate";
import type { TemplateDefinition } from "../types";

function makeTemplate(overrides: Partial<TemplateDefinition> = {}): TemplateDefinition {
  return {
    templateKey: "test",
    version: 1,
    profileCode: "po",
    status: "published",
    meta: { name: "Test", description: "Test", createdAt: "", updatedAt: "" },
    theme: { accent: "#6b1f2a", accentLight: "#f9f3f3", accentDark: "#3e1018", accentBorder: "#dfc8c8" },
    blocks: [],
    ...overrides,
  };
}

describe("validateTemplate", () => {
  it("accepts a valid empty template", () => {
    expect(validateTemplate(makeTemplate())).toHaveLength(0);
  });

  it("accepts a template with valid section block", () => {
    const errors = validateTemplate(makeTemplate({
      blocks: [{ type: "section", props: { title: "TEST" } }],
    }));
    expect(errors).toHaveLength(0);
  });

  it("rejects unknown block type", () => {
    const errors = validateTemplate(makeTemplate({
      blocks: [{ type: "nonexistent", props: {} }],
    }));
    expect(errors).toContainEqual(expect.objectContaining({ error: "unknown_block_type" }));
  });

  it("rejects section without title", () => {
    const errors = validateTemplate(makeTemplate({
      blocks: [{ type: "section", props: {} }],
    }));
    expect(errors).toContainEqual(expect.objectContaining({ error: "missing_required_prop" }));
  });
});
```

- [ ] **Step 3: Implement validateTemplate**

```typescript
// engine/template/validate.ts
import type { TemplateDefinition, TemplateBlock } from "./types";

export type ValidationError = {
  path: string;
  error: string;
  message: string;
};

const KNOWN_BLOCK_TYPES = new Set([
  "section", "dataTable", "repeatable", "repeatableItem",
  "richBlock", "paragraph", "heading", "bulletListItem",
  "numberedListItem", "image", "quote", "divider",
]);

const REQUIRED_PROPS: Record<string, string[]> = {
  section: ["title"],
  dataTable: ["label"],
  repeatable: ["label"],
  richBlock: ["label"],
};

export function validateTemplate(template: TemplateDefinition): ValidationError[] {
  const errors: ValidationError[] = [];
  validateBlocks(template.blocks, "blocks", errors);
  return errors;
}

function validateBlocks(blocks: TemplateBlock[], basePath: string, errors: ValidationError[]): void {
  for (let i = 0; i < blocks.length; i++) {
    const block = blocks[i];
    const path = `${basePath}[${i}]`;

    if (!KNOWN_BLOCK_TYPES.has(block.type)) {
      errors.push({ path, error: "unknown_block_type", message: `Unknown block type: ${block.type}` });
      continue;
    }

    const required = REQUIRED_PROPS[block.type];
    if (required) {
      for (const prop of required) {
        if (!block.props[prop]) {
          errors.push({ path: `${path}.props.${prop}`, error: "missing_required_prop", message: `Missing required prop: ${prop}` });
        }
      }
    }

    if (block.children) {
      validateBlocks(block.children, `${path}.children`, errors);
    }
  }
}
```

- [ ] **Step 4: Write failing instantiation tests**

```typescript
// engine/template/__tests__/instantiate.test.ts
import { describe, it, expect } from "vitest";
import { instantiateTemplate } from "../instantiate";
import type { TemplateDefinition } from "../types";

const template: TemplateDefinition = {
  templateKey: "po-standard",
  version: 1,
  profileCode: "po",
  status: "published",
  meta: { name: "PO", description: "PO", createdAt: "", updatedAt: "" },
  theme: { accent: "#6b1f2a", accentLight: "#f9f3f3", accentDark: "#3e1018", accentBorder: "#dfc8c8" },
  blocks: [
    { type: "section", props: { title: "IDENTIFICAÇÃO" }, capabilities: { locked: true } },
  ],
};

describe("instantiateTemplate", () => {
  it("creates an envelope with template_ref", () => {
    const envelope = instantiateTemplate(template);
    expect(envelope.template_ref.templateKey).toBe("po-standard");
    expect(envelope.template_ref.templateVersion).toBe(1);
    expect(envelope.template_ref.instantiatedAt).toBeTruthy();
  });

  it("deep clones blocks (mutation-safe)", () => {
    const envelope = instantiateTemplate(template);
    envelope.blocks[0].props.title = "MODIFIED";
    expect(template.blocks[0].props.title).toBe("IDENTIFICAÇÃO");
  });

  it("preserves capabilities on cloned blocks", () => {
    const envelope = instantiateTemplate(template);
    expect(envelope.blocks[0].capabilities).toEqual({ locked: true });
  });
});
```

- [ ] **Step 5: Implement instantiateTemplate**

```typescript
// engine/template/instantiate.ts
import type { TemplateDefinition, TemplateRef, TemplateBlock } from "./types";

export type MDDMTemplateEnvelope = {
  mddm_version: number;
  template_ref: TemplateRef;
  blocks: TemplateBlock[];
};

const CURRENT_MDDM_VERSION = 1;

export function instantiateTemplate(template: TemplateDefinition): MDDMTemplateEnvelope {
  return {
    mddm_version: CURRENT_MDDM_VERSION,
    template_ref: {
      templateKey: template.templateKey,
      templateVersion: template.version,
      instantiatedAt: new Date().toISOString(),
    },
    blocks: structuredClone(template.blocks),
  };
}
```

- [ ] **Step 6: Create index**

```typescript
// engine/template/index.ts
export type { TemplateDefinition, TemplateBlock, TemplateRef, TemplateMeta, TemplateTheme, TemplateStatus } from "./types";
export { validateTemplate, type ValidationError } from "./validate";
export { instantiateTemplate, type MDDMTemplateEnvelope } from "./instantiate";
```

- [ ] **Step 7: Run all template tests**

Run: `cd frontend/apps/web && rtk vitest run src/features/documents/mddm-editor/engine/template/`
Expected: PASS

- [ ] **Step 8: Commit**

```bash
rtk git add frontend/apps/web/src/features/documents/mddm-editor/engine/template/
rtk git commit -m "feat(mddm): add template schema types, validation, and instantiation"
```

---

### Task 11: PO Standard Template Definition

**Files:**
- Create: `templates/po-standard.ts`

- [ ] **Step 1: Define the PO template**

```typescript
// templates/po-standard.ts
import type { TemplateDefinition } from "../engine/template";

export const poStandardTemplate: TemplateDefinition = {
  templateKey: "po-standard",
  version: 1,
  profileCode: "po",
  status: "published",
  meta: {
    name: "Procedimento Operacional Padrão",
    description: "Template padrão para procedimentos operacionais",
    createdAt: "2026-04-13T00:00:00Z",
    updatedAt: "2026-04-13T00:00:00Z",
  },
  theme: {
    accent: "#6b1f2a",
    accentLight: "#f9f3f3",
    accentDark: "#3e1018",
    accentBorder: "#dfc8c8",
  },
  blocks: [
    {
      type: "section",
      props: { title: "IDENTIFICAÇÃO", locked: true, styleJson: "{}", capabilitiesJson: JSON.stringify({ locked: true, removable: false }) },
      children: [],
    },
    {
      type: "section",
      props: { title: "IDENTIFICAÇÃO DO PROCESSO", locked: true, styleJson: "{}", capabilitiesJson: JSON.stringify({ locked: true, removable: false }) },
      children: [
        { type: "richBlock", props: { label: "Objetivo", locked: true, styleJson: "{}", capabilitiesJson: JSON.stringify({ locked: true, editableZones: ["content"] }) } },
        { type: "richBlock", props: { label: "Escopo", locked: true, styleJson: "{}", capabilitiesJson: JSON.stringify({ locked: true, editableZones: ["content"] }) } },
        { type: "richBlock", props: { label: "Cargo responsável", locked: true, styleJson: "{}", capabilitiesJson: JSON.stringify({ locked: true, editableZones: ["content"] }) } },
        { type: "richBlock", props: { label: "Canal / Contexto", locked: true, styleJson: "{}", capabilitiesJson: JSON.stringify({ locked: true, editableZones: ["content"] }) } },
        { type: "richBlock", props: { label: "Participantes", locked: true, styleJson: "{}", capabilitiesJson: JSON.stringify({ locked: true, editableZones: ["content"] }) } },
      ],
    },
    {
      type: "section",
      props: { title: "ENTRADAS E SAÍDAS", locked: true, styleJson: "{}", capabilitiesJson: JSON.stringify({ locked: true, removable: false }) },
      children: [],
    },
    {
      type: "section",
      props: { title: "VISÃO GERAL DO PROCESSO", locked: true, styleJson: "{}", capabilitiesJson: JSON.stringify({ locked: true, removable: false }) },
      children: [
        { type: "richBlock", props: { label: "Descrição do processo", locked: true, styleJson: "{}", capabilitiesJson: JSON.stringify({ locked: true, editableZones: ["content"] }) } },
        { type: "richBlock", props: { label: "Diagrama", locked: true, styleJson: "{}", capabilitiesJson: JSON.stringify({ locked: true, editableZones: ["content"] }) } },
      ],
    },
    {
      type: "section",
      props: { title: "DETALHAMENTO DAS ETAPAS", locked: true, styleJson: "{}", capabilitiesJson: JSON.stringify({ locked: true, removable: false }) },
      children: [
        {
          type: "repeatable",
          props: { label: "Etapas", itemPrefix: "Etapa", locked: false, styleJson: "{}", capabilitiesJson: JSON.stringify({ locked: false, addItems: true, removeItems: true, maxItems: 50, minItems: 1 }) },
          children: [],
        },
      ],
    },
    {
      type: "section",
      props: { title: "INDICADORES", locked: true, styleJson: "{}", capabilitiesJson: JSON.stringify({ locked: true, removable: false }) },
      children: [],
    },
    {
      type: "section",
      props: { title: "RISCOS E CONTROLES", locked: true, styleJson: "{}", capabilitiesJson: JSON.stringify({ locked: true, removable: false }) },
      children: [],
    },
    {
      type: "section",
      props: { title: "REFERÊNCIAS", locked: true, styleJson: "{}", capabilitiesJson: JSON.stringify({ locked: true, removable: false }) },
      children: [],
    },
    {
      type: "section",
      props: { title: "GLOSSÁRIO", locked: true, styleJson: "{}", capabilitiesJson: JSON.stringify({ locked: true, removable: false }) },
      children: [],
    },
    {
      type: "section",
      props: { title: "HISTÓRICO DE REVISÕES", locked: true, styleJson: "{}", capabilitiesJson: JSON.stringify({ locked: true, removable: false }) },
      children: [],
    },
  ],
};
```

- [ ] **Step 2: Add a validation test for the PO template**

```typescript
// engine/template/__tests__/po-standard.test.ts
import { describe, it, expect } from "vitest";
import { validateTemplate } from "../validate";
import { poStandardTemplate } from "../../../templates/po-standard";

describe("PO Standard Template", () => {
  it("passes validation", () => {
    const errors = validateTemplate(poStandardTemplate);
    expect(errors).toHaveLength(0);
  });

  it("has 10 top-level sections", () => {
    const sections = poStandardTemplate.blocks.filter(b => b.type === "section");
    expect(sections.length).toBe(10);
  });

  it("has the correct theme", () => {
    expect(poStandardTemplate.theme.accent).toBe("#6b1f2a");
  });
});
```

- [ ] **Step 3: Run tests**

Run: `cd frontend/apps/web && rtk vitest run src/features/documents/mddm-editor/engine/template/__tests__/po-standard.test.ts`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
rtk git add frontend/apps/web/src/features/documents/mddm-editor/templates/ frontend/apps/web/src/features/documents/mddm-editor/engine/template/__tests__/po-standard.test.ts
rtk git commit -m "feat(mddm): add PO standard template definition (code-defined, Phase 1)"
```

---

### Task 12: Update IR Hash

**Files:**
- Modify: `engine/ir-hash/recorded-hash.ts`

- [ ] **Step 1: Run the IR hash drift test to get the new hash**

Run: `cd frontend/apps/web && rtk vitest run src/features/documents/mddm-editor/engine/ir-hash/ 2>&1 | head -20`
Expected: If Layout IR types changed (from Task 6), the test will fail and show the new hash.

- [ ] **Step 2: Update recorded-hash.ts with the new hash**

Update the `RECORDED_IR_HASH` value with the hash from the test output.

- [ ] **Step 3: Run the hash test again to verify**

Run: `cd frontend/apps/web && rtk vitest run src/features/documents/mddm-editor/engine/ir-hash/`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
rtk git add frontend/apps/web/src/features/documents/mddm-editor/engine/ir-hash/recorded-hash.ts
rtk git commit -m "chore(mddm): update IR hash after adding DataTable/Repeatable/RichBlock component rules"
```

---

### Task 13: DataTable Interpreter + React Cutover

**Files:**
- Create: `engine/layout-interpreter/data-table-interpreter.ts`
- Create: `engine/layout-interpreter/__tests__/data-table-interpreter.test.ts`
- Modify: `blocks/DataTable.tsx`

- [ ] **Step 1: Write failing interpreter tests**

```typescript
// engine/layout-interpreter/__tests__/data-table-interpreter.test.ts
import { describe, it, expect } from "vitest";
import { interpretDataTable } from "../data-table-interpreter";
import { defaultLayoutTokens } from "../../layout-ir";

describe("interpretDataTable", () => {
  it("defaults to dynamic mode", () => {
    const block = { props: { label: "Checklist", styleJson: "{}", capabilitiesJson: "{}" } };
    const vm = interpretDataTable(block as any, defaultLayoutTokens);
    expect(vm.mode).toBe("dynamic");
    expect(vm.canAddRows).toBe(true);
  });

  it("respects fixed mode from capabilities", () => {
    const block = {
      props: {
        label: "Fixed Table",
        styleJson: "{}",
        capabilitiesJson: JSON.stringify({ mode: "fixed", addRows: false, removeRows: false }),
      },
    };
    const vm = interpretDataTable(block as any, defaultLayoutTokens);
    expect(vm.mode).toBe("fixed");
    expect(vm.canAddRows).toBe(false);
    expect(vm.canRemoveRows).toBe(false);
  });

  it("resolves theme colors for header", () => {
    const block = { props: { label: "Table", styleJson: "{}", capabilitiesJson: "{}" } };
    const vm = interpretDataTable(block as any, defaultLayoutTokens);
    expect(vm.headerBg).toBe(defaultLayoutTokens.theme.accentLight);
  });
});
```

- [ ] **Step 2: Implement interpretDataTable**

```typescript
// engine/layout-interpreter/data-table-interpreter.ts
import type { LayoutTokens } from "../layout-ir";
import { defaultComponentRules } from "../layout-ir";
import { DataTableCodec } from "../codecs";
import { resolveThemeRef } from "../codecs/codec-utils";
import type { DataTableViewModel } from "./view-models";

export function interpretDataTable(
  block: { props: Record<string, unknown> },
  tokens: LayoutTokens,
): DataTableViewModel {
  const style = DataTableCodec.parseStyle((block.props.styleJson as string) ?? "{}");
  const caps = DataTableCodec.parseCaps((block.props.capabilitiesJson as string) ?? "{}");
  const rule = defaultComponentRules.dataTable;

  return {
    label: (block.props.label as string) ?? "",
    mode: caps.mode,
    headerBg: resolveThemeRef(style.headerBackground, tokens.theme) ?? tokens.theme.accentLight,
    headerColor: style.headerColor ?? rule.headerFontColor,
    headerFontWeight: style.headerFontWeight ?? rule.headerFontWeight,
    cellBorderColor: resolveThemeRef(style.cellBorderColor, tokens.theme) ?? tokens.theme.accentBorder,
    cellPadding: style.cellPadding ?? `${rule.cellPaddingMm}mm`,
    density: style.density ?? rule.defaultDensity,
    locked: caps.locked,
    removable: caps.removable,
    canAddRows: caps.addRows,
    canRemoveRows: caps.removeRows,
    canAddColumns: caps.addColumns,
    canRemoveColumns: caps.removeColumns,
    canResizeColumns: caps.resizeColumns,
    headerLocked: caps.headerLocked,
    maxRows: caps.maxRows,
  };
}
```

- [ ] **Step 3: Run interpreter tests**

Run: `cd frontend/apps/web && rtk vitest run src/features/documents/mddm-editor/engine/layout-interpreter/__tests__/data-table-interpreter.test.ts`
Expected: PASS

- [ ] **Step 4: Refactor DataTable.tsx addNodeView to read ViewModel**

In `blocks/DataTable.tsx`, update the `addNodeView()` function to call `interpretDataTable()` for label styling and capability-driven UI (show/hide add-row buttons based on `vm.canAddRows`). The ProseMirror node view reads tokens from the editor instance and calls the interpreter.

- [ ] **Step 5: Build + run tests**

Run: `cd frontend/apps/web && pnpm exec vite build 2>&1 | tail -3 && rtk vitest run src/features/documents/mddm-editor/`
Expected: Build succeeds, tests pass

- [ ] **Step 6: Commit**

```bash
rtk git add frontend/apps/web/src/features/documents/mddm-editor/engine/layout-interpreter/data-table-interpreter.ts frontend/apps/web/src/features/documents/mddm-editor/engine/layout-interpreter/__tests__/data-table-interpreter.test.ts frontend/apps/web/src/features/documents/mddm-editor/blocks/DataTable.tsx
rtk git commit -m "feat(mddm): DataTable interpreter + React render uses ViewModel"
```

---

### Task 14: Repeatable + RepeatableItem Interpreter + React Cutover

**Files:**
- Modify: `engine/layout-interpreter/repeatable-interpreter.ts`
- Create: `engine/layout-interpreter/repeatable-item-interpreter.ts`
- Create: `engine/layout-interpreter/__tests__/repeatable-item-interpreter.test.ts`
- Modify: `blocks/Repeatable.tsx`
- Modify: `blocks/RepeatableItem.tsx`

- [ ] **Step 1: Write failing tests for both interpreters**

Test `interpretRepeatable()`: verify it reads RepeatableCodec, resolves theme border colors, computes `canAddItems` from capabilities + currentItemCount vs maxItems.

Test `interpretRepeatableItem()`: verify it reads RepeatableItemCodec, computes numbered title from context (e.g., "3.1 Etapa 1").

- [ ] **Step 2: Refactor repeatable-interpreter.ts to use RepeatableCodec**

```typescript
// engine/layout-interpreter/repeatable-interpreter.ts
import type { LayoutTokens } from "../layout-ir";
import { defaultComponentRules } from "../layout-ir";
import { RepeatableCodec } from "../codecs";
import { resolveThemeRef } from "../codecs/codec-utils";
import type { RepeatableViewModel } from "./view-models";

export function interpretRepeatable(
  block: { props: Record<string, unknown>; children?: unknown[] },
  tokens: LayoutTokens,
): RepeatableViewModel {
  const style = RepeatableCodec.parseStyle((block.props.styleJson as string) ?? "{}");
  const caps = RepeatableCodec.parseCaps((block.props.capabilitiesJson as string) ?? "{}");
  const rule = defaultComponentRules.repeatable;

  const currentItemCount = block.children?.length ?? 0;

  return {
    label: (block.props.label as string) ?? "",
    itemPrefix: (block.props.itemPrefix as string) ?? "Item",
    borderColor: resolveThemeRef(style.borderColor, tokens.theme) ?? tokens.theme.accentBorder,
    itemAccentBorder: resolveThemeRef(style.itemAccentBorder, tokens.theme) ?? tokens.theme.accent,
    itemAccentWidth: style.itemAccentWidth ?? `${rule.itemAccentWidthPt}pt`,
    locked: caps.locked,
    removable: caps.removable,
    canAddItems: caps.addItems && currentItemCount < caps.maxItems,
    canRemoveItems: caps.removeItems && currentItemCount > caps.minItems,
    maxItems: caps.maxItems,
    minItems: caps.minItems,
    currentItemCount,
  };
}
```

- [ ] **Step 3: Implement interpretRepeatableItem**

```typescript
// engine/layout-interpreter/repeatable-item-interpreter.ts
import type { LayoutTokens } from "../layout-ir";
import { defaultComponentRules } from "../layout-ir";
import { RepeatableItemCodec } from "../codecs";
import { resolveThemeRef } from "../codecs/codec-utils";
import type { RepeatableItemViewModel } from "./view-models";

export function interpretRepeatableItem(
  block: { props: Record<string, unknown> },
  tokens: LayoutTokens,
  context: { itemIndex: number; parentNumber?: string },
): RepeatableItemViewModel {
  const style = RepeatableItemCodec.parseStyle((block.props.styleJson as string) ?? "{}");
  const caps = RepeatableItemCodec.parseCaps((block.props.capabilitiesJson as string) ?? "{}");
  const rule = defaultComponentRules.repeatable;

  const number = context.parentNumber
    ? `${context.parentNumber}.${context.itemIndex + 1}`
    : String(context.itemIndex + 1);

  return {
    title: (block.props.title as string) ?? "",
    number,
    accentBorderColor: resolveThemeRef(style.accentBorderColor, tokens.theme) ?? tokens.theme.accent,
    accentBorderWidth: style.accentBorderWidth ?? `${rule.itemAccentWidthPt}pt`,
    locked: caps.locked,
    removable: caps.removable,
  };
}
```

- [ ] **Step 4: Refactor Repeatable.tsx render() to use interpretRepeatable()**

Replace hardcoded prop reads with ViewModel values. Use `vm.canAddItems` instead of inline `!locked && children.length < maxItems` computation.

- [ ] **Step 5: Refactor RepeatableItem.tsx render() to use interpretRepeatableItem()**

- [ ] **Step 6: Run tests + build**

Run: `cd frontend/apps/web && rtk vitest run src/features/documents/mddm-editor/ && pnpm exec vite build 2>&1 | tail -3`
Expected: PASS + build succeeds

- [ ] **Step 7: Commit**

```bash
rtk git add frontend/apps/web/src/features/documents/mddm-editor/engine/layout-interpreter/ frontend/apps/web/src/features/documents/mddm-editor/blocks/Repeatable.tsx frontend/apps/web/src/features/documents/mddm-editor/blocks/RepeatableItem.tsx
rtk git commit -m "feat(mddm): Repeatable + RepeatableItem interpreters + React ViewModel cutover"
```

---

### Task 15: RichBlock Interpreter + React Cutover

**Files:**
- Create: `engine/layout-interpreter/rich-block-interpreter.ts`
- Create: `engine/layout-interpreter/__tests__/rich-block-interpreter.test.ts`
- Modify: `blocks/RichBlock.tsx`

- [ ] **Step 1: Write failing interpreter tests**

Test `interpretRichBlock()`: verify it reads RichBlockCodec, resolves label background from theme, applies style overrides.

- [ ] **Step 2: Implement interpretRichBlock**

```typescript
// engine/layout-interpreter/rich-block-interpreter.ts
import type { LayoutTokens } from "../layout-ir";
import { defaultComponentRules } from "../layout-ir";
import { RichBlockCodec } from "../codecs";
import { resolveThemeRef } from "../codecs/codec-utils";
import type { RichBlockViewModel } from "./view-models";

export function interpretRichBlock(
  block: { props: Record<string, unknown> },
  tokens: LayoutTokens,
): RichBlockViewModel {
  const style = RichBlockCodec.parseStyle((block.props.styleJson as string) ?? "{}");
  const caps = RichBlockCodec.parseCaps((block.props.capabilitiesJson as string) ?? "{}");
  const rule = defaultComponentRules.richBlock;

  return {
    label: (block.props.label as string) ?? "",
    chrome: (block.props.chrome as string) ?? "labeled",
    labelBackground: resolveThemeRef(style.labelBackground, tokens.theme) ?? tokens.theme.accentLight,
    labelFontSize: style.labelFontSize ?? `${rule.labelFontSizePt}pt`,
    labelColor: style.labelColor ?? rule.labelFontColor,
    borderColor: resolveThemeRef(style.borderColor, tokens.theme) ?? tokens.theme.accentBorder,
    locked: caps.locked,
    removable: caps.removable,
  };
}
```

- [ ] **Step 3: Refactor RichBlock.tsx render() to use interpretRichBlock()**

Replace hardcoded styling with ViewModel values from `interpretRichBlock()`.

- [ ] **Step 4: Run tests + build**

Run: `cd frontend/apps/web && rtk vitest run src/features/documents/mddm-editor/ && pnpm exec vite build 2>&1 | tail -3`
Expected: PASS + build succeeds

- [ ] **Step 5: Commit**

```bash
rtk git add frontend/apps/web/src/features/documents/mddm-editor/engine/layout-interpreter/rich-block-interpreter.ts frontend/apps/web/src/features/documents/mddm-editor/engine/layout-interpreter/__tests__/rich-block-interpreter.test.ts frontend/apps/web/src/features/documents/mddm-editor/blocks/RichBlock.tsx
rtk git commit -m "feat(mddm): RichBlock interpreter + React ViewModel cutover"
```

---

### Task 16: DOCX Emitter Migration — All Block Types

**Files:**
- Modify: `engine/docx-emitter/emitters/section.ts`
- Modify: `engine/docx-emitter/emitters/data-table.ts` (or equivalent)
- Modify: `engine/docx-emitter/emitters/repeatable.ts`
- Modify: `engine/docx-emitter/emitters/repeatable-item.ts`
- Modify: `engine/docx-emitter/emitters/rich-block.ts`

- [ ] **Step 1: Refactor Section DOCX emitter to call interpretSection()**

In `engine/docx-emitter/emitters/section.ts`, replace direct prop reading with:

```typescript
import { interpretSection } from "../../layout-interpreter/section-interpreter";

// Inside emit():
const vm = interpretSection({ props: block.props }, tokens, { sectionIndex: context.sectionIndex ?? 0 });
// Use vm.headerBg, vm.headerColor, vm.headerFontSize, vm.number, vm.title
// instead of reading block.props directly
```

- [ ] **Step 2: Refactor DataTable DOCX emitter to call interpretDataTable()**

Same pattern — replace prop reads with ViewModel values from `interpretDataTable()`.

- [ ] **Step 3: Refactor Repeatable DOCX emitter to call interpretRepeatable()**

- [ ] **Step 4: Refactor RepeatableItem DOCX emitter to call interpretRepeatableItem()**

- [ ] **Step 5: Refactor RichBlock DOCX emitter to call interpretRichBlock()**

- [ ] **Step 6: Run DOCX golden fixture tests**

Run: `cd frontend/apps/web && rtk vitest run src/features/documents/mddm-editor/engine/golden/`
Expected: PASS — golden fixtures should still match (ViewModel produces same values as the old direct-prop-reading path when no style overrides are set)

- [ ] **Step 7: Run full test suite**

Run: `cd frontend/apps/web && rtk vitest run src/features/documents/mddm-editor/`
Expected: ALL PASS

- [ ] **Step 8: Commit**

```bash
rtk git add frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/
rtk git commit -m "refactor(mddm): all DOCX emitters use shared interpreters + ViewModels"
```

---

### Task 17: Update IR Hash + Run Full Test Suite

- [ ] **Step 1: Run IR hash drift test**

Run: `cd frontend/apps/web && rtk vitest run src/features/documents/mddm-editor/engine/ir-hash/`
If fails, update `recorded-hash.ts` with the new hash.

- [ ] **Step 2: Run complete MDDM test suite**

Run: `cd frontend/apps/web && rtk vitest run src/features/documents/mddm-editor/`
Expected: ALL PASS — no regressions across codecs, interpreters, emitters, golden fixtures

- [ ] **Step 3: Fix any regressions found**

- [ ] **Step 4: Final commit**

```bash
rtk git add frontend/apps/web/src/features/documents/mddm-editor/engine/ir-hash/recorded-hash.ts
rtk git commit -m "chore(mddm): update IR hash + verify full test suite after template engine"
```
