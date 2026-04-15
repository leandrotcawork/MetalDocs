import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { ClassicEditor, Essentials, Paragraph, Plugin, Widget } from 'ckeditor5';
import { StandardEditingMode } from 'ckeditor5';

import { registerRepeatableConverters } from '../converters';
import { AddRepeatableItemCommand } from '../commands/AddRepeatableItemCommand';
import { InsertRepeatableCommand } from '../commands/InsertRepeatableCommand';
import { RemoveRepeatableItemCommand } from '../commands/RemoveRepeatableItemCommand';
import { registerRepeatableSchema } from '../schema';

class RepeatableCommandsTestPlugin extends Plugin {
  public override init(): void {
    registerRepeatableSchema(this.editor.model.schema);
    registerRepeatableConverters(this.editor);

    this.editor.commands.add('insertRepeatable', new InsertRepeatableCommand(this.editor));
    this.editor.commands.add('addRepeatableItem', new AddRepeatableItemCommand(this.editor));
    this.editor.commands.add('removeRepeatableItem', new RemoveRepeatableItemCommand(this.editor));
  }
}

describe('repeatable commands', () => {
  let editor: ClassicEditor;
  let element: HTMLElement;
  let insertCmd: InsertRepeatableCommand;
  let addCmd: AddRepeatableItemCommand;
  let removeCmd: RemoveRepeatableItemCommand;

  const countItems = (html: string): number => (html.match(/mddm-repeatable__item/g) ?? []).length;

  const findRepeatable = () => {
    const root = editor.model.document.getRoot()!;
    return Array.from(root.getChildren()).find((child) => child.name === 'mddmRepeatable') ?? null;
  };

  beforeEach(async () => {
    element = document.createElement('div');
    document.body.appendChild(element);

    editor = await ClassicEditor.create(element, {
      licenseKey: 'GPL',
      plugins: [
        Essentials,
        Paragraph,
        Widget,
        StandardEditingMode,
        RepeatableCommandsTestPlugin,
      ],
    });

    insertCmd = editor.commands.get('insertRepeatable') as InsertRepeatableCommand;
    addCmd = editor.commands.get('addRepeatableItem') as AddRepeatableItemCommand;
    removeCmd = editor.commands.get('removeRepeatableItem') as RemoveRepeatableItemCommand;

    insertCmd.refresh();
    addCmd.refresh();
    removeCmd.refresh();
  });

  afterEach(async () => {
    await editor.destroy();
  });

  it('inserts repeatable with initialCount 2', () => {
    insertCmd.execute({
      repeatableId: 'r1',
      label: 'Items',
      min: 1,
      max: 5,
      numberingStyle: 'decimal',
      initialCount: 2,
    });

    const html = editor.getData();
    expect(countItems(html)).toBe(2);
  });

  it('adds an item', () => {
    insertCmd.execute({
      repeatableId: 'r1',
      min: 1,
      max: 5,
      initialCount: 1,
    });

    editor.model.change((writer) => {
      const repeatable = findRepeatable()!;
      const firstItem = Array.from(repeatable.getChildren())[0];
      writer.setSelection(firstItem, 'in');
    });

    addCmd.refresh();
    addCmd.execute();

    const html = editor.getData();
    expect(countItems(html)).toBe(2);
  });

  it('removes an item', () => {
    insertCmd.execute({
      repeatableId: 'r1',
      min: 1,
      max: 5,
      initialCount: 2,
    });

    editor.model.change((writer) => {
      const repeatable = findRepeatable()!;
      const firstItem = Array.from(repeatable.getChildren())[0];
      writer.setSelection(firstItem, 'in');
    });

    removeCmd.refresh();
    removeCmd.execute();

    const html = editor.getData();
    expect(countItems(html)).toBe(1);
  });

  it('handles unbounded max without serializing Infinity and keeps add enabled after round-trip', async () => {
    insertCmd.execute({
      repeatableId: 'r-unbounded',
      min: 0,
      max: Infinity,
      initialCount: 1,
    });

    const html = editor.getData();
    expect(html).not.toContain('Infinity');

    await editor.setData(html);

    editor.model.change((writer) => {
      const repeatable = findRepeatable()!;
      const firstItem = Array.from(repeatable.getChildren())[0];
      writer.setSelection(firstItem, 'in');
    });

    addCmd.refresh();
    expect(addCmd.isEnabled).toBe(true);
  });
});
