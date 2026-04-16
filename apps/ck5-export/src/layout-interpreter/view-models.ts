// ViewModel types produced by Layout Interpreters.
// React render() and DOCX emitters both consume these — never raw block props.

export type SectionViewModel = {
  number: string;
  title: string;
  optional: boolean;
  headerHeight: string;
  headerBg: string;
  headerColor: string;
  headerFontSize: string;
  headerFontWeight: string;
  locked: boolean;
  removable: boolean;
};

export type DataTableViewModel = {
  label: string;
  mode: "fixed" | "dynamic";
  headerBg: string;
  headerColor: string;
  headerFontWeight: string;
  cellBorderColor: string;
  cellPadding: string;
  density: "normal" | "compact";
  locked: boolean;
  removable: boolean;
  canAddRows: boolean;
  canRemoveRows: boolean;
  canAddColumns: boolean;
  canRemoveColumns: boolean;
  canResizeColumns: boolean;
  headerLocked: boolean;
  maxRows: number;
};

export type RepeatableViewModel = {
  label: string;
  itemPrefix: string;
  borderColor: string;
  itemAccentBorder: string;
  itemAccentWidth: string;
  locked: boolean;
  removable: boolean;
  canAddItems: boolean;
  canRemoveItems: boolean;
  maxItems: number;
  minItems: number;
  currentItemCount: number;
};

export type RepeatableItemViewModel = {
  title: string;
  number: string;
  accentBorderColor: string;
  accentBorderWidth: string;
  locked: boolean;
  removable: boolean;
};

export type RichBlockViewModel = {
  label: string;
  chrome: string;
  labelBackground: string;
  labelFontSize: string;
  labelColor: string;
  borderColor: string;
  locked: boolean;
  removable: boolean;
};
