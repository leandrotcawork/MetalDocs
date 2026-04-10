import { execFileSync } from "node:child_process";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

import { expect, test, type Page } from "@playwright/test";

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

test("browser document editor opens as a single document surface", async ({ page }) => {
  await loginAsAdmin(page);
  const createdDocument = await createBrowserTemplateDocument(page);
  const apiContext = page.context().request;

  const templatesResponse = await apiContext.get("/api/v1/document-templates?profileCode=po", { headers: sameSiteHeaders });
  expect(templatesResponse.ok()).toBeTruthy();
  const templatesBody = (await templatesResponse.json()) as {
    items?: Array<{
      templateKey?: string;
      version?: number;
      profileCode?: string;
      editor?: string;
      contentFormat?: string;
    }>;
  };
  expect(Array.isArray(templatesBody.items)).toBeTruthy();
  const browserTemplate = templatesBody.items?.find(
    (item) =>
      item.templateKey === "po-mddm-canvas"
      && item.editor === "mddm-blocknote"
      && item.contentFormat === "mddm",
  );
  const available = templatesBody.items?.map((item) => `${item.templateKey}@${item.version}`).join(", ");
  expect(browserTemplate).toBeDefined();
  if (!browserTemplate) {
    throw new Error(`Expected a browser-compatible PO template in the template catalog; available: ${available || "none"}.`);
  }
  expect(browserTemplate.templateKey).toBeTruthy();
  expect(browserTemplate.version).toBeGreaterThan(0);

  const assignmentResponse = await apiContext.put(`/api/v1/documents/${encodeURIComponent(createdDocument.documentId)}/template-assignment`, {
    headers: sameSiteHeaders,
    data: {
      templateKey: browserTemplate.templateKey,
      templateVersion: browserTemplate.version,
    },
  });
  expect(assignmentResponse.ok()).toBeTruthy();

  await page.goto(`/#/documents/doc/${encodeURIComponent(createdDocument.documentId)}`);
  await page.getByRole("button", { name: "Abrir documento" }).click();

  const editorRoot = page.getByTestId("browser-document-editor");
  const editable = editorRoot.locator('[contenteditable="true"]').first();

  await expect(editorRoot).toBeVisible({ timeout: 20_000 });
  await expect(page.getByTestId("document-editor-header")).toBeVisible({ timeout: 5_000 });

  await expect(editable).toBeVisible();
  await editable.click();
  await page.keyboard.type(` Objetivo do teste ${Date.now()}`);

  const saveButton = page.getByRole("button", { name: "Salvar rascunho" });
  await expect(saveButton).toBeEnabled();
  await saveButton.click();
  await expect(page.getByText(/Salvo agora|Salvo ha pouco/)).toBeVisible();

  const exportResponse = await apiContext.post(`/api/v1/documents/${encodeURIComponent(createdDocument.documentId)}/export/docx`, {
    headers: sameSiteHeaders,
  });
  expect(exportResponse.ok()).toBeTruthy();
  expect(exportResponse.headers()["content-type"]).toContain("application/vnd.openxmlformats-officedocument.wordprocessingml.document");
});

test("native create flow opens the browser editor with a persisted document id", async ({ page }) => {
  await loginAsAdmin(page);

  const suffix = Date.now().toString();
  const documentTitle = `Native Editor ${suffix}`;

  await page.getByRole("button", { name: "Novo documento" }).first().click();
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
  const createdDocument = (await response.json()) as { documentId?: string };
  expect(createdDocument.documentId).toBeTruthy();

  const bundleResponse = await page.waitForResponse(
    (item) =>
      item.url().includes(`/api/v1/documents/${encodeURIComponent(createdDocument.documentId ?? "")}/browser-editor-bundle`)
      && item.request().method() === "GET"
      && item.status() >= 200
      && item.status() < 300,
    { timeout: 20_000 },
  );
  expect(bundleResponse.ok()).toBeTruthy();

  const editorRoot = page.getByTestId("browser-document-editor");
  const editable = editorRoot.locator('[contenteditable="true"]').first();

  await expect(editorRoot).toBeVisible({ timeout: 20_000 });
  await expect(editable).toBeVisible();
  await expect(page.getByTestId("document-editor-header")).toBeVisible({ timeout: 5_000 });
});

async function loginAsAdmin(page: Page) {
  await page.goto("/");
  await page.getByTestId("login-identifier").fill(adminUsername);
  await page.getByTestId("login-password").fill(adminPassword);
  await page.getByTestId("login-submit").click();
  await expect(page.getByRole("button", { name: "Todos Documentos" })).toBeVisible();
}

async function createBrowserTemplateDocument(page: Page) {
  const apiContext = page.context().request;
  const currentUser = await apiContext.get("/api/v1/auth/me", { headers: sameSiteHeaders });
  expect(currentUser.ok()).toBeTruthy();
  const currentUserBody = (await currentUser.json()) as { userId?: string };

  const suffix = Date.now().toString();
  const createResponse = await apiContext.post("/api/v1/documents", {
    headers: sameSiteHeaders,
    data: {
      title: `Browser Editor ${suffix}`,
      documentType: "po",
      documentProfile: "po",
      ownerId: currentUserBody.userId ?? "",
      businessUnit: "operations",
      department: "sgq",
      classification: "INTERNAL",
      tags: [],
    },
  });

  expect(createResponse.ok()).toBeTruthy();
  return (await createResponse.json()) as { documentId: string };
}
