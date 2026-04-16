import type { TemplateRecord } from './localStorageStub';

async function throwIfNotOk(res: Response): Promise<Response> {
  if (!res.ok) throw new Error(`API ${res.status}: ${res.statusText}`);
  return res;
}

export async function saveTemplate(
  id: string,
  contentHtml: string,
  manifest: TemplateRecord['manifest'],
): Promise<void> {
  await throwIfNotOk(
    await fetch(`/api/v1/templates/${encodeURIComponent(id)}/draft`, {
      method: 'PUT',
      headers: { 'content-type': 'application/json' },
      credentials: 'include',
      body: JSON.stringify({ contentHtml, manifest }),
    }),
  );
}

export async function loadTemplate(id: string): Promise<TemplateRecord | null> {
  const res = await fetch(`/api/v1/templates/${encodeURIComponent(id)}`, {
    credentials: 'include',
  });
  if (res.status === 404) return null;
  await throwIfNotOk(res);
  return res.json() as Promise<TemplateRecord>;
}

export async function saveDocument(id: string, contentHtml: string): Promise<void> {
  await throwIfNotOk(
    await fetch(`/api/v1/documents/${encodeURIComponent(id)}/content/browser`, {
      method: 'POST',
      headers: { 'content-type': 'application/json' },
      credentials: 'include',
      body: JSON.stringify({ contentHtml }),
    }),
  );
}

export async function loadDocument(id: string): Promise<string | null> {
  const res = await fetch(`/api/v1/documents/${encodeURIComponent(id)}`, {
    credentials: 'include',
  });
  if (res.status === 404) return null;
  await throwIfNotOk(res);
  const rec = (await res.json()) as { contentHtml?: string };
  return rec.contentHtml ?? null;
}
