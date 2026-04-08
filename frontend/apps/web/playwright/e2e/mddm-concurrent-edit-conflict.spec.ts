import { execFileSync } from "node:child_process";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

import { expect, test, type APIRequestContext, type BrowserContext, type Page } from "@playwright/test";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);
const repoRoot = resolve(__dirname, "../../../../../");
const seedScript = resolve(repoRoot, "scripts/e2e-seed.ps1");

const adminUsername = process.env.METALDOCS_E2E_ADMIN_USERNAME ?? "e2e.admin";
const adminPassword = process.env.METALDOCS_E2E_ADMIN_PASSWORD ?? "E2eAdmin123!";
const sameSiteHeaders = { Origin: "http://127.0.0.1:4173" };

test.beforeAll(() => {
  execFileSync(
    "powershell.exe",
    ["-ExecutionPolicy", "Bypass", "-File", seedScript],
    {
      cwd: repoRoot,
      stdio: "inherit",
    },
  );
});

test("mddm concurrent edit shows conflict for stale save", async ({ browser }) => {
  const ctxA = await browser.newContext();
  const ctxB = await browser.newContext();

  try {
    const pageA = await ctxA.newPage();
    await loginAsAdmin(pageA);

    const suffix = Date.now().toString();
    const documentTitle = `PO Concurrent ${suffix}`;
    const documentId = await createPoDocumentThroughUi(pageA, documentTitle);
    await assignBrowserTemplate(ctxA.request, documentId);

    const documentUrl = `/#/documents/doc/${encodeURIComponent(documentId)}`;
    await pageA.goto(documentUrl);
    await openDocumentEditorFromDetail(pageA);
    await ensureBrowserEditorReady(pageA);

    const pageB = await ctxB.newPage();
    await loginAsAdmin(pageB);
    await pageB.goto(documentUrl);
    await openDocumentEditorFromDetail(pageB);
    await ensureBrowserEditorReady(pageB);

    await appendEditorText(pageA, ` Context A ${suffix}`);
    await appendEditorText(pageB, ` Context B ${suffix}`);

    const saveA = waitForContentSave(pageA, documentId, (status) => status >= 200 && status < 300);
    await pageA.getByRole("button", { name: "Salvar rascunho" }).click();
    await saveA;
    await expect(pageA.getByText(/Salvo agora|Salvo ha pouco|Salvo/i)).toBeVisible({ timeout: 10_000 });

    const saveB = waitForContentSave(pageB, documentId, (status) => status === 409);
    await pageB.getByRole("button", { name: "Salvar rascunho" }).click();
    await saveB;

    const conflictBanner = pageB.getByRole("alert");
    await expect(conflictBanner).toContainText("Conflito de rascunho");
    await expect(conflictBanner).toContainText(/rascunho ficou desatualizado/i);
    await expect(pageB.getByRole("button", { name: "Salvar rascunho" })).toBeDisabled();
  } finally {
    await ctxB.close();
    await ctxA.close();
  }
});

async function loginAsAdmin(page: Page) {
  await page.goto("/");
  await page.getByTestId("login-identifier").fill(adminUsername);
  await page.getByTestId("login-password").fill(adminPassword);
  await page.getByTestId("login-submit").click();
  await expect(page.getByRole("button", { name: "Todos Documentos" })).toBeVisible();
}

async function createPoDocumentThroughUi(page: Page, documentTitle: string) {
  await page.getByRole("button", { name: /novo documento/i }).first().click();
  await expect(page.getByTestId("document-create-form")).toBeVisible();

  await page.getByTestId("document-title").fill(documentTitle);

  const createDocumentResponse = page.waitForResponse(
    (response) => {
      if (response.request().method() !== "POST") {
        return false;
      }
      if (response.status() < 200 || response.status() >= 300) {
        return false;
      }
      const url = new URL(response.url());
      if (url.pathname !== "/api/v1/documents") {
        return false;
      }
      const payload = response.request().postDataJSON() as { title?: string } | null;
      return payload?.title === documentTitle;
    },
    { timeout: 20_000 },
  );

  await page.getByTestId("document-submit").click();

  const response = await createDocumentResponse;
  const createdDocument = await response.json() as { documentId?: string };
  expect(createdDocument.documentId).toBeTruthy();
  return createdDocument.documentId as string;
}

async function assignBrowserTemplate(apiContext: APIRequestContext, documentId: string) {
  const selectedTemplate = await resolveAssignableTemplate(apiContext);
  const assignmentResponse = await apiContext.put(`/api/v1/documents/${encodeURIComponent(documentId)}/template-assignment`, {
    headers: sameSiteHeaders,
    data: {
      templateKey: selectedTemplate.templateKey,
      templateVersion: selectedTemplate.templateVersion,
    },
  });
  expect(assignmentResponse.ok(), `template assignment failed: ${assignmentResponse.status()} ${await assignmentResponse.text()}`).toBeTruthy();
}

type AssignableTemplateItem = {
  templateKey?: string;
  template_key?: string;
  version?: number;
  templateVersion?: number;
  template_version?: number;
  editor?: string;
  editorType?: string;
  editor_type?: string;
  contentFormat?: string;
  content_format?: string;
};

async function resolveAssignableTemplate(apiContext: APIRequestContext) {
  const templatesResponse = await apiContext.get("/api/v1/document-templates?profileCode=po", {
    headers: sameSiteHeaders,
  });
  expect(templatesResponse.ok(), `list templates failed: ${templatesResponse.status()} ${await templatesResponse.text()}`).toBeTruthy();

  const templatesBody = await templatesResponse.json() as { items?: AssignableTemplateItem[] };
  const assignableTemplates = Array.isArray(templatesBody.items)
    ? templatesBody.items
      .map(normalizeAssignableTemplate)
      .filter((item): item is { templateKey: string; version: number; editor?: string; contentFormat?: string } => item !== null)
    : [];

  expect(assignableTemplates.length, "no assignable templates returned for profileCode=po").toBeGreaterThan(0);

  const browserTemplate = assignableTemplates.find(
    (item) => item.editor?.toLowerCase() === "ckeditor5" && item.contentFormat?.toLowerCase() === "html",
  );
  const selectedTemplate = browserTemplate ?? assignableTemplates[0];

  return {
    templateKey: selectedTemplate.templateKey,
    templateVersion: selectedTemplate.version,
  };
}

function normalizeAssignableTemplate(item: AssignableTemplateItem): {
  templateKey: string;
  version: number;
  editor?: string;
  contentFormat?: string;
} | null {
  const templateKey = firstNonEmptyString(item.templateKey, item.template_key);
  const version = firstPositiveNumber(item.version, item.templateVersion, item.template_version);
  if (!templateKey || !version) {
    return null;
  }

  return {
    templateKey,
    version,
    editor: firstNonEmptyString(item.editor, item.editorType, item.editor_type),
    contentFormat: firstNonEmptyString(item.contentFormat, item.content_format),
  };
}

function firstNonEmptyString(...values: Array<unknown>) {
  for (const value of values) {
    if (typeof value === "string") {
      const trimmed = value.trim();
      if (trimmed.length > 0) {
        return trimmed;
      }
    }
  }
  return undefined;
}

function firstPositiveNumber(...values: Array<unknown>) {
  for (const value of values) {
    if (typeof value === "number" && Number.isFinite(value) && value > 0) {
      return value;
    }
    if (typeof value === "string") {
      const parsed = Number(value);
      if (Number.isFinite(parsed) && parsed > 0) {
        return parsed;
      }
    }
  }
  return undefined;
}

async function openDocumentEditorFromDetail(page: Page) {
  const editor = page.getByTestId("browser-document-editor");
  if (await editor.isVisible()) {
    return;
  }

  const openButton = page.getByRole("button", { name: "Abrir documento" });
  await expect(openButton).toBeVisible({ timeout: 20_000 });
  await openButton.click();
}

async function ensureBrowserEditorReady(page: Page) {
  const editorSurface = page.locator(".ck-editor__editable").first();
  const reloadButton = page.getByRole("button", { name: "Recarregar documento" });

  await expect(page.getByTestId("browser-document-editor")).toBeVisible({ timeout: 20_000 });

  for (let attempt = 0; attempt < 3; attempt += 1) {
    if (await editorSurface.isVisible()) {
      return;
    }

    const branch = await Promise.race([
      editorSurface.waitFor({ state: "visible", timeout: 8_000 }).then(() => "editor"),
      reloadButton.waitFor({ state: "visible", timeout: 8_000 }).then(() => "reload"),
    ]).catch(() => "timeout");

    if (branch === "editor") {
      return;
    }

    if (branch === "reload" && await reloadButton.isVisible()) {
      await reloadButton.click();
      continue;
    }
  }

  await expect(editorSurface).toBeVisible({ timeout: 20_000 });
}

async function appendEditorText(page: Page, value: string) {
  const editable = page.locator(".ck-editor__editable").first();
  await expect(editable).toBeVisible();
  await page.locator(".ck-editor__editable .restricted-editing-exception").first().click();
  await page.keyboard.type(value);
}

function waitForContentSave(page: Page, documentId: string, matcher: (status: number) => boolean) {
  const savePath = `/api/v1/documents/${encodeURIComponent(documentId)}/content/browser`;
  return page.waitForResponse(
    (response) => (
      response.request().method() === "POST"
      && matcher(response.status())
      && new URL(response.url()).pathname === savePath
    ),
    { timeout: 20_000 },
  );
}
