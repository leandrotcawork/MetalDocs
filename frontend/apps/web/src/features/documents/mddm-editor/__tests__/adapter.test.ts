import { describe, expect, it } from "vitest";
import { canonicalizeMDDM } from "../../../../../../../../shared/schemas/canonicalize";
import {
  blockNoteToMDDM,
  mddmToBlockNote,
  type MDDMEnvelope,
} from "../adapter";

function parseFieldGroupTable(input: MDDMEnvelope) {
  return mddmToBlockNote(input)[0] as any;
}

describe("MDDM ↔ BlockNote adapter", () => {
  it("imports 1-column fieldGroup as a native table with label and value cells", () => {
    const table = parseFieldGroupTable({
      mddm_version: 1,
      template_ref: null,
      blocks: [
        {
          id: "10000000-0000-0000-0000-000000000001",
          template_block_id: "20000000-0000-0000-0000-000000000001",
          type: "fieldGroup",
          props: { columns: 1, locked: true },
          children: [
            {
              id: "10000000-0000-0000-0000-000000000002",
              template_block_id: "20000000-0000-0000-0000-000000000002",
              type: "field",
              props: {
                label: "Nome",
                valueMode: "inline",
                locked: true,
                hint: "hint: d.nome",
                layout: "grid",
              },
              children: [
                { text: "Joao" },
                { text: " Silva", marks: [{ type: "italic" }] },
              ],
            },
            {
              id: "10000000-0000-0000-0000-000000000003",
              type: "field",
              props: {
                label: "Cargo",
                valueMode: "inline",
                locked: false,
                layout: "stacked",
              },
              children: [],
            },
          ],
        },
      ],
    });

    expect(table.type).toBe("table");
    expect(table.content?.type).toBe("tableContent");
    expect(table.content?.headerCols).toBe(1);
    expect(table.content?.headerRows).toBe(0);
    expect(table.content?.columnWidths).toEqual([180, 540]);
    expect(table.content?.rows).toHaveLength(2);
    expect(table.content?.rows[0].cells[0]).toEqual({
      type: "tableCell",
      props: { backgroundColor: "gray" },
      content: [{ type: "text", text: "Nome", styles: { bold: true } }],
    });
    expect(table.content?.rows[0].cells[1]).toEqual([
      { type: "text", text: "Joao" },
      { type: "text", text: " Silva", styles: { italic: true } },
    ]);
    expect(table.content?.rows[1].cells[1]).toEqual([{ type: "text", text: "" }]);
  });

  it("imports 2-column fieldGroup as a native table with paired field rows", () => {
    const table = parseFieldGroupTable({
      mddm_version: 1,
      template_ref: null,
      blocks: [
        {
          id: "10000000-0000-0000-0000-000000000010",
          type: "fieldGroup",
          props: { columns: 2, locked: false },
          children: [
            {
              id: "10000000-0000-0000-0000-000000000011",
              type: "field",
              props: { label: "A", valueMode: "inline", locked: true, layout: "grid" },
              children: [{ text: "1" }],
            },
            {
              id: "10000000-0000-0000-0000-000000000012",
              type: "field",
              props: { label: "B", valueMode: "inline", locked: true, layout: "grid" },
              children: [{ text: "2" }],
            },
            {
              id: "10000000-0000-0000-0000-000000000013",
              type: "field",
              props: { label: "C", valueMode: "inline", locked: true, layout: "grid" },
              children: [{ text: "3" }],
            },
          ],
        },
      ],
    });

    expect(table.type).toBe("table");
    expect(table.content?.headerCols).toBe(1);
    expect(table.content?.columnWidths).toEqual([150, 210, 150, 210]);
    expect(table.content?.rows).toHaveLength(2);
    expect(table.content?.rows[0].cells).toEqual([
      {
        type: "tableCell",
        props: { backgroundColor: "gray" },
        content: [{ type: "text", text: "A", styles: { bold: true } }],
      },
      [{ type: "text", text: "1" }],
      {
        type: "tableCell",
        props: { backgroundColor: "gray" },
        content: [{ type: "text", text: "B", styles: { bold: true } }],
      },
      [{ type: "text", text: "2" }],
    ]);
    expect(table.content?.rows[1].cells).toEqual([
      {
        type: "tableCell",
        props: { backgroundColor: "gray" },
        content: [{ type: "text", text: "C", styles: { bold: true } }],
      },
      [{ type: "text", text: "3" }],
      [{ type: "text", text: "" }],
      [{ type: "text", text: "" }],
    ]);
  });

  it("stores fieldGroup round-trip metadata in table props", () => {
    const table = parseFieldGroupTable({
      mddm_version: 1,
      template_ref: null,
      blocks: [
        {
          id: "10000000-0000-0000-0000-000000000020",
          template_block_id: "20000000-0000-0000-0000-000000000020",
          type: "fieldGroup",
          props: { columns: 1, locked: true },
          children: [
            {
              id: "10000000-0000-0000-0000-000000000021",
              template_block_id: "20000000-0000-0000-0000-000000000021",
              type: "field",
              props: {
                label: "Responsavel",
                valueMode: "inline",
                locked: true,
                hint: "hint: d.owner",
                layout: "stacked",
              },
              children: [{ text: "Joao" }],
            },
          ],
        },
      ],
    });

    expect(JSON.parse(table.props.__mddm_field_group)).toEqual({
      id: "10000000-0000-0000-0000-000000000020",
      templateBlockId: "20000000-0000-0000-0000-000000000020",
      columns: 1,
      locked: true,
      fields: [
        {
          id: "10000000-0000-0000-0000-000000000021",
          templateBlockId: "20000000-0000-0000-0000-000000000021",
          label: "Responsavel",
          valueMode: "inline",
          locked: true,
          hint: "hint: d.owner",
          layout: "stacked",
        },
      ],
    });
  });

  it("round-trips 1-column fieldGroup tables back to fieldGroup blocks", () => {
    const input: MDDMEnvelope = {
      mddm_version: 1,
      template_ref: null,
      blocks: [
        {
          id: "10000000-0000-0000-0000-000000000030",
          template_block_id: "20000000-0000-0000-0000-000000000030",
          type: "fieldGroup",
          props: { columns: 1, locked: true },
          children: [
            {
              id: "10000000-0000-0000-0000-000000000031",
              template_block_id: "20000000-0000-0000-0000-000000000031",
              type: "field",
              props: {
                label: "Nome",
                valueMode: "inline",
                locked: true,
                hint: "hint: d.nome",
                layout: "grid",
              },
              children: [
                { text: "Joao", marks: [{ type: "bold" }] },
                { text: " Silva", marks: [{ type: "italic" }] },
              ],
            },
            {
              id: "10000000-0000-0000-0000-000000000032",
              template_block_id: "20000000-0000-0000-0000-000000000032",
              type: "field",
              props: {
                label: "Cargo",
                valueMode: "inline",
                locked: false,
                layout: "stacked",
              },
              children: [{ text: "Analista" }],
            },
          ],
        },
      ],
    };

    const output = blockNoteToMDDM(mddmToBlockNote(input));
    const fieldGroup = output.blocks[0] as any;

    expect(fieldGroup.type).toBe("fieldGroup");
    expect(fieldGroup.id).toBe("10000000-0000-0000-0000-000000000030");
    expect(fieldGroup.template_block_id).toBe("20000000-0000-0000-0000-000000000030");
    expect(fieldGroup.props.columns).toBe(1);
    expect(fieldGroup.props.locked).toBe(true);
    expect(fieldGroup.children).toHaveLength(2);

    expect(fieldGroup.children[0]).toMatchObject({
      id: "10000000-0000-0000-0000-000000000031",
      template_block_id: "20000000-0000-0000-0000-000000000031",
      type: "field",
      props: {
        label: "Nome",
        valueMode: "inline",
        locked: true,
        hint: "hint: d.nome",
        layout: "grid",
      },
    });
    expect(fieldGroup.children[0].children).toEqual([
      { text: "Joao", marks: [{ type: "bold" }] },
      { text: " Silva", marks: [{ type: "italic" }] },
    ]);

    expect(fieldGroup.children[1]).toMatchObject({
      id: "10000000-0000-0000-0000-000000000032",
      template_block_id: "20000000-0000-0000-0000-000000000032",
      type: "field",
      props: {
        label: "Cargo",
        valueMode: "inline",
        locked: false,
        layout: "stacked",
      },
    });
    expect(fieldGroup.children[1].children).toEqual([{ text: "Analista" }]);
  });

  it("round-trips 2-column fieldGroup tables back to fieldGroup blocks", () => {
    const input: MDDMEnvelope = {
      mddm_version: 1,
      template_ref: null,
      blocks: [
        {
          id: "10000000-0000-0000-0000-000000000040",
          template_block_id: "20000000-0000-0000-0000-000000000040",
          type: "fieldGroup",
          props: { columns: 2, locked: false },
          children: [
            {
              id: "10000000-0000-0000-0000-000000000041",
              template_block_id: "20000000-0000-0000-0000-000000000041",
              type: "field",
              props: { label: "A", valueMode: "inline", locked: true, layout: "grid" },
              children: [{ text: "1" }],
            },
            {
              id: "10000000-0000-0000-0000-000000000042",
              template_block_id: "20000000-0000-0000-0000-000000000042",
              type: "field",
              props: { label: "B", valueMode: "inline", locked: false, layout: "grid" },
              children: [{ text: "2", marks: [{ type: "italic" }] }],
            },
            {
              id: "10000000-0000-0000-0000-000000000043",
              template_block_id: "20000000-0000-0000-0000-000000000043",
              type: "field",
              props: { label: "C", valueMode: "inline", locked: true, layout: "stacked" },
              children: [{ text: "3" }],
            },
            {
              id: "10000000-0000-0000-0000-000000000044",
              template_block_id: "20000000-0000-0000-0000-000000000044",
              type: "field",
              props: {
                label: "D",
                valueMode: "inline",
                locked: true,
                hint: "hint: d",
                layout: "grid",
              },
              children: [{ text: "4", marks: [{ type: "bold" }] }],
            },
          ],
        },
      ],
    };

    const output = blockNoteToMDDM(mddmToBlockNote(input));
    const fieldGroup = output.blocks[0] as any;

    expect(fieldGroup.type).toBe("fieldGroup");
    expect(fieldGroup.id).toBe("10000000-0000-0000-0000-000000000040");
    expect(fieldGroup.template_block_id).toBe("20000000-0000-0000-0000-000000000040");
    expect(fieldGroup.props.columns).toBe(2);
    expect(fieldGroup.children).toHaveLength(4);
    expect(fieldGroup.children.map((field: any) => field.id)).toEqual([
      "10000000-0000-0000-0000-000000000041",
      "10000000-0000-0000-0000-000000000042",
      "10000000-0000-0000-0000-000000000043",
      "10000000-0000-0000-0000-000000000044",
    ]);
    expect(fieldGroup.children.map((field: any) => field.props.label)).toEqual([
      "A",
      "B",
      "C",
      "D",
    ]);
    expect(fieldGroup.children.map((field: any) => field.props.valueMode)).toEqual([
      "inline",
      "inline",
      "inline",
      "inline",
    ]);
    expect(fieldGroup.children.map((field: any) => field.props.locked)).toEqual([
      true,
      false,
      true,
      true,
    ]);
    expect(fieldGroup.children[1].children).toEqual([
      { text: "2", marks: [{ type: "italic" }] },
    ]);
    expect(fieldGroup.children[3].children).toEqual([
      { text: "4", marks: [{ type: "bold" }] },
    ]);
  });

  it("exports fieldGroup values correctly when label cells use full tableCell objects", () => {
    const output = blockNoteToMDDM([
      {
        id: "10000000-0000-0000-0000-000000000060",
        type: "table",
        props: {
          __mddm_field_group: JSON.stringify({
            id: "10000000-0000-0000-0000-000000000060",
            columns: 1,
            locked: true,
            fields: [
              {
                id: "10000000-0000-0000-0000-000000000061",
                label: "Objetivo",
                valueMode: "inline",
                locked: true,
                layout: "grid",
              },
            ],
          }),
        },
        content: {
          type: "tableContent",
          columnWidths: [180, 540],
          headerRows: 0,
          headerCols: 1,
          rows: [
            {
              cells: [
                {
                  type: "tableCell",
                  props: { backgroundColor: "gray" },
                  content: [{ type: "text", text: "Objetivo", styles: { bold: true } }],
                },
                [{ type: "text", text: "Definir o escopo" }],
              ],
            },
          ],
        },
        children: [],
      } as any,
    ]);

    expect((output.blocks[0] as any).children[0]).toMatchObject({
      type: "field",
      props: {
        label: "Objetivo",
        valueMode: "inline",
        locked: true,
        layout: "grid",
      },
      children: [{ text: "Definir o escopo" }],
    });
  });

  it("rejects regular native tables without fieldGroup metadata on export", () => {
    expect(() =>
      blockNoteToMDDM([
        {
          id: "10000000-0000-0000-0000-000000000050",
          type: "table",
          props: {},
          content: {
            type: "tableContent",
            columnWidths: [180, 540],
            headerRows: 0,
            headerCols: 1,
            rows: [
              {
                cells: [
                  [{ type: "text", text: "Nome", styles: { bold: true } }],
                  [{ type: "text", text: "Joao" }],
                ],
              },
            ],
          },
          children: [],
        } as any,
      ]),
    ).toThrow(/unsupported block type: table/i);
  });

  it("exports quote edits from BlockNote content (regression lock)", () => {
    const input: MDDMEnvelope = {
      mddm_version: 1,
      template_ref: null,
      blocks: [
        {
          id: "10000000-0000-0000-0000-000000000001",
          type: "quote",
          props: {},
          children: [
            {
              id: "10000000-0000-0000-0000-000000000002",
              type: "paragraph",
              props: {},
              children: [{ text: "OLD" }],
            },
          ],
        },
      ],
    };

    const blocks = mddmToBlockNote(input);

    // Simulate editing quote content inside BlockNote: export should reflect this edit.
    (blocks[0] as any).content = [{ type: "text", text: "NEW" }];

    const output = blockNoteToMDDM(blocks);
    expect(((output.blocks[0] as any).children[0] as any).children[0].text).toBe(
      "NEW",
    );
  });

  it("fails closed on unsupported BlockNote block types (regression lock)", () => {
    expect(() =>
      blockNoteToMDDM([
        {
          id: "10000000-0000-0000-0000-000000000001",
          type: "audio",
          props: {},
        } as any,
      ]),
    ).toThrow(/unsupported block type/i);
  });

  it("fails closed on unsupported MDDM block types on import", () => {
    expect(() =>
      mddmToBlockNote({
        mddm_version: 1,
        template_ref: null,
        blocks: [
          {
            id: "10000000-0000-0000-0000-000000000001",
            type: "audio",
            props: {},
          },
        ],
      } as any),
    ).toThrow(/unsupported block type/i);
  });

  it("fails closed when field valueMode is not inline", () => {
    expect(() =>
      blockNoteToMDDM([
        {
          id: "10000000-0000-0000-0000-000000000001",
          type: "field",
          props: { label: "X", valueMode: "dropdown", locked: true },
          content: [{ type: "text", text: "value" }],
        } as any,
      ]),
    ).toThrow(/valueMode/i);
  });

  it("defaults repeatable max constraints to schema defaults", () => {
    const output = blockNoteToMDDM([
      {
        id: "10000000-0000-0000-0000-000000000001",
        type: "repeatable",
        props: { label: "R", itemPrefix: "Item", locked: true },
        children: [],
      } as any,
    ]);

    expect((output.blocks[0] as any).props.minItems).toBe(0);
    expect((output.blocks[0] as any).props.maxItems).toBe(100);
  });

  it("serializes dataTable with tableContent and no minRows/maxRows/columns", () => {
    const tableContent = {
      type: "tableContent",
      columnWidths: [null],
      headerRows: 1,
      rows: [{ cells: [[{ type: "text", text: "Item" }]] }],
    };
    const output = blockNoteToMDDM([
      {
        id: "10000000-0000-0000-0000-000000000002",
        type: "dataTable",
        props: { label: "T", locked: true, density: "normal" },
        content: tableContent,
        children: [],
      } as any,
    ]);

    const dt = output.blocks[0] as any;
    expect(dt.props.label).toBe("T");
    expect(dt.props.locked).toBe(true);
    expect(dt.props.density).toBe("normal");
    expect(dt.props.minRows).toBeUndefined();
    expect(dt.props.maxRows).toBeUndefined();
    expect(dt.props.columns).toBeUndefined();
    expect(dt.content).toEqual(tableContent);
  });

  it("preserves id and template_block_id through round-trip", () => {
    const input: MDDMEnvelope = {
      mddm_version: 1,
      template_ref: null,
      blocks: [
        {
          id: "11111111-1111-1111-1111-111111111111",
          template_block_id: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
          type: "section",
          props: { title: "S1", color: "#000000", locked: true },
          children: [],
        },
      ],
    };

    const blockNoteForm = mddmToBlockNote(input);
    const mddmForm = blockNoteToMDDM(blockNoteForm);

    expect(mddmForm.blocks[0].id).toBe(
      "11111111-1111-1111-1111-111111111111",
    );
    expect(mddmForm.blocks[0].template_block_id).toBe(
      "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
    );
  });

  it("round-trips section optional and field hint props", () => {
    const input: MDDMEnvelope = {
      mddm_version: 1,
      template_ref: null,
      blocks: [
        {
          id: "11111111-1111-1111-1111-111111111111",
          type: "section",
          props: {
            title: "7 - Indicadores",
            color: "#6b1f2a",
            locked: true,
            optional: true,
          },
          children: [
            {
              id: "22222222-2222-2222-2222-222222222222",
              type: "field",
              props: {
                label: "Indicador",
                valueMode: "inline",
                locked: true,
                hint: "hint: d.kpis[i].indicador",
              },
              children: [{ text: "" }],
            },
          ],
        },
      ],
    };

    const output = blockNoteToMDDM(mddmToBlockNote(input));
    expect(output.blocks[0].props.optional).toBe(true);
    expect((output.blocks[0].children?.[0] as any).props.hint).toBe(
      "hint: d.kpis[i].indicador",
    );
  });

  it("migrates old dataTable (dataTableRow/Cell children) to tableContent on import", () => {
    const input: MDDMEnvelope = {
      mddm_version: 1,
      template_ref: null,
      blocks: [
        {
          id: "10000000-0000-0000-0000-000000000001",
          type: "dataTable",
          props: {
            label: "Lista",
            columns: [
              { key: "item", label: "Item" },
              { key: "qty", label: "Quantidade" },
              { key: "value", label: "Valor" },
            ],
            locked: true,
            minRows: 0,
            maxRows: 500,
            density: "normal",
          },
          children: [
            {
              id: "10000000-0000-0000-0000-000000000002",
              type: "dataTableRow",
              props: {},
              children: [
                {
                  id: "10000000-0000-0000-0000-000000000003",
                  type: "dataTableCell",
                  props: { columnKey: "item" },
                  children: [{ text: "Parafuso M8" }],
                },
                {
                  id: "10000000-0000-0000-0000-000000000004",
                  type: "dataTableCell",
                  props: { columnKey: "qty" },
                  children: [{ text: "100" }],
                },
                {
                  id: "10000000-0000-0000-0000-000000000005",
                  type: "dataTableCell",
                  props: { columnKey: "value" },
                  children: [{ text: "R$ 5,00" }],
                },
              ],
            },
          ],
        },
      ],
    };

    const blocks = mddmToBlockNote(input);
    const dt = blocks[0] as any;

    // Should have tableContent with header row + 1 data row
    expect(dt.content?.type).toBe("tableContent");
    expect(dt.content?.headerRows).toBe(1);
    expect(dt.content?.rows).toHaveLength(2);

    // Header row has 3 cells with column labels
    const headerRow = dt.content?.rows[0];
    expect(headerRow.cells).toHaveLength(3);
    expect(headerRow.cells[0][0].text).toBe("Item");
    expect(headerRow.cells[1][0].text).toBe("Quantidade");
    expect(headerRow.cells[2][0].text).toBe("Valor");

    // Data row has cell values
    const dataRow = dt.content?.rows[1];
    expect(dataRow.cells[0][0].text).toBe("Parafuso M8");
    expect(dataRow.cells[1][0].text).toBe("100");
    expect(dataRow.cells[2][0].text).toBe("R$ 5,00");

    // columnWidths has 3 nulls
    expect(dt.content?.columnWidths).toEqual([null, null, null]);
  });

  it("keeps envelope metadata and canonically round-trips blocks that still preserve their native types", () => {
    const input: MDDMEnvelope = {
      mddm_version: 1,
      template_ref: {
        template_key: "po-default",
        template_version: 2,
        template_content_hash:
          "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
      },
      blocks: [
        {
          id: "10000000-0000-0000-0000-000000000001",
          template_block_id: "20000000-0000-0000-0000-000000000001",
          type: "section",
          props: { title: "Section", color: "#6b1f2a", locked: true, variant: "bar" },
          children: [
            {
              id: "10000000-0000-0000-0000-000000000005",
              template_block_id: "20000000-0000-0000-0000-000000000004",
              type: "repeatable",
              props: {
                label: "Etapas",
                itemPrefix: "Etapa",
                locked: true,
                minItems: 1,
                maxItems: 5,
              },
              children: [
                {
                  id: "10000000-0000-0000-0000-000000000006",
                  type: "repeatableItem",
                  props: { title: "Item 1", style: "bordered" },
                  children: [
                    {
                      id: "10000000-0000-0000-0000-000000000007",
                      type: "heading",
                      props: { level: 2 },
                      children: [{ text: "Subtitulo" }],
                    },
                  ],
                },
              ],
            },
            {
              id: "10000000-0000-0000-0000-000000000008",
              template_block_id: "20000000-0000-0000-0000-000000000005",
              type: "dataTable",
              props: {
                label: "Tabela",
                locked: true,
                density: "normal",
              },
              content: {
                type: "tableContent",
                columnWidths: [null],
                headerRows: 1,
                rows: [
                  { cells: [[{ type: "text", text: "Item" }]] },
                  { cells: [[{ type: "text", text: "valor" }]] },
                ],
              },
              children: [],
            },
            {
              id: "10000000-0000-0000-0000-000000000011",
              template_block_id: "20000000-0000-0000-0000-000000000006",
              type: "richBlock",
              props: { label: "Livre", locked: true, chrome: "labeled" },
              children: [
                {
                  id: "10000000-0000-0000-0000-000000000012",
                  type: "quote",
                  props: {},
                  children: [
                    {
                      id: "10000000-0000-0000-0000-000000000013",
                      type: "paragraph",
                      props: {},
                      children: [{ text: "Citacao" }],
                    },
                  ],
                },
                {
                  id: "10000000-0000-0000-0000-000000000014",
                  type: "code",
                  props: { language: "go" },
                  children: [{ type: "text", text: "fmt.Println(\"ok\")" }],
                },
                {
                  id: "10000000-0000-0000-0000-000000000015",
                  type: "divider",
                  props: {},
                },
              ],
            },
          ],
        },
        {
          id: "10000000-0000-0000-0000-000000000016",
          template_block_id: "20000000-0000-0000-0000-000000000007",
          type: "field",
          props: {
            label: "Responsavel",
            valueMode: "inline",
            locked: true,
            layout: "grid",
          },
          children: [
            {
              text: "Joao",
              marks: [{ type: "bold" }],
            },
            {
              text: " Procedimento",
              link: {
                href: "https://example.com",
                title: "Exemplo",
              },
              document_ref: {
                target_document_id: "30000000-0000-0000-0000-000000000001",
                target_revision_label: "v2",
              },
            },
          ],
        },
        {
          id: "10000000-0000-0000-0000-000000000017",
          type: "bulletListItem",
          props: { level: 0 },
          children: [{ text: "Bullet" }],
        },
        {
          id: "10000000-0000-0000-0000-000000000018",
          type: "numberedListItem",
          props: { level: 0 },
          children: [{ text: "Numbered" }],
        },
        {
          id: "10000000-0000-0000-0000-000000000019",
          type: "image",
          props: {
            src: "/api/images/40000000-0000-0000-0000-000000000001",
            alt: "Diagrama",
            caption: "Legenda",
          },
        },
      ],
    };

    const blockNoteForm = mddmToBlockNote(input);
    const mddmForm = blockNoteToMDDM(blockNoteForm);

    expect(canonicalizeMDDM(mddmForm)).toEqual(canonicalizeMDDM(input));
  });
});
