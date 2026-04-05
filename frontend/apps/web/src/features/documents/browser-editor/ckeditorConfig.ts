import {
  Bold,
  DecoupledEditor,
  Essentials,
  Heading,
  Italic,
  List,
  Paragraph,
  RestrictedEditingMode,
  Table,
  TableToolbar,
  type EditorConfig,
} from "ckeditor5";

const browserEditorLicenseKey = import.meta.env.VITE_CKEDITOR_LICENSE_KEY || "GPL";

export const browserDocumentEditorClass = DecoupledEditor;

export function buildBrowserDocumentEditorConfig(initialData: string): EditorConfig {
  return {
    licenseKey: browserEditorLicenseKey,
    plugins: [Essentials, Paragraph, Heading, Bold, Italic, List, Table, TableToolbar, RestrictedEditingMode],
    toolbar: ["heading", "|", "bold", "italic", "bulletedList", "numberedList", "insertTable", "|", "undo", "redo"],
    initialData,
    placeholder: "Comece a redigir o documento.",
  };
}
