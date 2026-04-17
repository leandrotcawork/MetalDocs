import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { ClassicEditor, Essentials, Paragraph, Heading } from 'ckeditor5';
import { MddmBlockIdentityPlugin } from '../../MddmBlockIdentityPlugin';
import { MddmSectionPlugin } from '../../MddmSectionPlugin';
import { findEnclosingSection } from '../SectionScope';

describe('findEnclosingSection', () => {
  let editor: ClassicEditor;
  let host: HTMLElement;

  beforeEach(async () => {
    host = document.createElement('div');
    document.body.appendChild(host);
    editor = await ClassicEditor.create(host, {
      licenseKey: 'GPL',
      plugins: [Essentials, Paragraph, Heading, MddmBlockIdentityPlugin, MddmSectionPlugin],
    });
  });

  afterEach(async () => {
    await editor.destroy();
    host.remove();
  });

  it('returns null for a root-level position with no section ancestor', () => {
    editor.setData('<p>A</p>');
    const root = editor.model.document.getRoot()!;
    const pos = editor.model.createPositionFromPath(root, [0]);
    expect(findEnclosingSection(pos)).toBeNull();
  });

  it('returns the enclosing mddmSection for a position inside a section', () => {
    editor.execute('insertMddmSection', { variant: 'editable' });

    const root = editor.model.document.getRoot()!;
    let sectionEl: ReturnType<typeof root.getChild> | null = null;
    for (let i = 0; i < root.childCount; i++) {
      const child = root.getChild(i);
      if (child && child.is('element') && child.name === 'mddmSection') {
        sectionEl = child;
        break;
      }
    }
    expect(sectionEl).not.toBeNull();

    // Find a descendant paragraph inside the section and build a position there.
    const iterator = editor.model.createRangeIn(sectionEl as never).getItems();
    let descendantPos: ReturnType<typeof editor.model.createPositionBefore> | null = null;
    for (const item of iterator) {
      if ((item as { is?: (t: string) => boolean }).is?.('element')) {
        descendantPos = editor.model.createPositionBefore(item as never);
        break;
      }
    }
    expect(descendantPos).not.toBeNull();

    const found = findEnclosingSection(descendantPos!);
    expect(found).toBe(sectionEl);
  });
});
