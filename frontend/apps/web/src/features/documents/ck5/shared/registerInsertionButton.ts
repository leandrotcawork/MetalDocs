import { ButtonView, type Editor } from 'ckeditor5';

export interface InsertionButtonOptions {
  componentName: string;
  commandName: string;
  label: string;
  tooltip?: string | boolean;
  icon?: string;
  executeOptions?: Record<string, unknown>;
}

export function registerInsertionButton(editor: Editor, opts: InsertionButtonOptions): void {
  editor.ui.componentFactory.add(opts.componentName, (locale) => {
    const view = new ButtonView(locale);
    view.set({
      label: opts.label,
      tooltip: opts.tooltip ?? true,
      withText: !opts.icon,
      icon: opts.icon,
    });

    const cmd = editor.commands.get(opts.commandName);
    if (cmd) {
      view.bind('isEnabled').to(cmd, 'isEnabled');
    }

    view.on('execute', () => {
      editor.execute(opts.commandName, opts.executeOptions ?? {});
      editor.editing.view.focus();
    });

    return view;
  });
}
