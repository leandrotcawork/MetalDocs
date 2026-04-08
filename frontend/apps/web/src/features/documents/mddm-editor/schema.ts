import { BlockNoteSchema, defaultBlockSpecs } from "@blocknote/core";
import { DataTable } from "./blocks/DataTable";
import { DataTableCell } from "./blocks/DataTableCell";
import { DataTableRow } from "./blocks/DataTableRow";
import { Field } from "./blocks/Field";
import { FieldGroup } from "./blocks/FieldGroup";
import { Repeatable } from "./blocks/Repeatable";
import { RepeatableItem } from "./blocks/RepeatableItem";
import { RichBlock } from "./blocks/RichBlock";
import { Section } from "./blocks/Section";

export const mddmSchema = BlockNoteSchema.create({
  blockSpecs: {
    ...defaultBlockSpecs,
    section: Section(),
    fieldGroup: FieldGroup(),
    field: Field(),
    repeatable: Repeatable(),
    repeatableItem: RepeatableItem(),
    dataTable: DataTable(),
    dataTableRow: DataTableRow(),
    dataTableCell: DataTableCell(),
    richBlock: RichBlock(),
  },
});

