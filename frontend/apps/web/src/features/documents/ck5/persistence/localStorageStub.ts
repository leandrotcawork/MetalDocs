import type { TemplateDraftStatus } from './templatePublishApi';

export interface TemplateRecord {
  id: string;
  contentHtml: string;
  manifest: { fields: Array<{ id: string; label?: string; type: string; required?: boolean }> };
  draft_status?: TemplateDraftStatus;
}

const DOC_KEY = (id: string) => `ck5.doc.${id}`;
const TPL_KEY = (id: string) => `ck5.tpl.${id}`;

export function saveDocument(id: string, contentHtml: string): void {
  localStorage.setItem(DOC_KEY(id), contentHtml);
}

export function loadDocument(id: string): string | null {
  return localStorage.getItem(DOC_KEY(id));
}

export function saveTemplate(id: string, contentHtml: string, manifest: TemplateRecord['manifest']): void {
  const rec: TemplateRecord = { id, contentHtml, manifest };
  localStorage.setItem(TPL_KEY(id), JSON.stringify(rec));
}

export function loadTemplate(id: string): TemplateRecord | null {
  const raw = localStorage.getItem(TPL_KEY(id));
  return raw ? (JSON.parse(raw) as TemplateRecord) : null;
}
