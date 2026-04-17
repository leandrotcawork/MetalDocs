import { describe, it, expect } from 'vitest';
import { Packer } from 'docx';
import JSZip from 'jszip';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { existsSync } from 'node:fs';
import { emitDocxFromExportTree } from '../../ck5-docx-emitter';
import { htmlToExportTree } from '../../html-to-export-tree';
import { defaultLayoutTokens } from '../../layout-ir';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const FONTS_DIR = path.resolve(__dirname, '..', '..', '..', 'fonts');
const fontsExist = existsSync(path.join(FONTS_DIR, 'Carlito-Regular.ttf'));

describe('Carlito font embed', () => {
  it.skipIf(!fontsExist)('embeds Carlito TTFs and references in fontTable', async () => {
    const nodes = htmlToExportTree('<p>x</p>');
    const doc = emitDocxFromExportTree(nodes, defaultLayoutTokens, new Map());
    const buf = await Packer.toBuffer(doc);
    const zip = await JSZip.loadAsync(buf);
    const fontTable = await zip.file('word/fontTable.xml')!.async('string');
    expect(fontTable).toContain('Carlito');
  });

  it('readyz: all 4 Carlito TTF files exist in fonts/', () => {
    const variants = ['Regular', 'Bold', 'Italic', 'BoldItalic'];
    for (const v of variants) {
      expect(existsSync(path.join(FONTS_DIR, `Carlito-${v}.ttf`)), `Carlito-${v}.ttf must exist`).toBe(true);
    }
  });
});