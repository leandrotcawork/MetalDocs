import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { ClassicEditor, Essentials, Paragraph, Heading } from 'ckeditor5';
import { MddmBlockIdentityPlugin } from '../../MddmBlockIdentityPlugin';
import { planBreaks } from '../BreakPlanner';

describe('planBreaks', () => {
  let editor: ClassicEditor;
  let host: HTMLElement;

  beforeEach(async () => {
    host = document.createElement('div');
    document.body.appendChild(host);
    editor = await ClassicEditor.create(host, {
      licenseKey: 'GPL',
      plugins: [Essentials, Paragraph, Heading, MddmBlockIdentityPlugin],
    });
  });

  afterEach(async () => {
    await editor.destroy();
    host.remove();
  });

  it('emits one candidate per paragraph', () => {
    editor.setData('<p>A</p><p>B</p><p>C</p>');
    const from = editor.model.createPositionFromPath(editor.model.document.getRoot()!, [0]);
    const cands = planBreaks(editor, from);
    expect(cands).toHaveLength(3);
  });

  it('respects dirtyStart — blocks before are skipped', () => {
    editor.setData('<p>A</p><p>B</p><p>C</p>');
    const from = editor.model.createPositionFromPath(editor.model.document.getRoot()!, [2]);
    const cands = planBreaks(editor, from);
    expect(cands).toHaveLength(1); // only after C
  });

  it('skips keep-with-next headings', () => {
    editor.setData('<p>A</p><h2>T</h2><p>B</p>');
    const from = editor.model.createPositionFromPath(editor.model.document.getRoot()!, [0]);
    const cands = planBreaks(editor, from);
    expect(cands).toHaveLength(2); // heading excluded
  });
});
