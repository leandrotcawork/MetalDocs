import { describe, it, expect } from 'vitest';
import { Packer } from 'docx';
import JSZip from 'jszip';
import { emitDocxFromExportTree } from '../../ck5-docx-emitter';
import { htmlToExportTree } from '../../html-to-export-tree';
import { defaultLayoutTokens } from '../../layout-ir';

describe('widowControl hygiene', () => {
  it('every paragraph in document.xml has w:widowControl w:val="false"', async () => {
    const nodes = htmlToExportTree('<p>x</p><p>y</p>');
    const doc = emitDocxFromExportTree(nodes, defaultLayoutTokens, new Map());
    const buf = await Packer.toBuffer(doc);
    const zip = await JSZip.loadAsync(buf);
    const xml = await zip.file('word/document.xml')!.async('string');
    const paragraphCount = (xml.match(/<w:p[\s>]/g) ?? []).length;
    const widowFalseCount = (xml.match(/<w:widowControl w:val="false"/g) ?? []).length;
    expect(widowFalseCount).toBe(paragraphCount);
  });
});