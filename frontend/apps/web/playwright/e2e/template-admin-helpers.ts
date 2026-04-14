import { execFileSync } from "node:child_process";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

import { expect, type Page } from "@playwright/test";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);
const repoRoot = resolve(__dirname, "../../../../../");
const seedScript = resolve(repoRoot, "scripts/e2e-seed.ps1");

const adminUsername = process.env.METALDOCS_E2E_ADMIN_USERNAME ?? "e2e.admin";
const adminPassword = process.env.METALDOCS_E2E_ADMIN_PASSWORD ?? "E2eAdmin123!";

type EditorBlock = {
  id: string;
  type: string;
  props: Record<string, unknown>;
  children?: EditorBlock[];
};

export function seedE2EWorkspace() {
  execFileSync(
    "powershell.exe",
    ["-ExecutionPolicy", "Bypass", "-File", seedScript],
    {
      cwd: repoRoot,
      stdio: "inherit",
    },
  );
}

export async function loginAsAdmin(page: Page) {
  await page.goto("/");
  await page.getByTestId("login-identifier").fill(adminUsername);
  await page.getByTestId("login-password").fill(adminPassword);
  await page.getByTestId("login-submit").click();
  await expect(page.locator("body")).toContainText(/Todos Documentos|Painel documental/);
}

export async function openRegistry(page: Page) {
  await page.goto("/#/registry");
  await expect(page.getByTestId("template-list-panel")).toBeVisible({ timeout: 20_000 });
}

export async function openTemplateEditor(page: Page, profileCode: string, templateKey: string) {
  await page.goto(`/#/registry/profiles/${encodeURIComponent(profileCode)}/templates/${encodeURIComponent(templateKey)}/edit`);
  await waitForTemplateEditor(page);
}

export async function waitForTemplateEditor(page: Page) {
  await expect(page.getByTestId("metadata-bar")).toBeVisible({ timeout: 20_000 });
  await expect(page.getByTestId("mddm-editor-root")).toBeVisible({ timeout: 20_000 });
  await page.waitForFunction(() => typeof (window as any).__mddmEditor?.document !== "undefined", undefined, {
    timeout: 20_000,
  });
}

export async function countBlocksByType(page: Page, type: string) {
  return page.evaluate((blockType) => {
    const editor = (window as any).__mddmEditor;
    const flat: Array<{ id: string; type: string }> = [];
    const walk = (blocks: any[]) => {
      for (const block of blocks ?? []) {
        flat.push({ id: block.id, type: block.type });
        walk(block.children ?? []);
      }
    };
    walk(editor?.document ?? []);
    return flat.filter((block) => block.type === blockType).length;
  }, type);
}

export async function findNthBlockIdByType(page: Page, type: string, index = 0) {
  return page.evaluate(({ blockType, blockIndex }) => {
    const editor = (window as any).__mddmEditor;
    const flat: Array<{ id: string; type: string }> = [];
    const walk = (blocks: any[]) => {
      for (const block of blocks ?? []) {
        flat.push({ id: block.id, type: block.type });
        walk(block.children ?? []);
      }
    };
    walk(editor?.document ?? []);
    return flat.filter((block) => block.type === blockType)[blockIndex]?.id ?? null;
  }, { blockType: type, blockIndex: index });
}

export async function selectBlock(page: Page, blockId: string) {
  await page.evaluate((id) => {
    const editor = (window as any).__mddmEditor;
    const block = editor?.getBlock?.(id);
    if (!block) {
      throw new Error(`Block not found: ${id}`);
    }
    editor.setTextCursorPosition(block, "start");
  }, blockId);
}

export async function updateBlockProps(page: Page, blockId: string, patch: Record<string, unknown>) {
  await page.evaluate(({ id, propsPatch }) => {
    const editor = (window as any).__mddmEditor;
    editor?.updateBlock?.(id, { props: propsPatch });
  }, { id: blockId, propsPatch: patch });
}

export async function getBlocks(page: Page): Promise<EditorBlock[]> {
  return page.evaluate(() => {
    const editor = (window as any).__mddmEditor;
    return structuredClone(editor?.document ?? []);
  });
}

export function flattenBlocks(blocks: EditorBlock[]): EditorBlock[] {
  const flat: EditorBlock[] = [];
  const walk = (items: EditorBlock[]) => {
    for (const item of items) {
      flat.push(item);
      walk(item.children ?? []);
    }
  };
  walk(blocks);
  return flat;
}
