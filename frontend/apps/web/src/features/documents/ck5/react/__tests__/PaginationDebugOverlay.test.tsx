import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import { PaginationDebugOverlay } from "../PaginationDebugOverlay";

describe("<PaginationDebugOverlay />", () => {
  it("renders nothing when debugFlag is false", () => {
    const { queryByTestId } = render(
      <PaginationDebugOverlay
        debugFlag={false}
        logs={{ exactMatches: 3, minorDrift: 1, majorDrift: 0, orphanedEditor: 0, serverOnly: 0 }}
      />,
    );

    expect(queryByTestId("pagination-debug-overlay")).toBeNull();
  });

  it("renders overlay with reconcile counters when debugFlag is true", () => {
    render(
      <PaginationDebugOverlay
        debugFlag
        logs={{ exactMatches: 3, minorDrift: 1, majorDrift: 0, orphanedEditor: 0, serverOnly: 0 }}
      />,
    );

    const overlay = screen.getByTestId("pagination-debug-overlay");
    const text = overlay.textContent ?? "";
    expect(text).toContain("exactMatches:3");
    expect(text).toContain("minorDrift:1");
    expect(text).toContain("majorDrift:0");
    expect(text).toContain("orphanedEditor:0");
    expect(text).toContain("serverOnly:0");
  });
});
