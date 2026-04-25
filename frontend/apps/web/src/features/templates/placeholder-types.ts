export type PlaceholderType = 'text' | 'date' | 'number' | 'select' | 'user' | 'picture' | 'computed';

export interface VisibilityCondition {
  placeholderID: string;
  operator: 'eq' | 'neq' | 'present' | 'absent';
  value?: string;
}

export interface Placeholder {
  id: string;
  label: string;
  type: PlaceholderType;
  required?: boolean;
  maxLength?: number;
  regex?: string;
  minNumber?: number;
  maxNumber?: number;
  minDate?: string;
  maxDate?: string;
  options?: string[];
  resolverKey?: string;
  visibleIf?: VisibilityCondition;
}

export interface ContentPolicy {
  allowTables: boolean;
  allowImages: boolean;
  allowHeadings: boolean;
  allowLists: boolean;
}

export interface SubBlockParam {
  name: string;
  type: 'string' | 'number' | 'boolean';
}

export interface SubBlockDef {
  key: string;
  label: string;
  params: SubBlockParam[];
}

export interface CompositionConfig {
  headerSubBlocks: string[];
  footerSubBlocks: string[];
  subBlockParams: Record<string, Record<string, string>>;
}
