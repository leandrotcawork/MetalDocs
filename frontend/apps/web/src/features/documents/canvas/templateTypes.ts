export type CanvasTemplateFieldKind = "scalar" | "rich" | "table" | "repeat";

export type CanvasTemplateNode =
  | { type: "page"; id: string; children: CanvasTemplateNode[] }
  | { type: "section-frame"; id: string; title?: string; children: CanvasTemplateNode[] }
  | { type: "label"; id: string; text: string }
  | { type: "field-slot"; id: string; path: string; fieldKind: "scalar" }
  | { type: "rich-slot"; id: string; path: string; fieldKind: "rich" }
  | { type: "table-slot"; id: string; path: string; fieldKind: "table" }
  | { type: "repeat-slot"; id: string; path: string; fieldKind: "repeat" };

export type CanvasTemplatePage = Extract<CanvasTemplateNode, { type: "page" }>;
