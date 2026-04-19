import { describe, it, expect, vi, afterEach } from 'vitest';
import { cleanup, render, screen } from '@testing-library/react';
import type { ReactNode } from 'react';
import { MetalDocsEditor } from '../src/MetalDocsEditor';

afterEach(cleanup);

vi.mock('@eigenpal/docx-js-editor', () => ({
  templatePlugin: { name: 'template', id: 'template' },
  PluginHost: ({
    plugins,
    children,
  }: {
    plugins: Array<{ name?: string }>;
    children: ReactNode;
  }) => (
    <div
      data-testid="plugin-host"
      data-plugins={plugins.length}
      data-plugin-names={plugins.map((p) => p.name ?? '?').join(',')}
    >
      {children}
    </div>
  ),
  DocxEditor: () => <div data-testid="docx-editor-mock" />,
}));

describe('template plugin wiring', () => {
  it('includes template plugin when no sidebar model is provided', () => {
    render(<MetalDocsEditor mode="document-edit" author="u1" />);
    const host = screen.getByTestId('plugin-host');
    expect(host.getAttribute('data-plugins')).toBe('1');
    expect(host.getAttribute('data-plugin-names')).toContain('template');
  });

  it('adds sidebar bridge plugin when sidebar model is provided', () => {
    render(
      <MetalDocsEditor
        mode="document-edit"
        author="u1"
        sidebarModel={{
          used: ['a'],
          missing: [],
          orphans: [],
          bannerError: false,
          errorCategories: [],
        }}
      />
    );
    const host = screen.getByTestId('plugin-host');
    const names = host.getAttribute('data-plugin-names') ?? '';
    expect(host.getAttribute('data-plugins')).toBe('2');
    expect(names).toContain('template');
    expect(names).toContain('metaldocs-sidebar-model');
  });

  it('includes external plugins', () => {
    render(
      <MetalDocsEditor
        mode="document-edit"
        author="u1"
        externalPlugins={[{ id: 'custom', name: 'custom' }]}
      />
    );
    const host = screen.getByTestId('plugin-host');
    const pluginCount = Number(host.getAttribute('data-plugins') ?? '0');
    expect(pluginCount).toBeGreaterThanOrEqual(2);
    expect(host.getAttribute('data-plugin-names')).toContain('custom');
  });
});
