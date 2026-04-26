export interface PlaceholderCatalogEntry {
  key: string;
  label: string;
  description: string;
}

export async function fetchPlaceholderCatalog(): Promise<PlaceholderCatalogEntry[]> {
  const r = await fetch('/api/v2/templates/v2/placeholder-catalog');
  if (!r.ok) throw new Error(`http_${r.status}`);
  const body = await r.json() as { items: PlaceholderCatalogEntry[] };
  return body.items ?? [];
}
