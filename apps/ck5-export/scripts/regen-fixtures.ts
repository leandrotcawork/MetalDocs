import { readdir, readFile, writeFile } from "node:fs/promises";
import { basename, dirname, extname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { parseHTML } from "linkedom";
import { reconcile, type ReconciledBreak } from "../src/pagination/reconcile.ts";
import { validateBids, validateEditorBidSet } from "../src/pagination/validator.ts";

type Golden = Readonly<{
  resolved: readonly ReconciledBreak[];
  logs: Readonly<{
    exactMatches: number;
    minorDrift: number;
    majorDrift: number;
    orphanedEditor: number;
    serverOnly: number;
  }>;
  drift: Readonly<{
    majorDrift: number;
    minorDrift: number;
    perBlockType: Readonly<Record<string, Readonly<{ total: number; majorDrift: number; minorDrift: number }>>>;
  }>;
  validator: Readonly<{
    bidCollision: boolean;
    editorServerDesync: boolean;
  }>;
}>;

const __dirname = dirname(fileURLToPath(import.meta.url));
const FIXTURES_DIR = join(__dirname, "..", "src", "__fixtures__", "pagination");
const CHECK = process.argv.includes("--check");
const MOCK = process.argv.includes("--mock") || process.env.PAGINATOR_MOCK === "1";
const MDDM_WIDGET_CLASSES = new Set([
  "mddm-section",
  "mddm-repeatable",
  "mddm-repeatable-item",
  "mddm-data-table",
  "mddm-field-group",
  "mddm-rich-block",
]);

function classifyBlocks(html: string): { editorBids: string[]; bidToType: Map<string, string>; typeTotals: Record<string, number> } {
  const { document } = parseHTML(`<!DOCTYPE html><html><body>${html}</body></html>`);
  const bidToType = new Map<string, string>();
  const typeTotals: Record<string, number> = {};
  const editorBids: string[] = [];

  for (const el of Array.from(document.querySelectorAll("[data-mddm-bid]"))) {
    const bid = (el as Element).getAttribute("data-mddm-bid");
    if (!bid) continue;
    const element = el as Element;
    const classType = Array.from(MDDM_WIDGET_CLASSES).find((name) => element.classList.contains(name));
    const blockType = classType ?? element.tagName.toLowerCase();
    editorBids.push(bid);
    bidToType.set(bid, blockType);
    typeTotals[blockType] = (typeTotals[blockType] ?? 0) + 1;
  }

  return { editorBids, bidToType, typeTotals };
}

function stableJson(value: unknown): string {
  return `${JSON.stringify(value, null, 2)}\n`;
}

async function main(): Promise<void> {
  const fixtureFiles = (await readdir(FIXTURES_DIR))
    .filter((name) => extname(name) === ".html")
    .sort((a, b) => a.localeCompare(b));

  let paginateWithChromiumFn:
    | ((pool: unknown, rawHtml: string, opts: { timeoutMs: number }) => Promise<Array<{ bid: string; pageNumber: number }>>)
    | null = null;
  let PlaywrightPoolClass: (new (opts: { size: number }) => { init(): Promise<void>; shutdown(): Promise<void> }) | null = null;
  if (!MOCK) {
    ({ paginateWithChromium: paginateWithChromiumFn } = await import("../src/pagination/paginate-with-chromium.ts"));
    ({ PlaywrightPool: PlaywrightPoolClass } = await import("../src/pagination/playwright-pool.ts"));
  }

  const pool = !MOCK ? new (PlaywrightPoolClass as NonNullable<typeof PlaywrightPoolClass>)({ size: 1 }) : null;
  if (pool) {
    await pool.init();
  }

  let changed = 0;
  let checked = 0;

  try {
    for (const fixtureFile of fixtureFiles) {
      const fixturePath = join(FIXTURES_DIR, fixtureFile);
      const goldenPath = join(FIXTURES_DIR, `${basename(fixtureFile, ".html")}.reconciled.json`);
      const html = await readFile(fixturePath, "utf8");
      const { editorBids, bidToType, typeTotals } = classifyBlocks(html);

      const serverBreaks = MOCK
        ? []
        : await (paginateWithChromiumFn as NonNullable<typeof paginateWithChromiumFn>)(pool, html, { timeoutMs: 15_000 });
      const editorBreaks = MOCK
        ? []
        : serverBreaks.map((s) => ({ afterBid: s.bid, pageNumber: s.pageNumber }));
      const reconciled = reconcile(editorBreaks, serverBreaks);

      const perBlockType: Record<string, { total: number; majorDrift: number; minorDrift: number }> = {};
      for (const [blockType, total] of Object.entries(typeTotals)) {
        perBlockType[blockType] = { total, majorDrift: 0, minorDrift: 0 };
      }
      for (const row of reconciled.resolved) {
        const blockType = bidToType.get(row.afterBid) ?? "unknown";
        const bucket = (perBlockType[blockType] ??= { total: 0, majorDrift: 0, minorDrift: 0 });
        if (row.source === "server") bucket.majorDrift += 1;
        if (row.source === "editor-minor-drift") bucket.minorDrift += 1;
      }

      const bidValidation = validateBids(html);
      const editorValidation = validateEditorBidSet(html, editorBids);

      const golden: Golden = {
        resolved: reconciled.resolved,
        logs: reconciled.logs,
        drift: {
          majorDrift: reconciled.logs.majorDrift,
          minorDrift: reconciled.logs.minorDrift,
          perBlockType,
        },
        validator: {
          bidCollision: !bidValidation.ok && bidValidation.error === "bid-collision",
          editorServerDesync: !editorValidation.ok,
        },
      };

      const next = stableJson(golden);
      if (CHECK) {
        checked += 1;
        const current = await readFile(goldenPath, "utf8").catch(() => "");
        if (current !== next) {
          throw new Error(`Golden mismatch: ${goldenPath}`);
        }
      } else {
        await writeFile(goldenPath, next, "utf8");
        changed += 1;
      }
    }
  } finally {
    if (pool) {
      await pool.shutdown();
    }
  }

  if (CHECK) {
    console.log(`Checked ${checked} fixture goldens.`);
    return;
  }
  console.log(`Wrote ${changed} fixture goldens.`);
}

await main();
