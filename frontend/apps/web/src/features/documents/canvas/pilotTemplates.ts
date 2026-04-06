import type { DocumentTemplateSnapshotItem } from "../../../lib.types";
import type { CanvasTemplatePage } from "./templateTypes";

export const PO_GOVERNED_CANVAS_TEMPLATE = {
  templateKey: "po-default-canvas",
  version: 1,
  profileCode: "po",
  schemaVersion: 3,
  definition: {
    type: "page",
    id: "po-root",
    children: [
      {
        type: "section-frame",
        id: "identificacao-processo",
        title: "Identificacao do Processo",
        children: [
          { type: "label", id: "lbl-objetivo", text: "Objetivo" },
          { type: "field-slot", id: "slot-objetivo", path: "identificacaoProcesso.objetivo", fieldKind: "scalar" },
          { type: "label", id: "lbl-descricao", text: "Descricao do processo" },
          { type: "rich-slot", id: "slot-descricao", path: "visaoGeral.descricaoProcesso", fieldKind: "rich" },
        ],
      },
    ],
  },
} as const satisfies DocumentTemplateSnapshotItem;

export const PO_GOVERNED_CANVAS_TEMPLATE_DEFINITION = PO_GOVERNED_CANVAS_TEMPLATE.definition as unknown as CanvasTemplatePage;

