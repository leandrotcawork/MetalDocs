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

type DocumentTemplateItem = {
  templateKey: string;
  version: number;
  profileCode?: string;
  editor?: string;
  contentFormat?: string;
};

type MddmBlock = {
  id: string;
  type: string;
  props: Record<string, unknown>;
  template_block_id?: string;
  children?: Array<MddmBlock | MddmTextRun>;
};

type MddmTextRun = {
  text: string;
  marks?: { type: string }[];
  link?: { href: string; title?: string };
  document_ref?: { target_document_id: string; target_revision_label?: string };
};

type MddmEnvelope = {
  mddm_version: number;
  template_ref: unknown;
  blocks: MddmBlock[];
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

test("mddm create from template renders a single etapa item with the expected rich scaffold", async ({ page }) => {
  await loginAsAdmin(page);

  const suffix = Date.now().toString();
  const documentTitle = `PO Template ${suffix}`;
  const apiContext = page.context().request;
  const documentId = await createPoDocumentThroughUi(page, documentTitle);
  await assignBrowserTemplate(apiContext, documentId);

  const documentUrl = `/#/documents/doc/${encodeURIComponent(documentId)}`;
  await page.goto(documentUrl);
  await openDocumentEditorFromDetail(page);
  await ensureBrowserEditorReady(page);

  const etapasBlock = page.locator('[data-mddm-block="repeatable"]').filter({ hasText: "Etapas" }).first();
  await expect(etapasBlock).toBeVisible();
  await expect(etapasBlock.locator('[data-mddm-block="repeatableItem"]')).toHaveCount(1);
  await expect(etapasBlock.locator('[data-mddm-block="repeatableItem"]').first()).toContainText("Etapa 1");
  await expect(etapasBlock.locator('[data-mddm-block="richBlock"]').first()).toContainText("Conteúdo da etapa");

  const bundle = await fetchBrowserBundle(apiContext, documentId);
  const envelope = JSON.parse(bundle.body) as MddmEnvelope;
  const etapas = findBlock(envelope.blocks, (block) => block.type === "repeatable" && block.props.label === "Etapas");
  expect(etapas).toBeTruthy();
  expect(etapas?.props.minItems).toBe(1);
  expect(etapas?.children).toHaveLength(1);

  const etapaChildren = (etapas?.children ?? []).filter(isBlock);
  const etapaItem = etapaChildren[0];
  expect(etapaItem?.type).toBe("repeatableItem");

  const etapaRichBlock = findBlock(etapaChildren, (block) => block.type === "richBlock" && block.props.label === "Conteúdo da etapa");
  expect(etapaRichBlock).toBeTruthy();

  await saveDraftViaUi(page, documentId);
  const savedBundle = await fetchBrowserBundle(apiContext, documentId);
  expect(savedBundle.body).toContain("Etapa 1");
  expect(savedBundle.body).toContain("Conteúdo da etapa");
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
  const browserTemplate = await findBrowserTemplate(apiContext);
  const assignmentResponse = await apiContext.put(`/api/v1/documents/${encodeURIComponent(documentId)}/template-assignment`, {
    headers: sameSiteHeaders,
    data: {
      templateKey: browserTemplate.templateKey,
      templateVersion: browserTemplate.version,
    },
  });
  expect(assignmentResponse.ok(), `template assignment failed: ${assignmentResponse.status()} ${await assignmentResponse.text()}`).toBeTruthy();
}

async function findBrowserTemplate(apiContext: APIRequestContext) {
  const templatesResponse = await apiContext.get("/api/v1/document-templates?profileCode=po", {
    headers: sameSiteHeaders,
  });
  expect(templatesResponse.ok(), `list templates failed: ${templatesResponse.status()} ${await templatesResponse.text()}`).toBeTruthy();

  const templatesBody = await templatesResponse.json() as { items?: DocumentTemplateItem[] };
  const templates = Array.isArray(templatesBody.items) ? templatesBody.items : [];
  const browserTemplate = templates.find(
    (item) =>
      item.templateKey === "po-mddm-canvas"
      && item.editor === "mddm-blocknote"
      && item.contentFormat === "mddm",
  );
  const available = templates.map((item) => `${item.templateKey}@${item.version}`).join(", ");
  expect(browserTemplate, `browser template missing; available: ${available || "none"}`).toBeTruthy();
  if (!browserTemplate) {
    throw new Error(`Expected a browser-compatible PO template in the template catalog; available: ${available || "none"}.`);
  }
  return browserTemplate;
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
  const editorRoot = page.getByTestId("browser-document-editor");
  const editorSurface = editorRoot.locator('[contenteditable="true"]').first();
  const reloadButton = page.getByRole("button", { name: "Recarregar documento" });

  await expect(editorRoot).toBeVisible({ timeout: 20_000 });

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

async function fetchBrowserBundle(apiContext: APIRequestContext, documentId: string) {
  const response = await apiContext.get(`/api/v1/documents/${encodeURIComponent(documentId)}/browser-editor-bundle`, {
    headers: sameSiteHeaders,
  });
  expect(response.ok(), `bundle fetch failed: ${response.status()} ${await response.text()}`).toBeTruthy();
  const payload = await response.json() as { body?: string; draftToken?: string };
  expect(typeof payload.body).toBe("string");
  expect(typeof payload.draftToken).toBe("string");
  return {
    body: payload.body ?? "",
    draftToken: payload.draftToken ?? "",
  };
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

function findBlock(
  blocks: MddmBlock[],
  predicate: (block: MddmBlock) => boolean,
): MddmBlock | undefined {
  for (const block of blocks) {
    if (predicate(block)) {
      return block;
    }
    const nested = findBlock(
      (block.children ?? []).filter((child): child is MddmBlock => isBlock(child)),
      predicate,
    );
    if (nested) {
      return nested;
    }
  }
  return undefined;
}

function isBlock(value: MddmBlock | MddmTextRun): value is MddmBlock {
  return typeof value === "object" && value !== null && "type" in value;
}
