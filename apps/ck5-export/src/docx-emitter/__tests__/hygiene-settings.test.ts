import { describe, it, expect } from 'vitest';
import { Packer } from 'docx';
import JSZip from 'jszip';
import { emitDocxFromExportTree } from '../../ck5-docx-emitter';
import { htmlToExportTree } from '../../html-to-export-tree';
import { defaultLayoutTokens } from '../../layout-ir';

describe('settings.xml hygiene', () => {
  it('disables autoHyphenation and sets defaultTabStop', async () => {
    const nodes = htmlToExportTree('<p>x</p>');
    const doc = emitDocxFromExportTree(nodes, defaultLayoutTokens, new Map());
    const buf = await Packer.toBuffer(doc);
    const zip = await JSZip.loadAsync(buf);
    const xml = await zip.file('word/settings.xml')!.async('string');
    expect(xml).toContain('w:autoHyphenation');
    expect(xml).toContain('w:defaultTabStop');
  });
});
