import { execFileSync } from "node:child_process";
import { randomUUID } from "node:crypto";
import { readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

import { expect, test, type APIRequestContext, type Locator, type Page } from "@playwright/test";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);
const repoRoot = resolve(__dirname, "../../../../../");
const seedScript = resolve(repoRoot, "scripts/e2e-seed.ps1");
const fixtureImagePath = resolve(__dirname, "fixtures/test-image.png");

const adminUsername = process.env.METALDOCS_E2E_ADMIN_USERNAME ?? "e2e.admin";
const adminPassword = process.env.METALDOCS_E2E_ADMIN_PASSWORD ?? "E2eAdmin123!";
const sameSiteHeaders = { Origin: "http://127.0.0.1:4173" };

const fixtureImageBuffer = readFileSync(fixtureImagePath);

type DocumentTemplateItem = {
  templateKey: string;
  version: number;
  profileCode?: string;
  editor?: string;
  contentFormat?: string;
};

type MddmTextRun = {
  text: string;
  marks?: { type: string }[];
  link?: { href: string; title?: string };
  document_ref?: { target_document_id: string; target_revision_label?: string };
};

type MddmBlock = {
  id: string;
  type: string;
  props: Record<string, unknown>;
  template_block_id?: string;
  children?: Array<MddmBlock | MddmTextRun>;
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

test("mddm etapas with images keeps minItems, rich edits, and repeatable persistence stable", async ({ page }) => {
  await loginAsAdmin(page);

  const suffix = Date.now().toString();
  const documentTitle = `PO E2E Etapas ${suffix}`;
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
  await expect(etapasBlock.locator('[data-mddm-block="richBlock"]').first()).toContainText("ConteÃºdo da etapa");

  const richBlock = etapasBlock.locator('[data-mddm-block="richBlock"]').first();
  const richEditors = richBlock.locator('[contenteditable="true"]');
  await expect(richEditors).toHaveCount(4);

  await appendToEditable(richEditors.nth(0), ` :: paragrafo-${suffix}`);
  await appendToEditable(richEditors.nth(1), ` :: bullets-${suffix}`);
  await appendToEditable(richEditors.nth(2), ` :: numerado-${suffix}`);
  await appendToEditable(richEditors.nth(3), ` :: tabela-${suffix}`);

  await saveDraftViaUi(page, documentId);
  let bundle = await fetchBrowserBundle(apiContext, documentId);
  expect(bundle.body).toContain(`paragrafo-${suffix}`);
  expect(bundle.body).toContain(`bullets-${suffix}`);
  expect(bundle.body).toContain(`numerado-${suffix}`);
  expect(bundle.body).toContain(`tabela-${suffix}`);

  const uploadedImageDownloadUrl = await uploadFixtureAttachmentAndGetDownloadUrl(apiContext, documentId);
  await insertImageViaSlashMenu(page, richEditors.nth(0), uploadedImageDownloadUrl);
  const image = richBlock.locator("img.bn-visual-media").last();
  await expect(image).toBeVisible();
  await expect(image).toHaveAttribute("src", uploadedImageDownloadUrl);

  await saveDraftViaUi(page, documentId);
  bundle = await fetchBrowserBundle(apiContext, documentId);
  expect(bundle.body).toContain(uploadedImageDownloadUrl);
  expect(bundle.body).toContain(`paragrafo-${suffix}`);

  await expandRepeatableViaBundle(apiContext, documentId);
  await page.reload({ waitUntil: "domcontentloaded" });
  await openDocumentEditorFromDetail(page);
  await ensureBrowserEditorReady(page);
  await expect(page.locator('[data-mddm-block="repeatable"]').filter({ hasText: "Etapas" }).first().locator('[data-mddm-block="repeatableItem"]')).toHaveCount(2);

  await contractRepeatableViaBundle(apiContext, documentId);
  await page.reload({ waitUntil: "domcontentloaded" });
  await openDocumentEditorFromDetail(page);
  await ensureBrowserEditorReady(page);
  await expect(page.locator('[data-mddm-block="repeatable"]').filter({ hasText: "Etapas" }).first().locator('[data-mddm-block="repeatableItem"]')).toHaveCount(1);
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

type BrowserTemplateItem = DocumentTemplateItem;

async function findBrowserTemplate(apiContext: APIRequestContext) {
  const templatesResponse = await apiContext.get("/api/v1/document-templates?profileCode=po", {
    headers: sameSiteHeaders,
  });
  expect(templatesResponse.ok(), `list templates failed: ${templatesResponse.status()} ${await templatesResponse.text()}`).toBeTruthy();

  const templatesBody = await templatesResponse.json() as { items?: BrowserTemplateItem[] };
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

  for (let attempt = 0; attempt < 3; attempt += 1) {
    const openButton = page.getByRole("button", { name: "Abrir documento" });
    const branch = await Promise.race([
      editor.waitFor({ state: "visible", timeout: 8_000 }).then(() => "editor"),
      openButton.waitFor({ state: "visible", timeout: 8_000 }).then(() => "open"),
    ]).catch(() => "timeout");

    if (branch === "editor") {
      return;
    }

    if (branch !== "open") {
      if (attempt === 2) {
        throw new Error("failed to detect either editor mount or 'Abrir documento' CTA");
      }
      continue;
    }

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

async function appendToEditable(locator: Locator, suffix: string) {
  await locator.click();
  await locator.press("End");
  await locator.type(suffix);
}

async function insertImageViaSlashMenu(page: Page, paragraph: Locator, imageUrl: string) {
  await paragraph.hover();
  const addButton = page.getByLabel("Adicionar bloco");
  await expect(addButton).toBeVisible({ timeout: 10_000 });
  await addButton.locator('[data-test="dragHandleAdd"]').click();

  const imageOption = page.getByRole("option", { name: "Imagem" });
  await expect(imageOption).toBeVisible({ timeout: 10_000 });
  await imageOption.click();

  const embedInput = page.getByTestId("embed-input");
  await expect(embedInput).toBeVisible({ timeout: 10_000 });
  await embedInput.fill(imageUrl);
  await page.getByTestId("embed-input-button").click();
}

async function uploadFixtureAttachmentAndGetDownloadUrl(apiContext: APIRequestContext, documentId: string) {
  const uploadResponse = await apiContext.post(`/api/v1/documents/${encodeURIComponent(documentId)}/attachments`, {
    headers: sameSiteHeaders,
    multipart: {
      file: {
        name: "test-image.png",
        mimeType: "image/png",
        buffer: fixtureImageBuffer,
      },
    },
  });
  expect(uploadResponse.ok(), `attachment upload failed: ${uploadResponse.status()} ${await uploadResponse.text()}`).toBeTruthy();
  const uploadBody = await uploadResponse.json() as { attachmentId?: string };
  const attachmentId = typeof uploadBody.attachmentId === "string" ? uploadBody.attachmentId.trim() : "";
  expect(attachmentId).toBeTruthy();

  const downloadUrlResponse = await apiContext.get(
    `/api/v1/documents/${encodeURIComponent(documentId)}/attachments/${encodeURIComponent(attachmentId)}/download-url`,
    { headers: sameSiteHeaders },
  );
  expect(
    downloadUrlResponse.ok(),
    `attachment download-url failed: ${downloadUrlResponse.status()} ${await downloadUrlResponse.text()}`,
  ).toBeTruthy();
  const downloadUrlBody = await downloadUrlResponse.json() as { downloadUrl?: string };
  const downloadUrl = typeof downloadUrlBody.downloadUrl === "string" ? downloadUrlBody.downloadUrl.trim() : "";
  expect(downloadUrl).toBeTruthy();
  return downloadUrl;
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

async function expandRepeatableViaBundle(apiContext: APIRequestContext, documentId: string) {
  const bundle = await fetchBrowserBundle(apiContext, documentId);
  const envelope = JSON.parse(bundle.body) as MddmEnvelope;
  const repeatable = findBlock(envelope.blocks, (block) => block.type === "repeatable" && block.props.label === "Etapas");
  if (!repeatable) {
    throw new Error("repeatable block Etapas was not found in the browser bundle");
  }

  const items = getBlockChildren(repeatable).filter(isBlock).filter((block) => block.type === "repeatableItem");
  expect(items.length).toBe(1);
  repeatable.children = [items[0], duplicateBlockTree(items[0])];

  await saveMutatedBundle(apiContext, documentId, bundle.draftToken, envelope);
}

async function contractRepeatableViaBundle(apiContext: APIRequestContext, documentId: string) {
  const bundle = await fetchBrowserBundle(apiContext, documentId);
  const envelope = JSON.parse(bundle.body) as MddmEnvelope;
  const repeatable = findBlock(envelope.blocks, (block) => block.type === "repeatable" && block.props.label === "Etapas");
  if (!repeatable) {
    throw new Error("repeatable block Etapas was not found in the browser bundle");
  }

  const items = getBlockChildren(repeatable).filter(isBlock).filter((block) => block.type === "repeatableItem");
  expect(items.length).toBeGreaterThanOrEqual(2);
  repeatable.children = [items[0]];

  await saveMutatedBundle(apiContext, documentId, bundle.draftToken, envelope);
}

async function saveMutatedBundle(
  apiContext: APIRequestContext,
  documentId: string,
  draftToken: string,
  envelope: MddmEnvelope,
) {
  const response = await apiContext.post(`/api/v1/documents/${encodeURIComponent(documentId)}/content/browser`, {
    headers: {
      ...sameSiteHeaders,
      "content-type": "application/json",
    },
    data: {
      body: JSON.stringify(envelope),
      draftToken,
    },
  });
  expect(response.ok(), `browser save failed: ${response.status()} ${await response.text()}`).toBeTruthy();
  const payload = await response.json() as { draftToken?: string };
  expect(typeof payload.draftToken).toBe("string");
}

function duplicateBlockTree(block: MddmBlock): MddmBlock {
  const clone = JSON.parse(JSON.stringify(block)) as MddmBlock;
  rewriteBlockIds(clone);
  return clone;
}

function rewriteBlockIds(block: MddmBlock) {
  block.id = randomUUID();
  delete block.template_block_id;

  for (const child of getBlockChildren(block)) {
    if (isBlock(child)) {
      rewriteBlockIds(child);
    }
  }
}

function findBlock(
  blocks: MddmBlock[],
  predicate: (block: MddmBlock) => boolean,
): MddmBlock | undefined {
  for (const block of blocks) {
    if (predicate(block)) {
      return block;
    }
    const nested = findBlock(getBlockChildren(block).filter(isBlock), predicate);
    if (nested) {
      return nested;
    }
  }
  return undefined;
}

function getBlockChildren(block: MddmBlock): Array<MddmBlock | MddmTextRun> {
  return Array.isArray(block.children) ? block.children : [];
}

function isBlock(value: MddmBlock | MddmTextRun): value is MddmBlock {
  return typeof value === "object" && value !== null && "type" in value;
}
