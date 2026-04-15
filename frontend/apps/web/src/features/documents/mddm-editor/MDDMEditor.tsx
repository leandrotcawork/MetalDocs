import { type PartialBlock } from "@blocknote/core";
import { BlockNoteView } from "@blocknote/mantine";
import {
  BasicTextStyleButton,
  BlockNoteViewEditor,
  BlockTypeSelect,
  ColorStyleButton,
  CreateLinkButton,
  FormattingToolbar,
  NestBlockButton,
  UnnestBlockButton,
  useCreateBlockNote,
} from "@blocknote/react";
import { MddmTextAlignButton } from "./toolbar/MddmTextAlignButton";
import { Plugin, PluginKey } from "@tiptap/pm/state";
import {
  useEffect,
  useMemo,
  useRef,
  useState,
  type CSSProperties,
} from "react";
import "@blocknote/core/fonts/inter.css";
import "@blocknote/mantine/style.css";
import "./mddm-editor-global.css";
import { getAttachmentDownloadURL, uploadAttachment } from "../../../api/documents";
import { mddmSchema } from "./schema";
import styles from "./MDDMEditor.module.css";
import { defaultLayoutTokens, tokensToCssVars } from "./engine/layout-ir";
import { setEditorTokens } from "./engine/editor-tokens";
import {
  mergePageRuntimeDimensions,
  type TemplatePageSettings,
} from "../../templates/page-settings";
import { computePageLayout, type PageLayout } from "./pagination";

export type MDDMTheme = {
  accent?: string;
  accentLight?: string;
  accentDark?: string;
  accentBorder?: string;
};

export type MDDMEditorProps = {
  initialContent?: PartialBlock[];
  onChange?: (blocks: unknown[]) => void;
  readOnly?: boolean;
  theme?: MDDMTheme;
  pageSettings?: TemplatePageSettings;
  onEditorReady?: (editor: unknown) => void;
  onSelectionChange?: (blockId: string | null) => void;
  documentId?: string;
};

const PX_PER_MM = 96 / 25.4;
const DEFAULT_PAGE_LAYOUT: PageLayout = {
  pageCount: 1,
  breakOffsetsByBlockId: {},
};

function parseCssLengthToPx(value: string | null | undefined): number | null {
  if (!value) return null;

  const trimmed = value.trim();
  if (!trimmed) return null;

  if (trimmed.endsWith("px")) {
    const numeric = Number.parseFloat(trimmed);
    return Number.isFinite(numeric) ? numeric : null;
  }

  if (trimmed.endsWith("mm")) {
    const numeric = Number.parseFloat(trimmed);
    return Number.isFinite(numeric) ? numeric * PX_PER_MM : null;
  }

  const numeric = Number.parseFloat(trimmed);
  return Number.isFinite(numeric) ? numeric : null;
}

function sameBreakOffsets(
  left: Record<string, number>,
  right: Record<string, number>,
): boolean {
  const leftKeys = Object.keys(left);
  const rightKeys = Object.keys(right);
  if (leftKeys.length !== rightKeys.length) return false;

  for (const key of leftKeys) {
    if (left[key] !== right[key]) return false;
  }

  return true;
}

export function MDDMEditor({
  initialContent,
  onChange,
  readOnly,
  theme,
  pageSettings,
  onEditorReady,
  onSelectionChange,
  documentId,
}: MDDMEditorProps) {
  const uploadFile = useMemo(() => {
    if (!documentId) return undefined;

    return async (file: File): Promise<string> => {
      const attachment = await uploadAttachment(documentId, file);
      return `/api/v1/documents/${documentId}/attachments/${attachment.attachmentId}/download-url`;
    };
  }, [documentId]);

  const resolveFileUrl = useMemo(() => {
    if (!documentId) return undefined;

    return async (url: string): Promise<string> => {
      const match = url.match(
        /^\/api\/v1\/documents\/([^/]+)\/attachments\/([^/]+)\/download-url$/,
      );
      if (!match) {
        return url;
      }

      const [, urlDocumentId, attachmentId] = match;
      if (urlDocumentId !== documentId) {
        return url;
      }

      try {
        const response = await getAttachmentDownloadURL(documentId, attachmentId);
        return response.downloadUrl || url;
      } catch {
        return url;
      }
    };
  }, [documentId]);

  const editor = useCreateBlockNote({
    schema: mddmSchema,
    initialContent: initialContent?.length ? initialContent : undefined,
    tables: {
      headers: true,
      cellBackgroundColor: true,
    },
    uploadFile,
    resolveFileUrl,
  });
  const editorRootRef = useRef<HTMLDivElement | null>(null);
  const recomputeLayoutRef = useRef<(() => void) | null>(null);
  const [pageLayout, setPageLayout] = useState<PageLayout>(DEFAULT_PAGE_LAYOUT);

  useEffect(() => {
    if (import.meta.env.DEV) {
      (window as any).__mddmEditor = editor;
      return () => { delete (window as any).__mddmEditor; };
    }
    return undefined;
  }, [editor]);

  const tokens = useMemo(() => {
    const page = pageSettings
      ? mergePageRuntimeDimensions(defaultLayoutTokens.page, pageSettings)
      : defaultLayoutTokens.page;

    if (!theme) {
      return {
        ...defaultLayoutTokens,
        page,
      };
    }

    return {
      ...defaultLayoutTokens,
      page,
      theme: {
        ...defaultLayoutTokens.theme,
        ...(theme.accent ? { accent: theme.accent } : {}),
        ...(theme.accentLight ? { accentLight: theme.accentLight } : {}),
        ...(theme.accentDark ? { accentDark: theme.accentDark } : {}),
        ...(theme.accentBorder ? { accentBorder: theme.accentBorder } : {}),
      },
    };
  }, [theme, pageSettings]);

  const cssVars = useMemo(() => tokensToCssVars(tokens), [tokens]);

  useEffect(() => {
    setEditorTokens(editor, tokens);
  }, [editor, tokens]);

  useEffect(() => {
    const tiptapRoot = (editor as any)?._tiptapEditor?.view?.dom;
    const paperElement = editorRootRef.current;
    if (!(tiptapRoot instanceof HTMLElement) || !(paperElement instanceof HTMLElement)) {
      setPageLayout(DEFAULT_PAGE_LAYOUT);
      recomputeLayoutRef.current = null;
      return undefined;
    }

    let frameId = 0;
    const recomputeLayout = () => {
      frameId = 0;

      const paperStyle = window.getComputedStyle(paperElement);
      const pageHeightPx =
        parseCssLengthToPx(paperStyle.getPropertyValue("--mddm-page-height")) ??
        tokens.page.heightMm * PX_PER_MM;
      const topMarginPx = Number.parseFloat(paperStyle.paddingTop) || 0;
      const bottomMarginPx = Number.parseFloat(paperStyle.paddingBottom) || 0;
      const contentOriginTop = paperElement.getBoundingClientRect().top + topMarginPx;

      const blockElements = Array.from(
        tiptapRoot.querySelectorAll<HTMLElement>(
          '.bn-block-content[data-id], .bn-block[data-id], .bn-block-outer[data-id], [data-id][data-content-type]',
        ),
      );

      const seenIds = new Set<string>();
      const blocks = blockElements
        .map((element) => {
          const id = element.dataset.id?.trim() ?? "";
          if (!id || seenIds.has(id)) return null;
          seenIds.add(id);

          const rect = element.getBoundingClientRect();
          const topPx = Math.max(0, rect.top - contentOriginTop);
          const heightPx = Math.max(0, rect.height);
          if (!Number.isFinite(topPx) || !Number.isFinite(heightPx)) return null;
          return {
            id,
            topPx,
            heightPx,
          };
        })
        .filter((block): block is { id: string; topPx: number; heightPx: number } => Boolean(block));

      const blocksForLayout =
        blocks.length > 0
          ? blocks
          : [
              {
                id: "__content__",
                topPx: 0,
                heightPx: Math.max(0, tiptapRoot.scrollHeight),
              },
            ];

      const nextLayout = computePageLayout({
        pageHeightPx,
        topMarginPx,
        bottomMarginPx,
        blocks: blocksForLayout,
      });

      setPageLayout((previous) => {
        if (
          previous.pageCount === nextLayout.pageCount &&
          sameBreakOffsets(previous.breakOffsetsByBlockId, nextLayout.breakOffsetsByBlockId)
        ) {
          return previous;
        }
        return nextLayout;
      });
    };

    const scheduleRecomputeLayout = () => {
      if (frameId !== 0) return;
      frameId = window.requestAnimationFrame(recomputeLayout);
    };

    recomputeLayoutRef.current = scheduleRecomputeLayout;
    scheduleRecomputeLayout();

    const mutationObserver = new MutationObserver(() => {
      scheduleRecomputeLayout();
    });
    mutationObserver.observe(tiptapRoot, {
      childList: true,
      subtree: true,
      characterData: true,
      attributes: true,
    });

    const resizeObserver =
      typeof ResizeObserver !== "undefined"
        ? new ResizeObserver(() => {
            scheduleRecomputeLayout();
          })
        : null;
    resizeObserver?.observe(tiptapRoot);
    resizeObserver?.observe(paperElement);
    window.addEventListener("resize", scheduleRecomputeLayout);

    return () => {
      if (frameId !== 0) {
        window.cancelAnimationFrame(frameId);
      }
      mutationObserver.disconnect();
      resizeObserver?.disconnect();
      window.removeEventListener("resize", scheduleRecomputeLayout);
      recomputeLayoutRef.current = null;
    };
  }, [editor, tokens.page.heightMm, tokens.page.marginTopMm, tokens.page.marginBottomMm]);

  useEffect(() => {
    const tiptapRoot = (editor as any)?._tiptapEditor?.view?.dom;
    if (!(tiptapRoot instanceof HTMLElement)) {
      return;
    }

    const elements = Array.from(
      tiptapRoot.querySelectorAll<HTMLElement>(".bn-block-content[data-id], .bn-block[data-id], .bn-block-outer[data-id], [data-id][data-content-type]"),
    );

    for (const element of elements) {
      const id = element.dataset.id?.trim() ?? "";
      if (!id) continue;

      const breakOffsetPx = pageLayout.breakOffsetsByBlockId[id];
      if (typeof breakOffsetPx === "number" && Number.isFinite(breakOffsetPx)) {
        element.dataset.mddmPageBreak = "true";
        element.style.setProperty("--mddm-page-break-offset", `${breakOffsetPx}px`);
      } else {
        delete element.dataset.mddmPageBreak;
        element.style.removeProperty("--mddm-page-break-offset");
      }
    }
  }, [editor, pageLayout.breakOffsetsByBlockId]);

  useEffect(() => {
    const root = (editor as any)?._tiptapEditor?.view?.dom;
    if (!(root instanceof HTMLElement)) {
      return undefined;
    }

    const lockHeaders = () => {
      // Lock th cells (produced by headerRows / headerCols on native tables).
      root.querySelectorAll("th").forEach((cell) => {
        (cell as HTMLElement).contentEditable = "false";
      });
      // Lock any td/th that carries data-background-color — BlockNote's DOM
      // attribute for tableCell / tableHeader backgroundColor prop.
      // fieldGroupToTable marks label cells with backgroundColor:"gray" which
      // renders as data-background-color="gray" in the DOM. This is the
      // template's explicit signal: "this cell is a header, not editable."
      // No column-position assumptions; works for any layout, any column count.
      root.querySelectorAll("td[data-background-color], th[data-background-color]").forEach((cell) => {
        (cell as HTMLElement).contentEditable = "false";
      });
    };

    lockHeaders();

    const observer = new MutationObserver(() => {
      lockHeaders();
    });
    observer.observe(root, { childList: true, subtree: true, attributes: true, attributeFilter: ["data-background-color"] });

    return () => {
      observer.disconnect();
    };
  }, [editor]);

  // Prevent deletion of any block whose ProseMirror node has locked=true in its
  // attrs. This is the universal lock — the template definition sets locked on
  // seed blocks; user-added blocks default to locked=false and remain deletable.
  // filterTransaction runs synchronously before every transaction is applied;
  // if the count of locked nodes would decrease, the transaction is rejected.
  useEffect(() => {
    const tiptap = (editor as any)?._tiptapEditor;
    if (!tiptap || typeof tiptap.registerPlugin !== "function") return;

    const pluginKey = new PluginKey("mddm-locked-blocks");
    const plugin = new Plugin({
      key: pluginKey,
      filterTransaction(tr, state) {
        if (!tr.docChanged) return true;

        const isLocked = (node: any): boolean => Boolean(node.attrs?.locked);

        let before = 0;
        let after = 0;
        state.doc.descendants((node: any) => { if (isLocked(node)) before++; });
        tr.doc.descendants((node: any) => { if (isLocked(node)) after++; });

        return after >= before;
      },
    });

    tiptap.registerPlugin(plugin);
    return () => { tiptap.unregisterPlugin(pluginKey); };
  }, [editor]);

  useEffect(() => {
    onEditorReady?.(editor);
  }, [editor, onEditorReady]);

  // Subscribe to Tiptap's selectionUpdate event and report the BlockNote block
  // that contains the cursor. Fires onSelectionChange(blockId) on selection moves
  // and onSelectionChange(null) when the editor loses its selection.
  useEffect(() => {
    if (!onSelectionChange) return undefined;

    const tiptap = (editor as any)?._tiptapEditor;
    if (!tiptap) return undefined;

    const handler = () => {
      const { from } = tiptap.state.selection;
      // resolvePos lets us walk the ProseMirror node tree to find the BlockNote block id.
      try {
        const resolvedPos = tiptap.state.doc.resolve(from);
        // Walk up to find a node with a blockId attribute (BlockNote block nodes carry `id`).
        for (let depth = resolvedPos.depth; depth >= 0; depth--) {
          const node = resolvedPos.node(depth);
          const blockId: string | undefined = node.attrs?.id;
          if (blockId) {
            onSelectionChange(blockId);
            return;
          }
        }
      } catch {
        // ignore resolve errors (e.g. out-of-range positions)
      }
      onSelectionChange(null);
    };

    tiptap.on("selectionUpdate", handler);
    return () => {
      tiptap.off("selectionUpdate", handler);
    };
  }, [editor, onSelectionChange]);

  // Place cursor in first inline-editable block on mount so toolbar items
  // have a ProseMirror selection and render immediately.
  useEffect(() => {
    if (readOnly) return;

    function findFirstInlineBlock(
      blocks: (typeof editor.document),
    ): (typeof editor.document)[number] | undefined {
      for (const block of blocks) {
        if (Array.isArray(block.content)) {
          return block;
        }
        if (block.children.length > 0) {
          const found = findFirstInlineBlock(block.children);
          if (found) return found;
        }
      }
      return undefined;
    }

    const firstBlock = findFirstInlineBlock(editor.document);
    if (firstBlock) {
      editor.setTextCursorPosition(firstBlock, "start");
    } else {
      editor.focus();
    }
  }, [editor, readOnly]);

  const visualPageCount = Math.max(1, pageLayout.pageCount);
  const visualPageHeightPx = tokens.page.heightMm * PX_PER_MM;
  const visualStackHeightPx = visualPageCount * visualPageHeightPx;

  const paperStyle = useMemo<CSSProperties>(
    () => ({
      ...(cssVars as CSSProperties),
      minHeight: `${Math.max(visualPageHeightPx, visualStackHeightPx)}px`,
    }),
    [cssVars, visualPageHeightPx, visualStackHeightPx],
  );

  return (
    <div className={styles.pageShell} data-testid="mddm-editor-root">
      <BlockNoteView
        editor={editor}
        editable={!readOnly}
        formattingToolbar={false}
        tableHandles={false}
        renderEditor={false}
        onChange={(currentEditor) => {
          onChange?.(currentEditor.document);
          recomputeLayoutRef.current?.();
        }}
      >
        <div className={styles.scrollShell} data-testid="mddm-editor-scroll-shell">
          {!readOnly && (
            <div
              className={styles.toolbarWrapper}
              data-testid="mddm-editor-toolbar"
              onMouseDown={(e) => {
                // Prevent toolbar buttons from stealing DOM focus from the editor.
                // Without this, clicking a button moves activeElement to the <button>,
                // which causes the cursor to vanish from native table cells (nested
                // contenteditable contexts) even though ProseMirror's internal
                // selection is preserved. This is the standard contenteditable toolbar
                // pattern — click events still fire normally.
                e.preventDefault();
              }}
            >
              <FormattingToolbar>
                <BlockTypeSelect key="blockTypeSelect" />
                <BasicTextStyleButton basicTextStyle="bold" key="boldStyleButton" />
                <BasicTextStyleButton basicTextStyle="italic" key="italicStyleButton" />
                <BasicTextStyleButton basicTextStyle="underline" key="underlineStyleButton" />
                <BasicTextStyleButton basicTextStyle="strike" key="strikeStyleButton" />
                <MddmTextAlignButton textAlignment="left" key="textAlignLeftButton" />
                <MddmTextAlignButton textAlignment="center" key="textAlignCenterButton" />
                <MddmTextAlignButton textAlignment="right" key="textAlignRightButton" />
                <ColorStyleButton key="colorStyleButton" />
                <NestBlockButton key="nestBlockButton" />
                <UnnestBlockButton key="unnestBlockButton" />
                <CreateLinkButton key="createLinkButton" />
              </FormattingToolbar>
            </div>
          )}
          <div className={styles.pageStack} data-testid="mddm-editor-page-stack">
            <div className={styles.surfaceStack} aria-hidden="true">
              {Array.from({ length: visualPageCount }).map((_, pageIndex) => (
                <div
                  key={`surface-${pageIndex}`}
                  className={styles.paperSurface}
                  data-testid="mddm-editor-paper-surface"
                />
              ))}
            </div>
            <div className={styles.editorLayer}>
            <div
              ref={editorRootRef}
              className={styles.editorRoot}
              style={paperStyle}
              data-editable={!readOnly}
              data-mddm-editor-root="true"
              data-testid="mddm-editor-paper"
              data-page-count={visualPageCount}
            >
              <BlockNoteViewEditor />
            </div>
            </div>
          </div>
        </div>
      </BlockNoteView>
    </div>
  );
}
