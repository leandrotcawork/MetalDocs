import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import {
  ClassicEditor,
  Essentials,
  Paragraph,
  StandardEditingMode,
  Widget,
} from 'ckeditor5';
import { MddmRichBlockPlugin } from '../index';

function countExceptionElements(editor: any): number {
  let count = 0;
  function visit(node: any): void {
    if (node.is && node.is('element', 'restrictedEditingException')) count++;
    if (node.getChildren) {
      for (const child of node.getChildren()) visit(child);
    }
  }
  visit(editor.model.document.getRoot());
  return count;
}

describe('MddmRichBlockPlugin', () => {
  let editor: ClassicEditor;

  beforeEach(async () => {
    const el = document.createElement('div');
    document.body.appendChild(el);
    editor = await ClassicEditor.create(el, {
      licenseKey: 'GPL',
      plugins: [
        Essentials,
        Paragraph,
        Widget,
        StandardEditingMode,
        MddmRichBlockPlugin,
      ],
    });
  });

  afterEach(async () => {
    await editor.destroy();
  });

  it('registers insertMddmRichBlock command', () => {
    expect(editor.commands.get('insertMddmRichBlock')).toBeDefined();
  });

  it('inserts a <div class="mddm-rich-block"> on execute', () => {
    (editor.commands.get('insertMddmRichBlock') as { refresh(): void }).refresh();
    editor.execute('insertMddmRichBlock');
    const html = editor.getData({ trim: false as const });
    expect(html).toContain('class="mddm-rich-block"');
  });

  it('plants a block exception marker on the rich block', () => {
    (editor.commands.get('insertMddmRichBlock') as { refresh(): void }).refresh();
    editor.execute('insertMddmRichBlock');
    expect(countExceptionElements(editor)).toBeGreaterThanOrEqual(1);
  });

  it('round-trips mddm-rich-block HTML via setData/getData', () => {
    editor.setData('<div class="mddm-rich-block"><p>Hello</p></div>');
    const out = editor.getData();
    expect(out).toContain('class="mddm-rich-block"');
    expect(out).toContain('Hello');
  });
});
