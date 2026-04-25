import { Schema } from "@tiptap/pm/model";
import { EditorState } from "@tiptap/pm/state";
import { describe, expect, test } from "vitest";

import { filterTransactionGuard } from "../filter-transaction-guard";

const schema = new Schema({
  nodes: {
    doc: { content: "block+" },
    paragraph: { group: "block", content: "inline*" },
    text: { group: "inline" },
    sdt: { group: "block", content: "block*", attrs: { sdtLock: { default: "" } } },
  },
});

function paragraph(text: string) {
  return schema.nodes.paragraph.create(null, schema.text(text));
}

function buildState() {
  const doc = schema.nodes.doc.create(null, [
    schema.nodes.sdt.create({ sdtLock: "sdtContentLocked" }, [paragraph("locked")]),
    schema.nodes.sdt.create({ sdtLock: "" }, [paragraph("open")]),
  ]);
  return EditorState.create({
    schema,
    doc,
    plugins: [filterTransactionGuard()],
  });
}

function getTextPos(state: EditorState, expectedText: string): number {
  let foundPos = -1;
  state.doc.descendants((node, pos) => {
    if (node.isText && node.text === expectedText) {
      foundPos = pos;
      return false;
    }
    return true;
  });
  return foundPos;
}

describe("filterTransactionGuard", () => {
  test("blocks doc-changing transaction inside locked sdt node", () => {
    const state = buildState();
    const lockedTextPos = getTextPos(state, "locked");
    expect(lockedTextPos).toBeGreaterThan(-1);

    const tr = state.tr.insertText("x", lockedTextPos, lockedTextPos + "locked".length);
    const filter = state.plugins[0].spec.filterTransaction;
    expect(filter).toBeDefined();
    expect(filter?.(tr, state)).toBe(false);
  });

  test("allows doc-changing transaction outside locked sdt node", () => {
    const state = buildState();
    const openTextPos = getTextPos(state, "open");
    expect(openTextPos).toBeGreaterThan(-1);

    const tr = state.tr.insertText("y", openTextPos, openTextPos + "open".length);
    const filter = state.plugins[0].spec.filterTransaction;
    expect(filter).toBeDefined();
    expect(filter?.(tr, state)).toBe(true);
  });
});
