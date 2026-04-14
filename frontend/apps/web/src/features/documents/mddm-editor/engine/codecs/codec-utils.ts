export function safeParse(json: string, fallback: Record<string, unknown>): Record<string, unknown> {
  if (!json || json === "") return fallback;
  try {
    const parsed = JSON.parse(json);
    if (typeof parsed !== "object" || parsed === null) return fallback;
    return parsed as Record<string, unknown>;
  } catch {
    return fallback;
  }
}

export function expectString(value: unknown): string | undefined {
  return typeof value === "string" ? value : undefined;
}

export function expectBoolean(value: unknown, defaultValue: boolean): boolean {
  return typeof value === "boolean" ? value : defaultValue;
}

export function expectNumber(value: unknown, defaultValue: number): number {
  return typeof value === "number" ? value : defaultValue;
}

export function stripUndefined<T extends Record<string, unknown>>(obj: T): Partial<T> {
  const result: Record<string, unknown> = {};
  for (const [key, val] of Object.entries(obj)) {
    if (val !== undefined) result[key] = val;
  }
  return result as Partial<T>;
}

type ThemeColors = {
  accent: string;
  accentLight: string;
  accentDark: string;
  accentBorder: string;
};

export function resolveThemeRef(value: string | undefined, theme: ThemeColors): string | undefined {
  if (value === undefined) return undefined;
  if (value.startsWith("theme.")) {
    const key = value.slice(6) as keyof ThemeColors;
    return theme[key] ?? value;
  }
  return value;
}

// ---------------------------------------------------------------------------
// Strict codec utilities — throw CodecStrictError instead of returning undefined
// ---------------------------------------------------------------------------

export class CodecStrictError extends Error {
  constructor(
    public readonly field: string,
    public readonly reason: string,
  ) {
    super(`[strict] ${field}: ${reason}`);
    this.name = "CodecStrictError";
  }
}

/** Like expectString but throws CodecStrictError if the value is missing or not a string. */
export function expectStringStrict(obj: Record<string, unknown>, key: string): string {
  const val = obj[key];
  if (val === undefined || val === null) {
    throw new CodecStrictError(key, "missing required field");
  }
  if (typeof val !== "string") {
    throw new CodecStrictError(key, `expected string, got ${typeof val}`);
  }
  return val;
}

/** Like expectNumber but throws CodecStrictError if the value is missing or not a number. */
export function expectNumberStrict(obj: Record<string, unknown>, key: string): number {
  const val = obj[key];
  if (val === undefined || val === null) {
    throw new CodecStrictError(key, "missing required field");
  }
  if (typeof val !== "number") {
    throw new CodecStrictError(key, `expected number, got ${typeof val}`);
  }
  return val;
}

/** Throws CodecStrictError if obj has any key not in allowedKeys. */
export function assertNoUnknownFields(
  obj: Record<string, unknown>,
  allowedKeys: string[],
  context: string,
): void {
  for (const key of Object.keys(obj)) {
    if (!allowedKeys.includes(key)) {
      throw new CodecStrictError(`${context}.${key}`, "unknown field");
    }
  }
}

/** Validates a boolean field strictly — throws CodecStrictError if missing or wrong type. */
export function expectBooleanStrict(obj: Record<string, unknown>, key: string): boolean {
  const val = obj[key];
  if (val === undefined || val === null) {
    throw new CodecStrictError(key, "missing required field");
  }
  if (typeof val !== "boolean") {
    throw new CodecStrictError(key, `expected boolean, got ${typeof val}`);
  }
  return val;
}
