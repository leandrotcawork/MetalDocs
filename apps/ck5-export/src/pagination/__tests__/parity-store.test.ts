import { describe, expect, it } from "vitest";
import { ParityStore } from "../parity-store";
import type { ParityReport } from "../parity-diff";

const sampleReport: ParityReport = {
  docId: "doc-a",
  editorBreaks: [{ afterBid: "aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa", pageNumber: 1, yPx: 0 }],
  serverBreaks: [{ bid: "aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa", pageNumber: 1 }],
  reconciled: [{ afterBid: "aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa", pageNumber: 1, source: "editor" }],
  logs: {
    exactMatches: 1,
    minorDrift: 0,
    majorDrift: 0,
    orphanedEditor: 0,
    serverOnly: 0,
  },
  driftStats: {
    totalBreaks: 1,
    driftRatio: 0,
  },
};

describe("ParityStore", () => {
  it("returns undefined for unknown documents", () => {
    const store = new ParityStore();
    expect(store.get("unknown-doc")).toBeUndefined();
  });

  it("stores and retrieves a report with updatedAt timestamp", () => {
    const store = new ParityStore();
    store.put("doc-a", sampleReport);

    const entry = store.get("doc-a");

    expect(entry?.report.docId).toBe("doc-a");
    expect(typeof entry?.updatedAt).toBe("number");
  });

  it("updates updatedAt on successive puts", async () => {
    const store = new ParityStore();
    store.put("doc-a", sampleReport);
    const first = store.get("doc-a");

    await new Promise((resolve) => setTimeout(resolve, 5));

    store.put("doc-a", { ...sampleReport, docId: "doc-a" });
    const second = store.get("doc-a");

    expect(first).toBeDefined();
    expect(second).toBeDefined();
    expect(second!.updatedAt).toBeGreaterThan(first!.updatedAt);
  });
});
