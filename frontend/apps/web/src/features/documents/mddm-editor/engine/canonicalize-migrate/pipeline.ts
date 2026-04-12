import { canonicalizeMDDM } from "../../../../../../../../../shared/schemas/canonicalize";
import type { MDDMEnvelope, MDDMBlock } from "../../adapter";

export const CURRENT_MDDM_VERSION = 2;

export class MigrationError extends Error {
  constructor(message: string, public readonly code: string) {
    super(message);
    this.name = "MigrationError";
  }
}

type Migration = (envelope: MDDMEnvelope) => MDDMEnvelope;

// --- Migration 1 → 2: DataTable children → tableContent ---
// Old format: dataTable with dataTableRow/dataTableCell children.
// New format: dataTable with content: { type:"tableContent", ... }, children: [].

function parseColumn(col: unknown): { key: string; label: string } {
  if (col && typeof col === "object" && !Array.isArray(col)) {
    const c = col as Record<string, unknown>;
    return { key: String(c["key"] ?? ""), label: String(c["label"] ?? c["key"] ?? "") };
  }
  return { key: "", label: "" };
}

function parseColumns(value: unknown): Array<{ key: string; label: string }> {
  if (Array.isArray(value)) return value.map(parseColumn);
  if (typeof value === "string") {
    try { return (JSON.parse(value) as unknown[]).map(parseColumn); } catch { /* ignore */ }
  }
  return [];
}

function migrateDataTableBlock(block: MDDMBlock): MDDMBlock {
  if (block.type !== "dataTable") return block;
  const children = Array.isArray(block.children) ? (block.children as MDDMBlock[]) : [];
  const hasOldChildren = children.length > 0 && (children[0] as MDDMBlock).type === "dataTableRow";
  if (!hasOldChildren) return block;

  const props = (block.props ?? {}) as Record<string, unknown>;
  const columns = parseColumns(props["columns"] ?? props["columnsJson"]);

  const headerRow = { cells: columns.map((col) => [{ type: "text" as const, text: col.label }]) };
  const dataRows = children.map((row) => {
    const rowCells = Array.isArray(row.children) ? (row.children as MDDMBlock[]) : [];
    const cells = columns.map((col) => {
      const cell = rowCells.find((c) => String((c.props ?? {})["columnKey"] ?? "") === col.key);
      if (!cell) return [{ type: "text" as const, text: "" }];
      const runs = Array.isArray(cell.children) ? (cell.children as Array<{ text?: string }>) : [];
      return [{ type: "text" as const, text: runs.map((r) => String(r.text ?? "")).join("") }];
    });
    return { cells };
  });

  const tableContent = {
    type: "tableContent" as const,
    columnWidths: columns.map(() => null),
    headerRows: 1,
    rows: [headerRow, ...dataRows],
  };

  const { columns: _c, columnsJson: _cj, minRows: _mr, maxRows: _mxr, ...cleanProps } = props as Record<string, unknown>;
  return { ...block, props: cleanProps, content: tableContent, children: [] };
}

function migrateBlocksRecursive(blocks: MDDMBlock[]): MDDMBlock[] {
  return blocks.map((block) => {
    const migrated = migrateDataTableBlock(block);
    const children = Array.isArray(migrated.children) ? migrated.children as MDDMBlock[] : [];
    return { ...migrated, children: migrateBlocksRecursive(children) };
  });
}

function migrateV1toV2(envelope: MDDMEnvelope): MDDMEnvelope {
  return {
    ...envelope,
    mddm_version: 2,
    blocks: migrateBlocksRecursive(envelope.blocks ?? []),
  };
}

const MIGRATIONS: Record<number, Migration> = {
  1: migrateV1toV2,
};

export type CanonicalizeAndMigrateOptions = {
  /** The version to migrate the envelope TO. Defaults to CURRENT_MDDM_VERSION.
   *  Set to a pinned version for released documents so they stay frozen. */
  targetVersion?: number;
};

export async function canonicalizeAndMigrate(
  envelope: MDDMEnvelope,
  options: CanonicalizeAndMigrateOptions = {},
): Promise<MDDMEnvelope> {
  const target = options.targetVersion ?? CURRENT_MDDM_VERSION;

  if (target > CURRENT_MDDM_VERSION) {
    throw new MigrationError(
      `Target version ${target} is newer than current engine version ${CURRENT_MDDM_VERSION}`,
      "TARGET_TOO_NEW",
    );
  }

  if (envelope === null || typeof envelope !== "object") {
    throw new MigrationError("Envelope is not an object", "INVALID_ENVELOPE");
  }

  const version = (envelope as { mddm_version?: unknown }).mddm_version;
  if (typeof version !== "number" || !Number.isInteger(version) || version < 1) {
    throw new MigrationError("Envelope missing a valid mddm_version", "MISSING_VERSION");
  }

  if (version > target) {
    throw new MigrationError(
      `Envelope version ${version} is newer than current engine version ${CURRENT_MDDM_VERSION}`,
      "VERSION_TOO_NEW",
    );
  }

  let current: MDDMEnvelope = envelope;
  while ((current.mddm_version ?? 0) < target) {
    const from = current.mddm_version ?? 0;
    const migrate = MIGRATIONS[from];
    if (!migrate) {
      throw new MigrationError(
        `No migration registered from version ${from} to ${from + 1}`,
        "MIGRATION_MISSING",
      );
    }
    current = migrate(current);
  }

  return canonicalizeMDDM(current) as MDDMEnvelope;
}
