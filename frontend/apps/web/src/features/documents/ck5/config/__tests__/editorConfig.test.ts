import { describe, it, expect } from 'vitest';
import { createAuthorConfig, createFillConfig } from '../editorConfig';

describe('createAuthorConfig', () => {
  it('includes licenseKey GPL', () => {
    const cfg = createAuthorConfig({ language: 'en' });
    expect(cfg.licenseKey).toBe('GPL');
  });

  it('includes AUTHOR_PLUGINS and AUTHOR_TOOLBAR', () => {
    const cfg = createAuthorConfig({ language: 'en' });
    expect(Array.isArray(cfg.plugins)).toBe(true);
    expect(cfg.plugins!.length).toBeGreaterThan(5);
    expect(cfg.toolbar).toEqual(expect.objectContaining({ items: expect.any(Array) }));
  });

  it('merges MetalDocs primitive plugins when provided', () => {
    class FakePlugin {}
    const cfg = createAuthorConfig({ language: 'en', extraPlugins: [FakePlugin as never] });
    expect(cfg.plugins).toContain(FakePlugin);
  });
});

describe('createFillConfig', () => {
  it('includes restrictedEditing.allowedCommands with a sensible default', () => {
    const cfg = createFillConfig({ language: 'en' });
    expect(cfg.restrictedEditing).toBeDefined();
    expect(Array.isArray(cfg.restrictedEditing!.allowedCommands)).toBe(true);
    expect(cfg.restrictedEditing!.allowedCommands).toContain('bold');
    expect(cfg.restrictedEditing!.allowedCommands).toContain('link');
  });
});
