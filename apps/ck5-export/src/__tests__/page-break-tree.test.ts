import { describe, it, expect } from 'vitest';
import { htmlToExportTree } from '../html-to-export-tree';

describe('html-to-export-tree page break', () => {
  it('emits pageBreak node for div.mddm-page-break', () => {
    const tree = htmlToExportTree('<p>a</p><div class="mddm-page-break"></div><p>b</p>');
    const kinds = tree.children.map((n: any) => n.kind);
    expect(kinds).toContain('pageBreak');
  });
});