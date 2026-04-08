import { execFileSync } from "node:child_process";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

import { expect, test, type APIRequestContext, type Page } from "@playwright/test";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);
const repoRoot = resolve(__dirname, "../../../../../");
const seedScript = resolve(repoRoot, "scripts/e2e-seed.ps1");

const adminUsername = process.env.METALDOCS_E2E_ADMIN_USERNAME ?? "e2e.admin";
const adminPassword = process.env.METALDOCS_E2E_ADMIN_PASSWORD ?? "E2eAdmin123!";
const sameSiteHeaders = { Origin: "http://127.0.0.1:4173" };
const deterministicTemplateKey = "po-default-canvas";
const deterministicTemplateVersion = 1;

type ApiErrorEnvelope = {
  error?: {
    code?: string;
    message?: string;
    details?: Record<string, unknown>;
    trace_id?: string;
  };
};

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

test("mddm save rejection returns structured validation envelope and shows inline error banner", async ({ page }) => {
  await loginAsAdmin(page);

  const suffix = Date.now().toString();
  const documentTitle = `PO Validation ${suffix}`;
  const documentId = await createPoDocumentThroughUi(page, documentTitle);
  await assignBrowserTemplate(page.context().request, documentId);

  const documentUrl = `/#/documents/doc/${encodeURIComponent(documentId)}`;
  await page.goto(documentUrl);
  await openDocumentEditorFromDetail(page);
  await ensureBrowserEditorReady(page);
  await appendEditorText(page, ` Invalid Save ${suffix}`);

  const savePath = `/api/v1/documents/${encodeURIComponent(documentId)}/content/browser`;
  const traceId = `trace-e2e-validation-rejection-${suffix}`;

  await page.route(`**${savePath}`, async (route, request) => {
    if (request.method() !== "POST") {
      await route.continue();
      return;
    }
    const payload = request.postDataJSON() as { body?: string; draftToken?: string } | null;
    await route.continue({
      headers: {
        ...request.headers(),
        "x-trace-id": traceId,
      },
      postData: JSON.stringify({
        body: payload?.body ?? "",
        draftToken: "",
      }),
    });
  });

  const saveRejection = waitForContentSave(page, documentId, (status) => status === 400);
  await page.getByRole("button", { name: "Salvar rascunho" }).click();

  const response = await saveRejection;
  const errorEnvelope = await response.json() as ApiErrorEnvelope;
  expect(errorEnvelope.error?.code).toBe("VALIDATION_ERROR");
  expect(errorEnvelope.error?.message).toBe("Invalid request data");
  expect(errorEnvelope.error?.details).toEqual({});
  expect(errorEnvelope.error?.trace_id).toBe(traceId);

  const errorBanner = page.getByRole("alert");
  await expect(errorBanner).toContainText("Falha no editor");
  await expect(errorBanner).toContainText("Nao foi possivel salvar o rascunho no editor do navegador.");
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
  await ensureDeterministicTemplateAvailable(apiContext);
  const assignmentResponse = await apiContext.put(`/api/v1/documents/${encodeURIComponent(documentId)}/template-assignment`, {
    headers: sameSiteHeaders,
    data: {
      templateKey: deterministicTemplateKey,
      templateVersion: deterministicTemplateVersion,
    },
  });
  expect(assignmentResponse.ok(), `template assignment failed: ${assignmentResponse.status()} ${await assignmentResponse.text()}`).toBeTruthy();
}

type DocumentTemplateItem = {
  templateKey: string;
  version: number;
};

async function ensureDeterministicTemplateAvailable(apiContext: APIRequestContext) {
  const templatesResponse = await apiContext.get("/api/v1/document-templates?profileCode=po", {
    headers: sameSiteHeaders,
  });
  expect(templatesResponse.ok(), `list templates failed: ${templatesResponse.status()} ${await templatesResponse.text()}`).toBeTruthy();

  const templatesBody = await templatesResponse.json() as { items?: DocumentTemplateItem[] };
  const templates = Array.isArray(templatesBody.items) ? templatesBody.items : [];
  const hasDeterministicTemplate = templates.some(
    (item) => item.templateKey === deterministicTemplateKey && item.version === deterministicTemplateVersion,
  );
  const available = templates
    .map((item) => `${item.templateKey}@${item.version}`)
    .join(", ");
  expect(
    hasDeterministicTemplate,
    `deterministic template ${deterministicTemplateKey}@${deterministicTemplateVersion} missing; available: ${available || "none"}`,
  ).toBeTruthy();
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
