import { Bold, DecoupledEditor, Essentials, GeneralHtmlSupport, Heading, Italic, List, Paragraph, RestrictedEditingMode, Table, TableToolbar, type EditorConfig } from "ckeditor5";

const browserEditorLicenseKey = import.meta.env.VITE_CKEDITOR_LICENSE_KEY || "GPL";

export const browserDocumentEditorClass = DecoupledEditor;

export const browserDocumentEditorConfig: EditorConfig = {
  licenseKey: browserEditorLicenseKey,
  plugins: [Essentials, Paragraph, Heading, Bold, Italic, List, Table, TableToolbar, RestrictedEditingMode, GeneralHtmlSupport],
  toolbar: ["heading", "|", "bold", "italic", "bulletedList", "numberedList", "insertTable", "|", "undo", "redo"],
  placeholder: "Comece a redigir o documento.",
  htmlSupport: {
    allow: [
      {
        name: /^(table|tbody|tr|td|th|thead)$/,
        styles: true,
        classes: true,
        attributes: true,
      },
    ],
  },
};
