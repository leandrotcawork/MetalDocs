import { expect, test, type Page, type Route } from "@playwright/test";

import { loginAsAdmin, seedE2EWorkspace } from "./template-admin-helpers";

const jsonMime = "application/json";

type TemplateListItem = {
  templateKey: string;
  version: number;
  profileCode: string;
  name: string;
  status: "draft" | "published" | "deprecated";
};

type TemplateDraft = {
  templateKey: string;
  profileCode: string;
  name: string;
  status: "draft";
  lockVersion: number;
  hasStrippedFields: boolean;
  blocks: unknown[];
  updatedAt: string;
};

test.beforeAll(() => {
  seedE2EWorkspace();
});

test("template list actions hit the expected endpoints and keep navigation consistent", async ({ page }) => {
  const drafts = new Map<string, TemplateDraft>([
    [
      "tpl-draft-delete",
      makeDraft({ templateKey: "tpl-draft-delete", name: "Draft delete" }),
    ],
    [
      "tpl-draft-discard",
      makeDraft({ templateKey: "tpl-draft-discard", name: "Draft discard" }),
    ],
    [
      "tpl-created",
      makeDraft({ templateKey: "tpl-created", name: "Novo template" }),
    ],
    [
      "tpl-clone",
      makeDraft({ templateKey: "tpl-clone", name: "Published source (copia)" }),
    ],
  ]);

  const listItems: TemplateListItem[] = [
    { templateKey: "tpl-draft-delete", version: 1, profileCode: "po", name: "Draft delete", status: "draft" },
    { templateKey: "tpl-draft-discard", version: 2, profileCode: "po", name: "Draft discard", status: "draft" },
    { templateKey: "tpl-published", version: 7, profileCode: "po", name: "Published source", status: "published" },
    { templateKey: "tpl-deprecated", version: 3, profileCode: "po", name: "Deprecated source", status: "deprecated" },
  ];

  const actions = {
    createBodies: [] as Array<{ profileCode: string; name: string }>,
    cloneBodies: [] as Array<{ newName: string }>,
    deprecateBodies: [] as Array<{ version: number }>,
    deleteKeys: [] as string[],
    discardKeys: [] as string[],
    exportRequests: [] as Array<{ key: string; version: string | null }>,
  };

  let activeProfileCode = "po";

  await page.route("**/api/v1/templates**", async (route) => {
    const url = new URL(route.request().url());
    const method = route.request().method();

    if (url.pathname === "/api/v1/templates" && method === "GET") {
      activeProfileCode = url.searchParams.get("profileCode") ?? activeProfileCode;
      await fulfillJson(route, 200, {
        items: listItems.filter((item) => item.profileCode === activeProfileCode),
      });
      return;
    }

    if (url.pathname === "/api/v1/templates" && method === "POST") {
      const payload = route.request().postDataJSON() as { profileCode: string; name: string };
      actions.createBodies.push(payload);
      const created = drafts.get("tpl-created")!;
      created.profileCode = payload.profileCode;
      const existing = listItems.find((item) => item.templateKey === created.templateKey);
      if (!existing) {
        listItems.unshift({
          templateKey: created.templateKey,
          version: 1,
          profileCode: payload.profileCode,
          name: created.name,
          status: "draft",
        });
      }
      await fulfillJson(route, 200, created);
      return;
    }

    await route.continue();
  });

  await page.route("**/api/v1/templates/**", async (route) => {
    const url = new URL(route.request().url());
    const method = route.request().method();
    const pathSegments = url.pathname.split("/").filter(Boolean);
    const templateKey = decodeURIComponent(pathSegments[3] ?? "");

    if (pathSegments.length === 4 && method === "GET" && drafts.has(templateKey)) {
      await fulfillJson(route, 200, drafts.get(templateKey)!);
      return;
    }

    if (url.pathname === `/api/v1/templates/${templateKey}` && method === "DELETE") {
      actions.deleteKeys.push(templateKey);
      removeItem(listItems, templateKey);
      await fulfillJson(route, 204, undefined);
      return;
    }

    if (url.pathname === `/api/v1/templates/${templateKey}/discard-draft` && method === "POST") {
      actions.discardKeys.push(templateKey);
      removeItem(listItems, templateKey);
      await fulfillJson(route, 204, undefined);
      return;
    }

    if (url.pathname === `/api/v1/templates/${templateKey}/clone` && method === "POST") {
      const payload = route.request().postDataJSON() as { newName: string };
      actions.cloneBodies.push(payload);
      const clone = drafts.get("tpl-clone")!;
      clone.profileCode = activeProfileCode;
      const existing = listItems.find((item) => item.templateKey === clone.templateKey);
      if (!existing) {
        listItems.unshift({
          templateKey: clone.templateKey,
          version: 1,
          profileCode: activeProfileCode,
          name: clone.name,
          status: "draft",
        });
      }
      await fulfillJson(route, 200, clone);
      return;
    }

    if (url.pathname === `/api/v1/templates/${templateKey}/deprecate` && method === "POST") {
      const payload = route.request().postDataJSON() as { version: number };
      actions.deprecateBodies.push(payload);
      const item = listItems.find((candidate) => candidate.templateKey === templateKey);
      if (item) {
        item.status = "deprecated";
      }
      await fulfillJson(route, 204, undefined);
      return;
    }

    if (url.pathname === `/api/v1/templates/${templateKey}/export` && method === "GET") {
      actions.exportRequests.push({ key: templateKey, version: url.searchParams.get("version") });
      await route.fulfill({
        status: 200,
        contentType: jsonMime,
        body: JSON.stringify({ templateKey, version: url.searchParams.get("version") }),
      });
      return;
    }

    await route.continue();
  });

  await loginAsAdmin(page);
  await page.goto("/#/registry");
  await expect(page.getByTestId("template-list-panel")).toBeVisible({ timeout: 20_000 });

  await page.getByTestId("template-create-btn").click();
  expect(actions.createBodies).toEqual([{ profileCode: activeProfileCode, name: "Novo template" }]);
  await expect(page).toHaveURL(new RegExp(`#\\/registry\\/profiles\\/${escapeRegex(activeProfileCode)}\\/templates\\/tpl-created\\/edit$`));
  await expect(page.getByTestId("metadata-bar")).toBeVisible();

  await page.goto("/#/registry");
  const cloneRow = page.getByTestId("template-row-tpl-published");
  await expect(cloneRow).toBeVisible();

  await cloneRow.getByTestId("template-action-clone-tpl-published").click();
  expect(actions.cloneBodies).toEqual([{ newName: "Published source (copia)" }]);
  await expect(page).toHaveURL(new RegExp(`#\\/registry\\/profiles\\/${escapeRegex(activeProfileCode)}\\/templates\\/tpl-clone\\/edit$`));

  await page.goto("/#/registry");
  const exportDownload = page.waitForEvent("download");
  await page.getByTestId("template-action-export-tpl-published").click();
  const exported = await exportDownload;
  expect(exported.suggestedFilename()).toBe("tpl-published-v7.json");
  expect(actions.exportRequests).toEqual([{ key: "tpl-published", version: "7" }]);

  await page.goto("/#/registry");
  await page.getByTestId("template-action-deprecate-tpl-published").click();
  expect(actions.deprecateBodies).toEqual([{ version: 7 }]);
  await expect(page.getByTestId("template-row-tpl-published")).toContainText("Depreciado");

  await page.goto("/#/registry");
  await page.getByTestId("template-action-delete-tpl-draft-delete").click();
  expect(actions.deleteKeys).toEqual(["tpl-draft-delete"]);
  await expect(page.getByTestId("template-row-tpl-draft-delete")).toHaveCount(0);

  await page.goto("/#/registry");
  await page.getByTestId("template-action-discard-tpl-draft-discard").click();
  expect(actions.discardKeys).toEqual(["tpl-draft-discard"]);
  await expect(page.getByTestId("template-row-tpl-draft-discard")).toHaveCount(0);
});

function makeDraft(overrides: Partial<TemplateDraft> = {}): TemplateDraft {
  return {
    templateKey: overrides.templateKey ?? "tpl-draft",
    profileCode: overrides.profileCode ?? "po",
    name: overrides.name ?? "Draft",
    status: "draft",
    lockVersion: overrides.lockVersion ?? 1,
    hasStrippedFields: false,
    blocks: overrides.blocks ?? [],
    updatedAt: overrides.updatedAt ?? "2026-04-14T00:00:00Z",
  };
}

function removeItem(items: TemplateListItem[], templateKey: string) {
  const index = items.findIndex((item) => item.templateKey === templateKey);
  if (index >= 0) {
    items.splice(index, 1);
  }
}

async function fulfillJson(route: Route, status: number, body: unknown) {
  await route.fulfill({
    status,
    contentType: jsonMime,
    body: body === undefined ? "" : JSON.stringify(body),
  });
}

function escapeRegex(value: string) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}
