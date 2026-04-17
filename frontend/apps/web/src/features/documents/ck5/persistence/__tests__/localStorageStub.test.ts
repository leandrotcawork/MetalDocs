import { describe, it, expect, beforeEach } from 'vitest';
import { saveDocument, loadDocument, saveTemplate, loadTemplate } from '../localStorageStub';

beforeEach(() => {
  localStorage.clear();
});

describe('localStorageStub', () => {
  it('saves and loads a document', () => {
    saveDocument('doc-1', '<p>Hello</p>');
    expect(loadDocument('doc-1')).toBe('<p>Hello</p>');
  });

  it('saves and loads a template with manifest', () => {
    saveTemplate('tpl-1', '<section class="mddm-section"/>', { fields: [] });
    const tpl = loadTemplate('tpl-1');
    expect(tpl?.contentHtml).toBe('<section class="mddm-section"/>');
    expect(tpl?.manifest).toEqual({ fields: [] });
  });

  it('returns null for unknown ids', () => {
    expect(loadDocument('missing')).toBeNull();
    expect(loadTemplate('missing')).toBeNull();
  });
});
