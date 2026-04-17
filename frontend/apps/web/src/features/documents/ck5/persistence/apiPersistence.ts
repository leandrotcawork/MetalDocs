import type { TemplateRecord } from './localStorageStub';

async function throwIfNotOk(res: Response): Promise<Response> {
  if (!res.ok) throw new Error(`API ${res.status}: ${res.statusText}`);
  return res;
}

// Per-template cache of the server-side draft envelope (blocks/theme/meta +
// lockVersion). The server's PUT /draft endpoint uses optimistic locking on
// lockVersion and expects the full blocks structure, not just the HTML.
// loadTemplate primes this cache; saveTemplate merges new HTML into the
// cached blocks and replays the lockVersion the server last returned.
interface DraftEnvelope {
  lockVersion: number;
  blocks: Record<string, unknown>;
  theme: Record<string, unknown>;
  meta: Record<string, unknown>;
}

const draftCache = new Map<string, DraftEnvelope>();

interface TemplateDraftResponse {
  templateKey: string;
  profileCode: string;
  name: string;
  status: string;
  lockVersion: number;
  hasStrippedFields: boolean;
  blocks: Record<string, unknown> | null;
  theme?: Record<string, unknown>;
  meta?: Record<string, unknown>;
  updatedAt?: string;
}

function extractContentHtml(blocks: Record<string, unknown> | null | undefined): string {
  if (!blocks || typeof blocks !== 'object') return '';
  const ck5 = (blocks as { _ck5?: { contentHtml?: string } })._ck5;
  return ck5?.contentHtml ?? '';
}

function withUpdatedContentHtml(
  blocks: Record<string, unknown>,
  contentHtml: string,
): Record<string, unknown> {
  const prevCk5 = (blocks._ck5 as Record<string, unknown> | undefined) ?? {};
  return { ...blocks, _ck5: { ...prevCk5, contentHtml } };
}

// ---------------------------------------------------------------------------
// Template persistence (Author editor)
// ---------------------------------------------------------------------------

export async function saveTemplate(
  id: string,
  contentHtml: string,
  _manifest: TemplateRecord['manifest'],
): Promise<void> {
  const cached = draftCache.get(id);
  if (!cached) {
    // No load happened yet — skip the save; AuthorPage will issue a save
    // after the initial load completes.
    return;
  }
  const body = {
    blocks: withUpdatedContentHtml(cached.blocks, contentHtml),
    theme: cached.theme,
    meta: cached.meta,
    lockVersion: cached.lockVersion,
  };
  const res = await fetch(`/api/v1/templates/${encodeURIComponent(id)}/draft`, {
    method: 'PUT',
    headers: { 'content-type': 'application/json' },
    credentials: 'include',
    body: JSON.stringify(body),
  });
  await throwIfNotOk(res);
  const updated = (await res.json()) as TemplateDraftResponse;
  draftCache.set(id, {
    lockVersion: updated.lockVersion,
    blocks: updated.blocks ?? body.blocks,
    theme: updated.theme ?? {},
    meta: updated.meta ?? {},
  });
}

export async function loadTemplate(id: string): Promise<TemplateRecord | null> {
  const res = await fetch(`/api/v1/templates/${encodeURIComponent(id)}`, {
    credentials: 'include',
  });
  if (res.status === 404) return null;
  await throwIfNotOk(res);
  const server = (await res.json()) as TemplateDraftResponse;
  const blocks = server.blocks ?? { id: 'po-root', type: 'page', children: [], _ck5: { contentHtml: '' } };
  draftCache.set(id, {
    lockVersion: server.lockVersion,
    blocks,
    theme: server.theme ?? {},
    meta: server.meta ?? {},
  });
  return {
    id,
    contentHtml: extractContentHtml(blocks),
    manifest: { fields: [] },
    draft_status: server.status === 'published' ? 'published' : server.status === 'pending_review' ? 'pending_review' : 'draft',
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
