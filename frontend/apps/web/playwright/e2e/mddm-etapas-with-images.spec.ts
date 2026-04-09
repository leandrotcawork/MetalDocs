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

test("mddm etapas with images saves draft, publishes via API, and exports DOCX", async ({ page }) => {
  await loginAsAdmin(page);

  const suffix = Date.now().toString();
  const documentTitle = `PO E2E Etapas ${suffix}`;
  const apiContext = page.context().request;

  const createdDocumentId = await createPoDocumentThroughUi(page, documentTitle);
  await ensureBrowserEditorReady(page, apiContext, createdDocumentId);

  await addThreeEtapasViaUi(page, suffix);
  await saveDraftViaUi(page, createdDocumentId);
  await publishDocumentViaApi(apiContext, createdDocumentId);

  const documentResponse = await apiContext.get(`/api/v1/documents/${encodeURIComponent(createdDocumentId)}`, {
    headers: sameSiteHeaders,
  });
  expect(documentResponse.ok()).toBeTruthy();
  const publishedDocument = await documentResponse.json() as { status?: string };
  expect(publishedDocument.status).toBe("PUBLISHED");

  const exportResponse = await apiContext.post(`/api/v1/documents/${encodeURIComponent(createdDocumentId)}/export/docx`, {
    headers: sameSiteHeaders,
  });
  expect(exportResponse.ok()).toBeTruthy();
  expect(exportResponse.headers()["content-type"]).toContain("application/vnd.openxmlformats-officedocument.wordprocessingml.document");

  const docxBytes = await exportResponse.body();
  expect(docxBytes.byteLength).toBeGreaterThan(1000);
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

async function addThreeEtapasViaUi(page: Page, suffix: string) {
  const editable = page.locator('[contenteditable="true"]').first();
  await expect(editable).toBeVisible({ timeout: 20_000 });
  await expect(editable).toContainText("Detalhamento das Etapas");

  await editable.click();
  await page.keyboard.type(` Etapa 1 - Preparar materiais (${suffix}) [imagem: etapa-1].`);
  await page.keyboard.press("Enter");
  await page.keyboard.type("Etapa 2 - Executar o fluxo principal [imagem: etapa-2].");
  await page.keyboard.press("Enter");
  await page.keyboard.type("Etapa 3 - Validar e liberar publicacao [imagem: etapa-3].");
}

async function ensureBrowserEditorReady(page: Page, apiContext: APIRequestContext, documentId: string) {
  const browserTemplate = await findBrowserTemplate(apiContext);
  const assignmentResponse = await apiContext.put(`/api/v1/documents/${encodeURIComponent(documentId)}/template-assignment`, {
    headers: sameSiteHeaders,
    data: {
      templateKey: browserTemplate.templateKey,
      templateVersion: browserTemplate.version,
    },
  });
  expect(assignmentResponse.ok(), `template assignment failed: ${assignmentResponse.status()} ${await assignmentResponse.text()}`).toBeTruthy();

  const editorSurface = page.locator('[contenteditable="true"]').first();
  const errorState = page.getByText("Editor indisponivel");
  if (await editorSurface.isVisible()) {
    return;
  }

  if (await errorState.isVisible()) {
    const reloadButton = page.getByRole("button", { name: "Recarregar documento" });
    if (await reloadButton.isVisible()) {
      await reloadButton.click();
    }
  }

  await expect(page.getByTestId("browser-document-editor")).toBeVisible({ timeout: 20_000 });
  await expect(editorSurface).toBeVisible({ timeout: 20_000 });
}

type DocumentTemplateItem = {
  templateKey: string;
  version: number;
  profileCode?: string;
  editor?: string;
  contentFormat?: string;
};

async function findBrowserTemplate(apiContext: APIRequestContext) {
  const templatesResponse = await apiContext.get("/api/v1/document-templates?profileCode=po", {
    headers: sameSiteHeaders,
  });
  expect(templatesResponse.ok(), `list templates failed: ${templatesResponse.status()} ${await templatesResponse.text()}`).toBeTruthy();

  const templatesBody = await templatesResponse.json() as { items?: DocumentTemplateItem[] };
  const templates = Array.isArray(templatesBody.items) ? templatesBody.items : [];
  const browserTemplate = templates.find((item) => item.profileCode === "po" && item.contentFormat === "html");
  const available = templates.map((item) => `${item.templateKey}@${item.version}`).join(", ");
  expect(browserTemplate, `browser template missing; available: ${available || "none"}`).toBeTruthy();
  if (!browserTemplate) {
    throw new Error("Expected a browser-compatible PO template in the template catalog.");
  }
  return browserTemplate;
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

async function publishDocumentViaApi(apiContext: APIRequestContext, documentId: string) {
  const userResponse = await apiContext.get("/api/v1/auth/me", { headers: sameSiteHeaders });
  expect(userResponse.ok()).toBeTruthy();
  const userBody = await userResponse.json() as { userId?: string };
  expect(userBody.userId).toBeTruthy();
  const actorId = userBody.userId as string;

  await releaseTransition(apiContext, documentId, "IN_REVIEW", actorId, "Enviar para revisao (e2e)");
  await releaseTransition(apiContext, documentId, "APPROVED", actorId, "Aprovar documento (e2e)");
  await releaseTransition(apiContext, documentId, "PUBLISHED", actorId, "Publicar documento (e2e)");
}

async function releaseTransition(
  apiContext: APIRequestContext,
  documentId: string,
  toStatus: string,
  actorId: string,
  reason: string,
) {
  const response = await apiContext.post(`/api/v1/workflow/documents/${encodeURIComponent(documentId)}/transitions`, {
    headers: sameSiteHeaders,
    data: {
      toStatus,
      assignedReviewer: toStatus === "IN_REVIEW" ? actorId : undefined,
      reason,
    },
  });
  expect(response.ok()).toBeTruthy();
  const body = await response.json() as { toStatus?: string };
  expect(body.toStatus).toBe(toStatus);
}
