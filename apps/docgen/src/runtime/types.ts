export type ScalarFieldType =
  | "text"
  | "textarea"
  | "number"
  | "date"
  | "select"
  | "checkbox";

export type FieldType = ScalarFieldType | "table" | "rich" | "repeat";

export interface ColumnDef {
  key: string;
  label: string;
  type: ScalarFieldType;
}

export interface ScalarFieldDef {
  key: string;
  label: string;
  type: ScalarFieldType;
}

export interface TableFieldDef {
  key: string;
  label: string;
  type: "table";
  columns: ColumnDef[];
}

export interface RichFieldDef {
  key: string;
  label: string;
  type: "rich";
}

export interface RepeatFieldDef {
  key: string;
  label: string;
  type: "repeat";
  itemFields: FieldDef[];
}

export type FieldDef = ScalarFieldDef | TableFieldDef | RichFieldDef | RepeatFieldDef;

export interface SectionDef {
  key: string;
  num: string;
  title: string;
  color?: string;
  fields: FieldDef[];
}

export interface DocumentTypeSchema {
  sections: SectionDef[];
}

export type RichTextRun = {
  text: string;
  bold?: boolean;
  italic?: boolean;
  underline?: boolean;
  color?: string;
};

export type RichBlock =
  | {
      type: "text";
      runs: RichTextRun[];
    }
  | {
      type: "image";
      data: string;
      mimeType?: string;
      altText?: string;
      width?: number;
      height?: number;
    }
  | {
      type: "table";
      rows: string[][];
      header?: boolean;
    }
  | {
      type: "list";
      ordered?: boolean;
      items: string[];
    };

export type SectionValues = Record<string, unknown>;
export type DocumentValues = Record<string, SectionValues>;

export interface DocumentMetadata {
  elaboradoPor: string;
  aprovadoPor: string;
  createdAt: string;
  approvedAt: string;
}

export interface DocumentRevision {
  versao: string;
  data: string;
  descricao: string;
  por: string;
}

export interface DocumentPayload {
  documentType: string;
  documentCode: string;
  title: string;
  version?: string;
  status?: string;
  schema: DocumentTypeSchema;
  values: DocumentValues;
  metadata?: DocumentMetadata;
  revisions?: DocumentRevision[];
}
