export type SchemaRuntimeStatus = "idle" | "loading" | "saving" | "error";

export type SchemaScalarFieldType =
  | "text"
  | "textarea"
  | "number"
  | "date"
  | "select"
  | "checkbox"
  | "checklist";

export interface SchemaFieldBase {
  key: string;
  label: string;
  required?: boolean;
  description?: string;
  itemType?: string;
}

export interface SchemaScalarField extends SchemaFieldBase {
  type: SchemaScalarFieldType;
  options?: string[];
}

export interface SchemaTableField extends SchemaFieldBase {
  type: "table";
  columns: SchemaField[];
}

export interface SchemaRichField extends SchemaFieldBase {
  type: "rich";
}

export interface SchemaRepeatField extends SchemaFieldBase {
  type: "repeat";
  itemFields: SchemaField[];
}

export type SchemaField = SchemaScalarField | SchemaTableField | SchemaRichField | SchemaRepeatField;

export interface SchemaSection {
  key: string;
  num?: string;
  title?: string;
  color?: string;
  description?: string;
  fields: SchemaField[];
}

export interface DocumentTypeSchema {
  sections: SchemaSection[];
}

export interface ChecklistItem {
  label: string;
  checked: boolean;
}

export interface SchemaDocumentTypeBundleResponse {
  typeKey: string;
  schema: DocumentTypeSchema;
  name?: string;
  description?: string;
  activeVersion?: number | null;
}

export interface SchemaDocumentSnapshot {
  documentId: string;
  title: string;
  documentCode: string;
  documentProfile: string;
  documentType: string;
  status?: string;
}

export interface SchemaDocumentEditorBundleResponse {
  document: SchemaDocumentSnapshot;
  schema: DocumentTypeSchema;
  values: Record<string, unknown>;
  version: number | null;
  pdfUrl: string;
  typeKey: string;
}

export interface SchemaDocumentContentSaveResponse {
  documentId: string;
  version: number | null;
  pdfUrl: string;
  values: Record<string, unknown>;
}

export interface SchemaDocumentEditorState {
  documentId: string;
  typeKey: string;
  schema: DocumentTypeSchema | null;
  values: Record<string, unknown>;
  version: number | null;
  pdfUrl: string;
  status: SchemaRuntimeStatus;
  error: string;
  bundle: SchemaDocumentTypeBundleResponse | null;
  document: SchemaDocumentSnapshot | null;
}

export const emptyDocumentTypeSchema: DocumentTypeSchema = {
  sections: [],
};

export const emptySchemaDocumentEditorState: SchemaDocumentEditorState = {
  documentId: "",
  typeKey: "",
  schema: null,
  values: {},
  version: null,
  pdfUrl: "",
  status: "idle",
  error: "",
  bundle: null,
  document: null,
};
