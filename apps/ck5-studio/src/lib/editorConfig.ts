import * as CKEditor from "ckeditor5";

const {
  Alignment,
  AutoImage,
  Autoformat,
  Base64UploadAdapter,
  BlockQuote,
  Bold,
  DecoupledEditor,
  Essentials,
  FontBackgroundColor,
  FontColor,
  FontFamily,
  FontSize,
  Heading,
  Image,
  ImageCaption,
  ImageInsert,
  ImageResize,
  ImageStyle,
  ImageToolbar,
  ImageUpload,
  Italic,
  Link,
  List,
  Paragraph,
  RestrictedEditingMode,
  StandardEditingMode,
  Table,
  TableToolbar,
  Underline,
} = CKEditor as Record<string, any>;

export const editorClass = DecoupledEditor as any;

export type EditingTemplateMode = "author" | "fill";

const BASE_PLUGINS = [
  Essentials,
  Paragraph,
  Heading,
  Bold,
  Italic,
  Underline,
  Link,
  List,
  Table,
  TableToolbar,
  Image,
  ImageUpload,
  ImageToolbar,
  ImageStyle,
  ImageResize,
  ImageCaption,
  ImageInsert,
  AutoImage,
  Base64UploadAdapter,
  Autoformat,
  Alignment,
  FontFamily,
  FontSize,
  FontColor,
  FontBackgroundColor,
  BlockQuote,
];

const BASE_TOOLBAR_ITEMS = [
  "undo",
  "redo",
  "|",
  "heading",
  "|",
  "fontFamily",
  "fontSize",
  "fontColor",
  "fontBackgroundColor",
  "|",
  "bold",
  "italic",
  "underline",
  "link",
  "|",
  "alignment",
  "bulletedList",
  "numberedList",
  "blockQuote",
  "insertTable",
  "uploadImage",
];

export function getEditorConfig(mode: EditingTemplateMode) {
  const plugins = [...BASE_PLUGINS];
  const toolbarItems = [...BASE_TOOLBAR_ITEMS];

  if (mode === "author" && StandardEditingMode) {
    plugins.push(StandardEditingMode);
    toolbarItems.splice(2, 0, "restrictedEditingException:dropdown", "|");
  }

  if (mode === "fill" && RestrictedEditingMode) {
    plugins.push(RestrictedEditingMode);
    toolbarItems.splice(2, 0, "restrictedEditing", "|");
  }

  return {
    licenseKey: "GPL",
    plugins,
    toolbar: {
      items: toolbarItems,
      shouldNotGroupWhenFull: true,
    },
    restrictedEditing:
      mode === "fill"
        ? {
            allowedCommands: ["bold", "italic", "link"],
            allowedAttributes: ["bold", "italic", "linkHref"],
          }
        : undefined,
  };
}
