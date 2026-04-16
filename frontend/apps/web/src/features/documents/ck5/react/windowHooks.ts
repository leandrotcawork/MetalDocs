import type { DecoupledEditor, ClassicEditor } from 'ckeditor5';

declare global {
  interface Window {
    __ck5?: {
      authorEditor?: DecoupledEditor;
      fillEditor?: ClassicEditor;
      save?: (html?: string) => Promise<void> | void;
    };
  }
}

export function installAuthorHook(editor: DecoupledEditor, save: (html?: string) => void): void {
  if (!import.meta.env.DEV) return;
  window.__ck5 = { ...(window.__ck5 ?? {}), authorEditor: editor, save };
}

export function installFillHook(editor: ClassicEditor, save: (html?: string) => void): void {
  if (!import.meta.env.DEV) return;
  window.__ck5 = { ...(window.__ck5 ?? {}), fillEditor: editor, save };
}

export function clearHooks(): void {
  if (!import.meta.env.DEV) return;
  delete window.__ck5;
}
