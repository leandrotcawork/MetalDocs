import { execFileSync } from "node:child_process";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

import { expect, test } from "@playwright/test";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);
const repoRoot = resolve(__dirname, "../../../../");
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

test("auth and document flow smoke", async ({ page }) => {
  const suffix = Date.now().toString();
  const newUsername = `e2e.user.${suffix}`;
  const initialPassword = "TempPass123!";
  const rotatedPassword = "TempPass124!";
  const documentTitle = `E2E Document ${suffix}`;

  await page.goto("/");

  await page.getByTestId("login-identifier").fill(adminUsername);
  await page.getByTestId("login-password").fill(adminPassword);
  await page.getByTestId("login-submit").click();

  await expect(page.getByText("MetalDocs Control Room")).toBeVisible();
  await expect(page.getByTestId("managed-users-panel")).toBeVisible();

  await page.getByTestId("user-username").fill(newUsername);
  await page.getByTestId("user-email").fill(`${newUsername}@local.test`);
  await page.getByTestId("user-display-name").fill(`Smoke ${suffix}`);
  await page.getByTestId("user-password").fill(initialPassword);
  await page.getByTestId("user-role").selectOption("editor");
  await page.getByTestId("user-submit").click();

  await expect(page.getByTestId("app-banner")).toContainText("Usuario criado");
  await expect(page.getByTestId("managed-users-panel")).toContainText(newUsername);

  await page.getByTestId("logout-button").click();
  await expect(page.getByTestId("login-submit")).toBeVisible();

  await page.getByTestId("login-identifier").fill(newUsername);
  await page.getByTestId("login-password").fill(initialPassword);
  await page.getByTestId("login-submit").click();

  await expect(page.getByText("Troca obrigatoria de senha")).toBeVisible();
  await page.getByTestId("password-new").fill(rotatedPassword);
  await page.getByTestId("password-confirm").fill(rotatedPassword);
  await page.getByTestId("password-submit").click();

  await expect(page.getByTestId("app-banner")).toContainText("Senha alterada com sucesso");
  await expect(page.getByTestId("document-create-form")).toBeVisible();

  await page.getByTestId("document-title").fill(documentTitle);
  await page.getByTestId("document-initial-content").fill("E2E smoke content");
  await page.getByTestId("document-submit").click();

  await expect(page.getByTestId("app-banner")).toContainText("Documento criado com sucesso");
  await expect(page.getByTestId("documents-panel")).toContainText(documentTitle);

  await page.getByTestId("logout-button").click();
  await expect(page.getByTestId("login-submit")).toBeVisible();

  await page.getByTestId("login-identifier").fill(newUsername);
  await page.getByTestId("login-password").fill(rotatedPassword);
  await page.getByTestId("login-submit").click();

  await expect(page.getByText("MetalDocs Control Room")).toBeVisible();
  await expect(page.getByText(newUsername)).toBeVisible();
  await expect(page.getByTestId("documents-panel")).toContainText(documentTitle);
});
