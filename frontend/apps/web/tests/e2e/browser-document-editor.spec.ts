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

  await page.goto(`/#/documents/doc/${encodeURIComponent(createdDocument.documentId)}`);
  await page.getByRole("button", { name: "Abrir documento" }).click();

  await expect(page.getByTestId("browser-document-editor")).toBeVisible({ timeout: 20_000 });
  await expect(page.locator(".content-builder-preview")).toHaveCount(0);

  const editable = page.locator(".ck-editor__editable").first();
  await editable.click();
  await editable.fill("Objetivo do teste");

  await page.getByRole("button", { name: "Salvar rascunho" }).click();
  await expect(page.getByText("Salvo agora")).toBeVisible();
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
  const currentUser = await apiContext.get("/api/v1/auth/me");
  expect(currentUser.ok()).toBeTruthy();
  const currentUserBody = (await currentUser.json()) as { userId?: string };

  const suffix = Date.now().toString();
  const createResponse = await apiContext.post("/api/v1/documents", {
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
