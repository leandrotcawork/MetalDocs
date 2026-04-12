import { defaultLayoutTokens, type LayoutTokens } from "./layout-ir";

const TOKEN_KEY = "__mddmTokens";

type EditorWithTokens = {
  [TOKEN_KEY]?: LayoutTokens;
};

export function setEditorTokens(editor: object, tokens: LayoutTokens): void {
  (editor as EditorWithTokens)[TOKEN_KEY] = tokens;
}

export function getEditorTokens(editor: object | null | undefined): LayoutTokens {
  return (editor as EditorWithTokens | null | undefined)?.[TOKEN_KEY] ?? defaultLayoutTokens;
}
