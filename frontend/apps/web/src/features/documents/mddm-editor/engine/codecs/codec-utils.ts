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
