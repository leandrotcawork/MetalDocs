import { test, expect } from "@playwright/test";

test("create PO from MDDM template, fill field, save draft", async ({ page }) => {
  await page.goto("/");

  // Click "Novo documento"
  await page.getByRole("button", { name: /novo documento/i }).click();

  // Title
  await page.getByLabel(/tÃ­tulo/i).fill("Teste E2E MDDM");

  // Pick PO type (already default)
  await page.getByRole("button", { name: /ir para o editor/i }).click();

  // Wait for the editor to mount
  await expect(page.getByText("IdentificaÃ§Ã£o do Processo")).toBeVisible({ timeout: 10000 });

  // Type in the Objetivo field
  const objetivoField = page.getByText("Objetivo").locator("..").locator("[contenteditable]").first();
  await objetivoField.fill("Garantir atendimento ao cliente em atÃ© 24h");

  // Save
  await page.getByRole("button", { name: /salvar/i }).click();

  // Confirm save toast
  await expect(page.getByText(/rascunho salvo/i)).toBeVisible({ timeout: 5000 });
});
