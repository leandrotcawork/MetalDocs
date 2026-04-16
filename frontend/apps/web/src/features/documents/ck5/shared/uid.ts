export function uid(prefix = 'id'): string {
  const raw =
    typeof crypto !== 'undefined' && crypto.randomUUID
      ? crypto.randomUUID()
      : Math.random().toString(36).slice(2) + Date.now().toString(36);
  return `${prefix}-${raw.replace(/-/g, '').slice(0, 12)}`;
}
