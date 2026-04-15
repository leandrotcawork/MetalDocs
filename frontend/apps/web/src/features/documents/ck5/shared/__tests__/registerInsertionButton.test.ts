import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { ClassicEditor, Essentials, Paragraph, Command } from 'ckeditor5';
import { registerInsertionButton } from '../registerInsertionButton';

describe('registerInsertionButton', () => {
  let editor: ClassicEditor;

  beforeEach(async () => {
    const el = document.createElement('div');
    document.body.appendChild(el);
    editor = await ClassicEditor.create(el, {
      licenseKey: 'GPL',
      plugins: [Essentials, Paragraph],
    });
    // Add a fake command (always enabled so execute events propagate)
    const fakeCmd = new (class extends Command {
      override refresh() { this.isEnabled = true; }
      override execute() {}
    })(editor);
    fakeCmd.refresh();
    editor.commands.add('fakeInsert', fakeCmd);
  });

  afterEach(async () => {
    await editor.destroy();
  });

  it('registers a ButtonView in the component factory', () => {
    registerInsertionButton(editor, {
      componentName: 'fakeInsertButton',
      commandName: 'fakeInsert',
      label: 'Fake insert',
    });
    expect(editor.ui.componentFactory.has('fakeInsertButton')).toBe(true);
  });

  it('button fires the command on execute', () => {
    let fired = false;
    editor.commands.get('fakeInsert')!.on('execute', () => {
      fired = true;
    });
    registerInsertionButton(editor, {
      componentName: 'fakeInsertButton',
      commandName: 'fakeInsert',
      label: 'Fake insert',
    });
    const view = editor.ui.componentFactory.create('fakeInsertButton');
    view.fire('execute');
    expect(fired).toBe(true);
  });
});
