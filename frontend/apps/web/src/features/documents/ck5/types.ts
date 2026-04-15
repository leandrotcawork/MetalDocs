export type SectionVariant = 'locked' | 'editable' | 'mixed';
export type TableVariant = 'fixed' | 'dynamic';
export type FieldType =
  | 'text'
  | 'date'
  | 'number'
  | `currency:${string}`
  | `select:${string}`
  | 'boolean';

export interface FieldDefinition {
  id: string;
  label: string;
  type: FieldType;
  required: boolean;
  defaultValue: string;
  group?: string;
}
