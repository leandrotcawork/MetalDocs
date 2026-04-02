export function readCanvasSlotValue(values: Record<string, unknown>, path: string): unknown {
  return path.split(".").reduce<unknown>((current, segment) => {
    if (!current || typeof current !== "object" || Array.isArray(current)) {
      return undefined;
    }
    return (current as Record<string, unknown>)[segment];
  }, values);
}

export function writeCanvasSlotValue(values: Record<string, unknown>, path: string, nextValue: unknown): Record<string, unknown> {
  const segments = path
    .split(".")
    .map((segment) => segment.trim())
    .filter(Boolean);

  if (segments.length === 0) {
    return values;
  }

  const next = { ...values };
  let cursor: Record<string, unknown> = next;

  for (let index = 0; index < segments.length - 1; index += 1) {
    const segment = segments[index];
    const current = cursor[segment];
    const nextCursor = current && typeof current === "object" && !Array.isArray(current) ? { ...(current as Record<string, unknown>) } : {};
    cursor[segment] = nextCursor;
    cursor = nextCursor;
  }

  cursor[segments[segments.length - 1]] = nextValue;
  return next;
}
