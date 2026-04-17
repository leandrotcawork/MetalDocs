import { describe, it, expect } from 'vitest';
import { Packer } from 'docx';
import JSZip from 'jszip';
import { emitDocxFromExportTree, injectDocxSettings } from '../../ck5-docx-emitter';
import { htmlToExportTree } from '../../html-to-export-tree';
import { defaultLayoutTokens } from '../../layout-ir';

describe('settings.xml hygiene', () => {
  it('disables autoHyphenation and sets defaultTabStop after post-pack inject', async () => {
    const nodes = htmlToExportTree('<p>x</p>');
    const doc = emitDocxFromExportTree(nodes, defaultLayoutTokens, new Map());
    const raw = await Packer.toBuffer(doc);
    const buf = await injectDocxSettings(raw);
    const zip = await JSZip.loadAsync(buf);
    const xml = await zip.file('word/settings.xml')!.async('string');
    expect(xml).toContain('w:autoHyphenation w:val="false"');
    expect(xml).toContain('w:defaultTabStop w:val="720"');
  });
});
