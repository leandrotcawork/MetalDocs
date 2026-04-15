export const MDDM_CLASSES = {
  section: 'mddm-section',
  sectionHeader: 'mddm-section__header',
  sectionBody: 'mddm-section__body',
  repeatable: 'mddm-repeatable',
  repeatableItem: 'mddm-repeatable__item',
  field: 'mddm-field',
  richBlock: 'mddm-rich-block',
  restrictedException: 'restricted-editing-exception',
} as const;

export const MDDM_DATA_ATTRS = {
  sectionId: 'data-section-id',
  sectionVariant: 'data-variant',
  repeatableId: 'data-repeatable-id',
  itemId: 'data-item-id',
  fieldId: 'data-field-id',
  fieldType: 'data-field-type',
  fieldLabel: 'data-field-label',
  fieldRequired: 'data-field-required',
  tableVariant: 'data-mddm-variant',
  schemaVersion: 'data-mddm-schema',
} as const;

export const MDDM_MODEL_ELEMENTS = {
  section: 'mddmSection',
  sectionHeader: 'mddmSectionHeader',
  sectionBody: 'mddmSectionBody',
  repeatable: 'mddmRepeatable',
  repeatableItem: 'mddmRepeatableItem',
  field: 'mddmField',
} as const;

export const SCHEMA_VERSION = 'v1';
