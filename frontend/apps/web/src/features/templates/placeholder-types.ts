export type PlaceholderType = 'text' | 'date' | 'number' | 'select' | 'user' | 'picture' | 'computed';

export interface VisibilityCondition {
  placeholderID: string;
  operator: 'eq' | 'neq' | 'present' | 'absent';
  value?: string;
}

export interface Placeholder {
  id: string;
  name?: string;
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

/**
 * Derive a URL-safe token slug from a human label.
 * "Customer Name" -> "customer_name"
 */
export function slugifyLabel(label: string): string {
  const cleaned = label
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '_')
    .replace(/^_+|_+$/g, '')
    .slice(0, 50);
  if (!cleaned) return 'field';
  return /^[a-z]/.test(cleaned) ? cleaned : `f_${cleaned}`.slice(0, 50);
}
