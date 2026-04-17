import type { TemplateRecord } from './localStorageStub';

async function throwIfNotOk(res: Response): Promise<Response> {
  if (!res.ok) throw new Error(`API ${res.status}: ${res.statusText}`);
  return res;
}

// ---------------------------------------------------------------------------
// Template persistence (Author editor)
// ---------------------------------------------------------------------------

export async function saveTemplate(
  id: string,
  contentHtml: string,
  manifest: TemplateRecord['manifest'],
): Promise<void> {
  await throwIfNotOk(
    await fetch(`/api/v1/templates/${encodeURIComponent(id)}/ck5-draft`, {
      method: 'PUT',
      headers: { 'content-type': 'application/json' },
      credentials: 'include',
      body: JSON.stringify({ contentHtml, manifest }),
    }),
  );
}

export async function loadTemplate(id: string): Promise<TemplateRecord | null> {
  const res = await fetch(`/api/v1/templates/${encodeURIComponent(id)}/ck5-draft`, {
    credentials: 'include',
  });
  if (res.status === 404) return null;
  await throwIfNotOk(res);
  const data = (await res.json()) as { contentHtml: string; manifest: TemplateRecord['manifest'] };
  return {
    id,
    contentHtml: data.contentHtml ?? '',
    manifest: data.manifest ?? { fields: [] },
  };
}

// ---------------------------------------------------------------------------
// Document persistence (Fill editor)
// ---------------------------------------------------------------------------

export async function saveDocument(id: string, contentHtml: string): Promise<void> {
  await throwIfNotOk(
    await fetch(`/api/v1/documents/${encodeURIComponent(id)}/content/ck5`, {
      method: 'POST',
      headers: { 'content-type': 'application/json' },
      credentials: 'include',
      body: JSON.stringify({ body: contentHtml }),
    }),
  );
}

export async function loadDocument(id: string): Promise<string | null> {
  const res = await fetch(`/api/v1/documents/${encodeURIComponent(id)}/content/ck5`, {
    credentials: 'include',
  });
  if (res.status === 404) return null;
  await throwIfNotOk(res);
  const data = (await res.json()) as { body?: string };
  return data.body ?? null;
}
