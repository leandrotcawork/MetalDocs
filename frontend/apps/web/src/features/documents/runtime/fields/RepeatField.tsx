import type { ReactNode } from "react";
import styles from "../DynamicEditor.module.css";
import type { RuntimeField, RuntimeRepeatField } from "../schemaRuntimeTypes";

type RepeatFieldProps = {
  field: RuntimeRepeatField;
  value: unknown;
  onChange?: (next: unknown) => void;
  renderField: (field: RuntimeField, value: unknown, onChange?: (next: unknown) => void) => ReactNode;
};

export function RepeatField({ field, value, onChange, renderField }: RepeatFieldProps) {
  const items = normalizeItems(value);
  const itemLabel = field.itemLabel ?? field.label ?? field.key;
  const singleValueField = field.itemFields.length === 1 && field.itemFields[0].kind === "scalar";

  return (
    <div className={styles.field}>
      <div className={styles.fieldLabel}>
        <span>{field.label ?? field.key}</span>
        {field.required && <span className={styles.requiredMark}>*</span>}
      </div>
      {field.description && <div className={styles.fieldDescription}>{field.description}</div>}
      <div className={styles.repeatShell}>
        {items.length === 0 ? (
          <div className={styles.repeatEmpty}>Nenhum item adicionado.</div>
        ) : (
          items.map((item, index) => (
            <div key={`${field.key}-${index}`} className={styles.repeatItem}>
              <div className={styles.repeatItemHead}>
                <span className={styles.repeatItemTitle}>
                  {itemLabel} {index + 1}
                </span>
                <button
                  type="button"
                  className={`${styles.repeatButton} ${styles.repeatButtonDanger}`}
                  onClick={() => onChange?.(items.filter((_, itemIndex) => itemIndex !== index).map(serializeRepeatItem))}
                >
                  Remover
                </button>
              </div>
              <div className={styles.repeatFields}>
                {field.itemFields.map((itemField) => {
                  const itemValue = getItemFieldValue(item, itemField.key, singleValueField);
                  return (
                    <div key={`${field.key}-${index}-${itemField.key}`} className={styles.repeatNested}>
                      {renderField(
                        itemField,
                        itemValue,
                        (nextValue) => {
                          const nextItems = items.slice();
                          nextItems[index] = setItemFieldValue(item, itemField.key, singleValueField, nextValue);
                          onChange?.(nextItems.map(serializeRepeatItem));
                        },
                      )}
                    </div>
                  );
                })}
              </div>
            </div>
          ))
        )}
        <div className={styles.repeatActions}>
          <button
            type="button"
            className={styles.repeatButton}
            onClick={() => onChange?.([...items, createBlankItem(field.itemFields, singleValueField)].map(serializeRepeatItem))}
          >
            Adicionar item
          </button>
        </div>
      </div>
    </div>
  );
}

function normalizeItems(value: unknown) {
  return Array.isArray(value) ? value : [];
}

function createBlankItem(itemFields: RuntimeRepeatField["itemFields"], singleValueField: boolean) {
  if (singleValueField && itemFields[0]) {
    return itemFields[0].kind === "scalar" && itemFields[0].input === "checkbox" ? false : "";
  }

  return itemFields.reduce<Record<string, unknown>>((acc, itemField) => {
    if (itemField.kind === "scalar" && itemField.input === "checkbox") {
      acc[itemField.key] = false;
      return acc;
    }
    acc[itemField.key] = "";
    return acc;
  }, {});
}

function getItemFieldValue(item: unknown, fieldKey: string, singleValueField: boolean) {
  if (singleValueField && !isRecordLike(item)) {
    return item;
  }
  if (!isRecordLike(item)) {
    if (fieldKey === "label") {
      return typeof item === "string" ? item : "";
    }
    if (fieldKey === "checked") {
      return typeof item === "boolean" ? item : false;
    }
  }
  if (isRecordLike(item)) {
    return item[fieldKey];
  }
  return undefined;
}

function setItemFieldValue(item: unknown, fieldKey: string, singleValueField: boolean, nextValue: unknown) {
  if (singleValueField && fieldKey === "value" && !isRecordLike(item)) {
    return nextValue;
  }
  if (!isRecordLike(item)) {
    if (fieldKey === "label") {
      return {
        label: nextValue,
        checked: false,
      };
    }
    if (fieldKey === "checked") {
      return {
        label: typeof item === "string" ? item : "",
        checked: nextValue,
      };
    }
  }
  const base = isRecordLike(item) ? item : {};
  return { ...base, [fieldKey]: nextValue };
}

function serializeRepeatItem(item: unknown) {
  return item;
}

function isRecordLike(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === "object" && !Array.isArray(value);
}
