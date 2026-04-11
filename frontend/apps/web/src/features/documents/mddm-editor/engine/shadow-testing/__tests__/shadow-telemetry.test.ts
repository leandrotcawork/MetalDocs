import { afterEach, describe, expect, it, vi } from "vitest";
import { postShadowDiff } from "../shadow-telemetry";

describe("postShadowDiff", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("POSTs the event to /api/v1/telemetry/mddm-shadow-diff with JSON content type", async () => {
    const spy = vi.fn().mockResolvedValue(new Response(null, { status: 202 }));
    vi.stubGlobal("fetch", spy);

    await postShadowDiff({
      document_id: "doc-1",
      version_number: 3,
      user_id_hash: "uh",
      current_xml_hash: "ch",
      shadow_xml_hash: "sh",
      diff_summary: { identical: true },
      current_duration_ms: 500,
      shadow_duration_ms: 800,
    });

    expect(spy).toHaveBeenCalledTimes(1);
    const [url, init] = spy.mock.calls[0];
    expect(url).toBe("/api/v1/telemetry/mddm-shadow-diff");
    expect(init?.method).toBe("POST");
    expect((init?.headers as Record<string, string>)["Content-Type"]).toBe("application/json");
    const body = JSON.parse(init?.body as string);
    expect(body.document_id).toBe("doc-1");
    expect(body.diff_summary.identical).toBe(true);
  });

  it("never throws (fire-and-forget semantics)", async () => {
    vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new Error("network down")));

    await expect(postShadowDiff({
      document_id: "d",
      version_number: 1,
      user_id_hash: "",
      current_xml_hash: "",
      shadow_xml_hash: "",
      diff_summary: {},
      current_duration_ms: 0,
      shadow_duration_ms: 0,
    })).resolves.not.toThrow();
  });
});
