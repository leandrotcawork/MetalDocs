import { canonicalizeMDDM } from "../../../../../../../../../shared/schemas/canonicalize";
import type { MDDMEnvelope } from "../../adapter";

export const CURRENT_MDDM_VERSION = 1;

export class MigrationError extends Error {
  constructor(message: string, public readonly code: string) {
    super(message);
    this.name = "MigrationError";
  }
}

type Migration = (envelope: MDDMEnvelope) => MDDMEnvelope;

const MIGRATIONS: Record<number, Migration> = {};

export async function canonicalizeAndMigrate(envelope: MDDMEnvelope): Promise<MDDMEnvelope> {
  if (envelope === null || typeof envelope !== "object") {
    throw new MigrationError("Envelope is not an object", "INVALID_ENVELOPE");
  }

  const version = (envelope as { mddm_version?: unknown }).mddm_version;
  if (typeof version !== "number" || !Number.isInteger(version) || version < 1) {
    throw new MigrationError("Envelope missing a valid mddm_version", "MISSING_VERSION");
  }

  if (version > CURRENT_MDDM_VERSION) {
    throw new MigrationError(
      `Envelope version ${version} is newer than current engine version ${CURRENT_MDDM_VERSION}`,
      "VERSION_TOO_NEW",
    );
  }

  let current: MDDMEnvelope = envelope;
  while ((current.mddm_version ?? 0) < CURRENT_MDDM_VERSION) {
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
