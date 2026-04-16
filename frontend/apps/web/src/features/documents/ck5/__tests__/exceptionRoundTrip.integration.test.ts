import { describe, it, expect } from 'vitest';
import { ClassicEditor } from 'ckeditor5';
import { createAuthorConfig, createFillConfig } from '../config/editorConfig';

async function mount(config: ReturnType<typeof createAuthorConfig>) {
  const el = document.createElement('div');
  document.body.appendChild(el);
  return ClassicEditor.create(el, config);
}

describe('exception element round-trip', () => {
  it('section insertion plants an exception element that survives Author save', async () => {
    const ed = await mount(createAuthorConfig({}));
    (ed.commands.get('insertMddmSection') as { refresh(): void }).refresh();
    ed.execute('insertMddmSection', { variant: 'editable' });
    const html = ed.getData();
    expect(html).toMatch(/class="restricted-editing-exception"/);
    await ed.destroy();
  });

  it('section exception element survives round-trip to Fill editor', async () => {
    const author = await mount(createAuthorConfig({}));
    (author.commands.get('insertMddmSection') as { refresh(): void }).refresh();
    author.execute('insertMddmSection', { variant: 'editable' });
    const html = author.getData();
    await author.destroy();

    const fill = await mount(createFillConfig({}));
    fill.setData(html);
    // RestrictedEditingMode upcasts the exception class into markers on Fill side - this is an internal CKEditor detail.
    const markers = Array.from(fill.model.markers).filter((m) =>
      m.name.startsWith('restrictedEditingException:'),
    );
    expect(markers.length).toBeGreaterThanOrEqual(1);
    await fill.destroy();
  });

  it('repeatable items each carry an exception element that survives Author to Fill', async () => {
    const author = await mount(createAuthorConfig({}));
    (author.commands.get('insertMddmRepeatable') as { refresh(): void }).refresh();
    author.execute('insertMddmRepeatable', { min: 1, max: 5, initialCount: 3 });
    const html = author.getData();
    expect((html.match(/class="restricted-editing-exception"/g) || []).length).toBeGreaterThanOrEqual(3);
    await author.destroy();

    const fill = await mount(createFillConfig({}));
    fill.setData(html);
    // RestrictedEditingMode upcasts the exception class into markers on Fill side - this is an internal CKEditor detail.
    const markers = Array.from(fill.model.markers).filter((m) =>
      m.name.startsWith('restrictedEditingException:'),
    );
    expect(markers.length).toBeGreaterThanOrEqual(3);
    const items = fill.getData().match(/class="mddm-repeatable__item"/g) || [];
    expect(items.length).toBe(3);
    await fill.destroy();
  });

  it('DataTable: per-cell exceptions applied at save survive Fill load', async () => {
    const { applyPerCellExceptions } = await import('../plugins/MddmDataTablePlugin');
    const author = await mount(createAuthorConfig({}));
    author.setData(
      '<figure class="table"><table data-mddm-variant="fixed"><tbody>' +
        '<tr><td>a</td><td>b</td></tr>' +
        '<tr><td>c</td><td>d</td></tr>' +
        '</tbody></table></figure>',
    );
    applyPerCellExceptions(author);
    const html = author.getData();
    expect((html.match(/class="restricted-editing-exception"/g) || []).length).toBe(4);
    await author.destroy();

    const fill = await mount(createFillConfig({}));
    fill.setData(html);
    // RestrictedEditingMode upcasts the exception class into markers on Fill side - this is an internal CKEditor detail.
    const markers = Array.from(fill.model.markers).filter((m) =>
      m.name.startsWith('restrictedEditingException:'),
    );
    expect(markers.length).toBe(4);
    const out = fill.getData();
    for (const letter of ['a', 'b', 'c', 'd']) {
      expect(out).toContain(letter);
    }
    await fill.destroy();
  });

  it('rich block exception element survives round-trip', async () => {
    const author = await mount(createAuthorConfig({}));
    (author.commands.get('insertMddmRichBlock') as { refresh(): void }).refresh();
    author.execute('insertMddmRichBlock');
    const html = author.getData({ trim: 'none' });
    expect(html).toMatch(/class="restricted-editing-exception"/);
    const fill = await mount(createFillConfig({}));
    fill.setData(html);
    // RestrictedEditingMode upcasts the exception class into markers on Fill side - this is an internal CKEditor detail.
    const markers = Array.from(fill.model.markers).filter((m) =>
      m.name.startsWith('restrictedEditingException:'),
    );
    expect(markers.length).toBeGreaterThanOrEqual(1);
    await fill.destroy();
  });
});
