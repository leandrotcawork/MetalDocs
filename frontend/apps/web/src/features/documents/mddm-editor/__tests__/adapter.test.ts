import { describe, expect, it } from "vitest";
import { canonicalizeMDDM } from "../../../../../../../../shared/schemas/canonicalize.ts";
import {
  blockNoteToMDDM,
  mddmToBlockNote,
  type MDDMEnvelope,
} from "../adapter";

describe("MDDM ↔ BlockNote adapter", () => {
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

  it("keeps envelope metadata and all 17 block types canonically equivalent in a round-trip", () => {
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
          props: { title: "Section", color: "#6b1f2a", locked: true },
          children: [
            {
              id: "10000000-0000-0000-0000-000000000002",
              template_block_id: "20000000-0000-0000-0000-000000000002",
              type: "fieldGroup",
              props: { columns: 1, locked: true },
              children: [
                {
                  id: "10000000-0000-0000-0000-000000000003",
                  template_block_id: "20000000-0000-0000-0000-000000000003",
                  type: "field",
                  props: {
                    label: "Responsavel",
                    valueMode: "inline",
                    locked: true,
                  },
                  children: [
                    {
                      id: "10000000-0000-0000-0000-000000000004",
                      type: "paragraph",
                      props: {},
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
                            target_document_id:
                              "30000000-0000-0000-0000-000000000001",
                            target_revision_label: "v2",
                          },
                        },
                      ],
                    },
                  ],
                },
              ],
            },
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
                  props: { title: "Item 1" },
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
                columns: [
                  {
                    key: "item",
                    label: "Item",
                    type: "text",
                    required: false,
                  },
                ],
                locked: true,
                minRows: 0,
                maxRows: 3,
              },
              children: [
                {
                  id: "10000000-0000-0000-0000-000000000009",
                  type: "dataTableRow",
                  props: {},
                  children: [
                    {
                      id: "10000000-0000-0000-0000-000000000010",
                      type: "dataTableCell",
                      props: { columnKey: "item" },
                      children: [{ text: "valor" }],
                    },
                  ],
                },
              ],
            },
            {
              id: "10000000-0000-0000-0000-000000000011",
              template_block_id: "20000000-0000-0000-0000-000000000006",
              type: "richBlock",
              props: { label: "Livre", locked: true },
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
          type: "bulletListItem",
          props: { level: 0 },
          children: [{ text: "Bullet" }],
        },
        {
          id: "10000000-0000-0000-0000-000000000017",
          type: "numberedListItem",
          props: { level: 0 },
          children: [{ text: "Numbered" }],
        },
        {
          id: "10000000-0000-0000-0000-000000000018",
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
