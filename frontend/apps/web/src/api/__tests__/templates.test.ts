import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import {
  listTemplates,
  createTemplate,
  saveDraft,
  publishTemplate,
  exportTemplate,
  importTemplate,
  TemplateLockConflictError,
  TemplatePublishValidationError,
  type TemplateListItemDTO,
  type TemplateDraftDTO,
  type TemplateVersionDTO,
  type PublishErrorDTO,
} from "../templates";

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function makeResponse(body: unknown, status = 200): Response {
  const isBlob = body instanceof Blob;
  return {
    ok: status >= 200 && status < 300,
    status,
    json: () => Promise.resolve(isBlob ? null : body),
    blob: () => Promise.resolve(isBlob ? body : new Blob()),
    headers: new Headers(),
  } as unknown as Response;
}

function makeErrorResponse(status: number, body: unknown): Response {
  return {
    ok: false,
    status,
    json: () => Promise.resolve(body),
    blob: () => Promise.resolve(new Blob()),
    headers: new Headers(),
  } as unknown as Response;
}

// ---------------------------------------------------------------------------
// Setup
// ---------------------------------------------------------------------------

beforeEach(() => {
  vi.stubGlobal("fetch", vi.fn());
});

afterEach(() => {
  vi.unstubAllGlobals();
});

const fetchMock = () => vi.mocked(fetch);

// ---------------------------------------------------------------------------
// 1. listTemplates
// ---------------------------------------------------------------------------

describe("listTemplates", () => {
  it("sends GET to /api/v1/templates with profileCode query param", async () => {
    const items: TemplateListItemDTO[] = [
      { templateKey: "tmpl-1", version: 2, profileCode: "MDDM", name: "Safety Manual", status: "published" },
    ];
    fetchMock().mockResolvedValueOnce(makeResponse(items));

    const result = await listTemplates("MDDM");

    expect(fetchMock()).toHaveBeenCalledOnce();
    const [url] = fetchMock().mock.calls[0] as [string, RequestInit?];
    expect(url).toContain("/templates");
    expect(url).toContain("profileCode=MDDM");

    expect(result).toEqual(items);
  });

  it("encodes special characters in profileCode", async () => {
    fetchMock().mockResolvedValueOnce(makeResponse([]));

    await listTemplates("MDDM/v2");

    const [url] = fetchMock().mock.calls[0] as [string, RequestInit?];
    expect(url).toContain("profileCode=MDDM%2Fv2");
  });
});

// ---------------------------------------------------------------------------
// 2. createTemplate
// ---------------------------------------------------------------------------

describe("createTemplate", () => {
  it("sends POST with profileCode and name in JSON body", async () => {
    const draft: TemplateDraftDTO = {
      templateKey: "tmpl-new",
      profileCode: "MDDM",
      name: "New Template",
      status: "draft",
      lockVersion: 1,
      hasStrippedFields: false,
      blocks: [],
      updatedAt: "2026-04-13T00:00:00Z",
    };
    fetchMock().mockResolvedValueOnce(makeResponse(draft, 201));

    const result = await createTemplate("MDDM", "New Template");

    expect(fetchMock()).toHaveBeenCalledOnce();
    const [url, init] = fetchMock().mock.calls[0] as [string, RequestInit?];
    expect(url).toContain("/templates");
    expect(init?.method).toBe("POST");
    expect(JSON.parse(init?.body as string)).toEqual({ profileCode: "MDDM", name: "New Template" });

    expect(result).toEqual(draft);
  });
});

// ---------------------------------------------------------------------------
// 3. saveDraft — 409 throws TemplateLockConflictError
// ---------------------------------------------------------------------------

describe("saveDraft", () => {
  it("returns updated draft on success", async () => {
    const draft: TemplateDraftDTO = {
      templateKey: "tmpl-1",
      profileCode: "MDDM",
      name: "Safety Manual",
      status: "draft",
      lockVersion: 2,
      hasStrippedFields: false,
      blocks: { type: "doc", content: [] },
      updatedAt: "2026-04-13T10:00:00Z",
    };
    fetchMock().mockResolvedValueOnce(makeResponse(draft));

    const result = await saveDraft("tmpl-1", {
      blocks: { type: "doc", content: [] },
      lockVersion: 1,
    });

    expect(result).toEqual(draft);
    const [url, init] = fetchMock().mock.calls[0] as [string, RequestInit?];
    expect(url).toContain("/templates/tmpl-1/draft");
    expect(init?.method).toBe("PUT");
  });

  it("throws TemplateLockConflictError on 409", async () => {
    fetchMock().mockResolvedValueOnce(
      makeErrorResponse(409, {
        error: { code: "LOCK_CONFLICT", message: "Stale lock version", details: {}, trace_id: "t1" },
      }),
    );

    await expect(
      saveDraft("tmpl-1", { blocks: {}, lockVersion: 0 }),
    ).rejects.toBeInstanceOf(TemplateLockConflictError);
  });

  it("preserves server message in TemplateLockConflictError", async () => {
    fetchMock().mockResolvedValueOnce(
      makeErrorResponse(409, {
        error: { code: "LOCK_CONFLICT", message: "Custom message", details: {}, trace_id: "t2" },
      }),
    );

    const err = await saveDraft("tmpl-1", { blocks: {}, lockVersion: 0 }).catch((e: unknown) => e);
    expect((err as TemplateLockConflictError).message).toBe("Custom message");
    expect((err as TemplateLockConflictError).status).toBe(409);
  });
});

// ---------------------------------------------------------------------------
// 4. publishTemplate — 422 throws TemplatePublishValidationError
// ---------------------------------------------------------------------------

describe("publishTemplate", () => {
  it("returns TemplateVersionDTO on success", async () => {
    const version: TemplateVersionDTO = {
      templateKey: "tmpl-1",
      version: 1,
      profileCode: "MDDM",
      name: "Safety Manual",
      status: "published",
    };
    fetchMock().mockResolvedValueOnce(makeResponse(version));

    const result = await publishTemplate("tmpl-1", 3);

    expect(result).toEqual(version);
    const [url, init] = fetchMock().mock.calls[0] as [string, RequestInit?];
    expect(url).toContain("/templates/tmpl-1/publish");
    expect(init?.method).toBe("POST");
    expect(JSON.parse(init?.body as string)).toEqual({ lockVersion: 3 });
  });

  it("throws TemplatePublishValidationError on 422 with errors array", async () => {
    const errors: PublishErrorDTO[] = [
      { blockId: "b1", blockType: "section", field: "title", reason: "Title is required" },
      { blockId: "b2", blockType: "table", field: "rows", reason: "At least one row required" },
    ];
    fetchMock().mockResolvedValueOnce(
      makeErrorResponse(422, {
        errors,
        error: { code: "VALIDATION_FAILED", message: "Validation failed", details: {}, trace_id: "t3" },
      }),
    );

    const err = await publishTemplate("tmpl-1", 1).catch((e: unknown) => e);
    expect(err).toBeInstanceOf(TemplatePublishValidationError);
    expect((err as TemplatePublishValidationError).errors).toEqual(errors);
    expect((err as TemplatePublishValidationError).status).toBe(422);
  });

  it("throws TemplatePublishValidationError with empty errors array when body has no errors key", async () => {
    fetchMock().mockResolvedValueOnce(
      makeErrorResponse(422, {
        error: { code: "VALIDATION_FAILED", message: "Validation failed", details: {}, trace_id: "t4" },
      }),
    );

    const err = await publishTemplate("tmpl-1", 1).catch((e: unknown) => e);
    expect(err).toBeInstanceOf(TemplatePublishValidationError);
    expect((err as TemplatePublishValidationError).errors).toEqual([]);
  });

  it("throws TemplateLockConflictError on 409", async () => {
    fetchMock().mockResolvedValueOnce(
      makeErrorResponse(409, {
        error: { code: "LOCK_CONFLICT", message: "Conflict", details: {}, trace_id: "t5" },
      }),
    );

    await expect(publishTemplate("tmpl-1", 1)).rejects.toBeInstanceOf(TemplateLockConflictError);
  });
});

// ---------------------------------------------------------------------------
// 5. exportTemplate — returns Blob
// ---------------------------------------------------------------------------

describe("exportTemplate", () => {
  it("returns a Blob for the exported template file", async () => {
    const content = JSON.stringify({ templateKey: "tmpl-1", version: 2 });
    const blob = new Blob([content], { type: "application/json" });
    fetchMock().mockResolvedValueOnce({
      ok: true,
      status: 200,
      blob: () => Promise.resolve(blob),
      headers: new Headers(),
    } as unknown as Response);

    const result = await exportTemplate("tmpl-1", 2);

    expect(result).toBeInstanceOf(Blob);
    const [url, init] = fetchMock().mock.calls[0] as [string, RequestInit?];
    expect(url).toContain("/templates/tmpl-1/export");
    expect(url).toContain("version=2");
    // exportTemplate uses GET (no method override)
    expect(init?.method ?? "GET").toBe("GET");
  });

  it("encodes special characters in template key", async () => {
    const blob = new Blob(["{}"], { type: "application/json" });
    fetchMock().mockResolvedValueOnce({
      ok: true,
      status: 200,
      blob: () => Promise.resolve(blob),
      headers: new Headers(),
    } as unknown as Response);

    await exportTemplate("tmpl/with-slash", 1);

    const [url] = fetchMock().mock.calls[0] as [string, RequestInit?];
    expect(url).toContain("tmpl%2Fwith-slash");
  });
});

// ---------------------------------------------------------------------------
// 6. importTemplate — uses FormData
// ---------------------------------------------------------------------------

describe("importTemplate", () => {
  it("sends POST with FormData body (not JSON)", async () => {
    const importResult = {
      templateKey: "tmpl-imported",
      hasStrippedFields: false,
      strippedFields: [],
    };
    fetchMock().mockResolvedValueOnce(makeResponse(importResult, 201));

    const file = new File(['{"name":"test"}'], "template.json", { type: "application/json" });
    const result = await importTemplate("MDDM", file);

    expect(result).toEqual(importResult);
    expect(fetchMock()).toHaveBeenCalledOnce();
    const [url, init] = fetchMock().mock.calls[0] as [string, RequestInit?];
    expect(url).toContain("/templates/import");
    expect(url).toContain("profileCode=MDDM");
    expect(init?.method).toBe("POST");
    // Body must be FormData, not a JSON string
    expect(init?.body).toBeInstanceOf(FormData);
    const form = init?.body as FormData;
    expect(form.get("file")).toBeInstanceOf(File);
    expect((form.get("file") as File).name).toBe("template.json");
  });

  it("does NOT set Content-Type application/json header when using FormData", async () => {
    const importResult = { templateKey: "t", hasStrippedFields: false, strippedFields: [] };
    fetchMock().mockResolvedValueOnce(makeResponse(importResult));

    const file = new File(["{}"], "t.json", { type: "application/json" });
    await importTemplate("MDDM", file);

    const [, init] = fetchMock().mock.calls[0] as [string, RequestInit?];
    const headers = init?.headers as Record<string, string> | undefined;
    // The client.ts request() helper skips Content-Type when body is FormData
    expect(headers?.["Content-Type"]).toBeUndefined();
  });
});
