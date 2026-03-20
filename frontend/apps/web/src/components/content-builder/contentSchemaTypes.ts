export type SchemaSection = {
  key: string;
  title?: string;
  description?: string;
  fields?: SchemaField[];
};

export type SchemaField = {
  key: string;
  label?: string;
  type?: string;
  required?: boolean;
  options?: string[];
  itemType?: string;
  columns?: SchemaField[];
};

export type ChecklistItem = {
  label: string;
  checked: boolean;
};
