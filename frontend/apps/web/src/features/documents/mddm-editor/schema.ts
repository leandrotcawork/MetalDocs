import { BlockNoteSchema, defaultBlockSpecs } from "@blocknote/core";
import { DataTable } from "./blocks/DataTable";
import { Repeatable } from "./blocks/Repeatable";
import { RepeatableItem } from "./blocks/RepeatableItem";
import { RichBlock } from "./blocks/RichBlock";
import { Section } from "./blocks/Section";

const {
  paragraph,
  heading,
  bulletListItem,
  numberedListItem,
  image,
  quote,
  divider,
  codeBlock,
  table,
} = defaultBlockSpecs;

export const mddmSchemaBlockSpecs = {
  section: Section(),
  repeatable: Repeatable(),
  repeatableItem: RepeatableItem(),
  dataTable: DataTable(),
  richBlock: RichBlock(),
};

export const mddmSchema = BlockNoteSchema.create({
  blockSpecs: {
    paragraph,
    heading,
    bulletListItem,
    numberedListItem,
    image,
    quote,
    divider,
    codeBlock,
    // The built-in `table` block is included solely to register the required
    // ProseMirror node types: tableRow, tableCell, tableHeader, tableParagraph.
    // BlockNote only registers these PM nodes (and the prosemirror-tables
    // plugins) when a block named "table" is present in blockSpecs. Our
    // `dataTable` block uses tableContent / tableRow+ content, so it depends
    // on those node types existing in the schema. The `table` block itself is
    // never exposed in any UI — it just seeds the PM schema.
    table,
    ...mddmSchemaBlockSpecs,
  },
});
