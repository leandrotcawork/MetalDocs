import { BlockNoteSchema, defaultBlockSpecs } from "@blocknote/core";
import { DataTable } from "./blocks/DataTable";
import { Field } from "./blocks/Field";
import { FieldGroup } from "./blocks/FieldGroup";
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
} = defaultBlockSpecs;

export const mddmSchemaBlockSpecs = {
  section: Section(),
  fieldGroup: FieldGroup(),
  field: Field(),
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
    ...mddmSchemaBlockSpecs,
  },
});
