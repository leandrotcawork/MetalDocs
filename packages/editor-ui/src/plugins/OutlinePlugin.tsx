import { useEffect, useState } from 'react';
import type { EditorPlugin, PluginPanelProps } from '@eigenpal/docx-js-editor';

type OutlineHeading = {
  id: string;
  level: number;
  text: string;
  pos: number;
};

type OutlineState = {
  headings: OutlineHeading[];
  activeId: string | null;
};

function OutlinePanel(props: PluginPanelProps<OutlineState>) {
  const [activeId, setActiveId] = useState<string | null>(null);
  const headings = props.pluginState?.headings ?? [];

  useEffect(() => {
    const ctx = props.renderedDomContext;
    if (!ctx) {
      setActiveId((prev) => (prev === null ? prev : null));
      return;
    }

    let raf = 0;
    const tick = () => {
      raf = 0;
      if (headings.length === 0) {
        setActiveId((prev) => (prev === null ? prev : null));
      } else {
        const targetY = ctx.pagesContainer.scrollTop + 80;
        let bestId: string | null = null;
        let bestDistance = Number.POSITIVE_INFINITY;
        for (const heading of headings) {
          const coords = ctx.getCoordinatesForPosition(heading.pos);
          if (!coords) continue;
          const distance = Math.abs(coords.y - targetY);
          if (distance < bestDistance) {
            bestDistance = distance;
            bestId = heading.id;
          }
        }
        setActiveId((prev) => (prev === bestId ? prev : bestId));
      }
    };

    const scheduleTick = () => {
      if (raf !== 0) return;
      raf = window.requestAnimationFrame(tick);
    };

    ctx.pagesContainer.addEventListener('scroll', scheduleTick);
    scheduleTick();

    return () => {
      ctx.pagesContainer.removeEventListener('scroll', scheduleTick);
      if (raf !== 0) window.cancelAnimationFrame(raf);
    };
  }, [props.renderedDomContext, headings]);

  return (
    <div className="outline-panel">
      {headings.length === 0 ? (
        <div style={{ padding: 8, color: '#6b7280', fontSize: 12 }}>(no headings)</div>
      ) : (
        headings.map((heading) => (
          <button
            key={heading.id}
            type="button"
            className={`outline-item${activeId === heading.id ? ' outline-item-active' : ''}`}
            style={{ paddingLeft: `${Math.max(0, heading.level - 1) * 12}px` }}
            onClick={() => props.scrollToPosition(heading.pos)}
          >
            {heading.text}
          </button>
        ))
      )}
    </div>
  );
}

export function createOutlinePlugin(): EditorPlugin<OutlineState> {
  let cachedDoc: unknown = null;

  return {
    id: 'outline',
    name: 'Outline',
    panelConfig: {
      position: 'left',
      defaultSize: 260,
      minSize: 220,
      maxSize: 400,
      resizable: true,
      collapsible: true,
    },
    initialize: () => ({ headings: [], activeId: null }),
    onStateChange(view) {
      if (view.state.doc === cachedDoc) return;
      cachedDoc = view.state.doc;

      const headings: OutlineHeading[] = [];

      type ParagraphLike = {
        type?: { name?: string };
        attrs?: { outlineLevel?: unknown; styleId?: unknown };
        textContent?: string;
      };

      view.state.doc.descendants((rawNode: unknown, pos: number) => {
        const node = rawNode as ParagraphLike;
        if (node.type?.name !== 'paragraph') return;
        const outline = node.attrs?.outlineLevel;
        let level: number | null = null;
        if (outline != null) {
          level = typeof outline === 'number' ? outline + 1 : 1;
        } else {
          const styleId = String(node.attrs?.styleId ?? '');
          const styleMatch = styleId.match(/^(T[íi]tulo|Heading)(\d+)$/i);
          if (styleMatch) {
            const parsedLevel = Number.parseInt(styleMatch[2], 10);
            if (Number.isFinite(parsedLevel)) {
              level = Math.max(1, Math.min(6, parsedLevel));
            }
          }
        }
        if (level == null) return;
        const text = (node.textContent ?? '').trim() || 'Untitled heading';
        headings.push({ id: String(headings.length), level, text, pos });
      });

      return { headings, activeId: null };
    },
    Panel: OutlinePanel,
    styles: `
      .outline-panel { height: 100%; overflow-y: auto; box-sizing: border-box; padding: 8px; }
      .outline-item { display: block; width: 100%; border: 0; background: transparent; text-align: left; font-size: 13px; line-height: 1.4; padding-top: 6px; padding-bottom: 6px; border-radius: 6px; cursor: pointer; color: #1f2937; }
      .outline-item:hover { background: #f3f4f6; }
      .outline-item-active { background: #e8f0fe; color: #1d4ed8; font-weight: 500; }
    `,
  };
}
