import { useEffect, useMemo } from "react";
import type { Editor } from "@tiptap/react";
import { Color } from "@tiptap/extension-color";
import Image from "@tiptap/extension-image";
import Table from "@tiptap/extension-table";
import TableCell from "@tiptap/extension-table-cell";
import TableHeader from "@tiptap/extension-table-header";
import TableRow from "@tiptap/extension-table-row";
import { EditorContent, useEditor } from "@tiptap/react";
import StarterKit from "@tiptap/starter-kit";
import { TextStyle } from "@tiptap/extension-text-style";
import type { RuntimeRichField } from "../schemaRuntimeTypes";
import styles from "../RichField.module.css";

type RuntimeMode = "edit" | "preview";

type RichFieldProps = {
  field: RuntimeRichField;
  value: unknown;
  mode: RuntimeMode;
  onChange?: (next: unknown) => void;
};

const EMPTY_HTML = "<p></p>";
const RICH_PREVIEW_STRIP_TAGS = new Set(["script", "style", "iframe", "object", "embed", "svg", "math"]);
const RICH_PREVIEW_ALLOWED_TAGS = new Set([
  "a",
  "b",
  "blockquote",
  "br",
  "code",
  "col",
  "colgroup",
  "em",
  "h1",
  "h2",
  "h3",
  "h4",
  "h5",
  "h6",
  "hr",
  "img",
  "li",
  "ol",
  "p",
  "pre",
  "s",
  "span",
  "strong",
  "sub",
  "sup",
  "table",
  "tbody",
  "td",
  "th",
  "thead",
  "tr",
  "u",
  "ul",
]);
const RICH_PREVIEW_ALLOWED_ATTRIBUTES: Partial<Record<string, Set<string>>> = {
  a: new Set(["href", "rel", "target", "title", "style"]),
  col: new Set(["span", "style"]),
  img: new Set(["alt", "height", "loading", "src", "title", "width", "style"]),
  table: new Set(["style"]),
  td: new Set(["colspan", "colwidth", "rowspan", "style"]),
  th: new Set(["colspan", "colwidth", "rowspan", "style"]),
  span: new Set(["style"]),
};
const RICH_PREVIEW_ALLOWED_STYLE_PROPERTIES = new Set([
  "background-color",
  "color",
  "font-family",
  "font-size",
  "font-style",
  "font-weight",
  "min-width",
  "text-align",
  "text-decoration",
  "text-decoration-line",
  "width",
]);

export function RichField({ field, value, mode, onChange }: RichFieldProps) {
  const content = useMemo(() => normalizeRichValue(value), [value]);

  if (mode === "preview") {
    return (
      <div className={styles.richRoot}>
        <div className={styles.editorShell}>
          <div className={styles.previewBody} dangerouslySetInnerHTML={{ __html: sanitizeRichPreviewHtml(content || EMPTY_HTML) }} />
        </div>
      </div>
    );
  }

  return <RichEditor field={field} value={content} onChange={onChange} />;
}

function RichEditor({ value, onChange }: Pick<RichFieldProps, "value" | "onChange"> & { field: RuntimeRichField }) {
  const editor = useEditor({
    extensions: [
      StarterKit,
      TextStyle,
      Color,
      Image.configure({ inline: true }),
      Table.configure({ resizable: true }),
      TableRow,
      TableHeader,
      TableCell,
    ],
    content: normalizeRichValue(value),
    editorProps: {
      attributes: {
        class: "tiptap-editor",
      },
    },
    onUpdate({ editor: tiptapEditor }) {
      onChange?.(tiptapEditor.getHTML());
    },
  });

  useEffect(() => {
    if (!editor) return;
    const nextValue = normalizeRichValue(value);
    if (editor.getHTML() !== nextValue) {
      editor.commands.setContent(nextValue, false);
    }
  }, [editor, value]);

  return (
    <div className={styles.richRoot}>
      <div className={styles.editorShell}>
        <div className={styles.toolbar}>
          <div className={styles.toolbarGroup}>
            <ToolbarButton editor={editor} command="bold">
              B
            </ToolbarButton>
            <ToolbarButton editor={editor} command="italic">
              I
            </ToolbarButton>
            <ToolbarButton editor={editor} command="strike">
              S
            </ToolbarButton>
          </div>
          <div className={styles.toolbarGroup}>
            <ToolbarButton editor={editor} command="paragraph">
              Paragrafo
            </ToolbarButton>
            <ToolbarButton editor={editor} command="heading" level={1}>
              H1
            </ToolbarButton>
            <ToolbarButton editor={editor} command="heading" level={2}>
              H2
            </ToolbarButton>
            <ToolbarButton editor={editor} command="heading" level={3}>
              H3
            </ToolbarButton>
          </div>
          <div className={styles.toolbarGroup}>
            <ToolbarButton editor={editor} command="bulletList">
              Lista
            </ToolbarButton>
            <ToolbarButton editor={editor} command="orderedList">
              Numerada
            </ToolbarButton>
            <ToolbarButton editor={editor} command="blockquote">
              Citar
            </ToolbarButton>
          </div>
          <div className={styles.toolbarGroup}>
            <ToolbarButton editor={editor} command="insertTable">
              Tabela
            </ToolbarButton>
            <ToolbarButton editor={editor} command="image">
              Imagem
            </ToolbarButton>
            <ToolbarButton editor={editor} command="undo">
              Undo
            </ToolbarButton>
            <ToolbarButton editor={editor} command="redo">
              Redo
            </ToolbarButton>
          </div>
          <div className={styles.toolbarGroup}>
            <input
              className={styles.colorInput}
              type="color"
              aria-label="Cor do texto"
              onChange={(event) => {
                if (!editor) return;
                editor.chain().focus().setColor(event.target.value).run();
              }}
            />
            <ToolbarButton editor={editor} command="clearColor">
              Cor padrao
            </ToolbarButton>
          </div>
        </div>
        <div className={styles.editorBody}>
          <EditorContent editor={editor} />
        </div>
      </div>
    </div>
  );
}

type ToolbarCommand =
  | "bold"
  | "italic"
  | "strike"
  | "paragraph"
  | "heading"
  | "bulletList"
  | "orderedList"
  | "blockquote"
  | "insertTable"
  | "image"
  | "undo"
  | "redo"
  | "clearColor";

type ToolbarButtonProps = {
  editor: Editor | null;
  command: ToolbarCommand;
  level?: 1 | 2 | 3 | 4 | 5 | 6;
  children: string;
};

function ToolbarButton({ editor, command, level, children }: ToolbarButtonProps) {
  if (!editor) {
    return (
      <button type="button" className={styles.toolbarButton} disabled>
        {children}
      </button>
    );
  }

  const active =
    (command === "bold" && editor.isActive("bold")) ||
    (command === "italic" && editor.isActive("italic")) ||
    (command === "strike" && editor.isActive("strike")) ||
    (command === "paragraph" && editor.isActive("paragraph")) ||
    (command === "heading" && level ? editor.isActive("heading", { level }) : false) ||
    (command === "bulletList" && editor.isActive("bulletList")) ||
    (command === "orderedList" && editor.isActive("orderedList")) ||
    (command === "blockquote" && editor.isActive("blockquote"));

  return (
    <button
      type="button"
      className={`${styles.toolbarButton} ${active ? styles.toolbarButtonActive : ""}`}
      onMouseDown={(event) => event.preventDefault()}
      onClick={() => {
        if (!editor) return;
        switch (command) {
          case "bold":
            editor.chain().focus().toggleBold().run();
            break;
          case "italic":
            editor.chain().focus().toggleItalic().run();
            break;
          case "strike":
            editor.chain().focus().toggleStrike().run();
            break;
          case "paragraph":
            editor.chain().focus().setParagraph().run();
            break;
          case "heading":
            editor.chain().focus().toggleHeading({ level: level ?? 1 }).run();
            break;
          case "bulletList":
            editor.chain().focus().toggleBulletList().run();
            break;
          case "orderedList":
            editor.chain().focus().toggleOrderedList().run();
            break;
          case "blockquote":
            editor.chain().focus().toggleBlockquote().run();
            break;
          case "insertTable":
            editor.chain().focus().insertTable({ rows: 2, cols: 2, withHeaderRow: true }).run();
            break;
          case "image": {
            const src = window.prompt("URL da imagem");
            if (src) {
              editor.chain().focus().setImage({ src }).run();
            }
            break;
          }
          case "undo":
            editor.chain().focus().undo().run();
            break;
          case "redo":
            editor.chain().focus().redo().run();
            break;
          case "clearColor":
            editor.chain().focus().unsetColor().run();
            break;
        }
      }}
    >
      {children}
    </button>
  );
}

function normalizeRichValue(value: unknown) {
  if (typeof value === "string") {
    return value.trim() ? value : EMPTY_HTML;
  }

  if (value && typeof value === "object" && !Array.isArray(value)) {
    const richRecord = value as Record<string, unknown>;
    if (typeof richRecord.html === "string") {
      return richRecord.html;
    }
    if (typeof richRecord.content === "string") {
      return richRecord.content;
    }
  }

  return EMPTY_HTML;
}

function sanitizeRichPreviewHtml(html: string) {
  const template = document.createElement("template");
  template.innerHTML = html;
  sanitizePreviewNode(template.content);
  return template.innerHTML;
}

function sanitizePreviewNode(node: ChildNode | DocumentFragment) {
  for (const child of Array.from(node.childNodes)) {
    if (child.nodeType === Node.TEXT_NODE) {
      continue;
    }

    if (child.nodeType !== Node.ELEMENT_NODE) {
      child.remove();
      continue;
    }

    const element = child as HTMLElement;
    const tagName = element.tagName.toLowerCase();

    if (RICH_PREVIEW_STRIP_TAGS.has(tagName)) {
      element.remove();
      continue;
    }

    sanitizePreviewNode(element);

    if (!RICH_PREVIEW_ALLOWED_TAGS.has(tagName)) {
      unwrapPreviewElement(element);
      continue;
    }

    sanitizePreviewAttributes(element);
  }
}

function sanitizePreviewAttributes(element: HTMLElement) {
  const tagName = element.tagName.toLowerCase();
  const allowedAttributes = RICH_PREVIEW_ALLOWED_ATTRIBUTES[tagName] ?? new Set<string>();

  for (const { name, value } of Array.from(element.attributes)) {
    if (name.startsWith("on")) {
      element.removeAttribute(name);
      continue;
    }

    if (!allowedAttributes.has(name)) {
      element.removeAttribute(name);
      continue;
    }

    if (name === "href" || name === "src") {
      if (!isSafeUrl(value, name === "src")) {
        element.removeAttribute(name);
      }
      continue;
    }

    if (name === "style") {
      const sanitizedStyle = sanitizeStyle(value, tagName);
      if (sanitizedStyle) {
        element.setAttribute(name, sanitizedStyle);
      } else {
        element.removeAttribute(name);
      }
    }
  }

  if (tagName === "a" && element.getAttribute("target") === "_blank" && !element.getAttribute("rel")) {
    element.setAttribute("rel", "noopener noreferrer");
  }
}

function sanitizeStyle(styleValue: string, tagName: string) {
  const probe = document.createElement("div");
  probe.setAttribute("style", styleValue);

  const allowedProperties = RICH_PREVIEW_ALLOWED_STYLE_PROPERTIES;
  if (tagName === "col") {
    return keepAllowedStyleDeclarations(probe, allowedProperties);
  }

  if (tagName === "table" || tagName === "span" || tagName === "a" || tagName === "img" || tagName === "td" || tagName === "th") {
    return keepAllowedStyleDeclarations(probe, allowedProperties);
  }

  return "";
}

function keepAllowedStyleDeclarations(element: HTMLElement, allowedProperties: Set<string>) {
  const declarations = element.getAttribute("style")?.split(";") ?? [];
  const kept = declarations
    .map((declaration) => declaration.trim())
    .filter(Boolean)
    .filter((declaration) => {
      const separatorIndex = declaration.indexOf(":");
      if (separatorIndex === -1) return false;
      const propertyName = declaration.slice(0, separatorIndex).trim().toLowerCase();
      return allowedProperties.has(propertyName);
    });

  return kept.join("; ");
}

function isSafeUrl(value: string, allowDataImage: boolean) {
  const trimmed = value.trim();
  if (!trimmed) return false;
  if (trimmed.startsWith("#") || trimmed.startsWith("/")) return true;
  if (/^https?:\/\//i.test(trimmed)) return true;
  if (allowDataImage && /^data:image\//i.test(trimmed)) return true;
  if (/^blob:/i.test(trimmed)) return true;

  return false;
}

function unwrapPreviewElement(element: HTMLElement) {
  const parent = element.parentNode;
  if (!parent) return;

  while (element.firstChild) {
    parent.insertBefore(element.firstChild, element);
  }

  element.remove();
}
