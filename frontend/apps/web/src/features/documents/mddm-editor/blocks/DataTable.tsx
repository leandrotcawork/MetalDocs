import { Node as TiptapNode } from "@tiptap/core";
import { propsToAttributes } from "@blocknote/core";
import { createReactBlockSpec } from "@blocknote/react";
import styles from "./DataTable.module.css";
import { DataTableExternalHTML } from "../engine/external-html";
import { getEditorTokens } from "../engine/editor-tokens";

const dataTablePropSchema = {
  label: { default: "" },
  locked: { default: false },
  density: { default: "normal" },
  __template_block_id: { default: "" },
  styleJson: { default: "{}" },
  capabilitiesJson: { default: "{}" },
};

// Proper ProseMirror / Tiptap node for the dataTable block.
//
// createReactBlockSpec only accepts content:"inline"|"none" at the TypeScript
// level, so we cannot use it to declare a table-content block. Instead we build
// the PM node explicitly here and later splice it into the spec returned by
// createReactBlockSpec. BlockNoteSchema.create() uses implementation.node when
// it is present (see addNodeAndExtensionsToSpec), so this completely replaces
// the "content:''" node that createReactBlockSpec would otherwise produce.
//
// content:"tableRow+" mirrors the built-in BlockNote table block, which is
// required so that blockToNode / nodeToBlock can round-trip tableContent data
// without calling createChecked() on an empty-content node.
const _dataTablePMNode = TiptapNode.create({
  name: "dataTable",
  content: "tableRow+",
  group: "blockContent",
  // tableRole tells prosemirror-tables that this is a table-like node so that
  // cell-selection and column-resizing plugins apply to it.
  tableRole: "table",
  selectable: true,
  isolating: true,
  defining: true,
  addAttributes() {
    return propsToAttributes(dataTablePropSchema);
  },
  parseHTML() {
    return [{ tag: "div[data-content-type='dataTable']" }];
  },
  addNodeView() {
    return ({ node }) => {
      let currentNode = node;

      const dom = document.createElement("div");
      dom.dataset.mddmBlock = "dataTable";

      const header = document.createElement("div");
      header.className = "mddm-dt-header";

      const label = document.createElement("strong");
      label.className = "mddm-dt-label";
      header.appendChild(label);

      const container = document.createElement("div");
      container.className = "mddm-dt-container";

      const table = document.createElement("table");
      table.className = "mddm-dt-table";
      container.appendChild(table);

      dom.append(header, container);

      const syncAttrs = (nextNode: typeof node) => {
        dom.dataset.density = nextNode.attrs.density || "normal";
        dom.dataset.locked = String(nextNode.attrs.locked);
        label.textContent = nextNode.attrs.label || "Data Table";
      };

      syncAttrs(currentNode);

      return {
        dom,
        contentDOM: table,
        update(updatedNode) {
          if (updatedNode.type !== currentNode.type) {
            return false;
          }

          currentNode = updatedNode;
          syncAttrs(currentNode);
          return true;
        },
      };
    };
  },
  renderHTML({ HTMLAttributes }) {
    return [
      "div",
      {
        "data-content-type": "dataTable",
        class: "bn-block-content",
        ...HTMLAttributes,
      },
      0,
    ];
  },
});

const _dataTableSpec = createReactBlockSpec(
  {
    type: "dataTable" as const,
    propSchema: dataTablePropSchema,
    // TypeScript-level declaration is "none"; the actual PM node registered
    // below has content:"tableRow+" — see _dataTablePMNode above.
    content: "none" as "none",
  },
  {
    render: (props) => (
      <div
        className={styles.dataTable}
        data-mddm-block="dataTable"
        data-density={props.block.props.density || "normal"}
        data-locked={String(props.block.props.locked)}
      >
        <div className={styles.dataTableHeader}>
          <strong className={styles.tableLabel}>
            {props.block.props.label || "Data Table"}
          </strong>
        </div>
        <div className={styles.tableContainer} ref={(props as any).contentRef} />
      </div>
    ),
    toExternalHTML: (props) => (
      <DataTableExternalHTML
        tokens={getEditorTokens(props.editor)}
        label={props.block.props.label as string}
        tableContent={props.block.content}
      />
    ),
  },
);

// Re-export with two patches applied so BlockNote treats dataTable as a proper
// table-content block:
//
//   spec.config.content  → "table"  (BlockNote API-level content type)
//   spec.implementation.node → _dataTablePMNode  (PM node with tableRow+ content)
//
// BlockNoteSchema.create() calls addNodeAndExtensionsToSpec(config, implementation),
// which picks up implementation.node when present and skips creating a new PM node
// from blockConfig.content. This ensures the registered PM node has the correct
// content expression so blockToNode/nodeToBlock can round-trip tableContent data.
export const DataTable: () => (typeof _dataTableSpec extends () => infer S ? S : never) =
  () => {
    const spec = (_dataTableSpec as () => any)();
    spec.config = { ...spec.config, content: "table" };
    spec.implementation = { ...spec.implementation, node: _dataTablePMNode };
    return spec;
  };
