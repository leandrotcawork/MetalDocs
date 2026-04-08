import { execFileSync } from "node:child_process";
import { readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

import { expect, test, type APIRequestContext, type Page } from "@playwright/test";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);
const repoRoot = resolve(__dirname, "../../../../../");
const seedScript = resolve(repoRoot, "scripts/e2e-seed.ps1");
const fixtureImagePath = resolve(__dirname, "fixtures/test-image.png");

const adminUsername = process.env.METALDOCS_E2E_ADMIN_USERNAME ?? "e2e.admin";
const adminPassword = process.env.METALDOCS_E2E_ADMIN_PASSWORD ?? "E2eAdmin123!";
const sameSiteHeaders = { Origin: "http://127.0.0.1:4173" };
const deterministicTemplateKey = "po-default-canvas";
const deterministicTemplateVersion = 1;

const fixtureImageDataUrl = `data:image/png;base64,${readFileSync(fixtureImagePath).toString("base64")}`;

type DocumentTemplateItem = {
  templateKey: string;
  version: number;
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

test("mddm image upload + save + reload roundtrip", async ({ page }) => {
  await loginAsAdmin(page);

  const suffix = Date.now().toString();
  const documentTitle = `PO Image Roundtrip ${suffix}`;
  const documentId = await createPoDocumentThroughUi(page, documentTitle);
  await assignBrowserTemplate(page.context().request, documentId);

  const documentUrl = `/#/documents/doc/${encodeURIComponent(documentId)}`;
  await page.goto(documentUrl);
  await openDocumentEditorFromDetail(page);
  await ensureBrowserEditorReady(page);

  await appendEditorText(page, ` Roundtrip marker ${suffix}`);

  const altText = `e2e-roundtrip-image-${suffix}`;
  const savePath = `/api/v1/documents/${encodeURIComponent(documentId)}/content/browser`;
  let injectedBody = "";

  // Current browser editor flow has no file-picker image control; inject a fixture-backed
  // <img> into the outgoing save payload to verify persisted save/reload behavior deterministically.
  await page.route(`**${savePath}`, async (route, request) => {
    if (request.method() !== "POST") {
      await route.continue();
      return;
    }

    const payload = request.postDataJSON() as { body?: string; draftToken?: string } | null;
    const currentBody = typeof payload?.body === "string" ? payload.body : "";
    injectedBody = `${currentBody}<p><img src="${fixtureImageDataUrl}" alt="${altText}" /></p>`;

    await route.continue({
      headers: {
        ...request.headers(),
        "content-type": "application/json",
      },
      postData: JSON.stringify({
        body: injectedBody,
        draftToken: payload?.draftToken ?? "",
      }),
    });
  });

  try {
    await saveDraftViaUi(page, documentId);
  } finally {
    await page.unroute(`**${savePath}`);
  }

  expect(injectedBody).toContain(altText);

  const bundleBodyAfterSave = await fetchBrowserBundleBody(page.context().request, documentId);
  expect(bundleBodyAfterSave).toContain(altText);
  expect(bundleBodyAfterSave).toContain("data:image/png;base64");

  await page.goto(documentUrl);
  const openButton = page.getByRole("button", { name: "Abrir documento" });
  await expect(openButton).toBeVisible({ timeout: 20_000 });

  const bundlePath = `/api/v1/documents/${encodeURIComponent(documentId)}/browser-editor-bundle`;
  const bundleReloadResponse = page.waitForResponse(
    (response) => (
      response.request().method() === "GET"
      && response.status() >= 200
      && response.status() < 300
      && new URL(response.url()).pathname === bundlePath
    ),
    { timeout: 20_000 },
  );

  // Force a deterministic SPA restart on the detail route before reopening the editor.
  await page.reload({ waitUntil: "domcontentloaded" });
  await expect(openButton).toBeVisible({ timeout: 20_000 });
  await openButton.click();

  const bundleResponse = await bundleReloadResponse;
  const bundlePayload = await bundleResponse.json() as { body?: string };
  expect(typeof bundlePayload.body).toBe("string");
  expect(bundlePayload.body).toContain(altText);
  expect(bundlePayload.body).toContain("data:image/png;base64");
  await ensureBrowserEditorReady(page);
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

  for (let attempt = 0; attempt < 3; attempt += 1) {
    const openButton = page.getByRole("button", { name: "Abrir documento" });
    await expect(openButton).toBeVisible({ timeout: 20_000 });
    try {
      await openButton.click({ timeout: 5_000 });
      return;
    } catch {
      if (await editor.isVisible()) {
        return;
      }
      if (attempt === 2) {
        throw new Error("failed to click 'Abrir documento' after retries");
      }
      await page.waitForTimeout(250);
    }
  }
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

async function saveDraftViaUi(page: Page, documentId: string) {
  const savePath = `/api/v1/documents/${encodeURIComponent(documentId)}/content/browser`;
  const saveResponse = page.waitForResponse(
    (response) => (
      response.request().method() === "POST"
      && response.status() >= 200
      && response.status() < 300
      && new URL(response.url()).pathname === savePath
    ),
    { timeout: 20_000 },
  );

  const saveButton = page.getByRole("button", { name: "Salvar rascunho" });
  await expect(saveButton).toBeEnabled();
  await saveButton.click();

  const response = await saveResponse;
  expect(response.ok()).toBeTruthy();
  await expect(page.getByText(/Salvo agora|Salvo ha pouco|Salvo/i)).toBeVisible({ timeout: 10_000 });
}

async function fetchBrowserBundleBody(apiContext: APIRequestContext, documentId: string) {
  const response = await apiContext.get(`/api/v1/documents/${encodeURIComponent(documentId)}/browser-editor-bundle`, {
    headers: sameSiteHeaders,
  });
  expect(response.ok(), `bundle reload failed: ${response.status()} ${await response.text()}`).toBeTruthy();
  const body = await response.json() as { body?: string };
  expect(typeof body.body).toBe("string");
  return body.body as string;
}
