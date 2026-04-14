import { expect, test, type Page, type Route } from "@playwright/test";

import {
  countBlocksByType,
  findNthBlockIdByType,
  flattenBlocks,
  getBlocks,
  loginAsAdmin,
  openTemplateEditor,
  seedE2EWorkspace,
  selectBlock,
  updateBlockProps,
} from "./template-admin-helpers";

const docxMime = "application/vnd.openxmlformats-officedocument.wordprocessingml.document";

type DraftDto = {
  templateKey: string;
  profileCode: string;
  name: string;
  status: "draft";
  lockVersion: number;
  hasStrippedFields: boolean;
  blocks: unknown[];
  theme?: unknown;
  meta?: unknown;
  updatedAt: string;
};

type VersionDto = {
  templateKey: string;
  version: number;
  profileCode: string;
  name: string;
  status: "published" | "deprecated";
};

test.beforeAll(() => {
  seedE2EWorkspace();
});

test("template editor keeps scroll inside the document pane", async ({ page }) => {
  const templateKey = "tpl-scroll-owner";
  const longSectionChildren = Array.from({ length: 26 }, (_, index) => ({
    id: `scroll-rich-${index + 1}`,
    type: "richBlock",
    props: {
      label: `Bloco ${index + 1}`,
      styleJson: "{}",
      capabilitiesJson: JSON.stringify({ locked: false, removable: true }),
    },
    children: [],
  }));
  const draft = makeDraft({
    templateKey,
    lockVersion: 2,
    blocks: [
      {
        id: "scroll-section-1",
        type: "section",
        props: {
          title: "Sessao longa",
          styleJson: "{}",
          capabilitiesJson: JSON.stringify({ locked: true, removable: false, reorderable: false }),
        },
        children: longSectionChildren,
      },
    ],
  });

  await page.route("**/api/v1/templates/**", async (route) => {
    const url = new URL(route.request().url());
    const method = route.request().method();

    if (url.pathname === `/api/v1/templates/${templateKey}` && method === "GET") {
      await fulfillJson(route, 200, draft);
      return;
    }

    await route.continue();
  });

  await loginAsAdmin(page);
  await openTemplateEditor(page, "po", templateKey);

  const scrollShell = page.getByTestId("mddm-editor-scroll-shell");
  await expect(scrollShell).toBeVisible();
  await expect(page.getByTestId("mddm-editor-page-stack")).toBeVisible();
  await expect(page.getByTestId("mddm-editor-paper")).toBeVisible();
  await expect(scrollShell).toHaveCSS("overflow-y", "auto");

  const scrollState = await page.evaluate(() => {
    const shell = document.querySelector('[data-testid="mddm-editor-scroll-shell"]') as HTMLElement | null;
    const layoutRoot = document.querySelector('[data-testid="template-editor-layout"]') as HTMLElement | null;
    if (!shell) {
      return null;
    }

    const before = shell.scrollTop;
    shell.scrollTop = 320;
    const after = shell.scrollTop;

    return {
      isScrollable: shell.scrollHeight > shell.clientHeight,
      before,
      after,
      layoutScrollTop: layoutRoot?.scrollTop ?? 0,
    };
  });

  expect(scrollState).toBeTruthy();
  expect(scrollState?.isScrollable).toBe(true);
  expect((scrollState?.after ?? 0) > (scrollState?.before ?? 0)).toBe(true);
  expect(scrollState?.layoutScrollTop ?? 0).toBe(0);
});

test("template editor uses a centered paper stack without oversized outer margins", async ({ page }) => {
  const templateKey = "tpl-paper-density";
  const draft = makeDraft({
    templateKey,
    lockVersion: 4,
    blocks: [
      {
        id: "density-section-1",
        type: "section",
        props: {
          title: "Secao de densidade",
          styleJson: "{}",
          capabilitiesJson: JSON.stringify({ locked: true, removable: false, reorderable: false }),
        },
        children: [],
      },
    ],
  });

  await page.route("**/api/v1/templates/**", async (route) => {
    const url = new URL(route.request().url());
    const method = route.request().method();

    if (url.pathname === `/api/v1/templates/${templateKey}` && method === "GET") {
      await fulfillJson(route, 200, draft);
      return;
    }

    await route.continue();
  });

  await loginAsAdmin(page);
  await openTemplateEditor(page, "po", templateKey);

  const metrics = await page.evaluate(() => {
    const pageStack = document.querySelector('[data-testid="mddm-editor-page-stack"]') as HTMLElement | null;
    const paper = document.querySelector('[data-testid="mddm-editor-paper"]') as HTMLElement | null;
    if (!pageStack || !paper) return null;

    const stackBox = pageStack.getBoundingClientRect();
    const paperBox = paper.getBoundingClientRect();
    const stackStyles = getComputedStyle(pageStack);

    return {
      leftInset: paperBox.left - stackBox.left,
      stackPaddingTop: parseFloat(stackStyles.paddingTop),
      stackPaddingBottom: parseFloat(stackStyles.paddingBottom),
      stackGap: parseFloat(stackStyles.rowGap || stackStyles.gap || "0"),
    };
  });

  expect(metrics).not.toBeNull();
  expect(metrics!.stackPaddingTop).toBeLessThanOrEqual(16);
  expect(metrics!.stackPaddingBottom).toBeLessThanOrEqual(24);
  expect(metrics!.stackGap).toBeLessThanOrEqual(20);
});

test("template authoring saves, previews DOCX, blocks invalid client publish, then publishes successfully", async ({ page }) => {
  const templateKey = "tpl-authoring-flow";
  let draft: DraftDto = makeDraft({
    templateKey,
    lockVersion: 3,
    blocks: [
      {
        id: "section-1",
        type: "section",
        props: {
          title: "Escopo",
          styleJson: "{}",
          capabilitiesJson: JSON.stringify({ locked: true, removable: false, reorderable: false }),
        },
        children: [
          {
            id: "rich-1",
            type: "richBlock",
            props: {
              label: "Objetivo",
              styleJson: "{}",
              capabilitiesJson: JSON.stringify({ locked: true, removable: false, editableZones: ["content"] }),
            },
            children: [],
          },
        ],
      },
    ],
  });

  const saveBodies: Array<{ blocks: unknown[]; lockVersion: number }> = [];
  const publishBodies: Array<{ lockVersion: number }> = [];

  await page.route("**/api/v1/templates/**", async (route) => {
    const url = new URL(route.request().url());
    const method = route.request().method();

    if (url.pathname === `/api/v1/templates/${templateKey}` && method === "GET") {
      await fulfillJson(route, 200, draft);
      return;
    }

    if (url.pathname === `/api/v1/templates/${templateKey}/draft` && method === "PUT") {
      const payload = route.request().postDataJSON() as { blocks: unknown[]; lockVersion: number };
      saveBodies.push(payload);
      draft = {
        ...draft,
        blocks: payload.blocks,
        lockVersion: payload.lockVersion + 1,
        updatedAt: "2026-04-14T03:00:00Z",
      };
      await fulfillJson(route, 200, draft);
      return;
    }

    if (url.pathname === `/api/v1/templates/${templateKey}/preview-docx` && method === "POST") {
      await route.fulfill({
        status: 200,
        contentType: docxMime,
        body: Buffer.from("template-preview-docx"),
      });
      return;
    }

    if (url.pathname === `/api/v1/templates/${templateKey}/publish` && method === "POST") {
      const payload = route.request().postDataJSON() as { lockVersion: number };
      publishBodies.push(payload);
      await fulfillJson(route, 200, {
        templateKey,
        version: 4,
        profileCode: draft.profileCode,
        name: draft.name,
        status: "published",
      } satisfies VersionDto);
      return;
    }

    await route.continue();
  });

  await loginAsAdmin(page);
  await page.goto("/#/registry");
  await openTemplateEditor(page, "po", templateKey);

  const firstRichBlockId = await findNthBlockIdByType(page, "richBlock", 0);
  expect(firstRichBlockId).toBeTruthy();
  await selectBlock(page, firstRichBlockId!);

  await page.getByTestId("palette-insert-richBlock").click();
  await expect.poll(async () => countBlocksByType(page, "richBlock")).toBe(2);

  const newRichBlockId = await findNthBlockIdByType(page, "richBlock", 1);
  expect(newRichBlockId).toBeTruthy();
  await selectBlock(page, newRichBlockId!);
  await expect(page.getByTestId("property-sidebar-block-type")).toContainText("richBlock");

  await page.getByTestId("template-prop-label").fill("Escopo detalhado");
  await page.getByTestId("property-tab-estilo").click();
  await page.getByTestId("template-style-labelFontSize").fill("12pt");
  await page.getByTestId("property-tab-capacidades").click();
  await page.getByTestId("template-caps-locked").uncheck();

  const saveResponse = waitForTemplateResponse(page, `/api/v1/templates/${templateKey}/draft`, 200);
  await page.getByTestId("template-save-btn").click();
  await saveResponse;

  expect(saveBodies).toHaveLength(1);
  const savedBlocks = flattenBlocks(saveBodies[0].blocks as any[]);
  const savedRichBlock = savedBlocks.find((block) => block.id === newRichBlockId);
  expect(savedRichBlock?.props.label).toBe("Escopo detalhado");
  expect(String(savedRichBlock?.props.styleJson ?? "")).toContain('"labelFontSize":"12pt"');
  expect(String(savedRichBlock?.props.capabilitiesJson ?? "")).toContain('"locked":false');

  const previewDownload = page.waitForEvent("download");
  await page.getByTestId("template-preview-docx-btn").click();
  const preview = await previewDownload;
  expect(preview.suggestedFilename()).toBe(`${templateKey}-preview.docx`);

  await updateBlockProps(page, newRichBlockId!, {
    styleJson: JSON.stringify({ labelFontSize: 12 }),
  });

  await page.getByTestId("template-publish-btn").click();
  await expect(page.getByTestId("validation-panel")).toBeVisible();
  expect(publishBodies).toHaveLength(0);
  await expect(page.getByTestId("validation-error-row-0")).toContainText("labelFontSize");

  await page.getByTestId("validation-error-row-0").click();
  await page.getByTestId("property-tab-estilo").click();
  await page.getByTestId("template-style-labelFontSize").fill("14pt");

  const publishResponse = waitForTemplateResponse(page, `/api/v1/templates/${templateKey}/publish`, 200);
  await page.getByTestId("template-publish-btn").click();
  await publishResponse;

  expect(publishBodies).toEqual([{ lockVersion: 5 }]);
  await expect(page).toHaveURL(/#\/registry$/);
});

test("published template edit path creates a draft and server-side publish validation populates the panel", async ({ page }) => {
  const templateKey = "tpl-published-flow";
  const publishedVersion: VersionDto = {
    templateKey,
    version: 7,
    profileCode: "po",
    name: "Template publicado",
    status: "published",
  };
  let draft = makeDraft({
    templateKey,
    lockVersion: 8,
    blocks: [
      {
        id: "section-published",
        type: "section",
        props: {
          title: "Publicado",
          styleJson: "{}",
          capabilitiesJson: JSON.stringify({ locked: true, removable: false, reorderable: false }),
        },
        children: [],
      },
    ],
  });

  let editCalls = 0;
  let publishCalls = 0;

  await page.route("**/api/v1/templates/**", async (route) => {
    const url = new URL(route.request().url());
    const method = route.request().method();

    if (url.pathname === `/api/v1/templates/${templateKey}` && method === "GET") {
      await fulfillJson(route, 200, publishedVersion);
      return;
    }

    if (url.pathname === `/api/v1/templates/${templateKey}/edit` && method === "POST") {
      editCalls += 1;
      await fulfillJson(route, 200, draft);
      return;
    }

    if (url.pathname === `/api/v1/templates/${templateKey}/draft` && method === "PUT") {
      const payload = route.request().postDataJSON() as { blocks: unknown[]; lockVersion: number };
      draft = {
        ...draft,
        blocks: payload.blocks,
        lockVersion: payload.lockVersion + 1,
        updatedAt: "2026-04-14T03:10:00Z",
      };
      await fulfillJson(route, 200, draft);
      return;
    }

    if (url.pathname === `/api/v1/templates/${templateKey}/publish` && method === "POST") {
      publishCalls += 1;
      if (publishCalls === 1) {
        await fulfillJson(route, 422, {
          errors: [{ blockId: "section-published", blockType: "section", field: "server.rule", reason: "Server-side publish rule blocked this draft" }],
          error: { message: "Publish blocked by server validation" },
        });
        return;
      }

      await fulfillJson(route, 200, {
        templateKey,
        version: 8,
        profileCode: draft.profileCode,
        name: draft.name,
        status: "published",
      } satisfies VersionDto);
      return;
    }

    await route.continue();
  });

  await loginAsAdmin(page);
  await page.goto("/#/registry");
  await openTemplateEditor(page, "po", templateKey);

  expect(editCalls).toBe(1);
  await expect(page.getByTestId("metadata-bar")).toContainText("Edicao #8");

  const rejectedPublish = waitForTemplateResponse(page, `/api/v1/templates/${templateKey}/publish`, 422);
  await page.getByTestId("template-publish-btn").click();
  await rejectedPublish;

  await expect(page.getByTestId("validation-panel")).toBeVisible();
  await expect(page.getByTestId("validation-error-row-0")).toContainText("Server-side publish rule blocked this draft");

  const acceptedPublish = waitForTemplateResponse(page, `/api/v1/templates/${templateKey}/publish`, 200);
  await page.getByTestId("template-publish-btn").click();
  await acceptedPublish;

  expect(publishCalls).toBe(2);
  await expect(page).toHaveURL(/#\/registry$/);
});

test("stripped-fields drafts require acknowledgement before publish", async ({ page }) => {
  const templateKey = "tpl-stripped-flow";
  let draft = makeDraft({
    templateKey,
    lockVersion: 5,
    hasStrippedFields: true,
  });
  let publishCalls = 0;
  let acknowledgeCalls = 0;

  await page.route("**/api/v1/templates/**", async (route) => {
    const url = new URL(route.request().url());
    const method = route.request().method();

    if (url.pathname === `/api/v1/templates/${templateKey}` && method === "GET") {
      await fulfillJson(route, 200, draft);
      return;
    }

    if (url.pathname === `/api/v1/templates/${templateKey}/acknowledge-stripped` && method === "POST") {
      acknowledgeCalls += 1;
      draft = {
        ...draft,
        hasStrippedFields: false,
        lockVersion: draft.lockVersion + 1,
        updatedAt: "2026-04-14T03:20:00Z",
      };
      await fulfillJson(route, 200, draft);
      return;
    }

    if (url.pathname === `/api/v1/templates/${templateKey}/draft` && method === "PUT") {
      const payload = route.request().postDataJSON() as { blocks: unknown[]; lockVersion: number };
      draft = {
        ...draft,
        blocks: payload.blocks,
        lockVersion: payload.lockVersion + 1,
      };
      await fulfillJson(route, 200, draft);
      return;
    }

    if (url.pathname === `/api/v1/templates/${templateKey}/publish` && method === "POST") {
      publishCalls += 1;
      if (draft.hasStrippedFields) {
        await fulfillJson(route, 422, {
          errors: [{ blockId: "", blockType: "template", field: "hasStrippedFields", reason: "Acknowledge stripped fields before publishing" }],
          error: { message: "Acknowledge stripped fields before publishing" },
        });
        return;
      }

      await fulfillJson(route, 200, {
        templateKey,
        version: 2,
        profileCode: draft.profileCode,
        name: draft.name,
        status: "published",
      } satisfies VersionDto);
      return;
    }

    await route.continue();
  });

  await loginAsAdmin(page);
  await page.goto("/#/registry");
  await openTemplateEditor(page, "po", templateKey);

  await expect(page.getByTestId("stripped-fields-banner")).toBeVisible();

  const blockedPublish = waitForTemplateResponse(page, `/api/v1/templates/${templateKey}/publish`, 422);
  await page.getByTestId("template-publish-btn").click();
  await blockedPublish;
  await expect(page.getByTestId("validation-panel")).toBeVisible();

  await page.getByTestId("stripped-fields-acknowledge-btn").click();
  await expect(page.getByTestId("stripped-fields-banner")).toBeHidden({ timeout: 20_000 });

  const acceptedPublish = waitForTemplateResponse(page, `/api/v1/templates/${templateKey}/publish`, 200);
  await page.getByTestId("template-publish-btn").click();
  await acceptedPublish;

  expect(acknowledgeCalls).toBe(1);
  expect(publishCalls).toBe(2);
  await expect(page).toHaveURL(/#\/registry$/);
});

test("template draft save conflicts surface an alert and do not advance publish state", async ({ page }) => {
  const templateKey = "tpl-lock-conflict";
  const draft = makeDraft({
    templateKey,
    lockVersion: 9,
  });

  await page.route("**/api/v1/templates/**", async (route) => {
    const url = new URL(route.request().url());
    const method = route.request().method();

    if (url.pathname === `/api/v1/templates/${templateKey}` && method === "GET") {
      await fulfillJson(route, 200, draft);
      return;
    }

    if (url.pathname === `/api/v1/templates/${templateKey}/draft` && method === "PUT") {
      await fulfillJson(route, 409, {
        error: { message: "Draft lock conflict" },
      });
      return;
    }

    await route.continue();
  });

  const dialogPromise = page.waitForEvent("dialog");

  await loginAsAdmin(page);
  await openTemplateEditor(page, "po", templateKey);

  const richBlockId = await findNthBlockIdByType(page, "richBlock", 0);
  expect(richBlockId).toBeTruthy();
  await selectBlock(page, richBlockId!);
  await page.getByTestId("template-prop-label").fill("Conflito");

  await page.getByTestId("template-save-btn").click();
  const dialog = await dialogPromise;
  expect(dialog.message()).toContain("Conflito de edicao");
  await dialog.accept();
  await expect(page.getByTestId("metadata-bar")).toContainText("Edicao #9");
});

test("blank draft supports section-first authoring, allows all palette blocks, and keeps A4-like canvas", async ({ page }) => {
  const templateKey = "tpl-blank-authoring";
  let draft = makeDraft({
    templateKey,
    lockVersion: 1,
    blocks: [],
  });

  await page.route("**/api/v1/templates/**", async (route) => {
    const url = new URL(route.request().url());
    const method = route.request().method();

    if (url.pathname === `/api/v1/templates/${templateKey}` && method === "GET") {
      await fulfillJson(route, 200, draft);
      return;
    }

    if (url.pathname === `/api/v1/templates/${templateKey}/draft` && method === "PUT") {
      const payload = route.request().postDataJSON() as { blocks: unknown[]; lockVersion: number };
      draft = {
        ...draft,
        blocks: payload.blocks,
        lockVersion: payload.lockVersion + 1,
      };
      await fulfillJson(route, 200, draft);
      return;
    }

    await route.continue();
  });

  await loginAsAdmin(page);
  await openTemplateEditor(page, "po", templateKey);

  // 1) Section insertion must work on a blank draft (no prior typing required).
  await page.getByTestId("palette-insert-section").click();
  await expect.poll(async () => countBlocksByType(page, "section")).toBe(1);

  const sectionId = await findNthBlockIdByType(page, "section", 0);
  expect(sectionId).toBeTruthy();
  await selectBlock(page, sectionId!);

  // 2) All non-section palette blocks must insert from a section context.
  await page.getByTestId("palette-insert-dataTable").click();
  await expect.poll(async () => countBlocksByType(page, "dataTable")).toBe(1);

  await selectBlock(page, sectionId!);
  await page.getByTestId("palette-insert-repeatable").click();
  await expect.poll(async () => countBlocksByType(page, "repeatable")).toBe(1);

  await selectBlock(page, sectionId!);
  await page.getByTestId("palette-insert-richBlock").click();
  await expect.poll(async () => countBlocksByType(page, "richBlock")).toBe(1);

  // 3) Canvas should be portrait-like (A4-ish), not a square card.
  const paper = page.getByTestId("mddm-editor-paper");
  const box = await paper.boundingBox();
  expect(box).toBeTruthy();
  const ratio = (box?.height ?? 0) / Math.max(box?.width ?? 1, 1);
  expect(ratio).toBeGreaterThan(1.2);
});

function makeDraft(overrides: Partial<DraftDto> = {}): DraftDto {
  return {
    templateKey: overrides.templateKey ?? "tpl-draft",
    profileCode: overrides.profileCode ?? "po",
    name: overrides.name ?? "Template de teste",
    status: "draft",
    lockVersion: overrides.lockVersion ?? 1,
    hasStrippedFields: overrides.hasStrippedFields ?? false,
    blocks: overrides.blocks ?? [
      {
        id: "section-base",
        type: "section",
        props: {
          title: "Base",
          styleJson: "{}",
          capabilitiesJson: JSON.stringify({ locked: true, removable: false, reorderable: false }),
        },
        children: [
          {
            id: "rich-base",
            type: "richBlock",
            props: {
              label: "Conteudo",
              styleJson: "{}",
              capabilitiesJson: JSON.stringify({ locked: true, removable: false, editableZones: ["content"] }),
            },
            children: [],
          },
        ],
      },
    ],
    updatedAt: overrides.updatedAt ?? "2026-04-14T00:00:00Z",
  };
}

async function fulfillJson(route: Route, status: number, body: unknown) {
  await route.fulfill({
    status,
    contentType: "application/json",
    body: JSON.stringify(body),
  });
}

function waitForTemplateResponse(page: Page, pathname: string, status: number) {
  return page.waitForResponse(
    (response) => response.status() === status && new URL(response.url()).pathname === pathname,
    { timeout: 20_000 },
  );
}
